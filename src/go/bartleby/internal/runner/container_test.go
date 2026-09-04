package runner

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// fakeDocker is a Docker API client that records what it was asked to do and
// answers from scripted values. Only the five calls a container run makes are
// implemented; the embedded interface is nil, so any other call panics loudly
// rather than passing silently.
type fakeDocker struct {
	dockerAPI

	mu sync.Mutex

	// scripted behaviour
	createErrs   []error // consumed one per ContainerCreate call
	exitCode     int64
	waitErr      error
	stdout       string
	stderr       string
	logsErr      error
	removeErr    error
	blockUntil   chan struct{} // when set, the wait blocks until closed
	waitConditio container.WaitCondition

	// recorded calls
	creates   int
	starts    int
	removed   []string
	createdAs []string
}

func (f *fakeDocker) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig,
	_ *network.NetworkingConfig, _ *ocispec.Platform, name string) (container.CreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.creates++
	f.createdAs = append(f.createdAs, name)

	if len(f.createErrs) > 0 {
		err := f.createErrs[0]
		f.createErrs = f.createErrs[1:]
		if err != nil {
			return container.CreateResponse{}, err
		}
	}
	return container.CreateResponse{ID: "container-id"}, nil
}

func (f *fakeDocker) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	return nil
}

func (f *fakeDocker) ContainerWait(ctx context.Context, _ string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	f.mu.Lock()
	f.waitConditio = condition
	block := f.blockUntil
	exit := f.exitCode
	waitErr := f.waitErr
	f.mu.Unlock()

	statusCh := make(chan container.WaitResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		if block != nil {
			select {
			case <-block:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
		if waitErr != nil {
			errCh <- waitErr
			return
		}
		statusCh <- container.WaitResponse{StatusCode: exit}
	}()

	return statusCh, errCh
}

func (f *fakeDocker) ContainerLogs(_ context.Context, _ string, _ container.LogsOptions) (io.ReadCloser, error) {
	if f.logsErr != nil {
		return nil, f.logsErr
	}

	// The daemon multiplexes stdout and stderr; stdcopy demultiplexes them.
	var buf strings.Builder
	writer := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
	_, _ = writer.Write([]byte(f.stdout))
	errWriter := stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
	_, _ = errWriter.Write([]byte(f.stderr))

	return io.NopCloser(strings.NewReader(buf.String())), nil
}

func (f *fakeDocker) ContainerRemove(_ context.Context, name string, _ container.RemoveOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, name)
	return f.removeErr
}

func (f *fakeDocker) Close() error { return nil }

func testSpec(stdout, stderr *strings.Builder) spec {
	return spec{
		Image:  "ghcr.io/neuronsphere/hmd-tf-bartleby:stable",
		Name:   "bartleby-inst_repo_html",
		Env:    []string{"BARTLEBY_SHELL=html"},
		Stdout: stdout,
		Stderr: stderr,
	}
}

// A container that exits non-zero is a failed build. This is the regression test
// for waiting on WaitConditionNotRunning, which returns immediately for a
// created-but-unstarted container and reported every failure as a success.
//
// Requirements: REQ_EXEC_009
func TestRunReportsNonZeroExit(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{exitCode: 2}

	err := run(context.Background(), fake, testSpec(&out, &errOut))
	if err == nil {
		t.Fatal("a container exiting 2 must be an error")
	}
	if !strings.Contains(err.Error(), "exited with code 2") {
		t.Errorf("error should name the exit code, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bartleby-inst_repo_html") {
		t.Errorf("error should name the container, got: %v", err)
	}
}

// Requirements: REQ_EXEC_009
func TestRunWaitsForTheNextExit(t *testing.T) {
	fake := &fakeDocker{}
	var out, errOut strings.Builder

	if err := run(context.Background(), fake, testSpec(&out, &errOut)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if fake.waitConditio != container.WaitConditionNextExit {
		t.Errorf("wait condition = %q, want %q — %q returns immediately for a container that has not started yet",
			fake.waitConditio, container.WaitConditionNextExit, container.WaitConditionNotRunning)
	}
	if fake.starts != 1 {
		t.Errorf("container started %d times, want 1", fake.starts)
	}
}

// Requirements: REQ_EXEC_009
func TestRunReportsAWaitFailure(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{waitErr: errors.New("daemon went away")}

	err := run(context.Background(), fake, testSpec(&out, &errOut))
	if err == nil || !strings.Contains(err.Error(), "daemon went away") {
		t.Errorf("err = %v, want the daemon's failure", err)
	}
}

// Requirements: REQ_EXEC_008
func TestRunStreamsContainerOutput(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{
		stdout: "Copying generated docs..\nTransform complete.\n",
		stderr: "WARNING: something\n",
	}

	if err := run(context.Background(), fake, testSpec(&out, &errOut)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if !strings.Contains(out.String(), "Transform complete.") {
		t.Errorf("container stdout not streamed: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "WARNING: something") {
		t.Errorf("container stderr not streamed: %q", errOut.String())
	}
}

// Requirements: REQ_EXEC_007
func TestRunRemovesALeftoverContainerAndRetries(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{
		createErrs: []error{errors.New(`Error response from daemon: Conflict. The container name "/bartleby-inst_repo_html" is already in use`)},
	}

	if err := run(context.Background(), fake, testSpec(&out, &errOut)); err != nil {
		t.Fatalf("run: %v", err)
	}

	if fake.creates != 2 {
		t.Errorf("ContainerCreate called %d times, want 2 (the retry)", fake.creates)
	}
	if len(fake.removed) == 0 || fake.removed[0] != "bartleby-inst_repo_html" {
		t.Errorf("the leftover container should be removed by name, removals: %v", fake.removed)
	}
	if !strings.Contains(errOut.String(), "leftover") {
		t.Errorf("the user should be told it happened: %q", errOut.String())
	}
}

// Requirements: REQ_EXEC_007
func TestRunGivesUpOnAnUnrelatedCreateFailure(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{createErrs: []error{errors.New("no such image")}}

	err := run(context.Background(), fake, testSpec(&out, &errOut))
	if err == nil || !strings.Contains(err.Error(), "no such image") {
		t.Fatalf("err = %v, want the create failure", err)
	}
	if fake.creates != 1 {
		t.Errorf("ContainerCreate called %d times, want 1 — only a name conflict is retried", fake.creates)
	}
}

// An interrupt cancels the build, and the container is still removed: cleanup
// runs on a context detached from the cancelled one.
//
// Requirements: REQ_EXEC_010
func TestRunRemovesTheContainerWhenCancelled(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{blockUntil: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- run(ctx, fake, testSpec(&out, &errOut)) }()

	cancel()

	err := <-done
	if err == nil {
		t.Fatal("a cancelled build must report an error")
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Errorf("err = %v, want an interruption", err)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.removed) == 0 {
		t.Error("the container must still be removed after cancellation")
	}
}

// Requirements: REQ_EXEC_009
func TestRunReportsALogAttachFailure(t *testing.T) {
	var out, errOut strings.Builder
	fake := &fakeDocker{logsErr: errors.New("cannot attach")}

	err := run(context.Background(), fake, testSpec(&out, &errOut))
	if err == nil || !strings.Contains(err.Error(), "cannot attach") {
		t.Errorf("err = %v, want the attach failure", err)
	}
}
