package reqtrace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/needs"
)

// The index makes a workspace listing cheap to repeat. It records every
// directory and candidate file it has seen, with modified times, and on each
// call re-reads only what moved:
//
//   - a directory's modified time changes when an entry is added, removed or
//     renamed directly inside it, which is how new and vanished files are
//     found without walking everything again;
//   - a file's modified time and size change when it is edited, which a
//     directory's time does not show;
//   - a repository whose git HEAD moved, or whose manifest changed, is walked
//     again in full, reusing every file whose stat is unchanged.
//
// There is deliberately no file watcher: on macOS one would hold a descriptor
// per watched directory, thousands across a large workspace, and it would
// need a resident process.

const indexSchema = 1

// absent is the recorded modified time of a scoped root that does not exist
// yet, so it is checked again on every refresh.
const absent int64 = -1

// scopedRoots are the trees within a repository that can hold needs or tests.
var scopedRoots = []string{"docs", "test", "src/go"}

// skipDirNames are never descended into: build output, dependencies, fixtures.
var skipDirNames = map[string]bool{
	"target": true, "build": true, "_build": true, "node_modules": true,
	"vendor": true, "testdata": true, ".artifacts": true, "__pycache__": true,
}

type workspaceIndex struct {
	Schema    int    `json:"schema"`
	Root      string `json:"root"`
	Refreshed string `json:"refreshed"`
	// RootMtime is the workspace directory's own modified time.
	RootMtime int64 `json:"root_mtime"`
	// Top lists the directories directly under the root that were examined.
	Top []string `json:"top"`
	// Containers are top-level directories that are not repositories, with
	// their modified times, so a repository created inside one is found.
	Containers map[string]int64 `json:"containers"`
	// Repos is keyed by the repository's path relative to the root.
	Repos map[string]*repoState `json:"repos"`
	// Signatures are the needs as of Refreshed, for -changes.
	Signatures map[string]string `json:"signatures"`
}

type repoState struct {
	// Dir is the repository's path relative to the workspace, "" when a
	// single repository is listed; Abs is never saved.
	Dir string `json:"dir"`
	Abs string `json:"-"`
	// Key is the name listings use when the manifest name is shared by more
	// than one folder in the workspace; see assignKeys.
	Key           string                `json:"-"`
	Config        repoConfig            `json:"config"`
	ManifestMtime int64                 `json:"manifest_mtime"`
	Head          string                `json:"head"`
	Dirs          map[string]int64      `json:"dirs"`
	Files         map[string]*fileState `json:"files"`
}

type fileState struct {
	Mtime int64        `json:"mtime"`
	Size  int64        `json:"size"`
	Needs []needs.Need `json:"needs,omitempty"`
	Tests []TestCase   `json:"tests,omitempty"`
}

// refreshStats says how much work a refresh did, for tests and timing.
type refreshStats struct {
	Parsed   int64
	Rewalked int64
}

// name is the repository as listings show it: its unique key when one was
// assigned, otherwise its manifest name.
func (r *repoState) name() string {
	if r.Key != "" {
		return r.Key
	}
	return r.baseName()
}

// baseName is the manifest name, or the folder name without a manifest.
func (r *repoState) baseName() string {
	if r.Config.Name != "" {
		return r.Config.Name
	}
	if r.Dir != "" {
		return path.Base(r.Dir)
	}
	return filepath.Base(r.Abs)
}

func (r *repoState) scheme() Scheme {
	return SchemeFromName(r.name())
}

// ListRepo lists one repository. It keeps no index: one repository is quick
// to read, and the output must not depend on what was read before.
func ListRepo(repoRoot string, o ListOptions) (Listing, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return Listing{}, err
	}
	r := &repoState{Abs: abs}
	var stats refreshStats
	if err := r.walk(nil, &stats); err != nil {
		return Listing{}, err
	}
	l := assemble([]*repoState{r})
	l.Generated = now(o).Format(time.RFC3339)
	return filter(l, o, map[string]string{r.name(): ""}), nil
}

