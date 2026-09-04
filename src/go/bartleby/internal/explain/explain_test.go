package explain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRequester records what it was asked and answers from a script.
type fakeRequester struct {
	calls  int
	system string
	user   string
	answer string
	err    error
}

func (f *fakeRequester) Explain(_ context.Context, system, user string) (string, error) {
	f.calls++
	f.system = system
	f.user = user
	return f.answer, f.err
}

// repoWithLogs builds a repository whose last build warned, matching what the
// transform image writes.
func repoWithLogs(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	write(t, filepath.Join(repo, "meta-data", "VERSION"), "1.4\n")
	write(t, filepath.Join(repo, "meta-data", "manifest.json"),
		`{"name": "test-docs", "bartleby": {"roots": {"main": {"builders": ["html"]}}}}`)

	write(t, filepath.Join(repo, "docs", "index.rst"), strings.Join([]string{
		"Title", "=====", "", "Intro paragraph.", "",
		"See :ref:`missing-label` for details.", "",
		".. toctree::", "   :maxdepth: 2", "", "   missing_page", "",
	}, "\n"))

	logs := filepath.Join(repo, "target", "bartleby", "logs")
	write(t, filepath.Join(logs, "html.log"), strings.Join([]string{
		"Running Sphinx v8.2.3",
		"building [html]: targets for 1 source files that are out of date",
		"/tmp/tmpabcdef12/source/index.rst:8: WARNING: toctree contains reference to nonexisting document 'missing_page' [toc.not_readable]",
		"build finished with problems",
	}, "\n"))
	write(t, filepath.Join(logs, "html-warnings.log"), strings.Join([]string{
		"/tmp/tmpabcdef12/source/index.rst:8: WARNING: toctree contains reference to nonexisting document 'missing_page' [toc.not_readable]",
		"/tmp/tmpabcdef12/source/index.rst:6: WARNING: undefined label: 'missing-label' [ref.ref]",
	}, "\n"))

	return repo
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Requirements: REQ_EXPL_004
func TestCollectGathersTheEvidence(t *testing.T) {
	repo := repoWithLogs(t)

	payload, err := Collect(Options{RepoPath: repo})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if payload.Builder != "html" {
		t.Errorf("builder = %q, want html", payload.Builder)
	}
	if payload.Version != "1.4" {
		t.Errorf("version = %q, want 1.4", payload.Version)
	}
	if !strings.Contains(payload.Warnings, "undefined label") {
		t.Errorf("warnings missing:\n%s", payload.Warnings)
	}
	if !strings.Contains(payload.LogTail, "Running Sphinx") {
		t.Errorf("log tail missing:\n%s", payload.LogTail)
	}
	if !strings.Contains(payload.Manifest, "test-docs") {
		t.Errorf("manifest missing:\n%s", payload.Manifest)
	}
	if len(payload.Excerpts) == 0 {
		t.Fatal("no source excerpts were collected")
	}

	rendered := payload.Render()
	for _, want := range []string{
		"Sphinx warnings and errors",
		"End of the build log",
		"## Source",
		"manifest.json",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered payload is missing the %q section", want)
		}
	}
}

// The warnings name a path inside the container's temp directory, which exists
// nowhere on the host; the excerpt has to come from the repository's own file.
//
// Requirements: REQ_EXPL_006
func TestCollectResolvesContainerPaths(t *testing.T) {
	repo := repoWithLogs(t)

	payload, err := Collect(Options{RepoPath: repo})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	excerpt := payload.Excerpts[0]
	if excerpt.Path != "docs/index.rst" {
		t.Errorf("excerpt path = %q, want docs/index.rst", excerpt.Path)
	}
	if !strings.Contains(excerpt.Text, "missing-label") {
		t.Errorf("excerpt should contain the cited source:\n%s", excerpt.Text)
	}
	if !strings.Contains(excerpt.Text, ">") {
		t.Errorf("the cited line should be marked:\n%s", excerpt.Text)
	}
}

