package runner

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Requirements: REQ_EXEC_011
func TestNewClientHonoursDockerHost(t *testing.T) {
	const host = "tcp://docker.internal.test:2375"
	t.Setenv("DOCKER_HOST", host)
	t.Setenv("DOCKER_TLS_VERIFY", "")

	cli, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer cli.Close()

	if got := cli.DaemonHost(); got != host {
		t.Errorf("DaemonHost = %q, want %q", got, host)
	}
}

// Requirements: REQ_EXEC_011
func TestDetectHostFallsBackToKnownSockets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket paths")
	}

	// An empty PATH makes the docker CLI unavailable, so context discovery
	// fails and detection has to fall back to the known socket locations.
	t.Setenv("PATH", t.TempDir())

	home := t.TempDir()
	t.Setenv("HOME", home)

	socket := filepath.Join(home, ".colima", "default", "docker.sock")
	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if got := detectHost(); got != "unix://"+socket {
		t.Errorf("detectHost = %q, want unix://%s", got, socket)
	}
}

// Requirements: REQ_EXEC_011
func TestDetectHostWithNothingToFind(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	if got := detectHost(); got != "" {
		t.Errorf("detectHost = %q, want empty when there is nothing to detect", got)
	}
}

// TestNoDockerComposeInvocation guards the decision to drive the container
// through the Docker API. Compose existed only to pass pip credentials as a
// secret, which a bind-mounted file does; reintroducing it would put back the
// external binary dependency and the YAML file written into target/.
//
// Requirements: REQ_EXEC_001
func TestNoDockerComposeInvocation(t *testing.T) {
	// The module root, from this package's directory.
	root := filepath.Join("..", "..")

	// String literals, not prose: the package comments explain why compose is
	// gone, and saying so must not trip its own guard.
	banned := []string{`"docker-compose`, `"compose"`, `"compose.yaml"`, `"compose.yml"`}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "build" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		// Test files are exempt: this one has to name what it forbids.
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, needle := range banned {
			if strings.Contains(string(data), needle) {
				t.Errorf("%s contains the literal %s: transforms must run through the Docker API", path, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
}
