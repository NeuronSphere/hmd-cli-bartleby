package reqtrace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	pythonOldRun = "2026-10-26T090000Z_hmd-ms-sample_01a17000-0000-7000-8000-000000000001"
	pythonNewRun = "2026-10-26T100000Z_hmd-ms-sample_01a17000-0000-7000-8000-000000000002"
	goRun        = "2026-10-26T110000Z_hmd-ms-sample_01a17000-0000-7000-8000-000000000003"
)

func runsFixture(name string) string {
	return filepath.Join("testdata", "verify", "runs", name)
}

// Requirements: REQ_VERIFY_006
func TestLoadResultsReadsABenderRunFolder(t *testing.T) {
	run := mustLoadResults(t, runsFixture(goRun))
	if run.Implementation != "go-0.1" {
		t.Errorf("label = %q, want go-0.1 from run.json", run.Implementation)
	}
	if run.Version != "0.4.0" {
		t.Errorf("version = %q, want 0.4.0 from run.json", run.Version)
	}
	if run.Generated != "2026-10-26T11:00:00Z" {
		t.Errorf("run time = %q, want the start time from run.json, not output.xml's", run.Generated)
	}
	if run.RunID != "01a17000-0000-7000-8000-000000000003" {
		t.Errorf("run ID = %q", run.RunID)
	}
	if !strings.HasSuffix(run.Source, "_go-0.1_output.xml") {
		t.Errorf("source = %q, want the folder's output.xml", run.Source)
	}
	if len(run.Results) != 3 {
		t.Errorf("results = %d, want 3", len(run.Results))
	}
}

// Requirements: REQ_VERIFY_006
func TestAGivenLabelAndVersionWinOverRunJSON(t *testing.T) {
	run := mustLoadResults(t, "port@9.9="+runsFixture(goRun))
	if run.Implementation != "port" || run.Version != "9.9" {
		t.Errorf("got %s@%s, want port@9.9", run.Implementation, run.Version)
	}
	if run.RunID == "" || run.Generated != "2026-10-26T11:00:00Z" {
		t.Errorf("run ID and start time must still come from run.json: %q %q", run.RunID, run.Generated)
	}
}

// Requirements: REQ_VERIFY_007
func TestRunLabelFallsBackToTagThenFileName(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(runsFixture(goRun), "2026-10-26T110000Z_hmd-ms-sample_go-0.1_output.xml"))
	if err != nil {
		t.Fatal(err)
	}

	// No run.json: the hmd_run_label: tag names the run.
	dir := filepath.Join(t.TempDir(), goRun)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2026-10-26T110000Z_hmd-ms-sample_other_output.xml"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	if run := mustLoadResults(t, dir); run.Implementation != "go-0.1" {
		t.Errorf("label = %q, want go-0.1 from the hmd_run_label: tag", run.Implementation)
	}

	// No run.json and no tag: the file name's label segment.
	untagged := strings.ReplaceAll(string(src), "<tag>hmd_run_label:go-0.1</tag>\n", "")
	if err := os.WriteFile(filepath.Join(dir, "2026-10-26T110000Z_hmd-ms-sample_other_output.xml"), []byte(untagged), 0o644); err != nil {
		t.Fatal(err)
	}
	if run := mustLoadResults(t, dir); run.Implementation != "other" {
		t.Errorf("label = %q, want other from the file name", run.Implementation)
	}

	// None of the three, as when hmd-bender ran without --label: "default".
	if err := os.Rename(filepath.Join(dir, "2026-10-26T110000Z_hmd-ms-sample_other_output.xml"),
		filepath.Join(dir, "2026-10-26T110000Z_hmd-ms-sample_output.xml")); err != nil {
		t.Fatal(err)
	}
	if run := mustLoadResults(t, dir); run.Implementation != "default" {
		t.Errorf("label = %q, want default", run.Implementation)
	}
}

