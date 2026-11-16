package reqtrace

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/needs"
)

// ListingSchema versions the listing's JSON shape and the meaning of each
// need's digest. Consumers that compare listings, such as DiffListings,
// refuse to compare two schemas.
const ListingSchema = 1

// ListOptions drives a listing: what to read, and what to print.
type ListOptions struct {
	// Workspace lists every repository under this directory instead of one
	// repository.
	Workspace string
	// Repos keeps only these repositories, matched by name or folder.
	Repos []string
	// Stages and Types keep only needs in these stages or of these types.
	Stages, Types []string
	// Open keeps needs that are not implemented or closed, and implemented
	// needs no test covers.
	Open bool
	// Summary prints per-repository counts instead of the needs.
	Summary bool
	// Changes adds what changed since the index's previous refresh.
	Changes bool
	// Rebuild ignores the index and builds it again.
	Rebuild bool
	// CacheDir holds the workspace indexes. Empty means $REQTRACE_CACHE_DIR,
	// then the user cache directory.
	CacheDir string
	// Now is the clock, for tests. Nil means time.Now.
	Now func() time.Time
}

// Listing is the JSON document "reqtrace -format json" prints.
type Listing struct {
	Schema    int             `json:"schema"`
	Generated string          `json:"generated"`
	Workspace string          `json:"workspace,omitempty"`
	Summary   ListingSummary  `json:"summary"`
	Needs     []ListedNeed    `json:"needs"`
	Problems  []ListedProblem `json:"problems"`
	Changes   *Changes        `json:"changes,omitempty"`
	Warnings  []string        `json:"warnings,omitempty"`
}

// ListingSummary counts what was found. Dirs and the repository counts
// describe the whole scan; the rest describe the needs after filtering.
type ListingSummary struct {
	Dirs              int            `json:"dirs,omitempty"`
	ReposWithNeeds    int            `json:"repos_with_needs"`
	ReposWithoutNeeds int            `json:"repos_without_needs"`
	Needs             int            `json:"needs"`
	Covered           int            `json:"covered"`
	Open              int            `json:"open"`
	ByStage           map[string]int `json:"by_stage"`
	Repos             []RepoSummary  `json:"repos,omitempty"`
}

// RepoSummary is one repository's line in a -summary listing.
type RepoSummary struct {
	Repo    string         `json:"repo"`
	Needs   int            `json:"needs"`
	Covered int            `json:"covered"`
	Open    int            `json:"open"`
	ByStage map[string]int `json:"by_stage"`
}

// ListedNeed is one need with where it lives, its stage and its coverage.
type ListedNeed struct {
	Repo      string   `json:"repo"`
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Stage     string   `json:"stage"`
	Tags      []string `json:"tags"`
	Links     []string `json:"links"`
	Source    string   `json:"source"`
	File      string   `json:"file"`
	Line      int      `json:"line"`
	Digest    string   `json:"digest"`
	CoveredBy []Cover  `json:"covered_by"`
}

// Cover is one test that declares it verifies a need.
type Cover struct {
	Test string `json:"test"`
	Kind Kind   `json:"kind"`
	File string `json:"file"`
}

