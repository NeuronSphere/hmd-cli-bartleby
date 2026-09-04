// Package explain assembles what is known about a failed documentation build and
// asks Claude to interpret it, in a single request.
//
// A Sphinx or LaTeX failure is usually explained somewhere in thousands of lines
// of log, in terms that assume you know Sphinx internals. The useful context is
// small and specific: the warnings Sphinx emitted, the tail of the build log, the
// LaTeX error if the PDF builder ran, and — the part that turns a vague warning
// into an answerable question — the actual source lines the warnings point at.
package explain

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

// Defaults for how much of each input to collect.
const (
	DefaultTailLines    = 120
	DefaultContextLines = 10
	DefaultLatexLines   = 40
	DefaultMaxBytes     = 100 << 10 // 100 KiB
)

// ErrNoLogs reports that there is no build log to explain.
var ErrNoLogs = errors.New("no build log found")

// Options controls collection.
type Options struct {
	// RepoPath is the repository being built; source excerpts are resolved
	// against it.
	RepoPath string
	// LogsDir defaults to <RepoPath>/target/bartleby/logs.
	LogsDir string
	// Builder selects <builder>.log. Empty means the most recently written.
	Builder string
	// LogPath names a log file directly, bypassing LogsDir and Builder.
	LogPath string

	TailLines    int
	ContextLines int
	LatexLines   int
	MaxBytes     int
}

func (o *Options) applyDefaults() {
	if o.LogsDir == "" {
		o.LogsDir = filepath.Join(o.RepoPath, "target", "bartleby", "logs")
	}
	if o.TailLines == 0 {
		o.TailLines = DefaultTailLines
	}
	if o.ContextLines == 0 {
		o.ContextLines = DefaultContextLines
	}
	if o.LatexLines == 0 {
		o.LatexLines = DefaultLatexLines
	}
	if o.MaxBytes == 0 {
		o.MaxBytes = DefaultMaxBytes
	}
}

// Citation is a file and line a warning refers to.
type Citation struct {
	// Raw is the path as the log spelled it, which is a path inside the
	// container's temporary build directory.
	Raw  string
	Path string // repo-relative, when it could be resolved
	Line int
}

// Excerpt is the source around a citation.
type Excerpt struct {
	Path      string
	FirstLine int
	Text      string
}

// Payload is everything gathered about one failed build.
type Payload struct {
	Repo    string
	Version string
	Builder string

	LogFile string
	LogTail string

	WarningsFile string
	Warnings     string

	LatexFile  string
	LatexSlice string

	Excerpts []Excerpt
	Manifest string

	// Notes records what was left out, so the report can say so rather than
	// quietly truncating.
	Notes []string
}

// Collect gathers the logs, citations, and source excerpts for a build.
func Collect(o Options) (*Payload, error) {
	o.applyDefaults()

	logPath, builder, err := findLog(o)
	if err != nil {
		return nil, err
	}

	p := &Payload{
		Repo:    filepath.Base(o.RepoPath),
		Builder: builder,
		LogFile: logPath,
	}

	if version, err := os.ReadFile(filepath.Join(o.RepoPath, "meta-data", "VERSION")); err == nil {
		p.Version = strings.TrimSpace(string(version))
	}
	if manifest, err := os.ReadFile(filepath.Join(o.RepoPath, "meta-data", "manifest.json")); err == nil {
		p.Manifest = strings.TrimSpace(string(manifest))
	}

	lines, err := readLines(logPath)
	if err != nil {
		return nil, err
	}
	p.LogTail = strings.Join(tail(lines, o.TailLines), "\n")
	if len(lines) > o.TailLines {
		p.Notes = append(p.Notes, fmt.Sprintf("%s is %d lines; the last %d are included",
			filepath.Base(logPath), len(lines), o.TailLines))
	}

	// Warnings first: this is the file that usually contains the answer.
	warningsPath := filepath.Join(o.LogsDir, builder+"-warnings.log")
	if data, err := os.ReadFile(warningsPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		p.WarningsFile = warningsPath
		p.Warnings = strings.TrimSpace(string(data))
	}

	if latexPath, slice := latexError(o, builder); slice != "" {
		p.LatexFile = latexPath
		p.LatexSlice = slice
	}

	p.Excerpts = excerpts(o, p.Warnings+"\n"+p.LogTail)

	// A LaTeX failure cites a line of the generated .tex, not of any .rst, so
	// nothing resolves and the model would be asked to explain an error with no
	// sight of the markup that caused it. The documents being built are the next
	// best thing.
	if len(p.Excerpts) == 0 {
		p.Excerpts = rootDocuments(o)
		if len(p.Excerpts) > 0 {
			p.Notes = append(p.Notes,
				"no warning named a source line, so the root document(s) are included instead")
		}
	}

	p.trim(o.MaxBytes)
	return p, nil
}

