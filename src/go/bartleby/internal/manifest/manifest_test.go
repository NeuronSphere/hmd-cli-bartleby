package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeManifest creates a repo skeleton containing the given manifest JSON.
func writeManifest(t *testing.T, contents string) string {
	t.Helper()
	repo := t.TempDir()
	metaData := filepath.Join(repo, "meta-data")
	if err := os.MkdirAll(metaData, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaData, "manifest.json"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// Requirements: REQ_MAN_001, REQ_MAN_003
func TestReadStringBuilders(t *testing.T) {
	repo := writeManifest(t, `{
		"name": "hmd-docs-example",
		"bartleby": {
			"roots": {
				"guide": {"root_doc": "guide", "builders": ["html", "pdf"]}
			}
		}
	}`)

	m, err := Read(repo)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if m.Name != "hmd-docs-example" {
		t.Errorf("Name = %q, want hmd-docs-example", m.Name)
	}

	root := m.Bartleby.Roots["guide"]
	if len(root.Builders) != 2 {
		t.Fatalf("got %d builders, want 2", len(root.Builders))
	}
	if root.Builders[0].Shell != "html" || root.Builders[1].Shell != "pdf" {
		t.Errorf("builders = %+v, want html then pdf", root.Builders)
	}
	if root.Builders[0].Config != nil {
		t.Errorf("bare builder should carry no config, got %+v", root.Builders[0].Config)
	}
}

// The Python CLI accepted object-form builders, so a manifest using them must
// not fail to parse here.
// Requirements: REQ_MAN_003
func TestReadObjectBuilders(t *testing.T) {
	repo := writeManifest(t, `{
		"bartleby": {
			"roots": {
				"api": {
					"builders": [
						"html",
						{"shell": "pdf", "config": {"theme": "classic", "depth": 3}}
					]
				}
			}
		}
	}`)

	m, err := Read(repo)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	builders := m.Bartleby.Roots["api"].Builders
	if len(builders) != 2 {
		t.Fatalf("got %d builders, want 2", len(builders))
	}
	if builders[1].Shell != "pdf" {
		t.Errorf("second builder shell = %q, want pdf", builders[1].Shell)
	}
	if got := builders[1].Config["theme"]; got != "classic" {
		t.Errorf("config theme = %v, want classic", got)
	}
}

// Requirements: REQ_MAN_003_SPEC001
func TestReadObjectBuilderMissingShell(t *testing.T) {
	repo := writeManifest(t, `{"bartleby": {"roots": {"api": {"builders": [{"config": {}}]}}}}`)

	if _, err := Read(repo); err == nil {
		t.Fatal("expected an error for a builder object with no shell")
	}
}

// Requirements: REQ_MAN_001
func TestReadMissingManifest(t *testing.T) {
	m, err := Read(t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if m == nil {
		t.Fatal("a missing manifest should still return an empty manifest")
	}
	if len(m.Bartleby.Roots) != 0 {
		t.Errorf("expected no roots, got %+v", m.Bartleby.Roots)
	}
}

// A malformed manifest must be reported rather than silently treated as absent,
// which is what the first implementation did.
// Requirements: REQ_MAN_002
func TestReadMalformedManifest(t *testing.T) {
	repo := writeManifest(t, `{"bartleby": {"roots": `)

	_, err := Read(repo)
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("malformed JSON must not be reported as a missing manifest")
	}
}

// Requirements: REQ_SEL_006
func TestRootDocDefault(t *testing.T) {
	if got := (Root{}).Doc(); got != DefaultRootDoc {
		t.Errorf("Doc() = %q, want %q", got, DefaultRootDoc)
	}
	if got := (Root{RootDoc: "guide"}).Doc(); got != "guide" {
		t.Errorf("Doc() = %q, want guide", got)
	}
}

// Requirements: REQ_SRC_001_SPEC001
func TestSourceDocsDefault(t *testing.T) {
	if got := (Source{}).Docs(); got != "docs" {
		t.Errorf("Docs() = %q, want docs", got)
	}
	if got := (Source{DocsRoot: "documentation"}).Docs(); got != "documentation" {
		t.Errorf("Docs() = %q, want documentation", got)
	}
}

// Requirements: REQ_MAN_005
func TestReadVersion(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		repo := t.TempDir()
		mustWrite(t, filepath.Join(repo, "meta-data", "VERSION"), "1.4\n")

		got, err := ReadVersion(repo)
		if err != nil {
			t.Fatalf("ReadVersion: %v", err)
		}
		if got != "1.4" {
			t.Errorf("version = %q, want 1.4", got)
		}
	})

	t.Run("missing is not an error", func(t *testing.T) {
		got, err := ReadVersion(t.TempDir())
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
		if got != DefaultVersion {
			t.Errorf("version = %q, want %q", got, DefaultVersion)
		}
	})

	t.Run("empty file reports a problem", func(t *testing.T) {
		repo := t.TempDir()
		mustWrite(t, filepath.Join(repo, "meta-data", "VERSION"), "   \n")

		got, err := ReadVersion(repo)
		if err == nil {
			t.Error("expected an error for an empty VERSION file")
		}
		if got != DefaultVersion {
			t.Errorf("version = %q, want the default when unusable", got)
		}
	})
}

// Requirements: REQ_CFG_001
func TestConfigBuilderConfig(t *testing.T) {
	cfg := Config{Builders: map[string]any{
		"html":     map[string]any{"theme": "furo"},
		"revealjs": "not-an-object",
	}}

	if got := cfg.BuilderConfig("html")["theme"]; got != "furo" {
		t.Errorf("theme = %v, want furo", got)
	}
	if got := cfg.BuilderConfig("revealjs"); got != nil {
		t.Errorf("non-object builder config should be ignored, got %v", got)
	}
	if got := cfg.BuilderConfig("pdf"); got != nil {
		t.Errorf("absent builder config should be nil, got %v", got)
	}
}

// Requirements: REQ_SEL_005
func TestDefaultRoots(t *testing.T) {
	roots := DefaultRoots()
	root, ok := roots[DefaultRootDoc]
	if !ok {
		t.Fatalf("default roots missing %q: %+v", DefaultRootDoc, roots)
	}
	if len(root.Builders) != len(DefaultBuilders) {
		t.Fatalf("got %d builders, want %d", len(root.Builders), len(DefaultBuilders))
	}
	for i, want := range DefaultBuilders {
		if root.Builders[i].Shell != want {
			t.Errorf("builder %d = %q, want %q", i, root.Builders[i].Shell, want)
		}
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
