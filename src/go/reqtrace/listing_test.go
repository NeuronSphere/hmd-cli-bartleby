package reqtrace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }

const listingDocs = `Things
======

.. req:: Covered requirement
    :id: HMD_CLI_ALPHA_REQ_A_001
    :status: implemented

.. req:: Uncovered requirement
    :id: HMD_CLI_ALPHA_REQ_A_002
    :status: Completed

.. spec:: A draft spec
    :id: HMD_CLI_ALPHA_REQ_A_002_SPEC001
    :links: HMD_CLI_ALPHA_REQ_A_002
    :status: draft

.. req:: Odd status
    :id: HMD_CLI_ALPHA_REQ_A_003
    :status: recorded-difference
`

const listingGoTest = `package thing

import "testing"

// Requirements: REQ_A_001
func TestThing(t *testing.T) {}
`

const listingRobot = `*** Test Cases ***
Robot Covers It
    [Tags]    REQ_A_001
    Log    hi
`

// writeListingRepo lays out a repository named name under parent, with the
// standard docs, a Go test and a Robot suite.
func writeListingRepo(t *testing.T, parent, dir, name string) string {
	t.Helper()
	root := filepath.Join(parent, dir)
	write(t, filepath.Join(root, "meta-data", "manifest.json"), `{"name": "`+name+`"}`)
	write(t, filepath.Join(root, "docs", "requirements", "things.rst"), listingDocs)
	write(t, filepath.Join(root, "src", "go", "thing", "thing_test.go"), listingGoTest)
	write(t, filepath.Join(root, "test", "things.robot"), listingRobot)
	return root
}

func needByID(l Listing, repo, id string) (ListedNeed, bool) {
	for _, n := range l.Needs {
		if n.Repo == repo && n.ID == id {
			return n, true
		}
	}
	return ListedNeed{}, false
}

func hasProblem(l Listing, kind, contains string) bool {
	for _, p := range l.Problems {
		if p.Kind == kind && strings.Contains(p.Message+p.ID, contains) {
			return true
		}
	}
	return false
}