// findLog resolves which build log to explain.
func findLog(o Options) (path, builder string, err error) {
	if o.LogPath != "" {
		if _, err := os.Stat(o.LogPath); err != nil {
			return "", "", fmt.Errorf("reading %s: %w", o.LogPath, err)
		}
		return o.LogPath, builderFromLogName(o.LogPath), nil
	}

	if o.Builder != "" {
		path = filepath.Join(o.LogsDir, o.Builder+".log")
		if _, err := os.Stat(path); err != nil {
			return "", "", fmt.Errorf("%w for builder %q at %s", ErrNoLogs, o.Builder, path)
		}
		return path, o.Builder, nil
	}

	entries, err := filepath.Glob(filepath.Join(o.LogsDir, "*.log"))
	if err != nil {
		return "", "", err
	}

	// The warnings and LaTeX files are companions of a build log, not build logs.
	var candidates []string
	for _, entry := range entries {
		name := filepath.Base(entry)
		if strings.Contains(name, "-warnings.") || strings.Contains(name, "-latex-") {
			continue
		}
		candidates = append(candidates, entry)
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("%w in %s — run a build first", ErrNoLogs, o.LogsDir)
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, errA := os.Stat(candidates[i])
		b, errB := os.Stat(candidates[j])
		if errA != nil || errB != nil {
			return candidates[i] < candidates[j]
		}
		return a.ModTime().After(b.ModTime())
	})

	return candidates[0], builderFromLogName(candidates[0]), nil
}

func builderFromLogName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// latexError returns the newest LaTeX log for a builder and the slice of it worth
// reading: LaTeX reports a failure on a line beginning with "! ", and what follows
// is the part that names the construct it could not set.
func latexError(o Options, builder string) (path, slice string) {
	matches, err := filepath.Glob(filepath.Join(o.LogsDir, builder+"-latex-*.log"))
	if err != nil || len(matches) == 0 {
		return "", ""
	}
	sort.Strings(matches)
	path = matches[len(matches)-1]

	lines, err := readLines(path)
	if err != nil {
		return "", ""
	}

	for i, line := range lines {
		if strings.HasPrefix(line, "! ") {
			end := min(i+o.LatexLines, len(lines))
			return path, strings.Join(lines[i:end], "\n")
		}
	}

	// No error marker: the tail is still the most likely place for a complaint.
	return path, strings.Join(tail(lines, o.LatexLines), "\n")
}

// citationRe matches the "<file>:<line>:" prefix Sphinx puts on a warning.
var citationRe = regexp.MustCompile(`(?m)^\s*([^\s:][^:]*?):(\d+):`)

// tmpSourceRe matches the container's copy of the docs directory, which is where
// Sphinx builds from: /tmp/tmpXXXXXXXX/source/<path>.
var tmpSourceRe = regexp.MustCompile(`^/tmp/[^/]+/source/(.+)$`)

// Citations extracts the file and line references from log text, in the order
// they first appear, deduplicated.
func Citations(text string) []Citation {
	var out []Citation
	seen := make(map[string]bool)

	for _, match := range citationRe.FindAllStringSubmatch(text, -1) {
		raw := strings.TrimSpace(match[1])
		line, err := strconv.Atoi(match[2])
		if err != nil || line <= 0 {
			continue
		}
		key := raw + ":" + match[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Citation{Raw: raw, Line: line})
	}

	return out
}

