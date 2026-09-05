package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeReqsFixture builds the smallest repository reqtrace will accept: a name
// to derive the ID scheme from, one requirement, and one test that covers it.
func writeReqsFixture(t *testing.T, extraRequirement string) string {
	t.Helper()
	root := t.TempDir()

	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("meta-data/manifest.json", `{"name": "hmd-cli-bartleby"}`)
	write("docs/requirements/selection.rst", `Selection
=========

.. req:: Do the thing
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    The CLI shall do the thing.
`+extraRequirement)
	write("src/go/thing/thing_test.go", `package thing

import "testing"

// Requirements: REQ_SEL_001
func TestTheThing(t *testing.T) {}
`)

	return root
}

// runReqs drives the reqs command and returns its output and error.
func runReqs(t *testing.T, opts reqsOptions) (string, error) {
	t.Helper()

	restore := reqsOpts
	reqsOpts = opts
	t.Cleanup(func() { reqsOpts = restore })

	var out strings.Builder
	reqsCmd.SetOut(&out)
	reqsCmd.SetErr(&out)
	t.Cleanup(func() {
		reqsCmd.SetOut(nil)
		reqsCmd.SetErr(nil)
	})

	err := reqsCmd.RunE(reqsCmd, nil)
	return out.String(), err
}

// Requirements: REQ_TRACE_010
func TestReqsGeneratesTheMatrix(t *testing.T) {
	root := writeReqsFixture(t, "")

	output, err := runReqs(t, reqsOptions{repo: root})
	if err != nil {
		t.Fatalf("bartleby reqs: %v\n%s", err, output)
	}

	generated := filepath.Join(root, "docs", "requirements", "traceability.rst")
	if _, err := os.Stat(generated); err != nil {
		t.Fatalf("traceability.rst was not written: %v", err)
	}
	if !strings.Contains(output, "traceability.rst") {
		t.Errorf("output does not name what it wrote:\n%s", output)
	}

	// Then the check passes against what was just written.
	output, err = runReqs(t, reqsOptions{repo: root, check: true})
	if err != nil {
		t.Fatalf("bartleby reqs --check: %v\n%s", err, output)
	}
	if !strings.Contains(output, "traceability ok") {
		t.Errorf("check did not report success:\n%s", output)
	}
}

// Requirements: REQ_TRACE_010
func TestReqsCheckFailsOnAGap(t *testing.T) {
	// Proves the command is really running the tool rather than reporting
	// success: a requirement nothing covers has to fail here.
	root := writeReqsFixture(t, `
.. req:: Do the other thing
    :id: HMD_CLI_BARTLEBY_REQ_SEL_002
    :status: implemented

    The CLI shall do the other thing.
`)

	if _, err := runReqs(t, reqsOptions{repo: root}); err == nil {
		t.Fatal("generating with an uncovered requirement should still report the gap")
	}

	output, err := runReqs(t, reqsOptions{repo: root, check: true})
	if err == nil {
		t.Fatal("bartleby reqs --check should fail when a requirement has no test")
	}
	if !strings.Contains(output, "HMD_CLI_BARTLEBY_REQ_SEL_002") {
		t.Errorf("output does not name the uncovered requirement:\n%s", output)
	}
}

// Requirements: REQ_TRACE_010
func TestReqsQuietPrintsNothingOnSuccess(t *testing.T) {
	root := writeReqsFixture(t, "")

	if _, err := runReqs(t, reqsOptions{repo: root}); err != nil {
		t.Fatalf("generate: %v", err)
	}

	output, err := runReqs(t, reqsOptions{repo: root, check: true, quiet: true})
	if err != nil {
		t.Fatalf("quiet check: %v", err)
	}
	if strings.TrimSpace(output) != "" {
		t.Errorf("--quiet printed:\n%s", output)
	}
}
