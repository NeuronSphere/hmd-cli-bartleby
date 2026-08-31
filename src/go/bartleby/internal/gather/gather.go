// Package gather implements the -g/--gather mode: assembling the documentation
// of several sibling repositories into the Bartleby docs repo before a build.
//
// This mode is narrow by design and inherited from the Python CLI. It only runs
// from a checkout of hmd-docs-bartleby that sits next to hmd-lib-bartleby-demos,
// because it replaces the contents of docs/ with the demo library's index plus
// one directory per gathered repo.
package gather

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DocsRepo is the only repo gather may run from.
	DocsRepo = "hmd-docs-bartleby"
	// DemosRepo supplies the index.rst that gathered docs are attached to.
	DemosRepo = "hmd-lib-bartleby-demos"
)

// indexMarker is the line gathered toctree entries are inserted before.
var indexMarkers = []string{"Indexes and tables", "Indices and tables"}

// Repos rebuilds docs/ in repoPath from the demo library index plus the docs of
// each named sibling repo. repos is the raw comma-separated flag value.
//
// It is destructive: everything in docs/ except the freshly copied index.rst is
// removed. That is the inherited behaviour, and the reason for the strict
// precondition check before anything is deleted.
func Repos(repoPath, repos string, logTo io.Writer) error {
	names := splitList(repos)
	if len(names) == 0 {
		return nil
	}

	if err := checkPreconditions(repoPath); err != nil {
		return err
	}

	parent := filepath.Dir(repoPath)
	docsPath := filepath.Join(repoPath, "docs")

	demoIndex := filepath.Join(parent, DemosRepo, "docs", "index.rst")
	if _, err := os.Stat(demoIndex); err != nil {
		return fmt.Errorf("gather requires %s: %w", demoIndex, err)
	}

	// Validate every repo before deleting anything, so a typo in the list does
	// not leave docs/ half-rebuilt.
	type gathered struct {
		name string
		docs string
	}
	var plan []gathered
	for _, name := range names {
		if !strings.Contains(name, "-") {
			if logTo != nil {
				fmt.Fprintf(logTo, "note: skipping %q — not a repository name\n", name)
			}
			continue
		}
		docs := filepath.Join(parent, name, "docs")
		info, err := os.Stat(docs)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("cannot gather %q: %s must exist — clone the repo next to %s",
				name, docs, DocsRepo)
		}
		plan = append(plan, gathered{name: name, docs: docs})
	}
	if len(plan) == 0 {
		return fmt.Errorf("no gatherable repositories in %q", repos)
	}

	if err := clearDocs(docsPath); err != nil {
		return err
	}

	indexPath := filepath.Join(docsPath, "index.rst")
	if err := copyFile(demoIndex, indexPath); err != nil {
		return fmt.Errorf("copying %s: %w", demoIndex, err)
	}

	for _, g := range plan {
		if logTo != nil {
			fmt.Fprintf(logTo, "Gathering docs from %s...\n", g.name)
		}
		if err := copyDir(g.docs, filepath.Join(docsPath, g.name)); err != nil {
			return fmt.Errorf("copying docs for %q: %w", g.name, err)
		}
		if err := AddToIndex(indexPath, g.name); err != nil {
			return err
		}
	}

	return nil
}

// checkPreconditions verifies gather is running somewhere it can work.
func checkPreconditions(repoPath string) error {
	if filepath.Base(repoPath) != DocsRepo {
		return fmt.Errorf("gather mode only runs from a %s checkout (current directory is %s)",
			DocsRepo, filepath.Base(repoPath))
	}

	demos := filepath.Join(filepath.Dir(repoPath), DemosRepo)
	if info, err := os.Stat(demos); err != nil || !info.IsDir() {
		return fmt.Errorf("gather mode requires %s alongside %s (looked for %s)",
			DemosRepo, DocsRepo, demos)
	}

	docs := filepath.Join(repoPath, "docs")
	if info, err := os.Stat(docs); err != nil || !info.IsDir() {
		return fmt.Errorf("gather mode requires a docs directory at %s", docs)
	}

	return nil
}

// clearDocs removes everything under docs/ except index.rst, which is replaced
// immediately afterwards.
func clearDocs(docsPath string) error {
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", docsPath, err)
	}

	for _, entry := range entries {
		if entry.Name() == "index.rst" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(docsPath, entry.Name())); err != nil {
			return fmt.Errorf("clearing %s: %w", filepath.Join(docsPath, entry.Name()), err)
		}
	}
	return nil
}

// AddToIndex inserts a toctree entry for repo into the index document, before
// the trailing "Indexes and tables" section when there is one and at the end
// otherwise. The Python version raised an IndexError when the marker was absent.
func AddToIndex(indexPath, repo string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", indexPath, err)
	}

	entry := fmt.Sprintf("   %s/index.rst\n", repo)
	text := string(data)
	lines := strings.SplitAfter(text, "\n")

	for i, line := range lines {
		if isIndexMarker(line) {
			lines = append(lines[:i], append([]string{entry}, lines[i:]...)...)
			return os.WriteFile(indexPath, []byte(strings.Join(lines, "")), 0o644)
		}
	}

	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return os.WriteFile(indexPath, []byte(text+entry), 0o644)
}

func isIndexMarker(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, marker := range indexMarkers {
		if strings.EqualFold(trimmed, marker) {
			return true
		}
	}
	return false
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func copyDir(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		switch {
		case entry.IsDir():
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		case entry.Type().IsRegular():
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}
