package reqtrace

// Verification joins what the documentation says with what a test run did.
//
// Traceability (the rest of this package) answers "which tests cover this
// requirement?" from the source tree. Verification answers "did they pass,
// against which implementation, and when?". Its two inputs are deliberately
// build artefacts rather than source:
//
//   - needs.json, which sphinx-needs exports from a bartleby build. It holds
//     every req and spec item and the test items this package generates into
//     traceability.rst, each test item linked to the requirements it covers.
//     The requirement-to-test linkage is read from there, so what is verified is
//     exactly what the published documentation claims.
//   - one or more result files from a run: a Robot Framework output.xml (what
//     bender writes) or the event stream of "go test -json". Each carries its
//     own timestamp.
//
// Results are labelled with the implementation they ran against ("python",
// "go", "go@0.3.1"), so one requirement gets one verdict per implementation:
// the conformance matrix a port is judged by.

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BaselineTag marks a test that existed before a port began. Baseline tests
// are the conformance floor; the verification report counts them separately
// from the tests a port adds.
const BaselineTag = "baseline"

// VerificationFile is the page Verify writes by default, beside
// traceability.rst. Unlike that page it describes one set of runs rather than
// the source tree, so it is not checked for staleness.
const VerificationFile = "verification_matrix.rst"

// Need is one item from a sphinx-needs needs.json export.
type Need struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
	Links  []string `json:"links"`
	// Docname is the document the item was declared in, without extension:
	// "requirements/cli", "proposals/NERD001_...".
	Docname string `json:"docname"`
}

// Needs is the content of a needs.json export that verification uses.
type Needs struct {
	Project string
	Created string
	Items   map[string]Need
}

