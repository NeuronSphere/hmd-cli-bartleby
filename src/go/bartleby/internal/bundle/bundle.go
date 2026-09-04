// Package bundle installs the assets bartleby carries inside its binary — the
// agent skills and the agents — where an agent will find them.
//
// Skills and agents differ only in what their file is called and where it
// lands, so the rules that matter (never clobber a local edit, report what
// happened, install twice for the same result) live here once.
package bundle

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Item is one bundled asset.
type Item struct {
	// Name is the directory it is bundled under, and its identity: what it is
	// called on the command line and what it installs as.
	Name string
	// Description comes from the front matter, for listing.
	Description string
	// Content is the whole file, front matter included.
	Content string
}

// Set is a bundled collection: the embedded files plus how they are named.
type Set struct {
	// FS holds one directory per item, each containing SourceFile.
	FS fs.FS
	// SourceFile is the filename inside each item's directory, e.g. "SKILL.md".
	SourceFile string
	// Kind names one item in messages, e.g. "skill".
	Kind string
	// Home is the directory under ~/.claude that these install into.
	Home string
	// DestName maps an item name to its path relative to the destination
	// directory. Skills install as <name>/SKILL.md; agents as <name>.md.
	DestName func(name string) string
}

// All returns every item in the set, by name.
func (s Set) All() []Item {
	entries, err := fs.ReadDir(s.FS, ".")
	if err != nil {
		// Only reachable if the embed pattern matched nothing, which is a build
		// error rather than a runtime one.
		return nil
	}

	var items []Item
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(s.FS, path(entry.Name(), s.SourceFile))
		if err != nil {
			continue
		}
		items = append(items, Item{
			Name:        entry.Name(),
			Description: Description(string(content)),
			Content:     string(content),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

// Names returns the item names, sorted.
func (s Set) Names() []string {
	all := s.All()
	names := make([]string, 0, len(all))
	for _, item := range all {
		names = append(names, item.Name)
	}
	return names
}

// Get returns one item by name. The error names what is available, because the
// likely cause of a miss is a typo or a half-remembered name.
func (s Set) Get(name string) (Item, error) {
	for _, item := range s.All() {
		if item.Name == name {
			return item, nil
		}
	}
	return Item{}, fmt.Errorf("no bundled %s named %q; available: %s",
		s.Kind, name, strings.Join(s.Names(), ", "))
}

// Select returns the named items, or all of them when none are named.
func (s Set) Select(names []string) ([]Item, error) {
	if len(names) == 0 {
		return s.All(), nil
	}
	var selected []Item
	for _, name := range names {
		item, err := s.Get(name)
		if err != nil {
			return nil, err
		}
		selected = append(selected, item)
	}
	return selected, nil
}

// DefaultDir is where these install unless told otherwise: the user-level
// directory an agent reads by default.
func (s Set) DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find your home directory to install %ss into: %w", s.Kind, err)
	}
	return filepath.Join(home, ".claude", s.Home), nil
}

// ProjectDir is the equivalent inside a repository, for assets that should
// travel with the project rather than the machine.
func (s Set) ProjectDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".claude", s.Home)
}

// Install writes the given items under dir.
//
// An item already present with identical content is left alone and reported as
// current. One that differs is *not* overwritten unless force is set: the local
// copy may carry someone's edits, and losing those silently is worse than
// declining to act.
func (s Set) Install(dir string, items []Item, force bool) ([]Result, error) {
	var results []Result

	for _, item := range items {
		dest := filepath.Join(dir, s.DestName(item.Name))

		existing, err := os.ReadFile(dest)
		switch {
		case err == nil && string(existing) == item.Content:
			results = append(results, Result{Item: item, Path: dest, Outcome: Unchanged})
			continue
		case err == nil && !force:
			results = append(results, Result{Item: item, Path: dest, Outcome: Differs})
			continue
		case err != nil && !os.IsNotExist(err):
			return results, fmt.Errorf("reading %s: %w", dest, err)
		}

		outcome := Written
		if err == nil {
			outcome = Updated
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return results, fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}
		if err := os.WriteFile(dest, []byte(item.Content), 0o644); err != nil {
			return results, fmt.Errorf("writing %s: %w", dest, err)
		}

		results = append(results, Result{Item: item, Path: dest, Outcome: outcome})
	}

	return results, nil
}

// Outcome is what installing one item did.
type Outcome int

const (
	// Written means the item was not there and now is.
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

// Result is what happened to one item.
type Result struct {
	Item    Item
	Path    string
	Outcome Outcome
}

// Description pulls the description out of the YAML front matter.
//
// This is deliberately not a YAML parse: the field is a single line in a fixed
// place, and a dependency to read it would not earn itself.
func Description(content string) string {
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

// path joins parts for an embedded FS, which always uses forward slashes.
func path(parts ...string) string {
	return strings.Join(parts, "/")
}
