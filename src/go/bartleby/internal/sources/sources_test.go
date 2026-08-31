package sources

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

// Requirements: REQ_SRC_003, REQ_SRC_005
func TestToctreeUsesTitleAndStagingPath(t *testing.T) {
	got := Toctree(map[string]manifest.Source{
		"widget": {ArtifactPath: "artifacts/widget", Title: "Widget Guide"},
		"inline": {},
	})

	if !strings.Contains(got, ":caption: Widget Guide") {
		t.Errorf("missing the declared caption:\n%s", got)
	}
	if !strings.Contains(got, StagingDir+"/widget/index") {
		t.Errorf("artifact-backed source should point into %s:\n%s", StagingDir, got)
	}
	if !strings.Contains(got, "   inline/index") {
		t.Errorf("in-docs source should point at docs/<key>/index:\n%s", got)
	}
	if !strings.Contains(got, ":caption: inline") {
		t.Errorf("caption should default to the key:\n%s", got)
	}
	// Keys are emitted in sorted order so builds are reproducible.
	if strings.Index(got, "inline/index") > strings.Index(got, "widget/index") {
		t.Errorf("sources should be emitted in key order:\n%s", got)
	}
}

// Requirements: REQ_SRC_003
func TestToctreeEmpty(t *testing.T) {
	if got := Toctree(nil); got != "" {
		t.Errorf("want empty string, got %q", got)
	}
}

// Requirements: REQ_SRC_003_SPEC001
func TestInjectReplacesMarker(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n=======\n\n"+Marker+"\n\nIndexes and tables\n")

	original, err := Inject(index, map[string]manifest.Source{"widget": {}})
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}

	got := read(t, index)
	if strings.Contains(got, Marker) {
		t.Errorf("marker should have been replaced:\n%s", got)
	}
	if !strings.Contains(got, ".. toctree::") {
		t.Errorf("toctree missing:\n%s", got)
	}
	// The marker sits above "Indexes and tables", so the toctree must too.
	if strings.Index(got, ".. toctree::") > strings.Index(got, "Indexes and tables") {
		t.Errorf("toctree placed after the index section:\n%s", got)
	}
	if !strings.Contains(original, Marker) {
		t.Error("returned original should be the pre-injection content")
	}
}

// Requirements: REQ_SRC_003_SPEC001
func TestInjectBeforeIndexSection(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n=======\n\nIndexes and tables\n==================\n")

	if _, err := Inject(index, map[string]manifest.Source{"widget": {}}); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	got := read(t, index)
	if strings.Index(got, ".. toctree::") > strings.Index(got, "Indexes and tables") {
		t.Errorf("toctree should precede the index section:\n%s", got)
	}
}

// "Indices" is the Sphinx default spelling; both must be recognised.
// Requirements: REQ_SRC_003_SPEC001
func TestInjectRecognisesIndicesSpelling(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n=======\n\nIndices and tables\n==================\n")

	if _, err := Inject(index, map[string]manifest.Source{"widget": {}}); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	got := read(t, index)
	if strings.Index(got, ".. toctree::") > strings.Index(got, "Indices and tables") {
		t.Errorf("toctree should precede the index section:\n%s", got)
	}
}

// Requirements: REQ_SRC_003_SPEC001
func TestInjectAppendsWhenThereIsNoAnchor(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n=======\n")

	if _, err := Inject(index, map[string]manifest.Source{"widget": {}}); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	got := read(t, index)
	if !strings.HasPrefix(got, "Welcome\n=======\n") {
		t.Errorf("existing content should be preserved:\n%s", got)
	}
	if !strings.Contains(got, ".. toctree::") {
		t.Errorf("toctree missing:\n%s", got)
	}
}

// Requirements: REQ_SRC_003
func TestInjectNoSourcesIsANoOp(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	write(t, index, "Welcome\n")

	original, err := Inject(index, nil)
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	if original != "" {
		t.Errorf("want empty original, got %q", original)
	}
	if got := read(t, index); got != "Welcome\n" {
		t.Errorf("file should be untouched, got %q", got)
	}
}

// Requirements: REQ_SRC_004
func TestRestoreReturnsTheFileToItsOriginalState(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.rst")
	const content = "Welcome\n=======\n\nIndexes and tables\n"
	write(t, index, content)

	original, err := Inject(index, map[string]manifest.Source{"widget": {}})
	if err != nil {
		t.Fatalf("Inject: %v", err)
	}
	if err := Restore(index, original); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if got := read(t, index); got != content {
		t.Errorf("after restore the file is:\n%q\nwant:\n%q", got, content)
	}
}

