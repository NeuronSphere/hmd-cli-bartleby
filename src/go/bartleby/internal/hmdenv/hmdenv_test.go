package hmdenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Requirements: REQ_ENV_003
func TestParse(t *testing.T) {
	content := `
# a comment
HMD_CUSTOMER_CODE=hmd

export HMD_REGION=reg1
QUOTED="a value with spaces"
SINGLE='literal $NOT_EXPANDED'
ESCAPED="line\nbreak"
INLINE=value # trailing comment
EMPTY=
`

	vars, err := Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := map[string]string{
		"HMD_CUSTOMER_CODE": "hmd",
		"HMD_REGION":        "reg1",
		"QUOTED":            "a value with spaces",
		"SINGLE":            "literal $NOT_EXPANDED",
		"ESCAPED":           "line\nbreak",
		"INLINE":            "value",
		"EMPTY":             "",
	}
	for key, wantValue := range want {
		if got, ok := vars[key]; !ok {
			t.Errorf("%s missing from parsed vars", key)
		} else if got != wantValue {
			t.Errorf("%s = %q, want %q", key, got, wantValue)
		}
	}
	if len(vars) != len(want) {
		t.Errorf("parsed %d vars, want %d: %+v", len(vars), len(want), vars)
	}
}

// Requirements: REQ_ENV_002
func TestParseMalformedLine(t *testing.T) {
	if _, err := Parse(strings.NewReader("VALID=1\nthis is not an assignment\n")); err == nil {
		t.Fatal("expected an error for a line without =")
	}
}

// A missing HMD_HOME is the normal case for a plain checkout: no file, no error.
// Requirements: REQ_ENV_002
func TestLoadWithoutHmdHome(t *testing.T) {
	t.Setenv("HMD_HOME", "")

	applied, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if applied != 0 {
		t.Errorf("applied = %d, want 0", applied)
	}
}

// Requirements: REQ_ENV_002
func TestLoadMissingFileIsNotAnError(t *testing.T) {
	t.Setenv("HMD_HOME", t.TempDir())

	if _, err := Load(); err != nil {
		t.Errorf("a missing hmd.env should not be an error, got %v", err)
	}
}

// Requirements: REQ_ENV_001, REQ_ENV_001_SPEC001
func TestLoadAppliesValuesWithoutOverriding(t *testing.T) {
	home := t.TempDir()
	writeEnv(t, home, "BARTLEBY_TEST_NEW=from-file\nBARTLEBY_TEST_EXISTING=from-file\n")

	t.Setenv("HMD_HOME", home)
	t.Setenv("BARTLEBY_TEST_EXISTING", "from-shell")
	t.Setenv("BARTLEBY_TEST_NEW", "")
	os.Unsetenv("BARTLEBY_TEST_NEW")

	applied, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if applied != 1 {
		t.Errorf("applied = %d, want 1 (only the unset variable)", applied)
	}
	if got := os.Getenv("BARTLEBY_TEST_NEW"); got != "from-file" {
		t.Errorf("BARTLEBY_TEST_NEW = %q, want from-file", got)
	}
	if got := os.Getenv("BARTLEBY_TEST_EXISTING"); got != "from-shell" {
		t.Errorf("the shell value must win, got %q", got)
	}
}

// Requirements: REQ_ENV_002
func TestLoadReportsMalformedFile(t *testing.T) {
	home := t.TempDir()
	writeEnv(t, home, "not an assignment\n")
	t.Setenv("HMD_HOME", home)

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a malformed hmd.env")
	}
}

// Requirements: REQ_ENV_004
func TestSetCreatesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HMD_HOME", home)

	if err := Set("HMD_BARTLEBY_DEFAULT_LOGO", "https://example.test/logo.png"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	vars := parseFile(t, filepath.Join(home, RelPath))
	if got := vars["HMD_BARTLEBY_DEFAULT_LOGO"]; got != "https://example.test/logo.png" {
		t.Errorf("logo = %q", got)
	}
}

// Requirements: REQ_ENV_004
func TestSetReplacesExistingKeyAndKeepsTheRest(t *testing.T) {
	home := t.TempDir()
	writeEnv(t, home, "# keep me\nHMD_REGION=reg1\nHMD_BARTLEBY_DEFAULT_LOGO=old\n")
	t.Setenv("HMD_HOME", home)

	if err := Set("HMD_BARTLEBY_DEFAULT_LOGO", "new"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	path := filepath.Join(home, RelPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "# keep me") {
		t.Error("comments should survive a Set")
	}
	if strings.Contains(string(raw), "=old") {
		t.Error("the old value should be gone")
	}

	vars := parseFile(t, path)
	if vars["HMD_REGION"] != "reg1" {
		t.Errorf("unrelated keys should survive, got %+v", vars)
	}
	if vars["HMD_BARTLEBY_DEFAULT_LOGO"] != "new" {
		t.Errorf("logo = %q, want new", vars["HMD_BARTLEBY_DEFAULT_LOGO"])
	}
}

// Requirements: REQ_ENV_004
func TestSetQuotesValuesNeedingIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HMD_HOME", home)

	statement := `Confidential — do not "share"`
	if err := Set("HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT", statement); err != nil {
		t.Fatalf("Set: %v", err)
	}

	vars := parseFile(t, filepath.Join(home, RelPath))
	if got := vars["HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT"]; got != statement {
		t.Errorf("statement round-tripped as %q, want %q", got, statement)
	}
}

// Requirements: REQ_ENV_004
func TestSetWithoutHmdHome(t *testing.T) {
	t.Setenv("HMD_HOME", "")

	if err := Set("KEY", "value"); err == nil {
		t.Fatal("expected an error when HMD_HOME is unset")
	}
}

func writeEnv(t *testing.T, home, content string) {
	t.Helper()
	path := filepath.Join(home, RelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func parseFile(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	vars, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse(%s): %v", path, err)
	}
	return vars
}
