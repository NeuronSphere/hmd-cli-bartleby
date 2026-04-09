package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// newDockerClient creates a Docker client, respecting DOCKER_HOST if set and
// falling back to the active Docker context endpoint (e.g. Colima) if not.
func newDockerClient() (*client.Client, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}

	if host := os.Getenv("DOCKER_HOST"); host != "" {
		opts = append(opts, client.WithHost(host))
	} else if host := dockerContextHost(); host != "" {
		opts = append(opts, client.WithHost(host))
	} else {
		opts = append(opts, client.FromEnv)
	}

	return client.NewClientWithOpts(opts...)
}

// dockerContextHost returns the Docker host from the active docker context,
// or empty string if it cannot be determined.
func dockerContextHost() string {
	out, err := exec.Command("docker", "context", "inspect",
		"--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// TransformInstanceContext matches the JSON structure the container expects.
type TransformInstanceContext struct {
	Name    string                 `json:"name"`
	Shell   string                 `json:"shell"`
	RootDoc string                 `json:"root_doc"`
	Config  map[string]interface{} `json:"config"`
}

// TransformConfig holds all parameters needed to run the Bartleby transform container.
type TransformConfig struct {
	ImageName      string
	InstanceName   string
	Context        TransformInstanceContext
	Environment    string
	Region         string
	Account        string
	CustomerCode   string
	DeploymentID   string
	Autodoc        bool
	DocRepo        string
	DocRepoVersion string
	InputPath      string
	OutputPath     string
	// PipConfigPath is the host path to a pip.conf file; used when Autodoc is true.
	// It will be bind-mounted to /run/secrets/pip_url inside the container.
	PipConfigPath     string
	DocumentTitle     string
	NoTimestampTitle  bool
	Confidential      bool
	ConfidentialityStatement string
	DefaultLogo       string
	HTMLDefaultLogo   string
	PDFDefaultLogo    string
	// GlobalStylesPath is the host path to $HMD_HOME/bartleby/styles, mounted read-only.
	GlobalStylesPath  string
}

// PumlConfig holds parameters for PlantUML image generation.
type PumlConfig struct {
	ImageName  string
	InputPath  string
	OutputPath string
	Files      []string
}

// RunTransform creates and runs the Bartleby transform container using the Docker API.
func RunTransform(cfg TransformConfig) error {
	ctx := context.Background()

	cli, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	contextJSON, err := json.Marshal(cfg.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal transform context: %w", err)
	}

	env := []string{
		fmt.Sprintf("TRANSFORM_INSTANCE_CONTEXT=%s", contextJSON),
		fmt.Sprintf("BARTLEBY_SHELL=%s", cfg.Context.Shell),
		fmt.Sprintf("HMD_ENVIRONMENT=%s", cfg.Environment),
		fmt.Sprintf("HMD_REGION=%s", cfg.Region),
		fmt.Sprintf("HMD_ACCOUNT=%s", cfg.Account),
		fmt.Sprintf("HMD_CUSTOMER_CODE=%s", cfg.CustomerCode),
		fmt.Sprintf("HMD_DID=%s", cfg.DeploymentID),
		fmt.Sprintf("AUTODOC=%v", cfg.Autodoc),
		fmt.Sprintf("HMD_DOC_REPO_NAME=%s", cfg.DocRepo),
		fmt.Sprintf("HMD_DOC_REPO_VERSION=%s", cfg.DocRepoVersion),
	}

	if v := os.Getenv("HMD_DOC_COMPANY_NAME"); v != "" {
		env = append(env, fmt.Sprintf("HMD_DOC_COMPANY_NAME=%s", v))
	}
	if cfg.DocumentTitle != "" {
		env = append(env, fmt.Sprintf("DOCUMENT_TITLE=%s", cfg.DocumentTitle))
	}
	if cfg.NoTimestampTitle {
		env = append(env, "NO_TIMESTAMP_TITLE=true")
	}
	if cfg.Confidential && cfg.ConfidentialityStatement != "" {
		env = append(env, fmt.Sprintf("CONFIDENTIALITY_STATEMENT=%s", cfg.ConfidentialityStatement))
	}
	if cfg.DefaultLogo != "" {
		env = append(env, fmt.Sprintf("DEFAULT_LOGO=%s", cfg.DefaultLogo))
	}
	if cfg.HTMLDefaultLogo != "" {
		env = append(env, fmt.Sprintf("HTML_DEFAULT_LOGO=%s", cfg.HTMLDefaultLogo))
	}
	if cfg.PDFDefaultLogo != "" {
		env = append(env, fmt.Sprintf("PDF_DEFAULT_LOGO=%s", cfg.PDFDefaultLogo))
	}
	if cfg.Autodoc && cfg.PipConfigPath != "" {
		env = append(env, "PIP_CONF=/run/secrets/pip_url")
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: cfg.InputPath,
			Target: "/hmd_transform/input",
		},
		{
			Type:   mount.TypeBind,
			Source: cfg.OutputPath,
			Target: "/hmd_transform/output",
		},
	}

	if cfg.GlobalStylesPath != "" {
		if _, err := os.Stat(cfg.GlobalStylesPath); err == nil {
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   cfg.GlobalStylesPath,
				Target:   "/hmd_transform/global_styles",
				ReadOnly: true,
			})
		}
	}

	if cfg.Autodoc && cfg.PipConfigPath != "" {
		if _, err := os.Stat(cfg.PipConfigPath); err == nil {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: cfg.PipConfigPath,
				Target: "/run/secrets/pip_url",
			})
		}
	}

	containerName := fmt.Sprintf("bartleby-inst_%s_%s", cfg.InstanceName, cfg.Context.Shell)

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.ImageName,
			Env:   env,
		},
		&container.HostConfig{
			Mounts:     mounts,
			AutoRemove: false,
		},
		nil, nil, containerName,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict") || strings.Contains(err.Error(), "already in use") {
			fmt.Printf("Container %s already exists — removing and retrying...\n", containerName)
			if rmErr := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); rmErr != nil {
				return fmt.Errorf("failed to remove conflicting container %s: %w", containerName, rmErr)
			}
			resp, err = cli.ContainerCreate(ctx,
				&container.Config{Image: cfg.ImageName, Env: env},
				&container.HostConfig{Mounts: mounts, AutoRemove: false},
				nil, nil, containerName,
			)
		}
		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}
	}

	defer func() {
		if err := cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove container %s: %v\n", containerName, err)
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Register wait before streaming logs to avoid a race.
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	logReader, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach container logs: %w", err)
	}
	defer logReader.Close()

	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, logReader); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "warning: log copy error: %v\n", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		if status.Error != nil {
			return fmt.Errorf("container error: %s", status.Error.Message)
		}
		if status.StatusCode != 0 {
			return fmt.Errorf("container exited with code %d", status.StatusCode)
		}
	}

	return nil
}