// ListWorkspace lists every repository under root, through the index.
func ListWorkspace(root string, o ListOptions) (Listing, error) {
	l, _, err := listWorkspace(root, o)
	return l, err
}

func listWorkspace(root string, o ListOptions) (Listing, refreshStats, error) {
	var stats refreshStats
	abs, err := filepath.Abs(root)
	if err != nil {
		return Listing{}, stats, err
	}

	var warnings []string
	indexPath, cacheErr := indexFile(o.CacheDir, abs)
	if cacheErr != nil {
		warnings = append(warnings, cacheErr.Error())
	}

	idx := (*workspaceIndex)(nil)
	if indexPath != "" && !o.Rebuild {
		idx = loadIndex(indexPath, abs)
	}
	previous := ""
	if idx != nil {
		previous = idx.Refreshed
	} else {
		idx = &workspaceIndex{Schema: indexSchema, Root: abs, RootMtime: absent,
			Containers: map[string]int64{}, Repos: map[string]*repoState{}}
	}

	if err := idx.refresh(&stats); err != nil {
		return Listing{}, stats, err
	}

	repos := make([]*repoState, 0, len(idx.Repos))
	for _, dir := range sortedKeys(idx.Repos) {
		repos = append(repos, idx.Repos[dir])
	}
	shared := assignKeys(repos)
	dirOf := map[string]string{}
	for _, r := range repos {
		dirOf[r.name()] = r.Dir
	}
	l := assemble(repos)
	l.Problems = append(shared, l.Problems...)
	l.Generated = now(o).Format(time.RFC3339)
	l.Workspace = filepath.Base(abs)
	l.Summary.Dirs = len(idx.Top)

	current := signatures(l.Needs)
	if o.Changes {
		old := map[NeedKey]string{}
		for k, sig := range idx.Signatures {
			repo, id, _ := strings.Cut(k, "\x00")
			old[NeedKey{Repo: repo, ID: id}] = sig
		}
		c := compare(previous, old, current)
		l.Changes = &c
	}
	idx.Signatures = map[string]string{}
	for k, sig := range current {
		idx.Signatures[k.Repo+"\x00"+k.ID] = sig
	}
	idx.Refreshed = l.Generated

	if indexPath != "" {
		if err := saveIndex(indexPath, idx); err != nil {
			warnings = append(warnings, fmt.Sprintf("index not saved, next call rescans: %v", err))
		}
	}
	l.Warnings = warnings
	return filter(l, o, dirOf), stats, nil
}

// assignKeys keeps repository names unique. Folders that share a manifest
// name — a copy or a second worktree of one repository — are each listed as
// name@folder, so their needs are never merged, and the sharing is reported.
func assignKeys(repos []*repoState) []ListedProblem {
	byName := map[string][]*repoState{}
	for _, r := range repos {
		r.Key = ""
		byName[r.baseName()] = append(byName[r.baseName()], r)
	}
	var problems []ListedProblem
	for _, name := range sortedKeys(byName) {
		group := byName[name]
		if len(group) < 2 {
			continue
		}
		var dirs []string
		for _, r := range group {
			r.Key = name + "@" + r.Dir
			dirs = append(dirs, r.Dir)
		}
		problems = append(problems, ListedProblem{
			Kind: ListProblemDuplicateRepo, Repo: name,
			Message: fmt.Sprintf("folders %s share the manifest name %s; each is listed as %s@<folder>", strings.Join(dirs, ", "), name, name),
		})
	}
	return problems
}

func now(o ListOptions) time.Time {
	if o.Now != nil {
		return o.Now().UTC()
	}
	return time.Now().UTC()
}

