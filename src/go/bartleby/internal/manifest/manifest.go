// Package manifest reads meta-data/manifest.json — the per-repo description of
// which documents Bartleby builds and with which builders.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultVersion is used as the repo version when meta-data/VERSION is absent.
// It doubles as the default image tag, which is why it is "stable" and not a
// semver-looking placeholder.
const DefaultVersion = "stable"

// DefaultRootDoc is the root document name assumed when a root omits root_doc.
const DefaultRootDoc = "index"

// DefaultBuilders are the builders used when the manifest declares no roots.
var DefaultBuilders = []string{"html", "pdf"}

// ErrNotFound reports that the repo has no meta-data/manifest.json. It is not a
// failure: a repo without a manifest builds index.rst with the default builders.
var ErrNotFound = errors.New("manifest not found")

// Builder names a Sphinx builder and, optionally, config specific to it.
//
// The manifest accepts two spellings, both of which the Python CLI tolerated:
//
//	"builders": ["html", "pdf"]
//	"builders": [{"shell": "html", "config": {"theme": "alabaster"}}]
type Builder struct {
	Shell  string         `json:"shell"`
	Config map[string]any `json:"config"`
}

// UnmarshalJSON accepts either a bare builder name or a {shell, config} object.
func (b *Builder) UnmarshalJSON(data []byte) error {
	var shell string
	if err := json.Unmarshal(data, &shell); err == nil {
		b.Shell = shell
		b.Config = nil
		return nil
	}

	var obj struct {
		Shell  string         `json:"shell"`
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("builder must be a name or a {shell, config} object: %w", err)
	}
	if obj.Shell == "" {
		return errors.New("builder object is missing the required \"shell\" field")
	}
	b.Shell = obj.Shell
	b.Config = obj.Config
	return nil
}

// MarshalJSON writes the compact form when there is no builder-specific config.
func (b Builder) MarshalJSON() ([]byte, error) {
	if len(b.Config) == 0 {
		return json.Marshal(b.Shell)
	}
	return json.Marshal(struct {
		Shell  string         `json:"shell"`
		Config map[string]any `json:"config"`
	}{b.Shell, b.Config})
}

// Root is one buildable document tree.
type Root struct {
	RootDoc  string         `json:"root_doc"`
	Builders []Builder      `json:"builders"`
	Config   map[string]any `json:"config"`
}

// Doc returns the root document name, defaulting to "index" when unset. The
// container has no default of its own — an empty root_doc reaches Sphinx as an
// empty string and fails there — so the default belongs here.
func (r Root) Doc() string {
	if r.RootDoc == "" {
		return DefaultRootDoc
	}
	return r.RootDoc
}

// Source is an external documentation tree stitched into this repo's docs.
type Source struct {
	// ArtifactPath is a path, relative to the repo root, holding a checked-out
	// or downloaded artifact. When empty, the source is expected to already
	// live at docs/<key>.
	ArtifactPath string `json:"artifact_path"`
	// DocsRoot is the docs directory within ArtifactPath. Defaults to "docs".
	DocsRoot string `json:"docs_root"`
	// Title is the toctree caption. Defaults to the source key.
	Title string `json:"title"`
}

// Docs returns DocsRoot with its default applied.
func (s Source) Docs() string {
	if s.DocsRoot == "" {
		return "docs"
	}
	return s.DocsRoot
}

// BartlebySection is the "bartleby" object in the manifest.
type BartlebySection struct {
	Roots   map[string]Root   `json:"roots"`
	Sources map[string]Source `json:"sources"`
	Config  Config            `json:"config"`
}

// Config is the "bartleby.config" object: repo-wide defaults, plus per-builder
// config under "builders".
type Config struct {
	DefaultLogo     string         `json:"default_logo"`
	HTMLDefaultLogo string         `json:"html_default_logo"`
	PDFDefaultLogo  string         `json:"pdf_default_logo"`
	Confidential    *bool          `json:"confidential"`
	Builders        map[string]any `json:"builders"`
}

// BuilderConfig returns the manifest config declared for one builder, i.e.
// bartleby.config.builders.<shell>. A non-object value is ignored.
func (c Config) BuilderConfig(shell string) map[string]any {
	raw, ok := c.Builders[shell]
	if !ok {
		return nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return obj
}

// Manifest is the subset of meta-data/manifest.json that Bartleby cares about.
type Manifest struct {
	Name     string          `json:"name"`
	Bartleby BartlebySection `json:"bartleby"`
}

// Path returns the manifest path for a repo.
func Path(repoPath string) string {
	return filepath.Join(repoPath, "meta-data", "manifest.json")
}

// Read loads meta-data/manifest.json.
//
// A repo with no manifest returns an empty Manifest wrapped with ErrNotFound, so
// callers can distinguish "no manifest, use defaults" (fine) from "manifest is
// unreadable or malformed" (not fine). The previous implementation collapsed
// both into silent defaults, which turned a permissions problem or a typo in
// JSON into a mysteriously wrong build.
func Read(repoPath string) (*Manifest, error) {
	path := Path(repoPath)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{}, ErrNotFound
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &m, nil
}

// ReadVersion reads meta-data/VERSION.
//
// A missing file returns (DefaultVersion, nil) — that is a normal repo shape. An
// unreadable file returns the default plus an error, so the caller can warn
// rather than silently building documents titled "<repo>-stable".
func ReadVersion(repoPath string) (string, error) {
	path := filepath.Join(repoPath, "meta-data", "VERSION")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultVersion, nil
		}
		return DefaultVersion, fmt.Errorf("reading %s: %w", path, err)
	}

	version := strings.TrimSpace(string(data))
	if version == "" {
		return DefaultVersion, fmt.Errorf("%s is empty", path)
	}
	return version, nil
}

// DefaultRoots is the single-root fallback used when the manifest declares none.
func DefaultRoots() map[string]Root {
	builders := make([]Builder, 0, len(DefaultBuilders))
	for _, shell := range DefaultBuilders {
		builders = append(builders, Builder{Shell: shell})
	}
	return map[string]Root{
		DefaultRootDoc: {RootDoc: DefaultRootDoc, Builders: builders},
	}
}