// Requirements: REQ_VERIFY_006
func TestARunFolderNeedsExactlyOneOutputXML(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadResults(ResultSpec{Implementation: "default", Path: dir}); err == nil {
		t.Error("an empty folder must be rejected")
	}
	for _, name := range []string{"a_r_x_output.xml", "b_r_x_output.xml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<robot/>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadResults(ResultSpec{Implementation: "default", Path: dir}); err == nil {
		t.Error("two output.xml files must be rejected")
	}
}

// Requirements: REQ_VERIFY_008
func TestLoadRunsDirKeepsEachLabelsLatestRun(t *testing.T) {
	runs, err := LoadRunsDir(runsFixture(""), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("runs = %d, want one per label (python, go-0.1)", len(runs))
	}
	byLabel := map[string]RunResults{}
	for _, r := range runs {
		byLabel[r.Implementation] = r
	}
	if got := byLabel["python"].RunID; got != "01a17000-0000-7000-8000-000000000002" {
		t.Errorf("python run = %q, want the later of its two runs", got)
	}
	if _, ok := byLabel["go-0.1"]; !ok {
		t.Error("go-0.1 run missing")
	}

	// The older python run failed Delete Entity; the newer one passed it.
	v := Verify(mustLoadNeeds(t).WithoutProposals(), runs)
	if c := cellFor(t, v, "HMD_MS_SAMPLE_REQ_CRUD_002", "python"); c.Verdict != VerdictPass {
		t.Errorf("python CRUD_002 = %s, want pass from the latest run", c.Verdict)
	}
	if c := cellFor(t, v, "HMD_MS_SAMPLE_REQ_CRUD_002", "go-0.1"); c.Verdict != VerdictFail {
		t.Errorf("go-0.1 CRUD_002 = %s, want fail", c.Verdict)
	}
}

// Requirements: REQ_VERIFY_008
func TestLoadRunsDirAllRunsKeepsEveryRun(t *testing.T) {
	runs, err := LoadRunsDir(runsFixture(""), true)
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, r := range runs {
		labels = append(labels, r.Implementation)
	}
	want := []string{"python 2026-10-26T090000Z", "python 2026-10-26T100000Z", "go-0.1 2026-10-26T110000Z"}
	if strings.Join(labels, "|") != strings.Join(want, "|") {
		t.Errorf("labels = %q, want %q (run order)", labels, want)
	}
	if _, err := LoadRunsDir(t.TempDir(), false); err == nil {
		t.Error("a directory with no run folders must be an error")
	}
}

// Requirements: REQ_VERIFY_009
func TestRunIDsAppearInThePageAndJSON(t *testing.T) {
	repo := t.TempDir()
	needs, err := filepath.Abs(verifyFixture("needs.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(runsFixture(""))
	if err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(repo, "matrix.rst")
	var out, errOut bytes.Buffer
	err = Run(RunOptions{
		Repo:   repo,
		Verify: &VerifyOptions{Needs: needs, Runs: []string{dir}, Out: page, JSON: true},
		Out:    &out, Err: &errOut,
	})
	if err == nil {
		t.Fatal("go-0.1 fails CRUD_002, so verification must fail")
	}
	var decoded Verification
	if jerr := json.Unmarshal(out.Bytes(), &decoded); jerr != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", jerr, out.String())
	}
	ids := map[string][]string{}
	for _, s := range decoded.Implementations {
		ids[s.Implementation] = s.RunIDs
	}
	if got := ids["go-0.1"]; len(got) != 1 || got[0] != "01a17000-0000-7000-8000-000000000003" {
		t.Errorf("go-0.1 run IDs = %q", got)
	}
	c := cellFor(t, decoded, "HMD_MS_SAMPLE_REQ_CRUD_001", "python")
	if len(c.Tests) == 0 || c.Tests[0].RunID != "01a17000-0000-7000-8000-000000000002" {
		t.Errorf("python CRUD_001 tests = %+v, want the run ID on each outcome", c.Tests)
	}
	rendered, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"     - Run ID", "``01a17000-0000-7000-8000-000000000003``", "2026-10-26T11:00:00Z"} {
		if !strings.Contains(string(rendered), want) {
			t.Errorf("page lacks %q", want)
		}
	}
}