// refresh brings the index up to date with the workspace on disk.
func (idx *workspaceIndex) refresh(stats *refreshStats) error {
	info, err := os.Stat(idx.Root)
	if err != nil {
		return err
	}

	// Which repositories exist. The root's own time says whether top-level
	// directories came or went; each container's time says the same one
	// level down.
	if mtime := info.ModTime().UnixNano(); mtime != idx.RootMtime {
		if err := idx.rediscover(); err != nil {
			return err
		}
		idx.RootMtime = mtime
	} else {
		for _, name := range sortedKeys(idx.Containers) {
			if err := idx.recheckContainer(name); err != nil {
				return err
			}
		}
	}

	// Each repository refreshes independently, so they run in parallel.
	work := make(chan *repoState)
	errs := make(chan error, len(idx.Repos))
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range work {
				if err := r.refresh(stats); err != nil {
					errs <- fmt.Errorf("%s: %w", r.Dir, err)
				}
			}
		}()
	}
	for _, dir := range sortedKeys(idx.Repos) {
		r := idx.Repos[dir]
		r.Abs = filepath.Join(idx.Root, filepath.FromSlash(dir))
		work <- r
	}
	close(work)
	wg.Wait()
	close(errs)
	return <-errs
}

// rediscover lists the top level again: new directories are classified, and
// repositories and containers that vanished are dropped.
func (idx *workspaceIndex) rediscover() error {
	entries, err := os.ReadDir(idx.Root)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	idx.Top = nil
	for _, e := range entries {
		if !eligibleDir(e) {
			continue
		}
		name := e.Name()
		present[name] = true
		idx.Top = append(idx.Top, name)
		if _, known := idx.Repos[name]; known {
			continue
		}
		if _, known := idx.Containers[name]; known {
			if err := idx.recheckContainer(name); err != nil {
				return err
			}
			continue
		}
		if err := idx.classify(name); err != nil {
			return err
		}
	}
	for dir := range idx.Repos {
		if top, _, _ := strings.Cut(dir, "/"); !present[top] {
			delete(idx.Repos, dir)
		}
	}
	for name := range idx.Containers {
		if !present[name] {
			delete(idx.Containers, name)
		}
	}
	return nil
}

// classify records a new top-level directory as a repository or a container.
func (idx *workspaceIndex) classify(name string) error {
	abs := filepath.Join(idx.Root, name)
	if isRepoDir(abs) {
		idx.Repos[name] = &repoState{Dir: name}
		return nil
	}
	idx.Containers[name] = absent
	return idx.recheckContainer(name)
}

