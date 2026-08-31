package gather

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspace lays out a parent directory containing hmd-docs-bartleby, the demos
// library, and any extra repos, mirroring what gather expects on disk.
func workspace(t *testing.T, repos ...string) (docsRepo string) {
	t.Helper()
	parent := t.TempDir()

	docsRepo = filepath.Join(parent, DocsRepo)
	write(t, filepath.Join(docsRepo, "docs", "index.rst"), "Old index\n")

	write(t, filepath.Join(parent, DemosRepo, "docs", "index.rst"),
		"Demos\n=====\n\nIndexes and tables\n")

	for _, repo := range repos {
		write(t, filepath.Join(parent, repo, "docs", "index.rst"), repo+" docs\n")
	}
	return docsRepo
}

// Requirements: REQ_GATH_003
func TestReposGathersSiblingDocs(t *testing.T) {
	docsRepo := workspace(t, "hmd-lib-widget", "hmd-cli-thing")

	// Something stale that must be cleared out.
	write(t, filepath.Join(docsRepo, "docs", "leftover", "old.rst"), "old\n")

	if err := Repos(docsRepo, "hmd-lib-widget,hmd-cli-thing", io.Discard); err != nil {
		t.Fatalf("Repos: %v", err)
	}

	docs := filepath.Join(docsRepo, "docs")
	if _, err := os.Stat(filepath.Join(docs, "leftover")); !os.IsNotExist(err) {
		t.Error("stale docs directories should be removed")
	}

	index := read(t, filepath.Join(docs, "index.rst"))
	if !strings.Contains(index, "Demos") {
		t.Errorf("index should come from the demos library:\n%s", index)
	}
	for _, repo := range []string{"hmd-lib-widget", "hmd-cli-thing"} {
		if got := read(t, filepath.Join(docs, repo, "index.rst")); got != repo+" docs\n" {
			t.Errorf("%s docs not copied, got %q", repo, got)
		}
		if !strings.Contains(index, repo+"/index.rst") {
			t.Errorf("index missing a toctree entry for %s:\n%s", repo, index)
		}
	}
	// Entries belong above the trailing index section.
	if strings.Index(index, "hmd-lib-widget/index.rst") > strings.Index(index, "Indexes and tables") {
		t.Errorf("entries inserted after the index section:\n%s", index)
	}
}

// Requirements: REQ_GATH_001
func TestReposRejectsWrongDirectory(t *testing.T) {
	parent := t.TempDir()
	wrong := filepath.Join(parent, "some-other-repo")
	write(t, filepath.Join(wrong, "docs", "index.rst"), "x\n")

	err := Repos(wrong, "hmd-lib-widget", io.Discard)
	if err == nil {
		t.Fatal("expected an error outside hmd-docs-bartleby")
	}
	if !strings.Contains(err.Error(), DocsRepo) {
		t.Errorf("error should name the required repo, got: %v", err)
	}
}

// Requirements: REQ_GATH_001
func TestReposRequiresDemosLibrary(t *testing.T) {
	parent := t.TempDir()
	docsRepo := filepath.Join(parent, DocsRepo)
	write(t, filepath.Join(docsRepo, "docs", "index.rst"), "x\n")

	err := Repos(docsRepo, "hmd-lib-widget", io.Discard)
	if err == nil {
		t.Fatal("expected an error without the demos library")
	}
	if !strings.Contains(err.Error(), DemosRepo) {
		t.Errorf("error should name the missing library, got: %v", err)
	}
}

// A typo in the repo list must fail before docs/ is emptied.
// Requirements: REQ_GATH_002
func TestReposValidatesBeforeDeleting(t *testing.T) {
	docsRepo := workspace(t, "hmd-lib-widget")
	keep := filepath.Join(docsRepo, "docs", "keep", "page.rst")
	write(t, keep, "keep me\n")

	err := Repos(docsRepo, "hmd-lib-widget,hmd-lib-typo", io.Discard)
	if err == nil {
		t.Fatal("expected an error for a repo that is not checked out")
	}
	if !strings.Contains(err.Error(), "hmd-lib-typo") {
		t.Errorf("error should name the missing repo, got: %v", err)
	}
	if _, statErr := os.Stat(keep); statErr != nil {
		t.Errorf("docs/ must be untouched when validation fails: %v", statErr)
	}
}

// Requirements: REQ_GATH_004
func TestReposEmptyListIsANoOp(t *testing.T) {
	docsRepo := workspace(t)
	before := read(t, filepath.Join(docsRepo, "docs", "index.rst"))

	if err := Repos(docsRepo, "", io.Discard); err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if after := read(t, filepath.Join(docsRepo, "docs", "index.rst")); after != before {
		t.Error("an empty gather list should change nothing")
	}
}

// Requirements: REQ_GATH_003
func TestAddToIndexWithoutMarker(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n=======\n")

	if err := AddToIndex(index, "hmd-lib-widget"); err != nil {
		t.Fatalf("AddToIndex: %v", err)
	}

	got := read(t, index)
	if !strings.Contains(got, "   hmd-lib-widget/index.rst") {
		t.Errorf("entry missing:\n%s", got)
	}
	if !strings.HasPrefix(got, "Welcome\n=======\n") {
		t.Errorf("existing content should be preserved:\n%s", got)
	}
}

// Requirements: REQ_GATH_003
func TestAddToIndexIndicesSpelling(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n\nIndices and tables\n")

	if err := AddToIndex(index, "hmd-lib-widget"); err != nil {
		t.Fatalf("AddToIndex: %v", err)
	}

	got := read(t, index)
	if strings.Index(got, "hmd-lib-widget/index.rst") > strings.Index(got, "Indices and tables") {
		t.Errorf("entry should precede the index section:\n%s", got)
	}
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

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
