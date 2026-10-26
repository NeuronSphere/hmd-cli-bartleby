package reqtrace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func verifyFixture(name string) string {
	return filepath.Join("testdata", "verify", name)
}

func mustLoadNeeds(t *testing.T) Needs {
	t.Helper()
	needs, err := LoadNeeds(verifyFixture("needs.json"))
	if err != nil {
		t.Fatal(err)
	}
	return needs
}

func mustLoadResults(t *testing.T, spec string) RunResults {
	t.Helper()
	parsed, err := ParseResultSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	run, err := LoadResults(parsed)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func cellFor(t *testing.T, v Verification, id, impl string) Cell {
	t.Helper()
	for _, row := range v.Rows {
		if row.ID != id {
			continue
		}
		for _, c := range row.Cells {
			if c.Implementation == impl {
				return c
			}
		}
	}
	t.Fatalf("no cell for %s × %s", id, impl)
	return Cell{}
}

// Requirements: REQ_VERIFY_001
func TestLoadNeedsReadsRequirementsAndTestItems(t *testing.T) {
	needs := mustLoadNeeds(t)
	if needs.Project != "hmd-ms-sample-1.0" {
		t.Errorf("project = %q", needs.Project)
	}
	if got := len(needs.Items); got != 10 {
		t.Fatalf("items = %d, want 10", got)
	}
	item := needs.Items["HMD_MS_SAMPLE_TEST_ROBOT_00000001"]
	if item.Type != "test" || len(item.Links) != 1 || item.Links[0] != "HMD_MS_SAMPLE_REQ_CRUD_001" {
		t.Errorf("test item = %+v", item)
	}
}

// Requirements: REQ_VERIFY_001
func TestWithoutProposalsDropsNERDItems(t *testing.T) {
	needs := mustLoadNeeds(t).WithoutProposals()
	if _, ok := needs.Items["HMD_MS_SAMPLE_NERD001"]; ok {
		t.Error("a req declared in docs/proposals must be dropped")
	}
	if len(needs.Items) != 9 {
		t.Errorf("items = %d, want 9: test items and docs/requirements items stay", len(needs.Items))
	}
}

// Requirements: REQ_VERIFY_002
func TestParseRobotOutputSchema5WithNestedSuites(t *testing.T) {
	run := mustLoadResults(t, "python="+verifyFixture("python-output.xml"))
	if run.Generated != "2026-10-26T09:00:00.000000" {
		t.Errorf("generated = %q", run.Generated)
	}
	if len(run.Results) != 3 {
		t.Fatalf("results = %d, want 3", len(run.Results))
	}
	first := run.Results[0]
	if first.Suite != "robot_test" || first.Name != "Create Entity" || first.Outcome != OutcomePass {
		t.Errorf("first = %+v", first)
	}
	// A keyword's own tags and status must not leak into the test's.
	if strings.Join(first.Tags, ",") != "REQ_CRUD_001,baseline" {
		t.Errorf("tags = %v", first.Tags)
	}
	if first.Start != "2026-10-26T09:00:00.100000" {
		t.Errorf("start = %q", first.Start)
	}
}

// Requirements: REQ_VERIFY_002
func TestParseRobotOutputSchema4(t *testing.T) {
	run := mustLoadResults(t, "go="+verifyFixture("go-output.xml"))
	if run.Generated != "20261026 09:30:00.000" {
		t.Errorf("generated = %q", run.Generated)
	}
	if len(run.Results) != 3 || run.Results[1].Outcome != OutcomeFail {
		t.Fatalf("results = %+v", run.Results)
	}
	if run.Results[1].Start != "20261026 09:30:00.400" {
		t.Errorf("start = %q", run.Results[1].Start)
	}
}

// Requirements: REQ_VERIFY_002
func TestParseGoTestJSONKeepsTopLevelTests(t *testing.T) {
	run := mustLoadResults(t, "go="+verifyFixture("go-test.json"))
	if len(run.Results) != 2 {
		t.Fatalf("results = %+v, want TestCreate and TestDeleteCLI only", run.Results)
	}
	if run.Results[0].Suite != "engine" || run.Results[0].Name != "TestCreate" || run.Results[0].Kind != KindGo {
		t.Errorf("first = %+v", run.Results[0])
	}
	if run.Generated != "2026-10-26T09:31:00Z" {
		t.Errorf("generated = %q", run.Generated)
	}
}

// Requirements: REQ_VERIFY_003
func TestParseResultSpec(t *testing.T) {
	cases := map[string]ResultSpec{
		"go@0.3.1=out.xml":  {Implementation: "go", Version: "0.3.1", Path: "out.xml"},
		"python=a/b.xml":    {Implementation: "python", Path: "a/b.xml"},
		"bender/output.xml": {Implementation: "default", Path: "bender/output.xml"},
	}
	for in, want := range cases {
		got, err := ParseResultSpec(in)
		if err != nil || got != want {
			t.Errorf("ParseResultSpec(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	if _, err := ParseResultSpec("=x"); err == nil {
		t.Error("an empty label must be rejected")
	}
}

// Requirements: REQ_VERIFY_003
func TestVerifyGivesOneVerdictPerImplementation(t *testing.T) {
	v := Verify(mustLoadNeeds(t).WithoutProposals(), []RunResults{
		mustLoadResults(t, "python="+verifyFixture("python-output.xml")),
		mustLoadResults(t, "go@0.1="+verifyFixture("go-output.xml")),
		mustLoadResults(t, "go@0.1="+verifyFixture("go-test.json")),
	})

	if len(v.Implementations) != 2 || v.Implementations[1].Implementation != "go" || v.Implementations[1].Version != "0.1" {
		t.Fatalf("implementations = %+v", v.Implementations)
	}

	checks := []struct {
		id, impl string
		want     Verdict
		tests    int
	}{
		{"HMD_MS_SAMPLE_REQ_CRUD_001", "python", VerdictPass, 1},
		{"HMD_MS_SAMPLE_REQ_CRUD_001", "go", VerdictPass, 2}, // the Robot test and the Go test
		{"HMD_MS_SAMPLE_REQ_CRUD_002", "python", VerdictPass, 1},
		{"HMD_MS_SAMPLE_REQ_CRUD_002", "go", VerdictFail, 2}, // Robot failed, main.TestDeleteCLI passed
		{"HMD_MS_SAMPLE_REQ_CRUD_003", "python", VerdictPass, 1},
		{"HMD_MS_SAMPLE_REQ_CRUD_003", "go", VerdictMissing, 0},
		{"HMD_MS_SAMPLE_REQ_DOC_001", "python", VerdictExempt, 0},
	}
	for _, c := range checks {
		cell := cellFor(t, v, c.id, c.impl)
		if cell.Verdict != c.want || len(cell.Tests) != c.tests {
			t.Errorf("%s × %s = %s with %d tests; want %s with %d", c.id, c.impl, cell.Verdict, len(cell.Tests), c.want, c.tests)
		}
	}
	if got := v.Failing(); got != 2 {
		t.Errorf("failing = %d, want 2 (go: CRUD_002 fail, CRUD_003 missing)", got)
	}
}

// Requirements: REQ_VERIFY_003, REQ_VERIFY_005
func TestVerifyReportsResultsTheDocsDoNotDeclare(t *testing.T) {
	v := Verify(mustLoadNeeds(t), []RunResults{mustLoadResults(t, "go="+verifyFixture("go-output.xml"))})
	if len(v.Unmatched) != 1 || !strings.Contains(v.Unmatched[0], "Brand New Test") {
		t.Errorf("unmatched = %v", v.Unmatched)
	}
}

// Requirements: REQ_VERIFY_004
func TestVerifyCountsBaselineApartFromNewTests(t *testing.T) {
	v := Verify(mustLoadNeeds(t), []RunResults{
		mustLoadResults(t, "python="+verifyFixture("python-output.xml")),
		mustLoadResults(t, "go="+verifyFixture("go-output.xml")),
	})
	py, gov := v.Implementations[0], v.Implementations[1]
	if py.BaselineTests != 2 || py.BaselinePassed != 2 || py.NewTests != 1 || py.NewPassed != 1 {
		t.Errorf("python = %+v", py)
	}
	if gov.BaselineTests != 2 || gov.BaselinePassed != 1 || gov.NewTests != 1 || gov.NewPassed != 1 {
		t.Errorf("go = %+v", gov)
	}
}

// Requirements: REQ_VERIFY_005
func TestRenderVerification(t *testing.T) {
	v := Verify(mustLoadNeeds(t).WithoutProposals(), []RunResults{
		mustLoadResults(t, "python="+verifyFixture("python-output.xml")),
		mustLoadResults(t, "go@0.1="+verifyFixture("go-output.xml")),
	})
	page := RenderVerification(v)
	for _, want := range []string{
		".. Generated by reqtrace -verify",
		"   * - :need:`HMD_MS_SAMPLE_REQ_CRUD_002`\n     - pass (1)\n     - **fail**: robot\\_test: Delete Entity",
		"     - **missing**",
		"   * - go 0.1",
		"     - 1 / 2",
		"Brand New Test",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q:\n%s", want, page)
		}
	}
	// No needs directives: the page must never be parsed as declaring requirements.
	if strings.Contains(page, ".. req::") || strings.Contains(page, ".. spec::") {
		t.Error("the verification page must not declare needs")
	}
}

// Requirements: REQ_VERIFY_005
func TestRunVerifyWritesPageJSONAndFailsOnGaps(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "docs", "requirements"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := func(name string) string {
		p, err := filepath.Abs(verifyFixture(name))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	var out, errOut bytes.Buffer
	err := Run(RunOptions{
		Repo: repo,
		Verify: &VerifyOptions{
			Needs:   abs("needs.json"),
			Results: []string{"python=" + abs("python-output.xml"), "go=" + abs("go-output.xml")},
			JSON:    true,
		},
		Out: &out, Err: &errOut,
	})
	if err == nil || !strings.Contains(err.Error(), "2 requirement result(s) failing or missing") {
		t.Fatalf("err = %v", err)
	}
	var decoded Verification
	if jerr := json.Unmarshal(out.Bytes(), &decoded); jerr != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", jerr, out.String())
	}
	if len(decoded.Rows) != 4 {
		t.Errorf("rows = %d, want 4", len(decoded.Rows))
	}
	if !strings.Contains(errOut.String(), "Brand New Test") {
		t.Errorf("stderr = %q", errOut.String())
	}
	if _, serr := os.Stat(filepath.Join(repo, "docs", "requirements", VerificationFile)); serr != nil {
		t.Errorf("page not written: %v", serr)
	}

	// All passing: python alone, against requirements its tests cover.
	out.Reset()
	errOut.Reset()
	err = Run(RunOptions{
		Repo: repo,
		Verify: &VerifyOptions{
			Needs:   abs("needs.json"),
			Results: []string{"python=" + abs("python-output.xml")},
			Out:     "-",
		},
		Out: &out, Err: &errOut,
	})
	if err != nil {
		t.Fatalf("all-pass run failed: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "python: 3 pass, 1 exempt; baseline 2/2 passed, new 1/1 passed") {
		t.Errorf("summary = %q", out.String())
	}

	if err := Run(RunOptions{Repo: repo, Verify: &VerifyOptions{Needs: abs("needs.json")}, Out: &out, Err: &errOut}); err == nil {
		t.Error("verify without results must fail")
	}
}