// recheckContainer looks inside a container when its modified time moved:
// for repositories one level down, or for the container having become a
// repository itself.
func (idx *workspaceIndex) recheckContainer(name string) error {
	abs := filepath.Join(idx.Root, name)
	info, err := os.Stat(abs)
	if os.IsNotExist(err) {
		delete(idx.Containers, name)
		idx.dropReposUnder(name)
		return nil
	}
	if err != nil {
		return err
	}
	mtime := info.ModTime().UnixNano()
	if mtime == idx.Containers[name] {
		return nil
	}

	if isRepoDir(abs) {
		delete(idx.Containers, name)
		idx.dropReposUnder(name)
		idx.Repos[name] = &repoState{Dir: name}
		return nil
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, e := range entries {
		if !eligibleDir(e) {
			continue
		}
		dir := name + "/" + e.Name()
		if isRepoDir(filepath.Join(abs, e.Name())) {
			present[dir] = true
			if _, known := idx.Repos[dir]; !known {
				idx.Repos[dir] = &repoState{Dir: dir}
			}
		}
	}
	for dir := range idx.Repos {
		if strings.HasPrefix(dir, name+"/") && !present[dir] {
			delete(idx.Repos, dir)
		}
	}
	idx.Containers[name] = mtime
	return nil
}

func (idx *workspaceIndex) dropReposUnder(name string) {
	for dir := range idx.Repos {
		if strings.HasPrefix(dir, name+"/") {
			delete(idx.Repos, dir)
		}
	}
}

// isRepoDir reports whether dir is a repository: it has a manifest, or it has
// one of the docs folders needs live in.
func isRepoDir(dir string) bool {
	for _, marker := range []string{
		filepath.Join("meta-data", "manifest.json"),
		filepath.Join("docs", "requirements"),
		filepath.Join("docs", "proposals"),
		filepath.Join("docs", "features"),
	} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

// hasReposBelow reports whether dir is a workspace: some directory up to two
// levels below it is a repository.
func hasReposBelow(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !eligibleDir(e) {
			continue
		}
		child := filepath.Join(dir, e.Name())
		if isRepoDir(child) {
			return true
		}
		grand, _ := os.ReadDir(child)
		for _, g := range grand {
			if eligibleDir(g) && isRepoDir(filepath.Join(child, g.Name())) {
				return true
			}
		}
	}
	return false
}

func eligibleDir(e fs.DirEntry) bool {
	return e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !skipDirNames[e.Name()]
}

// refresh brings one repository up to date. A new repository, a moved HEAD
// or a changed manifest means a full walk; otherwise only what moved is read.
func (r *repoState) refresh(stats *refreshStats) error {
	manifestMtime := statMtime(filepath.Join(r.Abs, "meta-data", "manifest.json"))
	head := gitHead(r.Abs)

	if r.Files == nil {
		return r.walk(nil, stats)
	}
	if manifestMtime != r.ManifestMtime {
		atomic.AddInt64(&stats.Rewalked, 1)
		// The manifest names the repository, and the name decides how test
		// annotations expand, so nothing parsed before can be reused.
		return r.walk(nil, stats)
	}
	if head != r.Head {
		atomic.AddInt64(&stats.Rewalked, 1)
		prev := r.Files
		return r.walk(prev, stats)
	}

	// Directories, parents before children, so a vanished parent takes its
	// subtree with it before the children are looked at.
	dirs := sortedKeys(r.Dirs)
	for _, rel := range dirs {
		recorded, ok := r.Dirs[rel]
		if !ok {
			continue // dropped with a parent
		}
		abs := filepath.Join(r.Abs, filepath.FromSlash(rel))
		mtime := statMtime(abs)
		switch {
		case mtime == recorded:
			continue
		case mtime == absent:
			r.dropSubtree(rel)
			if isScopedRoot(rel) {
				r.Dirs[rel] = absent
			}
		case recorded == absent:
			if err := r.walkDir(rel, nil, stats); err != nil {
				return err
			}
		default:
			if err := r.rescanDir(rel, mtime, stats); err != nil {
				return err
			}
		}
	}

	// Edits, which no directory time shows.
	for _, rel := range sortedKeys(r.Files) {
		f := r.Files[rel]
		info, err := os.Stat(filepath.Join(r.Abs, filepath.FromSlash(rel)))
		if err != nil {
			delete(r.Files, rel)
			continue
		}
		if info.ModTime().UnixNano() != f.Mtime || info.Size() != f.Size {
			if err := r.parseFile(rel, info, stats); err != nil {
				return err
			}
		}
	}
	return nil
}

// walk records the scoped trees from scratch. Files whose stat matches prev
// keep their parse.
func (r *repoState) walk(prev map[string]*fileState, stats *refreshStats) error {
	cfg, _, err := readRepoConfig(r.Abs)
	if err != nil {
		return err
	}
	r.Config = cfg
	r.ManifestMtime = statMtime(filepath.Join(r.Abs, "meta-data", "manifest.json"))
	r.Head = gitHead(r.Abs)
	r.Dirs = map[string]int64{}
	r.Files = map[string]*fileState{}
	for _, root := range scopedRoots {
		if statMtime(filepath.Join(r.Abs, filepath.FromSlash(root))) == absent {
			r.Dirs[root] = absent
			continue
		}
		if err := r.walkDir(root, prev, stats); err != nil {
			return err
		}
	}
	return nil
}

// walkDir records rel and, where the tree is recursive, everything below it.
func (r *repoState) walkDir(rel string, prev map[string]*fileState, stats *refreshStats) error {
	abs := filepath.Join(r.Abs, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		return nil // gone between listing and walking
	}
	r.Dirs[rel] = info.ModTime().UnixNano()

	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	for _, e := range entries {
		child := rel + "/" + e.Name()
		if e.IsDir() {
			if recursiveTree(rel) && eligibleDir(e) {
				if err := r.walkDir(child, prev, stats); err != nil {
					return err
				}
			}
			continue
		}
		if !isCandidate(child) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if old, ok := prev[child]; ok && old.Mtime == info.ModTime().UnixNano() && old.Size == info.Size() {
			r.Files[child] = old
			continue
		}
		if err := r.parseFile(child, info, stats); err != nil {
			return err
		}
	}
	return nil
}

// rescanDir reads a directory whose modified time moved: new entries are
// added, vanished ones dropped.
func (r *repoState) rescanDir(rel string, mtime int64, stats *refreshStats) error {
	abs := filepath.Join(r.Abs, filepath.FromSlash(rel))
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for _, e := range entries {
		child := rel + "/" + e.Name()
		present[child] = true
		if e.IsDir() {
			if _, known := r.Dirs[child]; !known && recursiveTree(rel) && eligibleDir(e) {
				if err := r.walkDir(child, nil, stats); err != nil {
					return err
				}
			}
			continue
		}
		if _, known := r.Files[child]; known || !isCandidate(child) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if err := r.parseFile(child, info, stats); err != nil {
			return err
		}
	}
	for d := range r.Dirs {
		if parentOf(d) == rel && !present[d] {
			r.dropSubtree(d)
		}
	}
	for f := range r.Files {
		if parentOf(f) == rel && !present[f] {
			delete(r.Files, f)
		}
	}
	r.Dirs[rel] = mtime
	return nil
}

func (r *repoState) dropSubtree(rel string) {
	for d := range r.Dirs {
		if d == rel || strings.HasPrefix(d, rel+"/") {
			delete(r.Dirs, d)
		}
	}
	for f := range r.Files {
		if strings.HasPrefix(f, rel+"/") {
			delete(r.Files, f)
		}
	}
}

// parseFile reads one candidate file into the index.
func (r *repoState) parseFile(rel string, info fs.FileInfo, stats *refreshStats) error {
	atomic.AddInt64(&stats.Parsed, 1)
	abs := filepath.Join(r.Abs, filepath.FromSlash(rel))
	f := &fileState{Mtime: info.ModTime().UnixNano(), Size: info.Size()}

	switch {
	case strings.HasPrefix(rel, "docs/"):
		found, err := needs.ParseFile(abs, r.Config.types())
		if err != nil {
			return fmt.Errorf("parsing %s: %w", rel, err)
		}
		for i := range found {
			found[i].Body = "" // the index keeps the digest, not the text
		}
		f.Needs = found
	case strings.HasSuffix(rel, ".robot"):
		tests, err := ParseRobotTests([]string{abs}, r.Abs, r.scheme())
		if err != nil {
			return err
		}
		f.Tests = tests
	case strings.HasSuffix(rel, "_test.go"):
		tests, err := ParseGoTests(abs, r.Abs, nil, r.scheme())
		if err != nil {
			// A test file that does not parse claims nothing; it must not
			// stop the whole workspace listing.
			tests = nil
		}
		f.Tests = tests
	}
	r.Files[rel] = f
	return nil
}

// isCandidate reports whether a repository-relative file can hold needs or
// tests: .rst under docs/ (not the generated page), .robot directly under
// test/, and _test.go under src/go/.
func isCandidate(rel string) bool {
	switch {
	case strings.HasPrefix(rel, "docs/"):
		return strings.EqualFold(path.Ext(rel), ".rst") && path.Base(rel) != GeneratedFile
	case strings.HasPrefix(rel, "test/"):
		return parentOf(rel) == "test" && strings.HasSuffix(rel, ".robot")
	case strings.HasPrefix(rel, "src/go/"):
		return strings.HasSuffix(rel, "_test.go")
	}
	return false
}

// recursiveTree reports whether subdirectories of rel are part of the scan.
// test/ is not: Robot suites are read from test/*.robot only.
func recursiveTree(rel string) bool {
	return rel == "docs" || strings.HasPrefix(rel, "docs/") || rel == "src/go" || strings.HasPrefix(rel, "src/go/")
}

func isScopedRoot(rel string) bool {
	for _, root := range scopedRoots {
		if rel == root {
			return true
		}
	}
	return false
}

func parentOf(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return ""
}

func statMtime(p string) int64 {
	info, err := os.Stat(p)
	if err != nil {
		return absent
	}
	return info.ModTime().UnixNano()
}

// gitHead returns what HEAD points at, resolved to a commit where the ref is
// readable, or "" outside git. It reads the files git keeps rather than
// running git, which across hundreds of repositories would dominate a refresh.
func gitHead(repo string) string {
	gitDir := filepath.Join(repo, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		// A worktree: .git is a file naming the real git directory.
		data, err := os.ReadFile(gitDir)
		if err != nil {
			return ""
		}
		target, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
		if !ok {
			return ""
		}
		gitDir = strings.TrimSpace(target)
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(repo, gitDir)
		}
	}

	data, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	head := strings.TrimSpace(string(data))
	ref, ok := strings.CutPrefix(head, "ref:")
	if !ok {
		return head // detached
	}
	ref = strings.TrimSpace(ref)

	common := gitDir
	if c, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		common = filepath.Join(gitDir, strings.TrimSpace(string(c)))
	}
	for _, dir := range []string{gitDir, common} {
		if sha, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(ref))); err == nil {
			return ref + "@" + strings.TrimSpace(string(sha))
		}
	}
	if packed, err := os.ReadFile(filepath.Join(common, "packed-refs")); err == nil {
		for _, line := range strings.Split(string(packed), "\n") {
			if sha, name, ok := strings.Cut(strings.TrimSpace(line), " "); ok && name == ref {
				return ref + "@" + sha
			}
		}
	}
	return ref
}