// resolve maps a path as the log spelled it to a repo-relative path.
//
// Sphinx runs against a copy of the docs tree inside the container, so its
// warnings name paths like /tmp/tmpyzfqp5i3/source/index.rst — which exists
// nowhere on the host. The stable part is the path below source/, which is the
// path below docs/ in the repository.
func resolve(repoPath, raw string) (string, bool) {
	var candidates []string

	if m := tmpSourceRe.FindStringSubmatch(raw); m != nil {
		candidates = append(candidates, filepath.Join("docs", m[1]), m[1])
	}

	if filepath.IsAbs(raw) {
		// Fall back to trying progressively shorter suffixes of the path, which
		// covers container layouts this code does not know about.
		parts := strings.Split(filepath.Clean(raw), string(filepath.Separator))
		for i := range parts {
			suffix := filepath.Join(parts[i:]...)
			if suffix != "" {
				candidates = append(candidates, suffix, filepath.Join("docs", suffix))
			}
		}
	} else {
		candidates = append(candidates, raw, filepath.Join("docs", raw))
	}

	for _, candidate := range candidates {
		full := filepath.Join(repoPath, candidate)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return filepath.ToSlash(candidate), true
		}
	}

	return "", false
}

// excerpts reads the source around every citation that resolves to a real file.
//
// Windows that overlap are merged: three warnings about neighbouring lines of one
// document should produce one excerpt with three marked lines, not three copies of
// the same page.
func excerpts(o Options, text string) []Excerpt {
	var order []string
	cited := make(map[string][]int)

	for _, citation := range Citations(text) {
		rel, ok := resolve(o.RepoPath, citation.Raw)
		if !ok {
			continue
		}
		if _, seen := cited[rel]; !seen {
			order = append(order, rel)
		}
		cited[rel] = append(cited[rel], citation.Line)
	}

	var out []Excerpt
	for _, rel := range order {
		lines, err := readLines(filepath.Join(o.RepoPath, rel))
		if err != nil {
			continue
		}

		marked := make(map[int]bool, len(cited[rel]))
		numbers := make([]int, 0, len(cited[rel]))
		for _, line := range cited[rel] {
			if line > len(lines) || marked[line] {
				continue
			}
			marked[line] = true
			numbers = append(numbers, line)
		}
		sort.Ints(numbers)

		for _, window := range mergeWindows(numbers, o.ContextLines, len(lines)) {
			numbered := make([]string, 0, window[1]-window[0]+1)
			for i := window[0]; i <= window[1]; i++ {
				marker := " "
				if marked[i] {
					marker = ">"
				}
				numbered = append(numbered, fmt.Sprintf("%s %4d | %s", marker, i, lines[i-1]))
			}
			out = append(out, Excerpt{
				Path:      rel,
				FirstLine: window[0],
				Text:      strings.Join(numbered, "\n"),
			})
		}
	}

	return out
}

// rootDocumentLines caps how much of a root document is attached when there is
// nothing more specific to send.
const rootDocumentLines = 150

// rootDocuments reads the documents the manifest builds, for the case where no
// warning names a line.
func rootDocuments(o Options) []Excerpt {
	m, err := manifest.Read(o.RepoPath)
	if err != nil && !errors.Is(err, manifest.ErrNotFound) {
		return nil
	}

	names := []string{}
	for _, root := range m.Bartleby.Roots {
		names = append(names, root.Doc())
	}
	if len(names) == 0 {
		names = []string{manifest.DefaultRootDoc}
	}
	sort.Strings(names)

	var out []Excerpt
	for _, name := range names {
		rel := filepath.Join("docs", name+".rst")
		lines, err := readLines(filepath.Join(o.RepoPath, rel))
		if err != nil {
			continue
		}

		shown := min(len(lines), rootDocumentLines)
		numbered := make([]string, 0, shown)
		for i := 1; i <= shown; i++ {
			numbered = append(numbered, fmt.Sprintf("  %4d | %s", i, lines[i-1]))
		}
		if shown < len(lines) {
			numbered = append(numbered, fmt.Sprintf("  ... %d more lines", len(lines)-shown))
		}

		out = append(out, Excerpt{
			Path:      filepath.ToSlash(rel),
			FirstLine: 1,
			Text:      strings.Join(numbered, "\n"),
		})

		// One document is enough context; more crowds out the error itself.
		break
	}

	return out
}