// ListedProblem is a defect found while listing. Problems never change the
// exit status of a listing: they are data for whoever reads it.
type ListedProblem struct {
	Kind    string `json:"kind"`
	Repo    string `json:"repo"`
	ID      string `json:"id,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

// Listing problem kinds.
const (
	ListProblemUnknownStatus = "unknown-status"
	ListProblemDuplicateID   = "duplicate-id"
	ListProblemUnknownLink   = "unknown-link"
	ListProblemAmbiguousLink = "ambiguous-link"
	ListProblemDuplicateRepo = "duplicate-repo"
)

// NeedKey identifies a need across repositories.
type NeedKey struct {
	Repo string `json:"repo"`
	ID   string `json:"id"`
}

// Changes says which needs were added, changed or removed since an earlier
// listing. A need is changed when its digest, status or covering tests are.
type Changes struct {
	Since   string    `json:"since,omitempty"`
	Added   []NeedKey `json:"added"`
	Changed []NeedKey `json:"changed"`
	Removed []NeedKey `json:"removed"`
}

// assemble turns scanned repositories into a listing: coverage joined, stages
// assigned, links resolved across every repository given, and the result
// sorted so the same tree always lists the same way.
func assemble(repos []*repoState) Listing {
	l := Listing{Schema: ListingSchema, Needs: []ListedNeed{}, Problems: []ListedProblem{}}

	// definedIn maps a need ID to the repositories that define it, for link
	// resolution.
	definedIn := map[string]map[string]bool{}
	for _, r := range repos {
		for _, f := range r.Files {
			for _, n := range f.Needs {
				if definedIn[n.ID] == nil {
					definedIn[n.ID] = map[string]bool{}
				}
				definedIn[n.ID][r.name()] = true
			}
		}
	}

	for _, r := range repos {
		name := r.name()
		cfg := r.Config

		covers := map[string][]Cover{}
		for file, f := range r.Files {
			for _, t := range f.Tests {
				for _, id := range t.Requirements {
					covers[id] = append(covers[id], Cover{Test: t.Title(), Kind: t.Kind, File: file})
				}
			}
		}

		seen := map[string]string{}
		count := 0
		for _, file := range sortedKeys(r.Files) {
			for _, n := range r.Files[file].Needs {
				count++
				stage, recognised := cfg.stageOf(n.Status)
				cov := append([]Cover{}, covers[n.ID]...)
				sort.Slice(cov, func(i, j int) bool {
					if cov[i].Kind != cov[j].Kind {
						return cov[i].Kind < cov[j].Kind
					}
					return cov[i].Test < cov[j].Test
				})
				l.Needs = append(l.Needs, ListedNeed{
					Repo: name, ID: n.ID, Type: n.Type, Title: n.Title,
					Status: n.Status, Stage: stage,
					Tags: nonNil(n.Tags), Links: nonNil(n.Links),
					Source: sourceOf(file), File: file, Line: n.Line,
					Digest: n.Digest, CoveredBy: cov,
				})

				if !recognised {
					l.Problems = append(l.Problems, ListedProblem{
						Kind: ListProblemUnknownStatus, Repo: name, ID: n.ID, File: file, Line: n.Line,
						Message: fmt.Sprintf("status %q maps to no stage", n.Status),
					})
				}
				if first, dup := seen[n.ID]; dup {
					l.Problems = append(l.Problems, ListedProblem{
						Kind: ListProblemDuplicateID, Repo: name, ID: n.ID, File: file, Line: n.Line,
						Message: fmt.Sprintf("%s is already defined at %s", n.ID, first),
					})
				} else {
					seen[n.ID] = fmt.Sprintf("%s:%d", file, n.Line)
				}
				l.Problems = append(l.Problems, linkProblems(name, file, n, definedIn)...)
			}
		}
		if count > 0 {
			l.Summary.ReposWithNeeds++
		} else {
			l.Summary.ReposWithoutNeeds++
		}
	}

	sort.SliceStable(l.Needs, func(i, j int) bool {
		a, b := l.Needs[i], l.Needs[j]
		if a.Repo != b.Repo {
			return a.Repo < b.Repo
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	sort.SliceStable(l.Problems, func(i, j int) bool {
		a, b := l.Problems[i], l.Problems[j]
		if a.Repo != b.Repo {
			return a.Repo < b.Repo
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Kind < b.Kind
	})
	return l
}

// linkProblems resolves each :links: target: the need's own repository first,
// then the single other repository that defines it.
func linkProblems(repo, file string, n needs.Need, definedIn map[string]map[string]bool) []ListedProblem {
	var out []ListedProblem
	for _, target := range n.Links {
		owners := definedIn[target]
		if owners[repo] {
			continue
		}
		switch len(owners) {
		case 1:
			continue
		case 0:
			out = append(out, ListedProblem{
				Kind: ListProblemUnknownLink, Repo: repo, ID: n.ID, File: file, Line: n.Line,
				Message: fmt.Sprintf("%s links to %s, which no repository defines", n.ID, target),
			})
		default:
			out = append(out, ListedProblem{
				Kind: ListProblemAmbiguousLink, Repo: repo, ID: n.ID, File: file, Line: n.Line,
				Message: fmt.Sprintf("%s links to %s, which is defined in %s", n.ID, target, strings.Join(sortedKeys(owners), ", ")),
			})
		}
	}
	return out
}

// sourceOf names the docs folder a file came from: "requirements" for
// docs/requirements/x.rst, "docs" for a file directly under docs/.
func sourceOf(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) >= 3 && parts[0] == "docs" {
		return parts[1]
	}
	return "docs"
}

// isOpen reports whether a need still asks for work.
func isOpen(n ListedNeed) bool {
	switch n.Stage {
	case StageImplemented:
		return len(n.CoveredBy) == 0
	case StageClosed:
		return false
	}
	return true
}

// filter applies the repository, stage, type and open filters, then fills in
// the summary from what is left.
func filter(l Listing, o ListOptions, dirOf map[string]string) Listing {
	keepRepo := matcher(o.Repos)
	keepStage := matcher(o.Stages)
	keepType := matcher(o.Types)

	kept := []ListedNeed{}
	for _, n := range l.Needs {
		if keepRepo != nil && !keepRepo[n.Repo] && !keepRepo[path.Base(dirOf[n.Repo])] {
			continue
		}
		if keepStage != nil && !keepStage[n.Stage] {
			continue
		}
		if keepType != nil && !keepType[n.Type] {
			continue
		}
		if o.Open && !isOpen(n) {
			continue
		}
		kept = append(kept, n)
	}
	if keepRepo != nil {
		problems := []ListedProblem{}
		for _, p := range l.Problems {
			if keepRepo[p.Repo] || keepRepo[path.Base(dirOf[p.Repo])] {
				problems = append(problems, p)
			}
		}
		l.Problems = problems
		if l.Changes != nil {
			l.Changes = filterChanges(*l.Changes, keepRepo, dirOf)
		}
	}

	l.Needs = kept
	l.Summary.ByStage = map[string]int{}
	l.Summary.Needs, l.Summary.Covered, l.Summary.Open = 0, 0, 0
	perRepo := map[string]*RepoSummary{}
	for _, n := range kept {
		rs := perRepo[n.Repo]
		if rs == nil {
			rs = &RepoSummary{Repo: n.Repo, ByStage: map[string]int{}}
			perRepo[n.Repo] = rs
		}
		l.Summary.Needs++
		rs.Needs++
		l.Summary.ByStage[n.Stage]++
		rs.ByStage[n.Stage]++
		if len(n.CoveredBy) > 0 {
			l.Summary.Covered++
			rs.Covered++
		}
		if isOpen(n) {
			l.Summary.Open++
			rs.Open++
		}
	}

	if o.Summary {
		for _, name := range sortedKeys(perRepo) {
			l.Summary.Repos = append(l.Summary.Repos, *perRepo[name])
		}
		l.Needs = []ListedNeed{}
	}
	return l
}

func filterChanges(c Changes, keep map[string]bool, dirOf map[string]string) *Changes {
	in := func(keys []NeedKey) []NeedKey {
		out := []NeedKey{}
		for _, k := range keys {
			if keep[k.Repo] || keep[path.Base(dirOf[k.Repo])] {
				out = append(out, k)
			}
		}
		return out
	}
	return &Changes{Since: c.Since, Added: in(c.Added), Changed: in(c.Changed), Removed: in(c.Removed)}
}

func matcher(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, v := range values {
		m[strings.TrimSpace(v)] = true
	}
	return m
}

// signatures gives each need the value whose change makes it "changed".
func signatures(needs []ListedNeed) map[NeedKey]string {
	out := make(map[NeedKey]string, len(needs))
	for _, n := range needs {
		var b strings.Builder
		b.WriteString(n.Digest)
		b.WriteString("|")
		b.WriteString(n.Status)
		for _, c := range n.CoveredBy {
			fmt.Fprintf(&b, "|%s:%s", c.Kind, c.Test)
		}
		out[NeedKey{Repo: n.Repo, ID: n.ID}] = b.String()
	}
	return out
}

// compare reports the changes from an old set of signatures to a new one.
func compare(since string, old, new map[NeedKey]string) Changes {
	c := Changes{Since: since, Added: []NeedKey{}, Changed: []NeedKey{}, Removed: []NeedKey{}}
	for k, sig := range new {
		prev, ok := old[k]
		switch {
		case !ok:
			c.Added = append(c.Added, k)
		case prev != sig:
			c.Changed = append(c.Changed, k)
		}
	}
	for k := range old {
		if _, ok := new[k]; !ok {
			c.Removed = append(c.Removed, k)
		}
	}
	for _, keys := range [][]NeedKey{c.Added, c.Changed, c.Removed} {
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].Repo != keys[j].Repo {
				return keys[i].Repo < keys[j].Repo
			}
			return keys[i].ID < keys[j].ID
		})
	}
	return c
}

// DiffListings reports what changed from one listing to another, by the same
// rule as -changes. It reads nothing but the two listings, so a consumer with
// no working tree can follow changes from build artifacts.
func DiffListings(old, new Listing) (Changes, error) {
	if old.Schema != new.Schema {
		return Changes{}, fmt.Errorf("listings have different schemas (%d and %d); regenerate the older one", old.Schema, new.Schema)
	}
	return compare(old.Generated, signatures(old.Needs), signatures(new.Needs)), nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
