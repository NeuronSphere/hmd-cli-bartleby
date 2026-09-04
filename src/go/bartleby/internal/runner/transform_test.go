package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/mount"
)

func baseConfig(t *testing.T) TransformConfig {
	t.Helper()
	dir := t.TempDir()
	return TransformConfig{
		ImageName:    "ghcr.io/neuronsphere/hmd-tf-bartleby:stable",
		InstanceName: "hmd-docs-example",
		Context: TransformInstanceContext{
			Name:    "guide",
			Shell:   "html",
			RootDoc: "guide",
			Config:  map[string]any{"theme": "furo"},
		},
		Environment:    "local",
		Region:         "reg1",
		CustomerCode:   "hmd",
		DeploymentID:   "aaa",
		DocRepo:        "hmd-docs-example",
		DocRepoVersion: "1.4",
		InputPath:      dir,
		OutputPath:     filepath.Join(dir, "target", "bartleby"),
	}
}

func envMap(t *testing.T, cfg TransformConfig) map[string]string {
	t.Helper()
	env, err := cfg.Env()
	if err != nil {
		t.Fatalf("Env: %v", err)
	}

	out := make(map[string]string, len(env))
	for _, kv := range env {
		key, value, _ := strings.Cut(kv, "=")
		out[key] = value
	}
	return out
}

// The transform image compares AUTODOC against Python's "True", so Go's
// lower-case "true" silently disabled autodoc.
// Requirements: REQ_EXEC_003
func TestEnvAutodocMatchesPythonBool(t *testing.T) {
	cfg := baseConfig(t)

	cfg.Autodoc = true
	if got := envMap(t, cfg)["AUTODOC"]; got != "True" {
		t.Errorf("AUTODOC = %q, want True", got)
	}

	cfg.Autodoc = false
	if got := envMap(t, cfg)["AUTODOC"]; got != "False" {
		t.Errorf("AUTODOC = %q, want False", got)
	}
}

// Requirements: REQ_EXEC_002
func TestEnvTransformContextIsValidJSON(t *testing.T) {
	env := envMap(t, baseConfig(t))

	var ctx TransformInstanceContext
	if err := json.Unmarshal([]byte(env["TRANSFORM_INSTANCE_CONTEXT"]), &ctx); err != nil {
		t.Fatalf("context is not valid JSON: %v", err)
	}
	if ctx.Name != "guide" || ctx.Shell != "html" || ctx.RootDoc != "guide" {
		t.Errorf("context = %+v", ctx)
	}
	if ctx.Config["theme"] != "furo" {
		t.Errorf("builder config lost: %+v", ctx.Config)
	}
	if env["BARTLEBY_SHELL"] != "html" {
		t.Errorf("BARTLEBY_SHELL = %q, want html", env["BARTLEBY_SHELL"])
	}
}

// The image treats these as "set or not", so empty values must be omitted rather
// than passed as empty strings.
// Requirements: REQ_EXEC_004
func TestEnvOmitsEmptyOptionalValues(t *testing.T) {
	env := envMap(t, baseConfig(t))

	for _, key := range []string{
		"HMD_DOC_COMPANY_NAME", "DOCUMENT_TITLE", "CONFIDENTIALITY_STATEMENT",
		"DEFAULT_LOGO", "HTML_DEFAULT_LOGO", "PDF_DEFAULT_LOGO",
		"NO_TIMESTAMP_TITLE", "PIP_CONF",
	} {
		if _, present := env[key]; present {
			t.Errorf("%s should be absent when unset", key)
		}
	}
}

// Requirements: REQ_EXEC_004
func TestEnvIncludesOptionalValuesWhenSet(t *testing.T) {
	cfg := baseConfig(t)
	cfg.CompanyName = "NeuronSphere"
	cfg.DocumentTitle = "example-1.4"
	cfg.ConfidentialityStatement = "Internal only"
	cfg.DefaultLogo = "https://example.test/logo.png"
	cfg.HTMLDefaultLogo = "https://example.test/html.png"
	cfg.PDFDefaultLogo = "https://example.test/pdf.png"
	cfg.NoTimestampTitle = true

	env := envMap(t, cfg)

	want := map[string]string{
		"HMD_DOC_COMPANY_NAME":      "NeuronSphere",
		"DOCUMENT_TITLE":            "example-1.4",
		"CONFIDENTIALITY_STATEMENT": "Internal only",
		"DEFAULT_LOGO":              "https://example.test/logo.png",
		"HTML_DEFAULT_LOGO":         "https://example.test/html.png",
		"PDF_DEFAULT_LOGO":          "https://example.test/pdf.png",
		"NO_TIMESTAMP_TITLE":        "true",
	}
	for key, wantValue := range want {
		if env[key] != wantValue {
			t.Errorf("%s = %q, want %q", key, env[key], wantValue)
		}
	}
}

// Requirements: REQ_AUTO_004
func TestEnvPipConfOnlyWithAutodocAndAPath(t *testing.T) {
	cfg := baseConfig(t)
	cfg.PipConfigPath = "/tmp/pip.conf"

	if _, present := envMap(t, cfg)["PIP_CONF"]; present {
		t.Error("PIP_CONF should not be set when autodoc is off")
	}

	cfg.Autodoc = true
	if got := envMap(t, cfg)["PIP_CONF"]; got != pipSecretMount {
		t.Errorf("PIP_CONF = %q, want %q", got, pipSecretMount)
	}
}