// mergeWindows turns cited line numbers into merged [first, last] ranges.
func mergeWindows(lines []int, context, total int) [][2]int {
	var windows [][2]int

	for _, line := range lines {
		first := max(line-context, 1)
		last := min(line+context, total)
		if first > total {
			continue
		}

		// Merge with the previous window when they touch or overlap.
		if n := len(windows); n > 0 && first <= windows[n-1][1]+1 {
			if last > windows[n-1][1] {
				windows[n-1][1] = last
			}
			continue
		}
		windows = append(windows, [2]int{first, last})
	}

	return windows
}

// trim drops the least valuable material until the rendered payload fits.
//
// Order matters. The warnings and the source excerpts are the point of the
// exercise, so the LaTeX slice and the log tail give way first; the warnings are
// only touched when they alone exceed the budget.
func (p *Payload) trim(maxBytes int) {
	if maxBytes <= 0 || p.Size() <= maxBytes {
		return
	}

	// Each note added below makes the payload slightly larger, so every cut
	// leaves room for the sentence describing it.
	const noteAllowance = 128
	const minWorthKeeping = 512

	steps := []struct {
		field *string
		what  string
	}{
		{&p.LatexSlice, "the LaTeX log slice"},
		{&p.LogTail, "the build log tail"},
		{&p.Warnings, "the warnings"},
	}

	for _, step := range steps {
		if *step.field == "" || p.Size() <= maxBytes {
			continue
		}

		keep := len(*step.field) - (p.Size() - maxBytes) - noteAllowance
		if keep < minWorthKeeping {
			*step.field = ""
			p.Notes = append(p.Notes, step.what+" was dropped to fit the size cap")
			continue
		}

		*step.field = lastBytes(*step.field, keep)
		p.Notes = append(p.Notes, step.what+" was shortened to fit the size cap")
	}

	// Last resort: keep the first excerpt, which is the one the first warning
	// pointed at.
	if p.Size() > maxBytes && len(p.Excerpts) > 1 {
		dropped := len(p.Excerpts) - 1
		p.Excerpts = p.Excerpts[:1]
		p.Notes = append(p.Notes,
			fmt.Sprintf("%d further source excerpt(s) were dropped to fit the size cap", dropped))
	}
}

// Render lays the payload out as the text sent to the model.
func (p *Payload) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Failed Bartleby build\n\nRepository: %s\n", p.Repo)
	if p.Version != "" {
		fmt.Fprintf(&b, "Version: %s\n", p.Version)
	}
	fmt.Fprintf(&b, "Builder: %s\nBuild log: %s\n", p.Builder, p.LogFile)

	if p.Warnings != "" {
		fmt.Fprintf(&b, "\n## Sphinx warnings and errors (%s)\n\n```\n%s\n```\n", p.WarningsFile, p.Warnings)
	} else {
		b.WriteString("\n## Sphinx warnings and errors\n\nNone were recorded.\n")
	}

	if p.LatexSlice != "" {
		fmt.Fprintf(&b, "\n## LaTeX log (%s)\n\n```\n%s\n```\n", p.LatexFile, p.LatexSlice)
	}

	if p.LogTail != "" {
		fmt.Fprintf(&b, "\n## End of the build log\n\n```\n%s\n```\n", p.LogTail)
	}

	if len(p.Excerpts) > 0 {
		b.WriteString("\n## Source\n")
		for _, e := range p.Excerpts {
			note := fmt.Sprintf("from line %d", e.FirstLine)
			if strings.Contains(e.Text, ">") {
				note += "; > marks a line a warning named"
			}
			fmt.Fprintf(&b, "\n### %s (%s)\n\n```\n%s\n```\n", e.Path, note, e.Text)
		}
	}

	if p.Manifest != "" {
		fmt.Fprintf(&b, "\n## meta-data/manifest.json\n\n```json\n%s\n```\n", p.Manifest)
	}

	if len(p.Notes) > 0 {
		b.WriteString("\n## What was left out\n\n")
		for _, note := range p.Notes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}

	return b.String()
}

// Size reports the rendered payload size, for telling the user what is being sent.
func (p *Payload) Size() int { return len(p.Render()) }

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	// Build logs contain very long lines — a LaTeX font warning can run to
	// thousands of characters — so the default 64 KiB limit is not enough.
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return lines, nil
}

func tail(lines []string, n int) []string {
	if n <= 0 || len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

func lastBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…\n" + s[len(s)-n:]
}
