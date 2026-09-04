package reqtrace

import (
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
	// RequirementsDir holds the requirement documents, relative to RepoRoot.
	RequirementsDir string
	// GoRoot is the Go module root to scan for tests, relative to RepoRoot.
	GoRoot string
	// GoSkipDirs are directory names not to scan, such as build output.
	GoSkipDirs []string
	// RobotGlob finds the Robot suites, relative to RepoRoot.
	RobotGlob string
}

// DefaultLayout is this repository's layout.
func DefaultLayout(repoRoot string) Layout {
	return Layout{
		RepoRoot:        repoRoot,
		RequirementsDir: filepath.Join("docs", "requirements"),
		GoRoot:          filepath.Join("src", "go", "bartleby"),
		GoSkipDirs:      []string{"build", "testdata"},
		RobotGlob:       filepath.Join("test", "*.robot"),
	}
}

// GeneratedPath is the full path of the generated traceability page.
func (l Layout) GeneratedPath() string {
	return filepath.Join(l.RepoRoot, l.RequirementsDir, GeneratedFile)
}

// Load parses the requirements and every annotated test.
func Load(l Layout) (Model, error) {
	var m Model

	requirementsDir := filepath.Join(l.RepoRoot, l.RequirementsDir)
	if _, err := os.Stat(requirementsDir); err != nil {
		return m, fmt.Errorf("requirements directory %s: %w", requirementsDir, err)
	}

	requirements, err := ParseRequirements(requirementsDir, l.RepoRoot)
	if err != nil {
		return m, err
	}
	m.Requirements = requirements

	goTests, err := ParseGoTests(filepath.Join(l.RepoRoot, l.GoRoot), l.RepoRoot, l.GoSkipDirs)
	if err != nil {
		return m, err
	}
	m.Tests = append(m.Tests, goTests...)

	robotFiles, err := filepath.Glob(filepath.Join(l.RepoRoot, l.RobotGlob))
	if err != nil {
		return m, fmt.Errorf("finding Robot suites: %w", err)
	}
	sort.Strings(robotFiles)

	robotTests, err := ParseRobotTests(robotFiles, l.RepoRoot)
	if err != nil {
		return m, err
	}
	m.Tests = append(m.Tests, robotTests...)

	m.Sort()
	return m, nil
}

// ErrStale reports that the generated page does not match the current
// requirements and annotations.
var ErrStale = errors.New("generated traceability page is out of date")

// Write regenerates the traceability page.
func Write(l Layout, m Model) error {
	path := l.GeneratedPath()
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
