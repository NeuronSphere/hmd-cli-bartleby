package reqtrace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleRequirements = `Sample
======

.. req:: Select builders with --shell
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    Body text that the parser ignores.

.. spec:: Any builder in the manifest is valid
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    More body text.

.. req:: Something no test can reach
    :id: HMD_CLI_BARTLEBY_REQ_CLI_004
    :status: implemented
    :tags: trace-exempt

    Verified by inspection.
`

// writeRepo lays out a miniature repository with requirements, a Go test file,
// and a Robot suite.
// testScheme matches the repository name writeRepo puts in meta-data/manifest.json,
// so direct parser calls expand IDs the same way Load would.
var testScheme = SchemeFromName("hmd-cli-bartleby")

func writeRepo(t *testing.T, requirements, goTest, robot string) string {
	t.Helper()
	root := t.TempDir()

	write(t, filepath.Join(root, "meta-data", "manifest.json"), `{"name": "hmd-cli-bartleby"}`)
	write(t, filepath.Join(root, "docs", "requirements", "sample.rst"), requirements)
	if goTest != "" {
		write(t, filepath.Join(root, "src", "go", "bartleby", "thing", "thing_test.go"), goTest)
	}
	if robot != "" {
		write(t, filepath.Join(root, "test", "sample.robot"), robot)
	}
	return root
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Requirements: REQ_TRACE_001
func TestParseRequirements(t *testing.T) {
	root := writeRepo(t, sampleRequirements, "", "")

	requirements, err := ParseRequirements(filepath.Join(root, "docs", "requirements"), root, testScheme)
	if err != nil {
		t.Fatalf("ParseRequirements: %v", err)
	}
	if len(requirements) != 3 {
		t.Fatalf("got %d items, want 3: %+v", len(requirements), requirements)
	}

	byID := map[string]Requirement{}
	for _, r := range requirements {
		byID[r.ID] = r
	}

	req := byID["HMD_CLI_BARTLEBY_REQ_SEL_001"]
	if req.Type != "req" {
		t.Errorf("type = %q, want req", req.Type)
	}
	if req.Title != "Select builders with --shell" {
		t.Errorf("title = %q", req.Title)
	}
	if req.Status != "implemented" {
		t.Errorf("status = %q, want implemented", req.Status)
	}
	if req.File != "docs/requirements/sample.rst" || req.Line == 0 {
		t.Errorf("location = %s:%d, want the repo-relative file and a line", req.File, req.Line)
	}

	spec := byID["HMD_CLI_BARTLEBY_REQ_SEL_001_SPEC001"]
	if spec.Type != "spec" {
		t.Errorf("type = %q, want spec", spec.Type)
	}
	if len(spec.Links) != 1 || spec.Links[0] != "HMD_CLI_BARTLEBY_REQ_SEL_001" {
		t.Errorf("links = %v", spec.Links)
	}

	if !byID["HMD_CLI_BARTLEBY_REQ_CLI_004"].Exempt() {
		t.Error("an item tagged trace-exempt should report Exempt")
	}
	if byID["HMD_CLI_BARTLEBY_REQ_SEL_001"].Exempt() {
		t.Error("an untagged item should not be exempt")
	}
}

// Requirements: REQ_TRACE_001
func TestParseRequirementsSkipsTheGeneratedPage(t *testing.T) {
	root := writeRepo(t, sampleRequirements, "", "")
	write(t, filepath.Join(root, "docs", "requirements", GeneratedFile),
		".. test:: x\n    :id: HMD_CLI_BARTLEBY_TEST_GO_00000000\n\n"+sampleRequirements)

	requirements, err := ParseRequirements(filepath.Join(root, "docs", "requirements"), root, testScheme)
	if err != nil {
		t.Fatalf("ParseRequirements: %v", err)
	}
	if len(requirements) != 3 {
		t.Errorf("got %d items, want 3 — the generated page must not be a source of requirements", len(requirements))
	}
}

const sampleGoTest = `package thing

import "testing"

// Requirements: REQ_SEL_001
func TestOne(t *testing.T) {}

// Some prose about the test.
// Requirements: REQ_SEL_001_SPEC001, REQ_SEL_001
func TestTwo(t *testing.T) {}

func TestUntagged(t *testing.T) {}

// Requirements: REQ_SEL_001
func helperNotATest(t *testing.T) {}

// Requirements: REQ_SEL_001
func BenchmarkThing(b *testing.B) {}

func TestWithBodyComment(t *testing.T) {
	// Requirements: REQ_SEL_001
	_ = 1
}
`

// Requirements: REQ_TRACE_002
func TestParseGoTests(t *testing.T) {
	root := writeRepo(t, sampleRequirements, sampleGoTest, "")

	tests, err := ParseGoTests(filepath.Join(root, "src", "go", "bartleby"), root, nil, testScheme)
	if err != nil {
		t.Fatalf("ParseGoTests: %v", err)
	}

	byName := map[string]TestCase{}
	for _, tc := range tests {
		byName[tc.Name] = tc
	}

	one, ok := byName["TestOne"]
	if !ok {
		t.Fatalf("TestOne missing from %+v", tests)
	}
	if len(one.Requirements) != 1 || one.Requirements[0] != "HMD_CLI_BARTLEBY_REQ_SEL_001" {
		t.Errorf("TestOne requirements = %v, want the expanded ID", one.Requirements)
	}
	if one.Suite != "thing" {
		t.Errorf("suite = %q, want the package name", one.Suite)
	}
	if one.Kind != KindGo || one.File != "src/go/bartleby/thing/thing_test.go" {
		t.Errorf("kind/file = %s %s", one.Kind, one.File)
	}

	if got := len(byName["TestTwo"].Requirements); got != 2 {
		t.Errorf("TestTwo claims %d requirements, want 2", got)
	}
	if len(byName["TestUntagged"].Requirements) != 0 {
		t.Error("TestUntagged should claim nothing")
	}
}

// Only real test functions count, so a helper taking *testing.T, a benchmark,
// and a comment inside a function body are not coverage claims.
//
// Requirements: REQ_TRACE_002_SPEC001
func TestParseGoTestsIgnoresNonTests(t *testing.T) {
	root := writeRepo(t, sampleRequirements, sampleGoTest, "")

	tests, err := ParseGoTests(filepath.Join(root, "src", "go", "bartleby"), root, nil, testScheme)
	if err != nil {
		t.Fatalf("ParseGoTests: %v", err)
	}

	for _, tc := range tests {
		if tc.Name == "helperNotATest" || tc.Name == "BenchmarkThing" {
			t.Errorf("%s should not be collected as a test", tc.Name)
		}
	}

	for _, tc := range tests {
		if tc.Name == "TestWithBodyComment" && len(tc.Requirements) != 0 {
			t.Errorf("a comment inside the body must not count as a claim, got %v", tc.Requirements)
		}
	}
}

// Requirements: REQ_TRACE_002
func TestParseGoTestsSkipsRequestedDirectories(t *testing.T) {
	root := writeRepo(t, sampleRequirements, sampleGoTest, "")
	write(t, filepath.Join(root, "src", "go", "bartleby", "build", "gen_test.go"),
		"package build\n\nimport \"testing\"\n\nfunc TestGenerated(t *testing.T) {}\n")

	tests, err := ParseGoTests(filepath.Join(root, "src", "go", "bartleby"), root, []string{"build"}, testScheme)
	if err != nil {
		t.Fatalf("ParseGoTests: %v", err)
	}
	for _, tc := range tests {
		if tc.Name == "TestGenerated" {
			t.Error("a skipped directory should not be scanned")
		}
	}
}

const sampleRobot = `*** Settings ***
Documentation     Sample

*** Variables ***
${BINARY}    bartleby

*** Test Cases ***
A Tagged Test
    [Documentation]    Something
    [Tags]    REQ_SEL_001    REQ_SEL_001_SPEC001    slow
    Log    hello

An Untagged Test
    Log    hello

*** Keywords ***
Not A Test
    [Tags]    REQ_SEL_001
    Log    hello
`

// Requirements: REQ_TRACE_002
func TestParseRobotTests(t *testing.T) {
	root := writeRepo(t, sampleRequirements, "", sampleRobot)

	tests, err := ParseRobotTests([]string{filepath.Join(root, "test", "sample.robot")}, root, testScheme)
	if err != nil {
		t.Fatalf("ParseRobotTests: %v", err)
	}
	if len(tests) != 2 {
		t.Fatalf("got %d tests, want 2 (keywords are not tests): %+v", len(tests), tests)
	}

	tagged := tests[0]
	if tagged.Name != "A Tagged Test" {
		t.Fatalf("first test = %q", tagged.Name)
	}
	if len(tagged.Requirements) != 2 {
		t.Errorf("requirements = %v, want the two REQ tags and not \"slow\"", tagged.Requirements)
	}
	if tagged.Suite != "sample" || tagged.Kind != KindRobot {
		t.Errorf("suite/kind = %s %s", tagged.Suite, tagged.Kind)
	}
	if len(tests[1].Requirements) != 0 {
		t.Errorf("second test should claim nothing, got %v", tests[1].Requirements)
	}
}

// Requirements: REQ_TRACE_002
func TestExpandID(t *testing.T) {
	cases := map[string]string{
		"REQ_SEL_001":                  "HMD_CLI_BARTLEBY_REQ_SEL_001",
		"HMD_CLI_BARTLEBY_REQ_SEL_001": "HMD_CLI_BARTLEBY_REQ_SEL_001",
		"HMD_TF_BARTLEBY_NERD001":      "HMD_TF_BARTLEBY_NERD001",
		"  REQ_SEL_001  ":              "HMD_CLI_BARTLEBY_REQ_SEL_001",
		"":                             "",
	}
	for in, want := range cases {
		if got := testScheme.ExpandID(in); got != want {
			t.Errorf("ExpandID(%q) = %q, want %q", in, got, want)
		}
	}

	// The prefix is derived from the repository name, not compiled in.
	schemes := map[string]Scheme{
		"hmd-cli-bartleby": {Prefix: "HMD_CLI_BARTLEBY_", Org: "HMD_"},
		"glint-jira":       {Prefix: "GLINT_JIRA_", Org: "GLINT_"},
		"bartleby":         {Prefix: "BARTLEBY_", Org: "BARTLEBY_"},
		"hmd_ms_gozer":     {Prefix: "HMD_MS_GOZER_", Org: "HMD_"},
		"weird..name--x":   {Prefix: "WEIRD_NAME_X_", Org: "WEIRD_"},
		"":                 {},
	}
	for name, want := range schemes {
		if got := SchemeFromName(name); got != want {
			t.Errorf("SchemeFromName(%q) = %+v, want %+v", name, got, want)
		}
	}

	// An unset scheme leaves IDs alone rather than corrupting them.
	if got := (Scheme{}).ExpandID("REQ_SEL_001"); got != "REQ_SEL_001" {
		t.Errorf("zero scheme ExpandID = %q, want the input unchanged", got)
	}
}

// Requirements: REQ_TRACE_003, REQ_TRACE_007
func TestValidateReportsUncoveredRequirements(t *testing.T) {
	model := Model{
		Requirements: []Requirement{
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req", File: "docs/requirements/sample.rst", Line: 4},
			{ID: "HMD_CLI_BARTLEBY_REQ_CLI_004", Type: "req", Tags: []string{ExemptTag},
				File: "docs/requirements/sample.rst", Line: 20},
		},
		Tests: []TestCase{
			{Kind: KindGo, Name: "TestSomething", Suite: "thing",
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_CLI_004"}, File: "x_test.go", Line: 9},
		},
	}

	problems := Validate(model)

	if len(problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(problems), problems)
	}
	if problems[0].Kind != ProblemUncovered {
		t.Errorf("kind = %q, want %q", problems[0].Kind, ProblemUncovered)
	}
	if !strings.Contains(problems[0].Message, "HMD_CLI_BARTLEBY_REQ_SEL_001") {
		t.Errorf("message should name the requirement: %s", problems[0].Message)
	}
	if problems[0].Where != "docs/requirements/sample.rst:4" {
		t.Errorf("where = %q, want the file and line", problems[0].Where)
	}
	if !strings.Contains(problems[0].Message, ExemptTag) {
		t.Errorf("message should say how to exempt it: %s", problems[0].Message)
	}
}

// An exempt requirement is not reported, which is the whole point of the tag.
//
// Requirements: REQ_TRACE_003
func TestValidateAcceptsExemptRequirements(t *testing.T) {
	model := Model{
		Requirements: []Requirement{
			{ID: "HMD_CLI_BARTLEBY_REQ_CLI_004", Type: "req", Tags: []string{"trace-exempt"}},
		},
	}

	for _, problem := range Validate(model) {
		if problem.Kind == ProblemUncovered {
			t.Errorf("exempt requirement was reported: %v", problem)
		}
	}
}

// Requirements: REQ_TRACE_004
func TestValidateReportsUntaggedTests(t *testing.T) {
	model := Model{
		Requirements: []Requirement{{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req"}},
		Tests: []TestCase{
			{Kind: KindGo, Name: "TestTagged", Suite: "thing",
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_SEL_001"}},
			{Kind: KindGo, Name: "TestUntagged", Suite: "thing", File: "thing_test.go", Line: 12},
			{Kind: KindRobot, Name: "An Untagged Test", Suite: "sample", File: "test/sample.robot", Line: 5},
		},
	}

	var untagged []Problem
	for _, problem := range Validate(model) {
		if problem.Kind == ProblemUntagged {
			untagged = append(untagged, problem)
		}
	}

	if len(untagged) != 2 {
		t.Fatalf("got %d untagged reports, want 2: %v", len(untagged), untagged)
	}
	// The hint has to match the annotation style of the test's own language.
	joined := untagged[0].Message + untagged[1].Message
	if !strings.Contains(joined, "// Requirements:") || !strings.Contains(joined, "[Tags]") {
		t.Errorf("hints should show the right syntax for each kind:\n%s", joined)
	}
}

// Requirements: REQ_TRACE_005
func TestValidateReportsUnresolvableReferences(t *testing.T) {
	model := Model{
		Requirements: []Requirement{
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req", File: "a.rst", Line: 1},
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req", File: "b.rst", Line: 1},
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_002", Type: "spec", File: "a.rst", Line: 9,
				Links: []string{"HMD_CLI_BARTLEBY_REQ_GONE_001"}},
		},
		Tests: []TestCase{
			{Kind: KindGo, Name: "TestOne", Suite: "thing", File: "t_test.go", Line: 3,
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_SEL_001"}},
			{Kind: KindGo, Name: "TestTypo", Suite: "thing", File: "t_test.go", Line: 9,
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_TYPO_001"}},
			{Kind: KindGo, Name: "TestSpec", Suite: "thing", File: "t_test.go", Line: 15,
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_SEL_002"}},
		},
	}

	kinds := map[string]int{}
	for _, problem := range Validate(model) {
		kinds[problem.Kind]++
	}

	for _, kind := range []string{ProblemDuplicateID, ProblemUnknownRef, ProblemBadLink} {
		if kinds[kind] != 1 {
			t.Errorf("%s reported %d times, want 1 (all problems: %v)", kind, kinds[kind], kinds)
		}
	}
}

// Requirements: REQ_TRACE_007
func TestValidateOrdersProblemsStably(t *testing.T) {
	model := Model{
		Requirements: []Requirement{
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_002", Type: "req", File: "b.rst", Line: 2},
			{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req", File: "a.rst", Line: 1},
		},
		Tests: []TestCase{
			{Kind: KindGo, Name: "TestB", Suite: "thing", File: "b_test.go", Line: 2},
			{Kind: KindGo, Name: "TestA", Suite: "thing", File: "a_test.go", Line: 1},
		},
	}

	first := Validate(model)
	for range 5 {
		again := Validate(model)
		if len(again) != len(first) {
			t.Fatalf("problem count changed between runs: %d then %d", len(first), len(again))
		}
		for i := range first {
			if first[i] != again[i] {
				t.Fatalf("problem %d changed between runs:\n%v\n%v", i, first[i], again[i])
			}
		}
	}
}

// Requirements: REQ_TRACE_006
func TestRenderIsDeterministic(t *testing.T) {
	model := Model{
		Requirements: []Requirement{{ID: "HMD_CLI_BARTLEBY_REQ_SEL_001", Type: "req", Title: "One"}},
		Tests: []TestCase{
			{Kind: KindRobot, Name: "B Test", Suite: "sample", File: "test/sample.robot", Line: 8,
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_SEL_001"}},
			{Kind: KindGo, Name: "TestA", Suite: "thing", File: "thing_test.go", Line: 4,
				Requirements: []string{"HMD_CLI_BARTLEBY_REQ_SEL_001"}},
		},
	}

	first := Render(model)
	for range 5 {
		if Render(model) != first {
			t.Fatal("Render is not deterministic")
		}
	}

	// Reversing the input order must not change the output.
	reversed := Model{
		Requirements: model.Requirements,
		Tests:        []TestCase{model.Tests[1], model.Tests[0]},
	}
	if Render(reversed) != first {
		t.Error("Render depends on input order")
	}

	if !strings.Contains(first, ".. test:: thing.TestA") {
		t.Errorf("Go test item missing:\n%s", first)
	}
	if !strings.Contains(first, ".. test:: sample: B Test") {
		t.Errorf("Robot test item missing:\n%s", first)
	}
	if !strings.Contains(first, ":links: HMD_CLI_BARTLEBY_REQ_SEL_001") {
		t.Errorf("test items must link to their requirements:\n%s", first)
	}
	if !strings.Contains(first, ".. needtable::") {
		t.Errorf("the matrix should be rendered by sphinx-needs:\n%s", first)
	}
}

// A generated test ID depends on the test's identity, not its position, so
// adding one test does not renumber the others.
//
// Requirements: REQ_TRACE_006_SPEC001
func TestNeedIDIsStable(t *testing.T) {
	prefix := testScheme.Prefix
	one := TestCase{Kind: KindGo, Name: "TestOne", Suite: "thing", Prefix: prefix}
	same := TestCase{Kind: KindGo, Name: "TestOne", Suite: "thing", File: "moved_test.go", Line: 99, Prefix: prefix}
	other := TestCase{Kind: KindGo, Name: "TestTwo", Suite: "thing", Prefix: prefix}
	otherKind := TestCase{Kind: KindRobot, Name: "TestOne", Suite: "thing", Prefix: prefix}

	if one.NeedID() != same.NeedID() {
		t.Error("moving a test within its file should not change its ID")
	}
	if one.NeedID() == other.NeedID() {
		t.Error("different tests must not share an ID")
	}
	if one.NeedID() == otherKind.NeedID() {
		t.Error("a Go test and a Robot test with the same name must not share an ID")
	}
	if !strings.HasPrefix(one.NeedID(), prefix+"TEST_GO_") {
		t.Errorf("ID = %q, want the repo prefix and kind", one.NeedID())
	}
}

// Requirements: REQ_TRACE_006
func TestCheckFreshDetectsStaleness(t *testing.T) {
	root := writeRepo(t, sampleRequirements, sampleGoTest, sampleRobot)
	layout := DefaultLayout(root)

	model, err := Load(layout)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := CheckFresh(layout, model); err == nil {
		t.Fatal("a missing generated page should be reported as stale")
	}

	if err := Write(layout, model); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := CheckFresh(layout, model); err != nil {
		t.Errorf("freshly written page should be current, got %v", err)
	}

	write(t, layout.GeneratedPath(), "hand edited\n")
	if err := CheckFresh(layout, model); err == nil {
		t.Error("an edited page should be reported as stale")
	}
}

// Requirements: REQ_TRACE_001, REQ_TRACE_002
func TestLoadReadsTheWholeRepository(t *testing.T) {
	root := writeRepo(t, sampleRequirements, sampleGoTest, sampleRobot)

	model, err := Load(DefaultLayout(root))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(model.Requirements) != 3 {
		t.Errorf("got %d requirements, want 3", len(model.Requirements))
	}

	var goCount, robotCount int
	for _, tc := range model.Tests {
		if tc.Kind == KindRobot {
			robotCount++
		} else {
			goCount++
		}
	}
	if goCount == 0 || robotCount == 0 {
		t.Errorf("expected both kinds of test, got %d Go and %d Robot", goCount, robotCount)
	}

	coverage := model.Coverage()
	if len(coverage["HMD_CLI_BARTLEBY_REQ_SEL_001"]) == 0 {
		t.Error("coverage should link the requirement to its tests")
	}
}

// Requirements: REQ_TRACE_001
func TestLoadWithoutARequirementsDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "meta-data", "manifest.json"), "{}")

	if _, err := Load(DefaultLayout(root)); err == nil {
		t.Fatal("expected an error when the requirements directory is absent")
	}
}

// Requirements: REQ_TRACE_001
func TestFindRepoRoot(t *testing.T) {
	root := writeRepo(t, sampleRequirements, "", "")
	nested := filepath.Join(root, "src", "go", "bartleby")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("FindRepoRoot: %v", err)
	}
	// t.TempDir can sit behind a symlink (/var vs /private/var on macOS).
	wantResolved, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(found)
	if gotResolved != wantResolved {
		t.Errorf("found %q, want %q", gotResolved, wantResolved)
	}

	if _, err := FindRepoRoot(t.TempDir()); err == nil {
		t.Error("expected an error where there is no manifest above the directory")
	}
}

// Requirements: REQ_TRACE_011
func TestParseGoTestsToleratesNoGoTree(t *testing.T) {
	tests, err := ParseGoTests(filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir(), nil, testScheme)
	if err != nil {
		t.Fatalf("a repository with no Go tree should not be an error: %v", err)
	}
	if len(tests) != 0 {
		t.Errorf("got %d tests from a nonexistent tree", len(tests))
	}
}

// Requirements: REQ_TRACE_011
func TestLoadWorksWithRobotTestsOnly(t *testing.T) {
	// The shape of a Python and Robot repository: requirements, a Robot suite,
	// and no src/go at all.
	root := t.TempDir()
	write(t, filepath.Join(root, "meta-data", "manifest.json"), `{"name": "hmd-cli-bartleby"}`)
	write(t, filepath.Join(root, "docs", "requirements", "sel.rst"), `
.. req:: Do the thing
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    It shall do the thing.
`)
	write(t, filepath.Join(root, "test", "suite.robot"), `*** Test Cases ***
The Thing Happens
    [Tags]    REQ_SEL_001
    No Operation
`)

	model, err := Load(DefaultLayout(root))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(model.Requirements) != 1 {
		t.Fatalf("got %d requirements", len(model.Requirements))
	}
	if len(model.Tests) != 1 {
		t.Fatalf("got %d tests", len(model.Tests))
	}
	if problems := Validate(model); len(problems) != 0 {
		t.Errorf("expected a clean model, got %v", problems)
	}
}