// indexFile names the index for a workspace: one file per absolute workspace
// path, under the cache directory.
func indexFile(cacheDir, workspace string) (string, error) {
	if cacheDir == "" {
		cacheDir = os.Getenv("REQTRACE_CACHE_DIR")
	}
	if cacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("no cache directory for the index (set REQTRACE_CACHE_DIR): %w", err)
		}
		cacheDir = filepath.Join(base, "reqtrace")
	}
	sum := sha256.Sum256([]byte(workspace))
	return filepath.Join(cacheDir, hex.EncodeToString(sum[:8])+".json"), nil
}

// loadIndex reads an index, or returns nil when it is missing, unreadable,
// from another schema or for another workspace — all of which mean rebuild.
func loadIndex(file, workspace string) *workspaceIndex {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var idx workspaceIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil
	}
	if idx.Schema != indexSchema || idx.Root != workspace || idx.Repos == nil {
		return nil
	}
	if idx.Containers == nil {
		idx.Containers = map[string]int64{}
	}
	for _, r := range idx.Repos {
		if r.Dirs == nil || r.Files == nil {
			return nil
		}
	}
	return &idx
}

// saveIndex writes the index atomically: a temporary file in the same
// directory, renamed over the old one, so a concurrent reader sees either.
func saveIndex(file string, idx *workspaceIndex) error {
	sort.Strings(idx.Top)
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".index-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := bytes.NewReader(data).WriteTo(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}