// Requirements: REQ_LIST_001, REQ_LIST_002
func TestListRepoListsNeedsWithCoverage(t *testing.T) {
	root := writeListingRepo(t, t.TempDir(), "alpha", "hmd-cli-alpha")

	l, err := ListRepo(root, ListOptions{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	if l.Schema != ListingSchema || l.Generated != "2026-11-16T12:00:00Z" || l.Workspace != "" {
		t.Errorf("header = schema %d, generated %q, workspace %q", l.Schema, l.Generated, l.Workspace)
	}
	if len(l.Needs) != 4 {
		t.Fatalf("listed %d needs, want 4", len(l.Needs))
	}

	covered, _ := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_REQ_A_001")
	if covered.Type != "req" || covered.Source != "requirements" || covered.File != "docs/requirements/things.rst" || covered.Line != 4 {
		t.Errorf("covered need = %+v", covered)
	}
	if len(covered.CoveredBy) != 2 {
		t.Fatalf("covered_by = %+v, want the Go and the Robot test", covered.CoveredBy)
	}
	kinds := covered.CoveredBy[0].Kind + "," + covered.CoveredBy[1].Kind
	if kinds != "go,robot" {
		t.Errorf("cover kinds = %s", kinds)
	}
	if covered.CoveredBy[0].Test != "thing.TestThing" || covered.CoveredBy[1].Test != "things: Robot Covers It" {
		t.Errorf("cover titles = %+v", covered.CoveredBy)
	}

	spec, _ := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_REQ_A_002_SPEC001")
	if spec.Stage != "draft" || len(spec.CoveredBy) != 0 || spec.Digest == "" {
		t.Errorf("draft spec = %+v", spec)
	}
}

// Requirements: REQ_LIST_001
func TestListRepoReadsEveryDocsFolderButNotTheGeneratedPage(t *testing.T) {
	root := writeListingRepo(t, t.TempDir(), "alpha", "hmd-cli-alpha")
	write(t, filepath.Join(root, "docs", "proposals", "NERD001.rst"), ".. req:: A NERD\n    :id: HMD_CLI_ALPHA_NERD001\n    :status: proposed\n")
	write(t, filepath.Join(root, "docs", "requirements", GeneratedFile), ".. test:: generated\n    :id: HMD_CLI_ALPHA_TEST_GO_1\n")

	l, err := ListRepo(root, ListOptions{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	nerd, ok := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_NERD001")
	if !ok || nerd.Source != "proposals" {
		t.Errorf("NERD = %+v, %v; want it listed with source proposals", nerd, ok)
	}
	if _, ok := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_TEST_GO_1"); ok {
		t.Error("the generated traceability page was listed")
	}
}

// Requirements: REQ_LIST_003
func TestStagesNormaliseStatuses(t *testing.T) {
	root := writeListingRepo(t, t.TempDir(), "alpha", "hmd-cli-alpha")

	l, err := ListRepo(root, ListOptions{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	completed, _ := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_REQ_A_002")
	if completed.Status != "Completed" || completed.Stage != "implemented" {
		t.Errorf("Completed → %q, want implemented", completed.Stage)
	}
	odd, _ := needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_REQ_A_003")
	if odd.Stage != "unknown" || !hasProblem(l, "unknown-status", "HMD_CLI_ALPHA_REQ_A_003") {
		t.Errorf("odd status → %q with problems %+v", odd.Stage, l.Problems)
	}

	// A manifest can teach the table a new status.
	write(t, filepath.Join(root, "meta-data", "manifest.json"), `{"name": "hmd-cli-alpha",
		"bartleby": {"roots": {"index": {"config": {"needs_status_stages": {"recorded-difference": "closed"}}}}}}`)
	l, err = ListRepo(root, ListOptions{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	odd, _ = needByID(l, "hmd-cli-alpha", "HMD_CLI_ALPHA_REQ_A_003")
	if odd.Stage != "closed" || hasProblem(l, "unknown-status", "") {
		t.Errorf("with a manifest mapping, stage = %q, problems %+v", odd.Stage, l.Problems)
	}
}

// Requirements: REQ_LIST_003
func TestStageMatchingIgnoresCaseSpacesAndHyphens(t *testing.T) {
	var cfg repoConfig
	for _, status := range []string{"In Progress", "in-progress", " IN_PROGRESS "} {
		if stage, ok := cfg.stageOf(status); !ok || stage != StageInProgress {
			t.Errorf("%q → %q (%v), want in_progress", status, stage, ok)
		}
	}
	if _, ok := cfg.stageOf("partly implemented"); ok {
		t.Error(`"partly implemented" was recognised; it is not in the table`)
	}
}

// Requirements: REQ_LIST_003
func TestStrictStatusesFailTheTraceabilityCheck(t *testing.T) {
	root := writeRepo(t, `.. req:: R
    :id: HMD_CLI_BARTLEBY_REQ_X_001
    :status: whatever
    :tags: trace-exempt
`, "", "")

	lenient := Validate(mustLoad(t, root))
	for _, p := range lenient {
		if p.Kind == ProblemUnknownStatus {
			t.Fatalf("unknown status reported without needs_status_strict: %v", p)
		}
	}

	write(t, filepath.Join(root, "meta-data", "manifest.json"), `{"name": "hmd-cli-bartleby",
		"bartleby": {"roots": {"index": {"config": {"needs_status_strict": true}}}}}`)
	strict := Validate(mustLoad(t, root))
	found := false
	for _, p := range strict {
		found = found || p.Kind == ProblemUnknownStatus
	}
	if !found {
		t.Errorf("needs_status_strict did not report the unknown status: %v", strict)
	}
}

func mustLoad(t *testing.T, root string) Model {
	t.Helper()
	m, err := Load(DefaultLayout(root))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// writeWorkspace lays out: two manifest repositories, one with docs and no
// manifest, one repository with no needs, a nested repository inside a
// container folder, and a hidden folder that must be ignored.
func writeWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	writeListingRepo(t, ws, "alpha", "hmd-cli-alpha")
	write(t, filepath.Join(ws, "beta", "meta-data", "manifest.json"), `{"name": "hmd-ms-beta"}`)
	write(t, filepath.Join(ws, "beta", "docs", "proposals", "NERD001.rst"), `.. req:: Beta NERD
    :id: NERD001
    :status: proposed

.. spec:: Links across repositories
    :id: HMD_MS_BETA_NERD001_SPEC001
    :links: NERD001, HMD_CLI_ALPHA_REQ_A_001, NERD009
    :status: proposed
`)
	write(t, filepath.Join(ws, "gamma", "docs", "requirements", "g.rst"), `.. req:: Gamma links to a shared name
    :id: GAMMA_REQ_001
    :links: NERD001
    :status: proposed

.. req:: Gamma also defines NERD001
    :id: NERD001
    :status: proposed

.. req:: Gamma twice
    :id: GAMMA_REQ_001
    :status: proposed
`)
	write(t, filepath.Join(ws, "delta", "meta-data", "manifest.json"), `{"name": "hmd-lib-delta"}`)
	write(t, filepath.Join(ws, "group", "epsilon", "meta-data", "manifest.json"), `{"name": "hmd-ui-epsilon"}`)
	write(t, filepath.Join(ws, "group", "epsilon", "docs", "requirements", "e.rst"), `.. req:: Epsilon links NERD001
    :id: EPS_REQ_001
    :links: NERD001
    :status: implemented
`)
	write(t, filepath.Join(ws, ".hidden", "docs", "requirements", "h.rst"), ".. req:: Hidden\n    :id: HIDDEN_1\n")
	return ws
}

func listWS(t *testing.T, ws string, o ListOptions) (Listing, refreshStats) {
	t.Helper()
	if o.CacheDir == "" {
		o.CacheDir = filepath.Join(t.TempDir(), "cache")
	}
	if o.Now == nil {
		o.Now = fixedNow
	}
	l, stats, err := listWorkspace(ws, o)
	if err != nil {
		t.Fatal(err)
	}
	return l, stats
}

// Requirements: REQ_LIST_004
func TestWorkspaceDiscoversRepositories(t *testing.T) {
	ws := writeWorkspace(t)
	l, _ := listWS(t, ws, ListOptions{})

	if l.Workspace != filepath.Base(ws) {
		t.Errorf("workspace = %q, want the directory name only", l.Workspace)
	}
	repos := map[string]bool{}
	for _, n := range l.Needs {
		repos[n.Repo] = true
	}
	for _, want := range []string{"hmd-cli-alpha", "hmd-ms-beta", "gamma", "hmd-ui-epsilon"} {
		if !repos[want] {
			t.Errorf("repository %s not listed; got %v", want, repos)
		}
	}
	if repos[".hidden"] {
		t.Error("a hidden directory was listed")
	}
	s := l.Summary
	if s.Dirs != 5 || s.ReposWithNeeds != 4 || s.ReposWithoutNeeds != 1 {
		t.Errorf("summary = %+v, want 5 dirs (alpha beta gamma delta group), 4 with needs, 1 without", s)
	}
}

// Requirements: REQ_LIST_005
func TestWorkspaceResolvesLinksAcrossRepositories(t *testing.T) {
	ws := writeWorkspace(t)
	l, _ := listWS(t, ws, ListOptions{})

	// beta's NERD001 link resolves inside beta; its link to alpha resolves to
	// the one sibling defining it; NERD009 is defined nowhere.
	if hasProblem(l, "unknown-link", "HMD_CLI_ALPHA_REQ_A_001") || hasProblem(l, "ambiguous-link", "HMD_MS_BETA_NERD001_SPEC001") {
		t.Errorf("resolvable links reported: %+v", l.Problems)
	}
	if !hasProblem(l, "unknown-link", "NERD009") {
		t.Errorf("NERD009 not reported unknown: %+v", l.Problems)
	}
	// epsilon links NERD001, which beta and gamma both define.
	if !hasProblem(l, "ambiguous-link", "EPS_REQ_001") {
		t.Errorf("ambiguous NERD001 from epsilon not reported: %+v", l.Problems)
	}
	// gamma links its own NERD001 — local first, so not ambiguous.
	if hasProblem(l, "ambiguous-link", "GAMMA_REQ_001") {
		t.Errorf("gamma's own NERD001 reported ambiguous: %+v", l.Problems)
	}
	if !hasProblem(l, "duplicate-id", "GAMMA_REQ_001") {
		t.Errorf("duplicate GAMMA_REQ_001 not reported: %+v", l.Problems)
	}
}

// Requirements: REQ_LIST_004
func TestWorkspaceSeparatesFoldersThatShareAManifestName(t *testing.T) {
	ws := t.TempDir()
	writeListingRepo(t, ws, "alpha", "hmd-cli-alpha")
	writeListingRepo(t, ws, "alpha-copy", "hmd-cli-alpha")

	l, _ := listWS(t, ws, ListOptions{})
	repos := map[string]int{}
	for _, n := range l.Needs {
		repos[n.Repo]++
	}
	if repos["hmd-cli-alpha@alpha"] != 4 || repos["hmd-cli-alpha@alpha-copy"] != 4 {
		t.Errorf("needs per repo = %v, want each folder listed apart", repos)
	}
	if hasProblem(l, "duplicate-id", "") {
		t.Errorf("the two folders' needs were merged: %+v", l.Problems)
	}
	if !hasProblem(l, "duplicate-repo", "hmd-cli-alpha") {
		t.Errorf("shared manifest name not reported: %+v", l.Problems)
	}

	// A folder-name filter still selects one of them.
	l, _ = listWS(t, ws, ListOptions{Repos: []string{"alpha-copy"}})
	if len(l.Needs) != 4 || l.Needs[0].Repo != "hmd-cli-alpha@alpha-copy" {
		t.Errorf("-repo alpha-copy = %d needs from %v", len(l.Needs), repos)
	}
}

// Requirements: REQ_LIST_006
func TestWorkspaceFiltersAndSummary(t *testing.T) {
	ws := writeWorkspace(t)

	l, _ := listWS(t, ws, ListOptions{Repos: []string{"alpha"}, Stages: []string{"draft"}})
	if len(l.Needs) != 1 || l.Needs[0].ID != "HMD_CLI_ALPHA_REQ_A_002_SPEC001" {
		t.Errorf("-repo alpha -stage draft = %+v", l.Needs)
	}

	l, _ = listWS(t, ws, ListOptions{Repos: []string{"hmd-cli-alpha"}, Open: true})
	open := map[string]bool{}
	for _, n := range l.Needs {
		open[n.ID] = true
	}
	// A_001 is implemented and covered: not open. A_002 is implemented with no
	// test: open. The draft and the unknown stage: open.
	want := []string{"HMD_CLI_ALPHA_REQ_A_002", "HMD_CLI_ALPHA_REQ_A_002_SPEC001", "HMD_CLI_ALPHA_REQ_A_003"}
	if len(open) != len(want) || open["HMD_CLI_ALPHA_REQ_A_001"] {
		t.Errorf("-open = %v, want %v", open, want)
	}

	l, _ = listWS(t, ws, ListOptions{Types: []string{"spec"}})
	for _, n := range l.Needs {
		if n.Type != "spec" {
			t.Errorf("-type spec listed a %s", n.Type)
		}
	}

	l, _ = listWS(t, ws, ListOptions{Summary: true})
	if len(l.Needs) != 0 || len(l.Summary.Repos) != 4 {
		t.Fatalf("-summary: %d needs, %d repos; want none and 4", len(l.Needs), len(l.Summary.Repos))
	}
	for _, r := range l.Summary.Repos {
		if r.Repo == "hmd-cli-alpha" && (r.Needs != 4 || r.Covered != 1 || r.Open != 3 || r.ByStage["implemented"] != 2) {
			t.Errorf("alpha summary = %+v", r)
		}
	}
}

// Requirements: REQ_LIST_007, REQ_LIST_008
func TestIndexRefreshesOnlyWhatChanged(t *testing.T) {
	ws := writeWorkspace(t)
	cache := filepath.Join(t.TempDir(), "cache")
	o := ListOptions{CacheDir: cache, Changes: true}

	first, stats := listWS(t, ws, o)
	// Changes are keyed by repository and ID, so gamma's duplicate counts once.
	if keys := signatures(first.Needs); stats.Parsed == 0 || len(first.Changes.Added) != len(keys) {
		t.Fatalf("cold run: parsed %d files, added %d of %d keys", stats.Parsed, len(first.Changes.Added), len(keys))
	}

	second, stats := listWS(t, ws, o)
	if stats.Parsed != 0 {
		t.Errorf("warm run with nothing changed parsed %d files", stats.Parsed)
	}
	if n := len(second.Changes.Added) + len(second.Changes.Changed) + len(second.Changes.Removed); n != 0 {
		t.Errorf("warm run reported %d changes: %+v", n, second.Changes)
	}
	if second.Changes.Since != first.Generated {
		t.Errorf("since = %q, want the previous refresh %q", second.Changes.Since, first.Generated)
	}

	// Edit one need: one file parsed, one need changed.
	path := filepath.Join(ws, "alpha", "docs", "requirements", "things.rst")
	edited := strings.Replace(listingDocs, ":status: draft", ":status: proposed", 1)
	write(t, path, edited)
	bumpMtime(t, path)
	third, stats := listWS(t, ws, o)
	if stats.Parsed != 1 {
		t.Errorf("after one edit, parsed %d files, want 1", stats.Parsed)
	}
	if len(third.Changes.Changed) != 1 || third.Changes.Changed[0].ID != "HMD_CLI_ALPHA_REQ_A_002_SPEC001" {
		t.Errorf("changes after edit = %+v", third.Changes)
	}
}

// Requirements: REQ_LIST_007, REQ_LIST_008
func TestIndexFindsNewRemovedAndRenamedFiles(t *testing.T) {
	ws := writeWorkspace(t)
	o := ListOptions{CacheDir: filepath.Join(t.TempDir(), "cache"), Changes: true}
	listWS(t, ws, o)

	// A new file in a new docs folder that did not exist before.
	write(t, filepath.Join(ws, "delta", "docs", "proposals", "NERD001.rst"), ".. req:: Delta NERD\n    :id: DELTA_NERD001\n    :status: draft\n")
	l, _ := listWS(t, ws, o)
	if len(l.Changes.Added) != 1 || l.Changes.Added[0].ID != "DELTA_NERD001" || l.Changes.Added[0].Repo != "hmd-lib-delta" {
		t.Fatalf("new docs folder: changes = %+v", l.Changes)
	}

	// Rename it: removed under the old name's file, but the need itself moved,
	// so it is changed (same key), not added and removed.
	oldPath := filepath.Join(ws, "delta", "docs", "proposals", "NERD001.rst")
	newPath := filepath.Join(ws, "delta", "docs", "proposals", "NERD001_renamed.rst")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	l, _ = listWS(t, ws, o)
	n, ok := needByID(l, "hmd-lib-delta", "DELTA_NERD001")
	if !ok || n.File != "docs/proposals/NERD001_renamed.rst" {
		t.Errorf("after rename, need = %+v, %v", n, ok)
	}
	if len(l.Changes.Added)+len(l.Changes.Removed) != 0 {
		t.Errorf("a rename reported added/removed: %+v", l.Changes)
	}

	// Delete it.
	if err := os.Remove(newPath); err != nil {
		t.Fatal(err)
	}
	l, _ = listWS(t, ws, o)
	if len(l.Changes.Removed) != 1 || l.Changes.Removed[0].ID != "DELTA_NERD001" {
		t.Errorf("after delete, changes = %+v", l.Changes)
	}

	// A new Go test anywhere under src/go adds coverage.
	write(t, filepath.Join(ws, "alpha", "src", "go", "deep", "er", "more_test.go"),
		"package er\n\nimport \"testing\"\n\n// Requirements: REQ_A_002\nfunc TestMore(t *testing.T) {}\n")
	l, _ = listWS(t, ws, o)
	if len(l.Changes.Changed) != 1 || l.Changes.Changed[0].ID != "HMD_CLI_ALPHA_REQ_A_002" {
		t.Errorf("new covering test: changes = %+v", l.Changes)
	}
}

// Requirements: REQ_LIST_007
func TestIndexFindsNewAndRemovedRepositories(t *testing.T) {
	ws := writeWorkspace(t)
	o := ListOptions{CacheDir: filepath.Join(t.TempDir(), "cache"), Changes: true}
	listWS(t, ws, o)

	write(t, filepath.Join(ws, "zeta", "docs", "requirements", "z.rst"), ".. req:: Zeta\n    :id: ZETA_1\n    :status: proposed\n")
	write(t, filepath.Join(ws, "group", "eta", "docs", "requirements", "e.rst"), ".. req:: Eta\n    :id: ETA_1\n    :status: proposed\n")
	l, _ := listWS(t, ws, o)
	added := map[string]bool{}
	for _, k := range l.Changes.Added {
		added[k.Repo+"/"+k.ID] = true
	}
	if !added["zeta/ZETA_1"] || !added["eta/ETA_1"] {
		t.Errorf("new repositories not found: %+v", l.Changes)
	}

	if err := os.RemoveAll(filepath.Join(ws, "beta")); err != nil {
		t.Fatal(err)
	}
	l, _ = listWS(t, ws, o)
	if len(l.Changes.Removed) != 2 {
		t.Errorf("removing beta: changes = %+v, want its 2 needs removed", l.Changes)
	}
}

// Requirements: REQ_LIST_007
func TestIndexRewalksARepositoryWhoseHeadMoved(t *testing.T) {
	ws := writeWorkspace(t)
	gitDir := filepath.Join(ws, "alpha", ".git")
	write(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(gitDir, "refs", "heads", "main"), "1111111111111111111111111111111111111111\n")
	o := ListOptions{CacheDir: filepath.Join(t.TempDir(), "cache")}
	listWS(t, ws, o)

	write(t, filepath.Join(gitDir, "refs", "heads", "main"), "2222222222222222222222222222222222222222\n")
	_, stats := listWS(t, ws, o)
	if stats.Rewalked != 1 {
		t.Errorf("rewalked %d repositories after HEAD moved, want 1", stats.Rewalked)
	}
}

// Requirements: REQ_LIST_007
func TestIndexSurvivesCorruptionAndRebuild(t *testing.T) {
	ws := writeWorkspace(t)
	cache := filepath.Join(t.TempDir(), "cache")
	o := ListOptions{CacheDir: cache}
	want, _ := listWS(t, ws, o)

	files, _ := filepath.Glob(filepath.Join(cache, "*.json"))
	if len(files) != 1 {
		t.Fatalf("index files = %v, want exactly one", files)
	}
	write(t, files[0], "{not json")
	got, stats := listWS(t, ws, o)
	if stats.Parsed == 0 || len(got.Needs) != len(want.Needs) {
		t.Errorf("corrupt index: parsed %d, listed %d of %d", stats.Parsed, len(got.Needs), len(want.Needs))
	}

	_, stats = listWS(t, ws, ListOptions{CacheDir: cache, Rebuild: true})
	if stats.Parsed == 0 {
		t.Error("-rebuild reused the index")
	}

	// An unwritable cache still lists, with a warning.
	blocker := filepath.Join(t.TempDir(), "file")
	write(t, blocker, "x")
	l, _, err := listWorkspace(ws, ListOptions{CacheDir: filepath.Join(blocker, "sub"), Now: fixedNow})
	if err != nil || len(l.Needs) != len(want.Needs) || len(l.Warnings) == 0 {
		t.Errorf("unwritable cache: err %v, %d needs, warnings %v", err, len(l.Needs), l.Warnings)
	}
}

// Requirements: REQ_LIST_009
func TestDiffComparesTwoListings(t *testing.T) {
	ws := t.TempDir()
	root := writeListingRepo(t, ws, "alpha", "hmd-cli-alpha")
	before, err := ListRepo(root, ListOptions{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	again, _ := ListRepo(root, ListOptions{Now: fixedNow})
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(again)
	if !bytes.Equal(a, b) {
		t.Error("two listings of the same tree differ")
	}
	if bytes.Contains(a, []byte(ws)) {
		t.Error("a single-repository listing contains an absolute path")
	}

	write(t, filepath.Join(root, "docs", "requirements", "things.rst"),
		strings.Replace(listingDocs, "Odd status", "Odd status, retitled", 1)+".. req:: New\n    :id: HMD_CLI_ALPHA_REQ_A_009\n    :status: proposed\n")
	after, _ := ListRepo(root, ListOptions{Now: fixedNow})

	changes, err := DiffListings(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes.Added) != 1 || changes.Added[0].ID != "HMD_CLI_ALPHA_REQ_A_009" ||
		len(changes.Changed) != 1 || changes.Changed[0].ID != "HMD_CLI_ALPHA_REQ_A_003" || len(changes.Removed) != 0 {
		t.Errorf("diff = %+v", changes)
	}

	after.Schema = ListingSchema + 1
	if _, err := DiffListings(before, after); err == nil {
		t.Error("listings with different schemas were compared")
	}
}

// Requirements: REQ_LIST_001, REQ_LIST_004
func TestRunPrintsJSONAndDefaultsToTheWorkspace(t *testing.T) {
	ws := writeWorkspace(t)
	t.Setenv("REQTRACE_CACHE_DIR", filepath.Join(t.TempDir(), "cache"))
	t.Chdir(ws)

	var out, errOut bytes.Buffer
	if err := Run(RunOptions{List: &ListOptions{Summary: true, Now: fixedNow}, Out: &out, Err: &errOut}); err != nil {
		t.Fatalf("Run: %v\n%s", err, errOut.String())
	}
	var l Listing
	if err := json.Unmarshal(out.Bytes(), &l); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if l.Workspace == "" || l.Summary.ReposWithNeeds != 4 {
		t.Errorf("default workspace listing = %+v", l.Summary)
	}

	// Inside a repository, the same call lists that repository alone.
	t.Chdir(filepath.Join(ws, "alpha", "docs"))
	out.Reset()
	if err := Run(RunOptions{List: &ListOptions{Now: fixedNow}, Out: &out, Err: &errOut}); err != nil {
		t.Fatal(err)
	}
	l = Listing{}
	if err := json.Unmarshal(out.Bytes(), &l); err != nil {
		t.Fatal(err)
	}
	if l.Workspace != "" || len(l.Needs) != 4 {
		t.Errorf("in-repository listing: workspace %q, %d needs", l.Workspace, len(l.Needs))
	}
}

// Requirements: REQ_LIST_009
func TestRunDiffReadsTwoFiles(t *testing.T) {
	root := writeListingRepo(t, t.TempDir(), "alpha", "hmd-cli-alpha")
	l, _ := ListRepo(root, ListOptions{Now: fixedNow})
	data, _ := json.Marshal(l)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.json"), string(data))
	l.Needs = l.Needs[1:]
	data, _ = json.Marshal(l)
	write(t, filepath.Join(dir, "b.json"), string(data))

	var out bytes.Buffer
	err := Run(RunOptions{Diff: []string{filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json")}, Out: &out})
	if err != nil {
		t.Fatal(err)
	}
	var c Changes
	if err := json.Unmarshal(out.Bytes(), &c); err != nil || len(c.Removed) != 1 {
		t.Errorf("diff output = %s (%v)", out.String(), err)
	}
}

// bumpMtime moves a file's modified time forward, so a test does not depend on
// the file system's timestamp resolution.
func bumpMtime(t *testing.T, path string) {
	t.Helper()
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
}
