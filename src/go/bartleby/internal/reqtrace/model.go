// Package reqtrace links the requirements written as sphinx-needs items in
// docs/requirements/ to the tests that verify them.
//
// Requirements are the source of truth for what the CLI must do; tests declare
// which requirements they cover, in the test source rather than in a separate
// document that would drift. Go tests declare coverage in a doc comment:
//
//	// Requirements: REQ_SHELL_001, REQ_PLAN_002
//	func TestBuildsFiltersByShell(t *testing.T) {
//
// Robot tests declare it with tags, which is what Robot's tag model is for:
//
//	[Tags]    REQ_SHELL_001
//
// From those three inputs this package generates the traceability page and
// reports the gaps: a reference to a requirement that does not exist, a
// duplicate requirement, a requirement nothing verifies, and a test that claims
// nothing.
package reqtrace

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// IDPrefix is prepended to the short form used in test annotations, so a test
// tags REQ_SHELL_001 and means HMD_CLI_BARTLEBY_REQ_SHELL_001. The prefix
// follows the HMD_<REPO_TYPE>_<NAME>_ convention the NERD documents use.
const IDPrefix = "HMD_CLI_BARTLEBY_"

// ExemptTag marks a requirement that no automated test can reasonably verify.
// Such a requirement must say in its own text how it is verified instead.
const ExemptTag = "trace-exempt"

// GeneratedFile is the traceability page this package writes. It is generated,
// committed, and checked for staleness — the docs have to build inside the
// transform container, which cannot run Go.
const GeneratedFile = "traceability.rst"

// Kind distinguishes where a test lives.
type Kind string

const (
	KindGo    Kind = "go"
	KindRobot Kind = "robot"
)

// Requirement is a sphinx-needs req or spec item parsed out of the docs.
type Requirement struct {
	ID     string
	Type   string // "req" or "spec"
	Title  string
	Status string
	Tags   []string
	Links  []string // :links: targets, e.g. a spec pointing at its req
	File   string   // path relative to the repo root
	Line   int
}

// Exempt reports whether this requirement is excused from needing a test.
func (r Requirement) Exempt() bool {
	for _, tag := range r.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), ExemptTag) {
			return true
		}
	}
	return false
}

// Area returns the area segment of an area-coded ID, e.g. "SHELL" for
// HMD_CLI_BARTLEBY_REQ_SHELL_001. It returns "" for IDs that do not follow the
// pattern, such as the NERD documents.
func (r Requirement) Area() string {
	rest, ok := cutPrefix(r.ID, IDPrefix+"REQ_")
	if !ok {
		return ""
	}
	idx := strings.LastIndex(rest, "_")
	if idx <= 0 {
		return ""
	}
	return rest[:idx]
}

// TestCase is one test that declares the requirements it verifies.
type TestCase struct {
	Kind         Kind
	Name         string // Go function name, or Robot test case name
	Suite        string // Go package name, or Robot suite file name
	File         string // path relative to the repo root
	Line         int
	Requirements []string // full requirement IDs, sorted and deduplicated
}

// NeedID returns the sphinx-needs ID for the generated test item. It is derived
// from a hash of the suite and name so that adding a test does not renumber
// every other one.
func (t TestCase) NeedID() string {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s\x00%s\x00%s", t.Kind, t.Suite, t.Name)
	return fmt.Sprintf("%sTEST_%s_%08X", IDPrefix, strings.ToUpper(string(t.Kind)), h.Sum32())
}

// Title returns the human-readable label for the generated test item.
func (t TestCase) Title() string {
	if t.Kind == KindRobot {
		return fmt.Sprintf("%s: %s", t.Suite, t.Name)
	}
	return fmt.Sprintf("%s.%s", t.Suite, t.Name)
}

// Model is everything parsed from the repository.
type Model struct {
	Requirements []Requirement
	Tests        []TestCase
}

// Sort puts the model in a stable order so the generated page and every report
// are byte-identical between runs.
func (m *Model) Sort() {
	sort.Slice(m.Requirements, func(i, j int) bool {
		return m.Requirements[i].ID < m.Requirements[j].ID
	})
	sort.Slice(m.Tests, func(i, j int) bool {
		a, b := m.Tests[i], m.Tests[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Suite != b.Suite {
			return a.Suite < b.Suite
		}
		return a.Name < b.Name
	})
}

// RequirementIndex maps requirement ID to requirement.
func (m Model) RequirementIndex() map[string]Requirement {
	index := make(map[string]Requirement, len(m.Requirements))
	for _, r := range m.Requirements {
		index[r.ID] = r
	}
	return index
}

// Coverage maps each requirement ID to the tests that verify it.
func (m Model) Coverage() map[string][]TestCase {
	coverage := make(map[string][]TestCase)
	for _, t := range m.Tests {
		for _, id := range t.Requirements {
			coverage[id] = append(coverage[id], t)
		}
	}
	return coverage
}

// ExpandID turns a short annotation such as REQ_SHELL_001 into a full ID. An ID
// that already carries the prefix is returned unchanged, so tests may spell it
// either way.
func ExpandID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, IDPrefix) {
		return id
	}
	if strings.HasPrefix(id, "HMD_") {
		// Another repo's ID, or a NERD reference: leave it alone.
		return id
	}
	return IDPrefix + id
}

// normalizeIDs expands, deduplicates, and sorts a list of referenced IDs.
func normalizeIDs(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	var out []string
	for _, id := range raw {
		full := ExpandID(id)
		if full == "" || seen[full] {
			continue
		}
		seen[full] = true
		out = append(out, full)
	}
	sort.Strings(out)
	return out
}

// cutPrefix is strings.CutPrefix, kept as a helper for readability at call sites.
func cutPrefix(s, prefix string) (string, bool) {
	return strings.CutPrefix(s, prefix)
}
