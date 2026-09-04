// Package skills carries the agent skills bartleby ships with, and installs
// them where an agent will find them.
//
// The skills are embedded in the binary rather than fetched, so a
// brew-installed bartleby can seed a machine with no network and no Python
// runtime. They live in this directory because go:embed cannot reach outside
// the module; the legacy Python package copies them from here too.
package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:*/SKILL.md
var files embed.FS

// File is the name every skill's instructions are stored under, which is what
// an agent looks for.
const File = "SKILL.md"

// Skill is one bundled skill.
type Skill struct {
	// Name is the directory name, and the identity used everywhere: on the
	// command line, and as the directory it installs into.
	Name string
	// Description comes from the front matter, for listing.
	Description string
	// Content is the whole SKILL.md, front matter included.
	Content string
}

// All returns every bundled skill, by name.
func All() []Skill {
	entries, err := files.ReadDir(".")
	if err != nil {
		// Only reachable if the embed pattern matched nothing, which is a build
		// error rather than a runtime one.
		return nil
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(files, filepath.Join(entry.Name(), File))
		if err != nil {
			continue
		}
		skills = append(skills, Skill{
			Name:        entry.Name(),
			Description: description(string(content)),
			Content:     string(content),
		})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}

// Names returns the bundled skill names, sorted.
func Names() []string {
	all := All()
	names := make([]string, 0, len(all))
	for _, s := range all {
		names = append(names, s.Name)
	}
	return names
}

// Get returns one skill by name. The error names what is available, because the
// most likely reason for missing is a typo or a half-remembered name.
func Get(name string) (Skill, error) {
	for _, s := range All() {
		if s.Name == name {
			return s, nil
		}
	}
	return Skill{}, fmt.Errorf("no bundled skill named %q; available: %s",
		name, strings.Join(Names(), ", "))
}

// Select returns the named skills, or all of them when none are named.
func Select(names []string) ([]Skill, error) {
	if len(names) == 0 {
		return All(), nil
	}
	var selected []Skill
	for _, name := range names {
		skill, err := Get(name)
		if err != nil {
			return nil, err
		}
		selected = append(selected, skill)
	}
	return selected, nil
}

// Outcome is what installing one skill did.
type Outcome int

const (
	// Written means the skill was not there and now is.
	Written Outcome = iota
	// Unchanged means an identical copy was already in place.
	Unchanged
	// Updated means an existing, differing copy was overwritten.
	Updated
	// Differs means an existing copy differs and was left alone.
	Differs
)

// String describes the outcome in the terms the CLI reports it.
func (o Outcome) String() string {
	switch o {
	case Written:
		return "installed"
	case Unchanged:
		return "already current"
	case Updated:
		return "updated"
	case Differs:
		return "left alone (differs)"
	}
	return "unknown"
}

// Result is what happened to one skill.
type Result struct {
	Skill   Skill
	Path    string
	Outcome Outcome
}

// DefaultDir is where skills go unless told otherwise: the user-level skills
// directory an agent reads by default.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find your home directory to install skills into: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// ProjectDir is the equivalent inside a repository, for skills that should
// travel with the project rather than the machine.
func ProjectDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".claude", "skills")
}

// Install writes the given skills under dir as <dir>/<name>/SKILL.md.
//
// A skill already present with identical content is left alone and reported as
// current. One that differs is *not* overwritten unless force is set: the local
// copy may carry someone's edits, and losing those silently is worse than
// refusing to act.
func Install(dir string, skills []Skill, force bool) ([]Result, error) {
	var results []Result

	for _, skill := range skills {
		path := filepath.Join(dir, skill.Name, File)

		existing, err := os.ReadFile(path)
		switch {
		case err == nil && string(existing) == skill.Content:
			results = append(results, Result{Skill: skill, Path: path, Outcome: Unchanged})
			continue
		case err == nil && !force:
			results = append(results, Result{Skill: skill, Path: path, Outcome: Differs})
			continue
		case err != nil && !os.IsNotExist(err):
			return results, fmt.Errorf("reading %s: %w", path, err)
		}

		outcome := Written
		if err == nil {
			outcome = Updated
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return results, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(skill.Content), 0o644); err != nil {
			return results, fmt.Errorf("writing %s: %w", path, err)
		}

		results = append(results, Result{Skill: skill, Path: path, Outcome: outcome})
	}

	return results, nil
}

// description pulls the description out of the YAML front matter.
//
// This is deliberately not a YAML parse: the field is a single line in a fixed
// place, and a dependency to read it would not earn itself.
func description(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return ""
		}
		if rest, ok := strings.CutPrefix(line, "description:"); ok {
			return strings.Trim(strings.TrimSpace(rest), `"'`)
		}
	}
	return ""
}
