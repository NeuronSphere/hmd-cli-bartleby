package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/hmdenv"
)

// runConfigure drives the configure command with scripted answers and returns
// what it printed.
func runConfigure(t *testing.T, answers string) string {
	t.Helper()

	var out strings.Builder
	configureCmd.SetOut(&out)
	configureCmd.SetErr(&out)
	configureCmd.SetIn(strings.NewReader(answers))
	t.Cleanup(func() {
		configureCmd.SetIn(nil)
		configureCmd.SetOut(nil)
		configureCmd.SetErr(nil)
	})

	if err := configureCmd.RunE(configureCmd, nil); err != nil {
		t.Fatalf("configure: %v", err)
	}
	return out.String()
}

// Requirements: REQ_CLI_005, REQ_ENV_004
func TestConfigureKeepsShownDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HMD_HOME", home)
	t.Setenv("HMD_BARTLEBY_DEFAULT_LOGO", "")
	t.Setenv("HMD_CONTAINER_REGISTRY", "")
	os.Unsetenv("HMD_BARTLEBY_DEFAULT_LOGO")
	os.Unsetenv("HMD_CONTAINER_REGISTRY")

	// Three empty answers: keep whatever each prompt offers.
	output := runConfigure(t, "\n\n\n")

	if !strings.Contains(output, filepath.Join(home, hmdenv.RelPath)) {
		t.Errorf("output should name the file being written:\n%s", output)
	}

	written := readEnvFile(t, filepath.Join(home, hmdenv.RelPath))

	if got := written["HMD_BARTLEBY_DEFAULT_LOGO"]; got != DefaultLogoURL {
		t.Errorf("logo = %q, want the offered default %q", got, DefaultLogoURL)
	}
	if got := written["HMD_CONTAINER_REGISTRY"]; got != defaultRegistry {
		t.Errorf("registry = %q, want the offered default %q", got, defaultRegistry)
	}
	// The confidentiality statement has no default, and an empty answer means
	// there is nothing to write.
	if _, present := written["HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT"]; present {
		t.Error("an empty answer with no default should not write a setting")
	}
}

// Requirements: REQ_CLI_005, REQ_ENV_004
func TestConfigureWritesAnswers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HMD_HOME", home)

	runConfigure(t, "https://example.test/logo.png\nInternal use only\nregistry.internal.test/ns\n")

	written := readEnvFile(t, filepath.Join(home, hmdenv.RelPath))

	want := map[string]string{
		"HMD_BARTLEBY_DEFAULT_LOGO":              "https://example.test/logo.png",
		"HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT": "Internal use only",
		"HMD_CONTAINER_REGISTRY":                 "registry.internal.test/ns",
	}
	for key, wantValue := range want {
		if written[key] != wantValue {
			t.Errorf("%s = %q, want %q", key, written[key], wantValue)
		}
	}
}

// Requirements: REQ_CLI_005
func TestConfigureRequiresHmdHome(t *testing.T) {
	t.Setenv("HMD_HOME", "")

	configureCmd.SetIn(strings.NewReader("\n\n\n"))
	configureCmd.SetOut(new(strings.Builder))
	t.Cleanup(func() { configureCmd.SetIn(nil); configureCmd.SetOut(nil) })

	err := configureCmd.RunE(configureCmd, nil)
	if err == nil {
		t.Fatal("expected an error when HMD_HOME is unset")
	}
	if !strings.Contains(err.Error(), "HMD_HOME") {
		t.Errorf("error should name the variable, got: %v", err)
	}
}

func readEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	vars, err := hmdenv.Parse(f)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return vars
}