// Requirements: REQ_SRC_002
func TestValidateSkipsMissingSources(t *testing.T) {
	repo := t.TempDir()
	docs := filepath.Join(repo, "docs")
	mkdir(t, filepath.Join(repo, "artifacts", "present", "docs"))
	mkdir(t, filepath.Join(docs, "inline"))

	var warnings strings.Builder
	valid := Validate(repo, docs, map[string]manifest.Source{
		"present": {ArtifactPath: "artifacts/present"},
		"absent":  {ArtifactPath: "artifacts/absent"},
		"inline":  {},
		"missing": {},
	}, &warnings)

	if len(valid) != 2 {
		t.Fatalf("got %d valid sources, want 2: %+v", len(valid), valid)
	}
	if _, ok := valid["present"]; !ok {
		t.Error("artifact-backed source with docs should be valid")
	}
	if _, ok := valid["inline"]; !ok {
		t.Error("in-docs source should be valid")
	}
	if !strings.Contains(warnings.String(), "absent") || !strings.Contains(warnings.String(), "missing") {
		t.Errorf("both skips should be warned about:\n%s", warnings.String())
	}
}

// Requirements: REQ_SRC_001_SPEC001
func TestValidateHonoursDocsRoot(t *testing.T) {
	repo := t.TempDir()
	docs := filepath.Join(repo, "docs")
	mkdir(t, filepath.Join(repo, "artifacts", "widget", "documentation"))

	valid := Validate(repo, docs, map[string]manifest.Source{
		"widget": {ArtifactPath: "artifacts/widget", DocsRoot: "documentation"},
	}, io.Discard)

	if len(valid) != 1 {
		t.Errorf("docs_root should be honoured, got %+v", valid)
	}
}

// Requirements: REQ_SRC_001, REQ_SRC_004
func TestStageAndCleanup(t *testing.T) {
	repo := t.TempDir()
	docs := filepath.Join(repo, "docs")
	mkdir(t, docs)

	src := filepath.Join(repo, "artifacts", "widget", "docs")
	mkdir(t, filepath.Join(src, "nested"))
	write(t, filepath.Join(src, "index.rst"), "Widget\n")
	write(t, filepath.Join(src, "nested", "page.rst"), "Page\n")

	staged, err := Stage(repo, docs, map[string]manifest.Source{
		"widget": {ArtifactPath: "artifacts/widget"},
		"inline": {},
	})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if len(staged) != 1 {
		t.Fatalf("only artifact-backed sources need staging, got %+v", staged)
	}

	if got := read(t, filepath.Join(docs, StagingDir, "widget", "index.rst")); got != "Widget\n" {
		t.Errorf("staged index = %q", got)
	}
	if got := read(t, filepath.Join(docs, StagingDir, "widget", "nested", "page.rst")); got != "Page\n" {
		t.Errorf("nested file not staged: %q", got)
	}

	if err := Cleanup(docs); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docs, StagingDir)); !os.IsNotExist(err) {
		t.Error("staging directory should be gone after Cleanup")
	}
}

// Re-staging must not merge into a stale directory.
// Requirements: REQ_SRC_001_SPEC002
func TestStageReplacesStaleStaging(t *testing.T) {
	repo := t.TempDir()
	docs := filepath.Join(repo, "docs")
	src := filepath.Join(repo, "artifacts", "widget", "docs")
	mkdir(t, src)
	write(t, filepath.Join(src, "index.rst"), "fresh\n")

	stale := filepath.Join(docs, StagingDir, "widget")
	mkdir(t, stale)
	write(t, filepath.Join(stale, "removed.rst"), "stale\n")

	if _, err := Stage(repo, docs, map[string]manifest.Source{
		"widget": {ArtifactPath: "artifacts/widget"},
	}); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	if _, err := os.Stat(filepath.Join(stale, "removed.rst")); !os.IsNotExist(err) {
		t.Error("stale staged files should be removed before copying")
	}
}

// Requirements: REQ_SRC_004
func TestCleanupWithNothingStaged(t *testing.T) {
	if err := Cleanup(t.TempDir()); err != nil {
		t.Errorf("Cleanup on a clean tree should succeed, got %v", err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
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