// Requirements: REQ_EXPL_006
func TestResolveCitationPaths(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "docs", "guide", "page.rst"), "x\n")
	write(t, filepath.Join(repo, "notes.rst"), "y\n")

	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"container source path", "/tmp/tmpabc123/source/guide/page.rst", "docs/guide/page.rst", true},
		{"repo-relative path", "docs/guide/page.rst", "docs/guide/page.rst", true},
		{"docs-relative path", "guide/page.rst", "docs/guide/page.rst", true},
		{"repo root file", "notes.rst", "notes.rst", true},
		{"unknown absolute path", "/usr/local/lib/python3.13/sphinx/thing.py", "", false},
		{"nonexistent file", "/tmp/tmpabc123/source/nope.rst", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolve(repo, tt.raw)
			if ok != tt.ok {
				t.Fatalf("resolved = %v, want %v (got %q)", ok, tt.ok, got)
			}
			if got != tt.want {
				t.Errorf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

// Requirements: REQ_EXPL_004
func TestCitations(t *testing.T) {
	text := strings.Join([]string{
		"/tmp/tmpabc/source/index.rst:8: WARNING: one",
		"/tmp/tmpabc/source/index.rst:8: WARNING: one again",
		"docs/other.rst:42: WARNING: two",
		"not a citation at all",
		"WARNING: no line number here",
		"/tmp/tmpabc/source/third.rst:0: WARNING: line zero is not a line",
	}, "\n")

	got := Citations(text)

	if len(got) != 2 {
		t.Fatalf("got %d citations, want 2 (deduplicated, line 0 rejected): %+v", len(got), got)
	}
	if got[0].Line != 8 || got[1].Line != 42 {
		t.Errorf("citations = %+v", got)
	}
	if got[1].Raw != "docs/other.rst" {
		t.Errorf("raw path = %q", got[1].Raw)
	}
}

// Three warnings about neighbouring lines should produce one excerpt with three
// marked lines, not three near-identical copies of the same page.
//
// Requirements: REQ_EXPL_004_SPEC001
func TestMergeWindows(t *testing.T) {
	tests := []struct {
		name    string
		lines   []int
		context int
		total   int
		want    [][2]int
	}{
		{"single", []int{10}, 3, 100, [][2]int{{7, 13}}},
		{"overlapping merge", []int{6, 8, 13}, 10, 13, [][2]int{{1, 13}}},
		{"adjacent merge", []int{5, 12}, 3, 100, [][2]int{{2, 15}}},
		{"separate", []int{5, 40}, 3, 100, [][2]int{{2, 8}, {37, 43}}},
		{"clamped to the file", []int{2}, 10, 5, [][2]int{{1, 5}}},
		{"beyond the file is dropped", []int{99}, 2, 5, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeWindows(tt.lines, tt.context, tt.total)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("window %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// A LaTeX failure cites a line of the generated .tex, so no warning resolves to a
// source file and the model would otherwise see the error with none of the markup
// that caused it.
//
// Requirements: REQ_EXPL_004_SPEC002
func TestCollectFallsBackToTheRootDocument(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "meta-data", "manifest.json"),
		`{"name": "pdf-docs", "bartleby": {"roots": {"main": {"root_doc": "index", "builders": ["pdf"]}}}}`)
	write(t, filepath.Join(repo, "docs", "index.rst"),
		"Title\n=====\n\n.. raw:: latex\n\n   \\thisMacroDoesNotExist\n")

	logs := filepath.Join(repo, "target", "bartleby", "logs")
	write(t, filepath.Join(logs, "pdf.log"), "latexmk output with no rst citations\n")
	write(t, filepath.Join(logs, "pdf-latex-doc.log"),
		"! Undefined control sequence.\nl.91 \\thisMacroDoesNotExist\n")

	payload, err := Collect(Options{RepoPath: repo})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(payload.Excerpts) != 1 {
		t.Fatalf("got %d excerpts, want the root document: %+v", len(payload.Excerpts), payload.Excerpts)
	}
	if payload.Excerpts[0].Path != "docs/index.rst" {
		t.Errorf("excerpt path = %q, want docs/index.rst", payload.Excerpts[0].Path)
	}
	if !strings.Contains(payload.Excerpts[0].Text, "thisMacroDoesNotExist") {
		t.Error("the root document should carry the offending markup")
	}
	if !strings.Contains(payload.Render(), "no warning named a source line") {
		t.Error("the substitution should be stated, not silent")
	}
}

// Requirements: REQ_EXPL_001
func TestCollectPicksTheMostRecentLog(t *testing.T) {
	repo := repoWithLogs(t)
	logs := filepath.Join(repo, "target", "bartleby", "logs")

	// A pdf build after the html one, plus companion files that are not build
	// logs and must not be chosen.
	write(t, filepath.Join(logs, "pdf.log"), "latexmk output\n")
	write(t, filepath.Join(logs, "pdf-warnings.log"), "a warning\n")
	write(t, filepath.Join(logs, "pdf-latex-doc.log"), "! Undefined control sequence.\nl.42 \\bogus\n")

	payload, err := Collect(Options{RepoPath: repo})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if payload.Builder != "pdf" {
		t.Errorf("builder = %q, want pdf — the newest build log", payload.Builder)
	}
	if !strings.Contains(payload.LatexSlice, "Undefined control sequence") {
		t.Errorf("the LaTeX error should be included:\n%s", payload.LatexSlice)
	}
	if !strings.Contains(payload.Render(), "LaTeX log") {
		t.Error("the rendered payload should have a LaTeX section")
	}
}

// Requirements: REQ_EXPL_001
func TestCollectHonoursBuilderAndLogPath(t *testing.T) {
	repo := repoWithLogs(t)
	logs := filepath.Join(repo, "target", "bartleby", "logs")
	write(t, filepath.Join(logs, "pdf.log"), "latexmk output\n")

	byBuilder, err := Collect(Options{RepoPath: repo, Builder: "html"})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if byBuilder.Builder != "html" {
		t.Errorf("builder = %q, want the requested html", byBuilder.Builder)
	}

	byPath, err := Collect(Options{RepoPath: repo, LogPath: filepath.Join(logs, "pdf.log")})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if byPath.Builder != "pdf" {
		t.Errorf("builder = %q, want pdf from the explicit log path", byPath.Builder)
	}
}

// Requirements: REQ_EXPL_001
func TestCollectWithoutLogs(t *testing.T) {
	_, err := Collect(Options{RepoPath: t.TempDir()})
	if !errors.Is(err, ErrNoLogs) {
		t.Fatalf("err = %v, want ErrNoLogs", err)
	}
	if !strings.Contains(err.Error(), "run a build first") {
		t.Errorf("the error should say what to do: %v", err)
	}
}

// Requirements: REQ_EXPL_005
func TestCollectCapsThePayloadAndSaysSo(t *testing.T) {
	repo := repoWithLogs(t)
	logs := filepath.Join(repo, "target", "bartleby", "logs")

	// A log far larger than the cap, as a real latexmk run produces.
	var big strings.Builder
	for i := range 20000 {
		big.WriteString("noise line with some length to it, number ")
		big.WriteString(strings.Repeat("x", 40))
		big.WriteString("\n")
		_ = i
	}
	write(t, filepath.Join(logs, "html.log"), big.String())

	const cap = 8 << 10
	payload, err := Collect(Options{RepoPath: repo, MaxBytes: cap})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if payload.Size() > cap {
		t.Errorf("payload is %d bytes, want at most %d", payload.Size(), cap)
	}
	if len(payload.Notes) == 0 {
		t.Error("trimming must be reported, not silent")
	}
	if !strings.Contains(payload.Render(), "What was left out") {
		t.Error("the rendered payload should say what was left out")
	}
	// The warnings are the point of the exercise and should survive trimming.
	if !strings.Contains(payload.Warnings, "undefined label") {
		t.Error("the warnings should be kept in preference to log noise")
	}
}

// Requirements: REQ_EXPL_008
func TestResolvePromptPrecedence(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, RepoPromptPath), "repo prompt")

	flagFile := filepath.Join(t.TempDir(), "flag.md")
	write(t, flagFile, "flag prompt")
	envFile := filepath.Join(t.TempDir(), "env.md")
	write(t, envFile, "env file prompt")

	full := map[string]string{
		EnvPromptFile: envFile,
		EnvPrompt:     "inline prompt",
	}
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	tests := []struct {
		name string
		o    PromptOptions
		want string
	}{
		{"flag wins", PromptOptions{File: flagFile, RepoPath: repo, Env: env(full)}, "flag prompt"},
		{"prompt file env next", PromptOptions{RepoPath: repo, Env: env(full)}, "env file prompt"},
		{"inline env next", PromptOptions{RepoPath: repo, Env: env(map[string]string{EnvPrompt: "inline prompt"})}, "inline prompt"},
		{"repo file next", PromptOptions{RepoPath: repo, Env: env(nil)}, "repo prompt"},
		{"built-in last", PromptOptions{RepoPath: t.TempDir(), Env: env(nil)}, DefaultSystemPrompt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := ResolvePrompt(tt.o)
			if err != nil {
				t.Fatalf("ResolvePrompt: %v", err)
			}
			if prompt.System != tt.want {
				t.Errorf("prompt = %q, want %q", truncate(prompt.System), truncate(tt.want))
			}
			if prompt.Source == "" {
				t.Error("the prompt source should be reported so the CLI can name it")
			}
		})
	}
}

// Requirements: REQ_EXPL_008
func TestResolvePromptRejectsUnusableFiles(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.md")
	if _, err := ResolvePrompt(PromptOptions{File: missing, Env: func(string) string { return "" }}); err == nil {
		t.Error("a missing --prompt-file should be an error, not a silent fallback")
	}

	empty := filepath.Join(t.TempDir(), "empty.md")
	write(t, empty, "   \n")
	if _, err := ResolvePrompt(PromptOptions{File: empty, Env: func(string) string { return "" }}); err == nil {
		t.Error("an empty prompt file should be an error")
	}
}

// Requirements: REQ_EXPL_001, REQ_EXPL_010, REQ_EXPL_011
func TestRunAsksOnceAndSavesTheAnswer(t *testing.T) {
	repo := repoWithLogs(t)
	fake := &fakeRequester{answer: "The toctree names a document that does not exist."}

	var out, status strings.Builder
	answer, err := Run(context.Background(), RunOptions{
		Collect:   Options{RepoPath: repo},
		Prompt:    PromptOptions{RepoPath: repo, Env: func(string) string { return "" }},
		Requester: fake,
		Out:       &out,
		Status:    &status,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if fake.calls != 1 {
		t.Errorf("the requester was called %d times, want exactly 1", fake.calls)
	}
	if answer != fake.answer {
		t.Errorf("answer = %q", answer)
	}
	if !strings.Contains(out.String(), fake.answer) {
		t.Errorf("the answer should be printed:\n%s", out.String())
	}
	if !strings.Contains(fake.user, "WARNING") || !strings.Contains(fake.user, "docs/index.rst") {
		t.Error("the request should carry the warnings and the cited source")
	}
	if fake.system == "" {
		t.Error("the request should carry a system prompt")
	}

	saved := filepath.Join(repo, "target", "bartleby", "logs", "html-explain.md")
	data, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("the explanation should be saved beside the logs: %v", err)
	}
	if !strings.Contains(string(data), fake.answer) {
		t.Errorf("saved file = %q", data)
	}
	if !strings.Contains(status.String(), "Sending") {
		t.Errorf("the status line should say what is being sent:\n%s", status.String())
	}
}

// Requirements: REQ_EXPL_003
func TestRunDryRunSendsNothing(t *testing.T) {
	repo := repoWithLogs(t)
	fake := &fakeRequester{answer: "should not be reached"}

	var out, status strings.Builder
	rendered, err := Run(context.Background(), RunOptions{
		Collect:   Options{RepoPath: repo},
		Prompt:    PromptOptions{RepoPath: repo, Env: func(string) string { return "" }},
		Requester: fake,
		Out:       &out,
		Status:    &status,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if fake.calls != 0 {
		t.Error("a dry run must not call the API")
	}
	if !strings.Contains(status.String(), "Would send") {
		t.Errorf("a dry run should say it is not sending:\n%s", status.String())
	}
	if !strings.Contains(out.String(), "Sphinx warnings") || !strings.Contains(rendered, "Sphinx warnings") {
		t.Error("a dry run should print the exact payload")
	}

	if _, err := os.Stat(filepath.Join(repo, "target", "bartleby", "logs", "html-explain.md")); !os.IsNotExist(err) {
		t.Error("a dry run should not write an explanation file")
	}
}

// Requirements: REQ_EXPL_002
func TestRunSurfacesRequestFailures(t *testing.T) {
	repo := repoWithLogs(t)
	fake := &fakeRequester{err: errors.New("429 rate limited")}

	_, err := Run(context.Background(), RunOptions{
		Collect:   Options{RepoPath: repo},
		Prompt:    PromptOptions{RepoPath: repo, Env: func(string) string { return "" }},
		Requester: fake,
		Out:       new(strings.Builder),
		Status:    new(strings.Builder),
	})
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("err = %v, want the request failure", err)
	}
}

// Requirements: REQ_EXPL_007
func TestHasCredentials(t *testing.T) {
	none := func(string) string { return "" }
	noProfile := func() bool { return false }

	if HasCredentials(none, noProfile) {
		t.Error("no key, no token, no profile should report no credentials")
	}

	for _, key := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_IDENTITY_TOKEN"} {
		env := func(k string) string {
			if k == key {
				return "value"
			}
			return ""
		}
		if !HasCredentials(env, noProfile) {
			t.Errorf("%s should count as credentials", key)
		}
	}

	// An `ant auth login` profile is credentials even with no environment set.
	if !HasCredentials(none, func() bool { return true }) {
		t.Error("a stored profile should count as credentials")
	}
}

// Requirements: REQ_EXPL_009
func TestClaudeDefaults(t *testing.T) {
	if DefaultModel != "claude-opus-5" {
		t.Errorf("default model = %q; the default should be the strongest model", DefaultModel)
	}
	if got := displayModel(""); got != DefaultModel {
		t.Errorf("displayModel(\"\") = %q, want %q", got, DefaultModel)
	}
	if got := displayModel("claude-haiku-4-5"); got != "claude-haiku-4-5" {
		t.Errorf("displayModel should report the override, got %q", got)
	}
}

func truncate(s string) string {
	if len(s) <= 40 {
		return s
	}
	return s[:40] + "..."
}