// LoadNeeds reads a needs.json written by sphinx-needs (needs_build_json). It
// uses the current version's needs, which is the only version a bartleby build
// writes.
func LoadNeeds(path string) (Needs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Needs{}, err
	}
	var raw struct {
		Created        string `json:"created"`
		CurrentVersion string `json:"current_version"`
		Project        string `json:"project"`
		Versions       map[string]struct {
			Needs map[string]Need `json:"needs"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Needs{}, fmt.Errorf("%s: %w", path, err)
	}
	version, ok := raw.Versions[raw.CurrentVersion]
	if !ok {
		return Needs{}, fmt.Errorf("%s: no needs for current_version %q", path, raw.CurrentVersion)
	}
	return Needs{Project: raw.Project, Created: raw.Created, Items: version.Needs}, nil
}

// WithoutProposals drops the req and spec items declared in docs/proposals,
// the NERD documents. Traceability leaves them out unless asked
// (-include-proposals), and verification judges the same set.
func (n Needs) WithoutProposals() Needs {
	kept := make(map[string]Need, len(n.Items))
	for id, item := range n.Items {
		if (item.Type == "req" || item.Type == "spec") && strings.HasPrefix(item.Docname, "proposals/") {
			continue
		}
		kept[id] = item
	}
	n.Items = kept
	return n
}

// Outcome is the status of one test in one run, as the runner reported it.
type Outcome string

const (
	OutcomePass Outcome = "PASS"
	OutcomeFail Outcome = "FAIL"
	OutcomeSkip Outcome = "SKIP"
)

// TestResult is one test's result in one run.
type TestResult struct {
	Kind    Kind
	Suite   string // Robot suite file stem, or Go package (last import path element)
	Name    string
	Outcome Outcome
	Tags    []string
	Start   string // as the runner recorded it
}

// Baseline reports whether the test carries the baseline tag.
func (r TestResult) Baseline() bool {
	for _, tag := range r.Tags {
		if strings.EqualFold(tag, BaselineTag) {
			return true
		}
	}
	return false
}

// RunResults is one result file, labelled with the implementation it ran
// against.
type RunResults struct {
	Implementation string // e.g. "python", "go"
	Version        string // optional, e.g. "0.3.1"
	Source         string // the result file
	Generated      string // the run's own timestamp
	RunID          string // Bender's run ID, for a run folder
	Stamp          string // a run folder's <stamp>, which orders runs
	Results        []TestResult
}

// ResultSpec is one -results argument: LABEL[@VERSION]=PATH. A bare PATH is
// labelled "default", except that a Bender run folder then names its own label.
type ResultSpec struct {
	Implementation string
	Version        string
	Path           string
}

// ParseResultSpec parses LABEL[@VERSION]=PATH.
func ParseResultSpec(s string) (ResultSpec, error) {
	label, file, found := strings.Cut(s, "=")
	if !found {
		return ResultSpec{Implementation: "default", Path: s}, nil
	}
	if file == "" || label == "" {
		return ResultSpec{}, fmt.Errorf("results %q: want LABEL[@VERSION]=PATH", s)
	}
	impl, version, _ := strings.Cut(label, "@")
	return ResultSpec{Implementation: impl, Version: version, Path: file}, nil
}

// LoadResults reads one result file, or one Bender run folder
// (bender/runs/<stamp>_<repo>_<run_id>/). A file's parser is chosen by
// content: an XML document is a Robot output.xml, anything else the JSON event
// stream of "go test -json".
func LoadResults(spec ResultSpec) (RunResults, error) {
	if isRunFolder(spec.Path) {
		return loadRunFolder(spec)
	}
	return loadResultFile(spec)
}

func loadResultFile(spec ResultSpec) (RunResults, error) {
	data, err := os.ReadFile(spec.Path)
	if err != nil {
		return RunResults{}, err
	}
	run := RunResults{Implementation: spec.Implementation, Version: spec.Version, Source: spec.Path}
	if strings.HasPrefix(strings.TrimSpace(string(data)), "<") {
		run.Generated, run.Results, err = parseRobotOutput(strings.NewReader(string(data)))
	} else {
		run.Generated, run.Results, err = parseGoTestJSON(strings.NewReader(string(data)))
	}
	if err != nil {
		return RunResults{}, fmt.Errorf("%s: %w", spec.Path, err)
	}
	return run, nil
}

// parseRobotOutput reads a Robot Framework output.xml. It handles schema 4
// (Robot 6: starttime) and 5 (Robot 7: start), and nested suites such as the
// ones pabot produces; a test belongs to the nearest suite whose source is a
// .robot file, which is how reqtrace names the suite in traceability.rst.
func parseRobotOutput(r io.Reader) (generated string, results []TestResult, err error) {
	dec := xml.NewDecoder(r)
	var (
		sources []string // suite source stack
		inTest  *TestResult
		depth   int // element depth inside the current <test>
		inTag   bool
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if inTest != nil {
				depth++
				switch {
				case t.Name.Local == "tag" && depth == 1:
					inTag = true
				case t.Name.Local == "status" && depth == 1:
					inTest.Outcome = Outcome(attr(t, "status"))
					inTest.Start = firstNonEmpty(attr(t, "start"), attr(t, "starttime"))
				}
				continue
			}
			switch t.Name.Local {
			case "robot":
				generated = attr(t, "generated")
			case "suite":
				sources = append(sources, attr(t, "source"))
			case "test":
				inTest = &TestResult{Kind: KindRobot, Suite: robotSuite(sources), Name: attr(t, "name")}
				depth = 0
			}
		case xml.CharData:
			if inTag {
				inTest.Tags = append(inTest.Tags, strings.TrimSpace(string(t)))
			}
		case xml.EndElement:
			if inTest != nil {
				if depth == 0 && t.Name.Local == "test" {
					results = append(results, *inTest)
					inTest = nil
					continue
				}
				if t.Name.Local == "tag" {
					inTag = false
				}
				depth--
				continue
			}
			if t.Name.Local == "suite" && len(sources) > 0 {
				sources = sources[:len(sources)-1]
			}
		}
	}
	if generated == "" {
		return "", nil, errors.New("not a Robot Framework output.xml (no <robot generated=...>)")
	}
	return generated, results, nil
}

// robotSuite is the stem of the innermost .robot source on the stack.
func robotSuite(sources []string) string {
	for i := len(sources) - 1; i >= 0; i-- {
		if strings.HasSuffix(sources[i], ".robot") {
			base := path.Base(filepath.ToSlash(sources[i]))
			return strings.TrimSuffix(base, ".robot")
		}
	}
	return ""
}

// parseGoTestJSON reads "go test -json" output. Only top-level tests count;
// subtests roll up into their parent, which is what a requirement annotation
// sits on.
func parseGoTestJSON(r io.Reader) (generated string, results []TestResult, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var ev struct {
			Time    time.Time
			Action  string
			Package string
			Test    string
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return "", nil, fmt.Errorf("not go test -json output: %w", err)
		}
		if generated == "" && !ev.Time.IsZero() {
			generated = ev.Time.Format(time.RFC3339)
		}
		if ev.Test == "" || strings.Contains(ev.Test, "/") {
			continue
		}
		var outcome Outcome
		switch ev.Action {
		case "pass":
			outcome = OutcomePass
		case "fail":
			outcome = OutcomeFail
		case "skip":
			outcome = OutcomeSkip
		default:
			continue
		}
		results = append(results, TestResult{
			Kind:    KindGo,
			Suite:   path.Base(ev.Package),
			Name:    ev.Test,
			Outcome: outcome,
			Start:   ev.Time.Format(time.RFC3339),
		})
	}
	if err := scanner.Err(); err != nil {
		return "", nil, err
	}
	if generated == "" {
		return "", nil, errors.New("no go test -json events")
	}
	return generated, results, nil
}

func attr(e xml.StartElement, name string) string {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Verdict is one requirement's result against one implementation.
type Verdict string

const (
	VerdictPass    Verdict = "pass"
	VerdictFail    Verdict = "fail"
	VerdictSkip    Verdict = "skip"    // every linked test that ran was skipped
	VerdictMissing Verdict = "missing" // no linked test ran
	VerdictExempt  Verdict = "exempt"  // trace-exempt: verified as its text says
)

// Failing reports whether the verdict fails verification.
func (v Verdict) Failing() bool { return v == VerdictFail || v == VerdictMissing }

// TestOutcome is a linked test's result within one implementation.
type TestOutcome struct {
	Test     string  `json:"test"` // the test item's title in needs.json
	NeedID   string  `json:"need_id"`
	Outcome  Outcome `json:"outcome"`
	Baseline bool    `json:"baseline"`
	Source   string  `json:"source"`
	RunAt    string  `json:"run_at"`
	RunID    string  `json:"run_id,omitempty"`
}

// Cell is one requirement × implementation.
type Cell struct {
	Implementation string        `json:"implementation"`
	Version        string        `json:"version,omitempty"`
	Verdict        Verdict       `json:"verdict"`
	Tests          []TestOutcome `json:"tests"`
}

// Row is one requirement and its verdict per implementation.
type Row struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Cells []Cell `json:"cells"`
}

// ImplementationSummary counts one implementation's results.
type ImplementationSummary struct {
	Implementation string          `json:"implementation"`
	Version        string          `json:"version,omitempty"`
	Sources        []string        `json:"sources"`
	RunAt          []string        `json:"run_at"`
	RunIDs         []string        `json:"run_ids,omitempty"`
	Verdicts       map[Verdict]int `json:"verdicts"`
	BaselineTests  int             `json:"baseline_tests"`
	BaselinePassed int             `json:"baseline_passed"`
	NewTests       int             `json:"new_tests"`
	NewPassed      int             `json:"new_passed"`
}

// Verification is the joined result.
type Verification struct {
	Project         string                  `json:"project"`
	NeedsCreated    string                  `json:"needs_created"`
	Implementations []ImplementationSummary `json:"implementations"`
	Rows            []Row                   `json:"rows"`
	// Unmatched lists results for tests the documentation does not declare,
	// which means needs.json predates the tests: rebuild the docs.
	Unmatched []string `json:"unmatched"`
}

// Failing counts requirement × implementation cells that fail verification.
func (v Verification) Failing() int {
	n := 0
	for _, row := range v.Rows {
		for _, c := range row.Cells {
			if c.Verdict.Failing() {
				n++
			}
		}
	}
	return n
}

// Verify joins needs with runs. Runs with the same implementation label are
// one implementation: a Go port is typically verified by its own go test -json
// and by the Robot suites run against it.
func Verify(needs Needs, runs []RunResults) Verification {
	v := Verification{Project: needs.Project, NeedsCreated: needs.Created}

	// The test items, keyed so a result can find its item.
	testsByKey := map[string]Need{}
	goTestsByName := map[string][]Need{}
	linksBack := map[string][]Need{}
	for _, n := range needs.Items {
		if n.Type != "test" {
			continue
		}
		kind, suite, name := splitTestTitle(n)
		testsByKey[string(kind)+"\x00"+suite+"\x00"+name] = n
		if kind == KindGo {
			goTestsByName[name] = append(goTestsByName[name], n)
		}
		for _, id := range n.Links {
			linksBack[id] = append(linksBack[id], n)
		}
	}

	// Group runs by implementation, in first-seen order.
	type implRuns struct {
		summary ImplementationSummary
		byNeed  map[string][]TestOutcome
	}
	var order []string
	impls := map[string]*implRuns{}
	unmatched := map[string]bool{}
	for _, run := range runs {
		ir, ok := impls[run.Implementation]
		if !ok {
			ir = &implRuns{
				summary: ImplementationSummary{Implementation: run.Implementation, Version: run.Version, Verdicts: map[Verdict]int{}},
				byNeed:  map[string][]TestOutcome{},
			}
			impls[run.Implementation] = ir
			order = append(order, run.Implementation)
		}
		ir.summary.Sources = append(ir.summary.Sources, run.Source)
		ir.summary.RunAt = append(ir.summary.RunAt, run.Generated)
		if run.RunID != "" {
			ir.summary.RunIDs = append(ir.summary.RunIDs, run.RunID)
		}
		for _, res := range run.Results {
			need, found := matchResult(res, testsByKey, goTestsByName)
			if res.Baseline() {
				ir.summary.BaselineTests++
				if res.Outcome == OutcomePass {
					ir.summary.BaselinePassed++
				}
			} else {
				ir.summary.NewTests++
				if res.Outcome == OutcomePass {
					ir.summary.NewPassed++
				}
			}
			if !found {
				unmatched[fmt.Sprintf("%s %s: %s (%s)", res.Kind, res.Suite, res.Name, run.Source)] = true
				continue
			}
			ir.byNeed[need.ID] = append(ir.byNeed[need.ID], TestOutcome{
				Test: need.Title, NeedID: need.ID, Outcome: res.Outcome,
				Baseline: res.Baseline(), Source: run.Source, RunAt: run.Generated, RunID: run.RunID,
			})
		}
	}

	var ids []string
	for id, n := range needs.Items {
		if n.Type == "req" || n.Type == "spec" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	for _, id := range ids {
		req := needs.Items[id]
		row := Row{ID: id, Type: req.Type, Title: req.Title}
		linked := linksBack[id]
		sort.Slice(linked, func(i, j int) bool { return linked[i].Title < linked[j].Title })
		for _, implName := range order {
			ir := impls[implName]
			cell := Cell{Implementation: implName, Version: ir.summary.Version}
			for _, t := range linked {
				cell.Tests = append(cell.Tests, ir.byNeed[t.ID]...)
			}
			cell.Verdict = verdictFor(req, cell.Tests)
			ir.summary.Verdicts[cell.Verdict]++
			row.Cells = append(row.Cells, cell)
		}
		v.Rows = append(v.Rows, row)
	}

	for _, implName := range order {
		v.Implementations = append(v.Implementations, impls[implName].summary)
	}
	for u := range unmatched {
		v.Unmatched = append(v.Unmatched, u)
	}
	sort.Strings(v.Unmatched)
	return v
}

func verdictFor(req Need, tests []TestOutcome) Verdict {
	if len(tests) == 0 {
		for _, tag := range req.Tags {
			if strings.EqualFold(tag, ExemptTag) {
				return VerdictExempt
			}
		}
		return VerdictMissing
	}
	skipped := 0
	for _, t := range tests {
		switch t.Outcome {
		case OutcomeFail:
			return VerdictFail
		case OutcomeSkip:
			skipped++
		case OutcomePass:
		default: // NOT RUN and anything unrecognised
			skipped++
		}
	}
	if skipped == len(tests) {
		return VerdictSkip
	}
	return VerdictPass
}

// splitTestTitle recovers kind, suite and name from a generated test item:
// "suite: Name" for Robot, "pkg.TestName" for Go (see TestCase.Title).
func splitTestTitle(n Need) (Kind, string, string) {
	kind := KindGo
	for _, tag := range n.Tags {
		if tag == string(KindRobot) {
			kind = KindRobot
		}
	}
	if kind == KindRobot {
		suite, name, _ := strings.Cut(n.Title, ": ")
		return kind, suite, name
	}
	suite, name, _ := strings.Cut(n.Title, ".")
	return kind, suite, name
}

// matchResult finds a result's test item. Go results carry the import path's
// last element while reqtrace records the package clause, so an external test
// package (pkg_test) and package main are matched by test name when that name
// is unique.
func matchResult(res TestResult, byKey map[string]Need, goByName map[string][]Need) (Need, bool) {
	key := func(suite string) string { return string(res.Kind) + "\x00" + suite + "\x00" + res.Name }
	if n, ok := byKey[key(res.Suite)]; ok {
		return n, true
	}
	if res.Kind == KindGo {
		if n, ok := byKey[key(res.Suite+"_test")]; ok {
			return n, true
		}
		if candidates := goByName[res.Name]; len(candidates) == 1 {
			return candidates[0], true
		}
	}
	return Need{}, false
}

// RenderVerification writes the verification page.
func RenderVerification(v Verification) string {
	var b strings.Builder
	writeLines(&b,
		".. Generated by reqtrace -verify — do not edit.",
		"   It records one set of test runs; regenerate it after the next run.",
		"",
		"Verification Matrix",
		"===================",
		"",
		"Each requirement's verdict against each implementation, from the documented",
		"requirement-to-test links (sphinx-needs ``needs.json``) joined with the",
		"results of the runs below. A requirement passes when every linked test that",
		"ran passed; it is *missing* when none of its tests ran.",
		"",
		fmt.Sprintf("Requirements from ``%s``, built %s.", v.Project, v.NeedsCreated),
		"",
		"Runs",
		"----",
		"",
		".. list-table::",
		"   :header-rows: 1",
		"",
		"   * - Implementation",
		"     - Run at",
		"     - Run ID",
		"     - Results",
		"     - Verdicts",
		"     - Baseline tests passed",
		"     - New tests passed",
	)
	for _, s := range v.Implementations {
		label := s.Implementation
		if s.Version != "" {
			label += " " + s.Version
		}
		writeLines(&b,
			"   * - "+label,
			"     - "+strings.Join(s.RunAt, ", "),
			"     - "+literalList(s.RunIDs),
			"     - "+literalList(s.Sources),
			"     - "+verdictCounts(s.Verdicts),
			fmt.Sprintf("     - %d / %d", s.BaselinePassed, s.BaselineTests),
			fmt.Sprintf("     - %d / %d", s.NewPassed, s.NewTests),
		)
	}
	b.WriteString("\n")

	writeLines(&b, "Matrix", "------", "", ".. list-table::", "   :header-rows: 1", "", "   * - Requirement")
	for _, s := range v.Implementations {
		writeLines(&b, "     - "+s.Implementation)
	}
	for _, row := range v.Rows {
		writeLines(&b, fmt.Sprintf("   * - :need:`%s`", row.ID))
		for _, c := range row.Cells {
			writeLines(&b, "     - "+verdictText(c))
		}
	}
	b.WriteString("\n")

	if len(v.Unmatched) > 0 {
		writeLines(&b,
			"Results the documentation does not declare",
			"------------------------------------------",
			"",
			"These tests ran but have no test item in ``needs.json``: the documentation was",
			"built before they existed. Rebuild it, then verify again.",
			"",
		)
		for _, u := range v.Unmatched {
			writeLines(&b, "- "+escapeRST(u))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func verdictText(c Cell) string {
	switch c.Verdict {
	case VerdictPass:
		return fmt.Sprintf("pass (%d)", len(c.Tests))
	case VerdictFail:
		var failed []string
		for _, t := range c.Tests {
			if t.Outcome == OutcomeFail {
				failed = append(failed, escapeRST(t.Test))
			}
		}
		return "**fail**: " + strings.Join(failed, "; ")
	case VerdictMissing:
		return "**missing**"
	default:
		return string(c.Verdict)
	}
}

func verdictCounts(m map[Verdict]int) string {
	var parts []string
	for _, v := range []Verdict{VerdictPass, VerdictFail, VerdictMissing, VerdictSkip, VerdictExempt} {
		if m[v] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", m[v], v))
		}
	}
	return strings.Join(parts, ", ")
}

func literalList(items []string) string {
	var out []string
	for _, i := range items {
		out = append(out, "``"+i+"``")
	}
	return strings.Join(out, ", ")
}

// escapeRST keeps test names such as "a *star*" from turning into markup.
func escapeRST(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "*", `\*`, "`", "\\`", "|", `\|`, "_", `\_`)
	return r.Replace(s)
}

// VerifyOptions drives one verification pass.
type VerifyOptions struct {
	// Needs is the needs.json path. Empty means the bartleby HTML build output,
	// target/bartleby/html/needs.json, under the repository root.
	Needs string
	// Results are LABEL[@VERSION]=PATH result files or Bender run folders.
	Results []string
	// Runs are directories of Bender run folders, such as bender/runs. Each
	// label's latest run is used, or every run when AllRuns is set.
	Runs    []string
	AllRuns bool
	// Out is the page to write. Empty means docs/requirements/verification_matrix.rst;
	// "-" writes nothing.
	Out string
	// JSON prints the verification as JSON on standard output.
	JSON bool
}

// DefaultNeedsPath is where "bartleby html" leaves needs.json.
func DefaultNeedsPath(repoRoot string) string {
	return filepath.Join(repoRoot, "target", "bartleby", "html", "needs.json")
}

func runVerify(o RunOptions, repo string, out, errOut io.Writer) error {
	vo := o.Verify
	if len(vo.Results) == 0 && len(vo.Runs) == 0 {
		return errors.New("verify: give at least one -results LABEL[@VERSION]=PATH (a Robot output.xml, go test -json output or Bender run folder) or -runs DIR (such as bender/runs)")
	}
	needsPath := vo.Needs
	if needsPath == "" {
		needsPath = DefaultNeedsPath(repo)
	}
	needs, err := LoadNeeds(needsPath)
	if err != nil {
		return fmt.Errorf("verify: reading needs.json (build the docs with \"bartleby html\" first): %w", err)
	}
	if !o.IncludeProposals {
		needs = needs.WithoutProposals()
	}
	var runs []RunResults
	for _, raw := range vo.Results {
		spec, err := ParseResultSpec(raw)
		if err != nil {
			return err
		}
		run, err := LoadResults(spec)
		if err != nil {
			return err
		}
		runs = append(runs, run)
	}
	for _, dir := range vo.Runs {
		found, err := LoadRunsDir(dir, vo.AllRuns)
		if err != nil {
			return err
		}
		runs = append(runs, found...)
	}

	v := Verify(needs, runs)
	for _, u := range v.Unmatched {
		fmt.Fprintf(errOut, "result for a test the documentation does not declare (rebuild the docs): %s\n", u)
	}

	page := vo.Out
	if page == "" {
		page = filepath.Join(repo, "docs", "requirements", VerificationFile)
	}
	if page != "-" {
		if err := os.WriteFile(page, []byte(RenderVerification(v)), 0o644); err != nil {
			return err
		}
	}

	if vo.JSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			return err
		}
	} else if !o.Quiet {
		for _, s := range v.Implementations {
			fmt.Fprintf(out, "%s: %s; baseline %d/%d passed, new %d/%d passed\n",
				s.Implementation, verdictCounts(s.Verdicts), s.BaselinePassed, s.BaselineTests, s.NewPassed, s.NewTests)
		}
		if page != "-" {
			fmt.Fprintf(out, "wrote %s\n", page)
		}
	}

	if n := v.Failing(); n > 0 {
		return fmt.Errorf("verification failed: %d requirement result(s) failing or missing", n)
	}
	return nil
}
