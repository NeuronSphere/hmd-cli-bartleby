package reqtrace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Layout says where in the repository the requirements, the Go tests, and the
// Robot suites live.
type Layout struct {
	// RepoRoot is the directory holding meta-data/, docs/, src/, and test/.
	RepoRoot string
	// RequirementsDir holds the requirement documents, relative to RepoRoot,
	// and is where the generated traceability page is written.
	RequirementsDir string
	// ExtraRequirementsDirs holds additional directories, relative to
	// RepoRoot, also scanned for requirement documents. NERD documents live
	// under docs/proposals alongside — not instead of — docs/requirements, so
	// this is additive: a directory that does not exist is skipped rather than
	// treated as an error, as long as at least one requirements directory does.
	ExtraRequirementsDirs []string
	// GoRoot is the tree to scan for annotated Go tests, relative to RepoRoot.
	// It may contain more than one module.
	GoRoot string
	// GoSkipDirs are directory names not to scan, such as build output.
	GoSkipDirs []string
	// RobotGlob finds the Robot suites, relative to RepoRoot.
	RobotGlob string
	// Scheme is the ID naming scheme. When zero, Load derives it from the
	// repository name in meta-data/manifest.json.
	Scheme Scheme
}

// DefaultLayout is this repository's layout.
func DefaultLayout(repoRoot string) Layout {
	return Layout{
		RepoRoot:        repoRoot,
		RequirementsDir: filepath.Join("docs", "requirements"),
		GoRoot:          filepath.Join("src", "go"),
		GoSkipDirs:      []string{"build", "testdata"},
		RobotGlob:       filepath.Join("test", "*.robot"),
	}
}

// ProposalsDir is where NERD documents live: req/spec items under
// docs/proposals, alongside — never instead of — docs/requirements.
//
// Scanning it is opt-in (see RunOptions.IncludeProposals) rather than part of
// DefaultLayout, because a repository already running "reqtrace -check" in CI
// may itself own NERD proposals with no test annotations yet; turning this on
// unconditionally would fail that build the moment this package adopted it,
// which is exactly the gap this tool exists to surface, not to spring on
// someone as a side effect of an unrelated change.
func ProposalsDir() string {
	return filepath.Join("docs", "proposals")
}

// GeneratedPath is the full path of the generated traceability page.
func (l Layout) GeneratedPath() string {
	return filepath.Join(l.RepoRoot, l.RequirementsDir, GeneratedFile)
}

// existingRequirementsDirs returns the absolute paths of RequirementsDir and
// ExtraRequirementsDirs that actually exist, in that order. A repository is
// free to keep its requirements in only one of them — e.g. only
// docs/proposals, for a repository whose requirements are all NERD documents
// — so a missing directory is skipped rather than treated as an error, unless
// none of them exist.
func (l Layout) existingRequirementsDirs() ([]string, error) {
	candidates := append([]string{l.RequirementsDir}, l.ExtraRequirementsDirs...)

	var dirs []string
	for _, candidate := range candidates {
		dir := filepath.Join(l.RepoRoot, candidate)
		if _, err := os.Stat(dir); err == nil {
			dirs = append(dirs, dir)
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no requirements directory found (tried %s)", strings.Join(candidates, ", "))
	}
	return dirs, nil
}

// Load parses the requirements and every annotated test.
func Load(l Layout) (Model, error) {
	var m Model

	requirementsDirs, err := l.existingRequirementsDirs()
	if err != nil {
		return m, err
	}

	scheme := l.Scheme
	if scheme.IsZero() {
		derived, err := SchemeFromManifest(l.RepoRoot)
		if err != nil {
			return m, err
		}
		scheme = derived
	}

	for _, dir := range requirementsDirs {
		requirements, err := ParseRequirements(dir, l.RepoRoot, scheme)
		if err != nil {
			return m, err
		}
		m.Requirements = append(m.Requirements, requirements...)
	}

	goTests, err := ParseGoTests(filepath.Join(l.RepoRoot, l.GoRoot), l.RepoRoot, l.GoSkipDirs, scheme)
	if err != nil {
		return m, err
	}
	m.Tests = append(m.Tests, goTests...)

	robotFiles, err := filepath.Glob(filepath.Join(l.RepoRoot, l.RobotGlob))
	if err != nil {
		return m, fmt.Errorf("finding Robot suites: %w", err)
	}
	sort.Strings(robotFiles)

	robotTests, err := ParseRobotTests(robotFiles, l.RepoRoot, scheme)
	if err != nil {
		return m, err
	}
	m.Tests = append(m.Tests, robotTests...)

	m.Sort()
	return m, nil
}

// SchemeFromManifest derives the ID naming scheme from the repository name in
// meta-data/manifest.json — the same file FindRepoRoot uses to recognise a
// repository root.
func SchemeFromManifest(repoRoot string) (Scheme, error) {
	path := filepath.Join(repoRoot, "meta-data", "manifest.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return Scheme{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Scheme{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	if strings.TrimSpace(manifest.Name) == "" {
		return Scheme{}, fmt.Errorf(`%s has no "name", so the requirement ID prefix cannot be derived`, path)
	}

	scheme := SchemeFromName(manifest.Name)
	if scheme.IsZero() {
		return Scheme{}, fmt.Errorf("%s: name %q yields no usable ID prefix", path, manifest.Name)
	}
	return scheme, nil
}

// ErrStale reports that the generated page does not match the current
// requirements and annotations.
var ErrStale = errors.New("generated traceability page is out of date")

// Write regenerates the traceability page.
func Write(l Layout, m Model) error {
	path := l.GeneratedPath()
	// RequirementsDir may not exist yet — a repository whose requirements are
	// all NERD documents under docs/proposals has no reason to have created
	// docs/requirements before now, and the generated page still has to live
	// somewhere predictable.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(Render(m)), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// CheckFresh compares the generated page on disk with what the model would
// produce, returning ErrStale when they differ.
func CheckFresh(l Layout, m Model) error {
	path := l.GeneratedPath()

	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s does not exist — run \"make reqs\"", ErrStale, path)
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}

	if string(existing) != Render(m) {
		return fmt.Errorf("%w: %s — run \"make reqs\"", ErrStale, path)
	}
	return nil
}

// FindRepoRoot walks up from start looking for the directory that holds
// meta-data/manifest.json, so the tool can be run from anywhere in the tree.
func FindRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "meta-data", "manifest.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no repository root (a directory containing meta-data/manifest.json) at or above %s", start)
		}
		dir = parent
	}
}

// Summary is a one-line description of the model, for command output.
func Summary(m Model) string {
	coverage := m.Coverage()

	covered, exempt := 0, 0
	for _, r := range m.Requirements {
		switch {
		case len(coverage[r.ID]) > 0:
			covered++
		case r.Exempt():
			exempt++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d requirements, %d covered", len(m.Requirements), covered)
	if exempt > 0 {
		fmt.Fprintf(&b, ", %d exempt", exempt)
	}
	fmt.Fprintf(&b, "; %d tests", len(m.Tests))
	return b.String()
}
