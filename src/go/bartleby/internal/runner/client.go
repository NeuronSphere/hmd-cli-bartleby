// Package runner drives the Bartleby transform container through the Docker
// Engine API.
//
// It talks to the API directly rather than shelling out to docker compose. The
// Python CLI needed compose to hand pip credentials to the container as a
// secret; a bind-mounted file does the same job, so the compose file, the YAML
// it was written into target/, and the docker-compose dependency are all gone.
package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
)

// NewClient connects to the Docker daemon.
//
// Standard DOCKER_* environment variables are always honoured, including the TLS
// settings a remote daemon needs. When DOCKER_HOST is not set, the active docker
// context is consulted so that Colima, Rancher Desktop, and Podman sockets work
// without the user exporting anything.
func NewClient() (*client.Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if os.Getenv("DOCKER_HOST") == "" {
		if host := detectHost(); host != "" {
			opts = append(opts, client.WithHost(host))
		}
	}

	return client.NewClientWithOpts(opts...)
}

// detectHost finds a daemon endpoint when the environment does not name one:
// first from the active docker context, then from the sockets the common macOS
// and Linux runtimes create.
func detectHost() string {
	if host := contextHost(); host != "" {
		return host
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	candidates := []string{
		filepath.Join(home, ".colima", "default", "docker.sock"),
		filepath.Join(home, ".docker", "run", "docker.sock"),
		filepath.Join(home, ".rd", "docker.sock"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return "unix://" + path
		}
	}

	return ""
}

// contextHost asks the docker CLI for the active context's endpoint. The CLI is
// optional: when it is missing or errors, detection falls through to the known
// socket locations.
func contextHost() string {
	out, err := exec.Command("docker", "context", "inspect",
		"--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