// Requirements: REQ_EXEC_005
func TestMountsAlwaysIncludeInputAndOutput(t *testing.T) {
	cfg := baseConfig(t)
	mounts := cfg.Mounts()

	if len(mounts) != 2 {
		t.Fatalf("got %d mounts, want 2: %+v", len(mounts), mounts)
	}
	if !hasMount(mounts, cfg.InputPath, inputMount) {
		t.Errorf("input mount missing: %+v", mounts)
	}
	if !hasMount(mounts, cfg.OutputPath, outputMount) {
		t.Errorf("output mount missing: %+v", mounts)
	}
}

// Requirements: REQ_EXEC_005
func TestMountsSkipMissingOptionalPaths(t *testing.T) {
	cfg := baseConfig(t)
	cfg.GlobalStylesPath = filepath.Join(t.TempDir(), "does-not-exist")
	cfg.Autodoc = true
	cfg.PipConfigPath = filepath.Join(t.TempDir(), "no-pip.conf")

	if got := len(cfg.Mounts()); got != 2 {
		t.Errorf("got %d mounts, want 2 — absent optional paths must not be mounted", got)
	}
}

// Requirements: REQ_EXEC_005, REQ_ENV_005
func TestMountsIncludeStylesReadOnly(t *testing.T) {
	cfg := baseConfig(t)
	styles := t.TempDir()
	cfg.GlobalStylesPath = styles

	mounts := cfg.Mounts()
	idx := slices.IndexFunc(mounts, func(m mount.Mount) bool { return m.Target == globalStylesMount })
	if idx < 0 {
		t.Fatalf("styles mount missing: %+v", mounts)
	}
	if !mounts[idx].ReadOnly {
		t.Error("the styles mount should be read-only")
	}
}

// Requirements: REQ_EXEC_005, REQ_AUTO_002
func TestMountsIncludePipConfigWhenPresent(t *testing.T) {
	cfg := baseConfig(t)
	cfg.Autodoc = true

	pipConf := filepath.Join(t.TempDir(), "pip.conf")
	if err := os.WriteFile(pipConf, []byte("[global]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.PipConfigPath = pipConf

	mounts := cfg.Mounts()
	idx := slices.IndexFunc(mounts, func(m mount.Mount) bool { return m.Target == pipSecretMount })
	if idx < 0 {
		t.Fatalf("pip config mount missing: %+v", mounts)
	}
	if !mounts[idx].ReadOnly {
		t.Error("a credential mount should be read-only")
	}
}

// Requirements: REQ_EXEC_006
func TestContainerNameIsSanitized(t *testing.T) {
	tests := []struct {
		instance string
		shell    string
		want     string
	}{
		{"hmd-docs-example", "html", "bartleby-inst_hmd-docs-example_html"},
		{"my_repo", "pdf", "bartleby-inst_my_repo_pdf"},
		{"My Docs (v2)", "html", "bartleby-inst_My-Docs-v2_html"},
		{".hidden", "html", "bartleby-inst_hidden_html"},
		{"///", "html", "bartleby-inst_repo_html"},
		{"repo", "", "bartleby-inst_repo"},
	}

	for _, tt := range tests {
		cfg := baseConfig(t)
		cfg.InstanceName = tt.instance
		cfg.Context.Shell = tt.shell

		if got := cfg.ContainerName(); got != tt.want {
			t.Errorf("ContainerName(%q, %q) = %q, want %q", tt.instance, tt.shell, got, tt.want)
		}
	}
}

// Docker rejects names that do not start with an alphanumeric, so whatever the
// repo is called the result has to be usable.
// Requirements: REQ_EXEC_006
func TestContainerNameIsAlwaysValid(t *testing.T) {
	for _, instance := range []string{"", "_", "...", "  ", "a b c", "über/docs"} {
		cfg := baseConfig(t)
		cfg.InstanceName = instance

		got := cfg.ContainerName()
		if got == "" {
			t.Fatalf("empty container name for instance %q", instance)
		}
		first := got[0]
		if !(first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z' || first >= '0' && first <= '9') {
			t.Errorf("container name %q for instance %q does not start with an alphanumeric", got, instance)
		}
	}
}

// Requirements: REQ_PUML_001
func TestPumlEnvAndMounts(t *testing.T) {
	dir := t.TempDir()
	cfg := PumlConfig{
		ImageName:  "img",
		InputPath:  dir,
		OutputPath: filepath.Join(dir, "out"),
		Files:      []string{"a.puml", "nested/b.puml"},
	}

	env := cfg.Env()
	if len(env) != 1 || env[0] != "PUML_FILES=a.puml,nested/b.puml" {
		t.Errorf("env = %+v", env)
	}

	mounts := cfg.Mounts()
	if !hasMount(mounts, cfg.InputPath, inputMount) || !hasMount(mounts, cfg.OutputPath, outputMount) {
		t.Errorf("mounts = %+v", mounts)
	}
}

func hasMount(mounts []mount.Mount, source, target string) bool {
	return slices.ContainsFunc(mounts, func(m mount.Mount) bool {
		return m.Source == source && m.Target == target && m.Type == mount.TypeBind
	})
}
