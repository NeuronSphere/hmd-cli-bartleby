package pipconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Requirements: REQ_AUTO_002
func TestResolveGeneratesConfigFromCredentials(t *testing.T) {
	t.Setenv("PIP_USERNAME", "builder")
	t.Setenv("PIP_PASSWORD", "p@ss w/ord")

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer cleanup()

	if !resolved.Generated {
		t.Error("credentials should produce a generated config")
	}

	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		t.Fatalf("reading generated pip.conf: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "[global]") || !strings.Contains(content, "extra-index-url") {
		t.Errorf("unexpected pip.conf:\n%s", content)
	}
	if strings.Contains(content, "p@ss w/ord") {
		t.Errorf("the password must be URL-escaped:\n%s", content)
	}
	if !strings.Contains(content, DefaultIndexPath) {
		t.Errorf("index host missing:\n%s", content)
	}

	// The file holds a credential, so it must not be world- or group-readable.
	info, err := os.Stat(resolved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 600", perm)
	}
}

// Requirements: REQ_AUTO_002
func TestResolveCleanupRemovesTheCredential(t *testing.T) {
	t.Setenv("PIP_USERNAME", "builder")
	t.Setenv("PIP_PASSWORD", "secret")

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	cleanup()

	if _, err := os.Stat(resolved.Path); !os.IsNotExist(err) {
		t.Errorf("generated pip.conf should be gone after cleanup, stat err = %v", err)
	}
}

// Requirements: REQ_AUTO_002_SPEC001
func TestResolveHonoursIndexOverride(t *testing.T) {
	t.Setenv("PIP_USERNAME", "builder")
	t.Setenv("PIP_PASSWORD", "secret")
	t.Setenv("HMD_PIP_EXTRA_INDEX_HOST", "pypi.internal.test/simple")

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "pypi.internal.test/simple") {
		t.Errorf("override ignored:\n%s", data)
	}
}

// Requirements: REQ_AUTO_003
func TestResolveFallsBackToUserConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PIP_USERNAME", "")
	t.Setenv("PIP_PASSWORD", "")

	pipConf := UserConfigPath()
	if err := os.MkdirAll(filepath.Dir(pipConf), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pipConf, []byte("[global]\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer cleanup()

	if resolved.Path != pipConf {
		t.Errorf("path = %q, want %q", resolved.Path, pipConf)
	}
	if resolved.Generated {
		t.Error("an existing user config is not generated")
	}
}

// No credentials and no user pip config is a normal setup for a package whose
// dependencies are all public.
// Requirements: REQ_AUTO_003_SPEC001
func TestResolveWithNothingConfigured(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("PIP_USERNAME", "")
	t.Setenv("PIP_PASSWORD", "")

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer cleanup()

	if resolved.Path != "" {
		t.Errorf("path = %q, want empty", resolved.Path)
	}
}

// Half-configured credentials should not silently produce a broken config.
// Requirements: REQ_AUTO_003
func TestResolveIgnoresPartialCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("PIP_USERNAME", "builder")
	t.Setenv("PIP_PASSWORD", "")

	resolved, cleanup, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defer cleanup()

	if resolved.Generated {
		t.Error("a username with no password should not generate a config")
	}
}
