package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/mount"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/sanitize"
)

// Container paths the transform image reads and writes.
const (
	inputMount        = "/hmd_transform/input"
	outputMount       = "/hmd_transform/output"
	globalStylesMount = "/hmd_transform/global_styles"
	pipSecretMount    = "/run/secrets/pip_url"
)

// TransformInstanceContext is the JSON document the container parses out of
// TRANSFORM_INSTANCE_CONTEXT to learn what it is building.
type TransformInstanceContext struct {
	Name    string         `json:"name"`
	Shell   string         `json:"shell"`
	RootDoc string         `json:"root_doc"`
	Config  map[string]any `json:"config"`
}

// TransformConfig is everything one transform run needs.
type TransformConfig struct {
	ImageName    string
	InstanceName string
	Context      TransformInstanceContext

	Environment  string
	Region       string
	Account      string
	CustomerCode string
	DeploymentID string
	CompanyName  string

	DocRepo        string
	DocRepoVersion string

	InputPath  string
	OutputPath string

	Autodoc bool
	// PipConfigPath is a host pip.conf bind-mounted at /run/secrets/pip_url for
	// autodoc builds that install from a private index.
	PipConfigPath string

	DocumentTitle            string
	NoTimestampTitle         bool
	ConfidentialityStatement string

	DefaultLogo     string
	HTMLDefaultLogo string
	PDFDefaultLogo  string

	// GlobalStylesPath is $HMD_HOME/bartleby/styles, mounted read-only when it
	// exists.
	GlobalStylesPath string
}

// Env builds the container environment.
//
// AUTODOC is deliberately "True"/"False" and not Go's "true"/"false": the image
// compares it against the Python repr of a bool ("True"), so a lower-cased value
// disables autodoc silently. Optional values are omitted entirely rather than
// passed as empty strings, because the image treats presence as meaningful.
func (c TransformConfig) Env() ([]string, error) {
	contextJSON, err := json.Marshal(c.Context)
	if err != nil {
		return nil, fmt.Errorf("encoding transform context: %w", err)
	}

	env := []string{
		"TRANSFORM_INSTANCE_CONTEXT=" + string(contextJSON),
		"BARTLEBY_SHELL=" + c.Context.Shell,
		"HMD_ENVIRONMENT=" + c.Environment,
		"HMD_REGION=" + c.Region,
		"HMD_ACCOUNT=" + c.Account,
		"HMD_CUSTOMER_CODE=" + c.CustomerCode,
		"HMD_DID=" + c.DeploymentID,
		"AUTODOC=" + pythonBool(c.Autodoc),
		"HMD_DOC_REPO_NAME=" + c.DocRepo,
		"HMD_DOC_REPO_VERSION=" + c.DocRepoVersion,
	}

	optional := []struct {
		key   string
		value string
	}{
		{"HMD_DOC_COMPANY_NAME", c.CompanyName},
		{"DOCUMENT_TITLE", c.DocumentTitle},
		{"CONFIDENTIALITY_STATEMENT", c.ConfidentialityStatement},
		{"DEFAULT_LOGO", c.DefaultLogo},
		{"HTML_DEFAULT_LOGO", c.HTMLDefaultLogo},
		{"PDF_DEFAULT_LOGO", c.PDFDefaultLogo},
	}
	for _, o := range optional {
		if o.value != "" {
			env = append(env, o.key+"="+o.value)
		}
	}

	// The image only checks whether NO_TIMESTAMP_TITLE is set, so it must be
	// absent — not "false" — when timestamps are wanted.
	if c.NoTimestampTitle {
		env = append(env, "NO_TIMESTAMP_TITLE=true")
	}

	if c.Autodoc && c.PipConfigPath != "" {
		env = append(env, "PIP_CONF="+pipSecretMount)
	}

	return env, nil
}

// Mounts builds the bind mounts for a transform run. Optional mounts are
// included only when their host path exists, so a missing styles directory or
// pip config is not a container-create failure.
func (c TransformConfig) Mounts() []mount.Mount {
	mounts := []mount.Mount{
		{Type: mount.TypeBind, Source: c.InputPath, Target: inputMount},
		{Type: mount.TypeBind, Source: c.OutputPath, Target: outputMount},
	}

	if c.GlobalStylesPath != "" && isDir(c.GlobalStylesPath) {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   c.GlobalStylesPath,
			Target:   globalStylesMount,
			ReadOnly: true,
		})
	}

	if c.Autodoc && c.PipConfigPath != "" && exists(c.PipConfigPath) {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   c.PipConfigPath,
			Target:   pipSecretMount,
			ReadOnly: true,
		})
	}

	return mounts
}

// ContainerName returns the container name for a run, sanitized to Docker's
// character rules. A repo or instance name with spaces or punctuation would
// otherwise fail container creation with an opaque API error.
func (c TransformConfig) ContainerName() string {
	instance := sanitize.ContainerName(c.InstanceName)
	if instance == "" {
		instance = "repo"
	}
	shell := sanitize.ContainerName(c.Context.Shell)
	if shell == "" {
		return "bartleby-inst_" + instance
	}
	return fmt.Sprintf("bartleby-inst_%s_%s", instance, shell)
}

// RunTransform renders one document with one builder.
func RunTransform(ctx context.Context, cfg TransformConfig) error {
	env, err := cfg.Env()
	if err != nil {
		return err
	}

	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer cli.Close()

	return run(ctx, cli, spec{
		Image:  cfg.ImageName,
		Name:   cfg.ContainerName(),
		Env:    env,
		Mounts: cfg.Mounts(),
	})
}

// pythonBool renders a bool the way Python's str() does, which is what the
// transform image compares against.
func pythonBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