// RunPuml runs the Bartleby container with the PlantUML entrypoint.
func RunPuml(cfg PumlConfig) error {
	ctx := context.Background()

	cli, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	filesCSV := ""
	for i, f := range cfg.Files {
		if i > 0 {
			filesCSV += ","
		}
		filesCSV += f
	}

	env := []string{
		fmt.Sprintf("PUML_FILES=%s", filesCSV),
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: cfg.InputPath,
			Target: "/hmd_transform/input",
		},
		{
			Type:   mount.TypeBind,
			Source: cfg.OutputPath,
			Target: "/hmd_transform/output",
		},
	}

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.ImageName,
			Env:   env,
			Cmd:   []string{"python", "entry_puml.py"},
		},
		&container.HostConfig{
			Mounts:     mounts,
			AutoRemove: false,
		},
		nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("failed to create puml container: %w", err)
	}

	defer func() {
		cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start puml container: %w", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	logReader, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach puml container logs: %w", err)
	}
	defer logReader.Close()

	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, logReader); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "warning: log copy error: %v\n", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for puml container: %w", err)
		}
	case status := <-statusCh:
		if status.Error != nil {
			return fmt.Errorf("puml container error: %s", status.Error.Message)
		}
		if status.StatusCode != 0 {
			return fmt.Errorf("puml container exited with code %d", status.StatusCode)
		}
	}

	return nil
}

// PullImage pulls the specified Docker image, printing progress to stdout.
func PullImage(imageName string) error {
	ctx := context.Background()

	cli, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	fmt.Printf("Pulling %s...\n", imageName)
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	io.Copy(os.Stdout, reader)
	return nil
}

// RemoveImage removes the specified Docker image.
func RemoveImage(imageName string) error {
	ctx := context.Background()

	cli, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	fmt.Printf("Removing %s...\n", imageName)
	_, err = cli.ImageRemove(ctx, imageName, image.RemoveOptions{Force: false, PruneChildren: false})
	if err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}
	return nil
}

// PipConfigPath returns the host path to the user's pip configuration file.
func DefaultPipConfigPath() string {
	if os.Getenv("PIP_USERNAME") != "" && os.Getenv("PIP_PASSWORD") != "" {
		return "" // caller should generate a temp pip.conf
	}
	home, _ := os.UserHomeDir()
	if os.PathSeparator == '\\' {
		return filepath.Join(home, "pip", "pip.ini")
	}
	return filepath.Join(home, ".pip", "pip.conf")
}
