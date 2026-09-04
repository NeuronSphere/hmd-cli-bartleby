package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// dockerAPI is the slice of the Docker client that running a container needs.
// Narrowing the dependency to five calls is what makes the run loop — exit
// codes, log streaming, name conflicts, cancellation — testable without a
// daemon. *client.Client satisfies it.
type dockerAPI interface {
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig,
		networkingConfig *network.NetworkingConfig, platform *ocispec.Platform,
		containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, container string, options container.StartOptions) error
	ContainerWait(ctx context.Context, container string,
		condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)
	ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error)
	ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error
}

// spec is one container run: what to start, and how to identify it.
type spec struct {
	Image  string
	Name   string
	Env    []string
	Cmd    []string
	Mounts []mount.Mount
	Stdout io.Writer
	Stderr io.Writer
}

// run creates, starts, streams, and removes a container, returning an error
// unless it exits 0.
//
// Removal happens through a context detached from ctx, so an interrupted build
// still cleans up after itself instead of leaving a container behind to collide
// with the next run.
func run(ctx context.Context, cli dockerAPI, s spec) error {
	if s.Stdout == nil {
		s.Stdout = os.Stdout
	}
	if s.Stderr == nil {
		s.Stderr = os.Stderr
	}

	id, err := create(ctx, cli, s)
	if err != nil {
		return err
	}

	defer func() {
		cleanupCtx := context.WithoutCancel(ctx)
		if err := cli.ContainerRemove(cleanupCtx, id, container.RemoveOptions{Force: true}); err != nil {
			fmt.Fprintf(s.Stderr, "warning: could not remove container %s: %v\n", displayName(s.Name, id), err)
		}
	}()

	// The wait is registered before the start so that a container which exits
	// immediately cannot do so before anyone is listening for its status.
	//
	// The condition has to be WaitConditionNextExit, not WaitConditionNotRunning:
	// a created-but-not-yet-started container is already "not running", so
	// NotRunning returns straight away reporting status 0 — which silently turns
	// every failed build into a successful one.
	statusCh, errCh := cli.ContainerWait(ctx, id, container.WaitConditionNextExit)

	if err := cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting container %s: %w", displayName(s.Name, id), err)
	}

	if err := streamLogs(ctx, cli, id, s); err != nil {
		return err
	}

	select {
	case err := <-errCh:
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("build interrupted: %w", ctxErr)
			}
			return fmt.Errorf("waiting for container %s: %w", displayName(s.Name, id), err)
		}
	case status := <-statusCh:
		if status.Error != nil {
			return fmt.Errorf("container %s failed: %s", displayName(s.Name, id), status.Error.Message)
		}
		if status.StatusCode != 0 {
			return fmt.Errorf("container %s exited with code %d", displayName(s.Name, id), status.StatusCode)
		}
	case <-ctx.Done():
		return fmt.Errorf("build interrupted: %w", ctx.Err())
	}

	return nil
}

// create makes the container, clearing a leftover container of the same name
// once before giving up. A name collision is expected after an interrupted run.
func create(ctx context.Context, cli dockerAPI, s spec) (string, error) {
	config := &container.Config{
		Image: s.Image,
		Env:   s.Env,
		Cmd:   s.Cmd,
	}
	hostConfig := &container.HostConfig{Mounts: s.Mounts}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, s.Name)
	if err == nil {
		return resp.ID, nil
	}

	if !isNameConflict(err) {
		return "", fmt.Errorf("creating container: %w", err)
	}

	fmt.Fprintf(s.Stderr, "note: removing leftover container %s\n", s.Name)
	if rmErr := cli.ContainerRemove(ctx, s.Name, container.RemoveOptions{Force: true}); rmErr != nil {
		return "", fmt.Errorf("removing leftover container %s: %w", s.Name, rmErr)
	}

	resp, err = cli.ContainerCreate(ctx, config, hostConfig, nil, nil, s.Name)
	if err != nil {
		return "", fmt.Errorf("creating container %s after removing the previous one: %w", s.Name, err)
	}
	return resp.ID, nil
}

// streamLogs copies the container's output to the caller's streams until it
// exits. A closed stream mid-build is reported but is not itself a build failure —
// the exit status decides that.
func streamLogs(ctx context.Context, cli dockerAPI, id string, s spec) error {
	logs, err := cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("attaching to container logs: %w", err)
	}
	defer logs.Close()

	if _, err := stdcopy.StdCopy(s.Stdout, s.Stderr, logs); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return nil
		}
		fmt.Fprintf(s.Stderr, "warning: container log stream ended early: %v\n", err)
	}
	return nil
}

// isNameConflict reports whether err is Docker's "name already in use" error.
func isNameConflict(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Conflict") || strings.Contains(msg, "already in use")
}

// displayName prefers the container name, falling back to a short id.
func displayName(name, id string) string {
	if name != "" {
		return name
	}
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
