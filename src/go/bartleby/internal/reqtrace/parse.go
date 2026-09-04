package reqtrace

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// needDirective matches the start of a sphinx-needs item we care about.
var needDirective = regexp.MustCompile(`^\s*\.\.\s+(req|spec)::\s*(.*)$`)

// needOption matches an option line inside a needs directive.
var needOption = regexp.MustCompile(`^\s*:([a-z_]+):\s*(.*)$`)

// goAnnotation matches the coverage declaration in a Go test's doc comment.
var goAnnotation = regexp.MustCompile(`(?i)^\s*(?://\s*)?Requirements?:\s*(.+)$`)

// robotRequirementTag matches a Robot tag that names a requirement.
var robotRequirementTag = regexp.MustCompile(`^(?:HMD_[A-Z0-9_]*_)?(?:REQ|NERD)_?[A-Z0-9_]*$`)

// ParseRequirements reads every .rst file under root, skipping the generated
// page, and returns the req and spec items it finds.
//
// The parse is line-oriented rather than a full RST parse: a needs directive is
// a directive line followed by indented option lines, which is unambiguous
// enough, and it keeps this tool free of a docutils dependency.
func ParseRequirements(root, repoRoot string) ([]Requirement, error) {
	var requirements []Requirement

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".rst") {
			return nil
		}
		if d.Name() == GeneratedFile {
			return nil
		}

		found, err := parseRequirementFile(path, repoRoot)
		if err != nil {
			return err
		}
		requirements = append(requirements, found...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s for requirements: %w", root, err)
	}

	return requirements, nil
}

func parseRequirementFile(path, repoRoot string) ([]Requirement, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		requirements []Requirement
		current      *Requirement
		scanner      = bufio.NewScanner(file)
		lineNo       int
	)

	flush := func() {
		if current != nil && current.ID != "" {
			requirements = append(requirements, *current)
		}
		current = nil
	}

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		if match := needDirective.FindStringSubmatch(line); match != nil {
			flush()
			current = &Requirement{
				Type:  match[1],
				Title: strings.TrimSpace(match[2]),
				File:  relativeTo(repoRoot, path),
				Line:  lineNo,
			}
			continue
		}

		if current == nil {
			continue
		}

		// A non-blank line that is not an option ends the option block; the
		// directive body follows, which this parser does not need.
		match := needOption.FindStringSubmatch(line)
		if match == nil {
			if strings.TrimSpace(line) != "" {
				if current.ID != "" {
					flush()
				} else {
					current = nil
				}
			}
			continue
		}

		value := strings.TrimSpace(match[2])
		switch match[1] {
		case "id":
			current.ID = value
		case "status":
			current.Status = value
		case "tags":
			current.Tags = splitCommaList(value)
		case "links":
			current.Links = normalizeIDs(splitCommaList(value))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()

	return requirements, nil
}

// ParseGoTests walks Go test files under root and returns every test function
// along with the requirements its doc comment declares.
//
// Test functions are found through go/ast rather than a regex so that a comment
// which merely mentions a requirement in passing, somewhere in a function body,
// is not mistaken for a coverage claim.
func ParseGoTests(root, repoRoot string, skipDirs []string) ([]TestCase, error) {
	var tests []TestCase
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkip(path, skipDirs) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		parsed, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !isTestFunc(fn) {
				continue
			}

			tests = append(tests, TestCase{
				Kind:         KindGo,
				Name:         fn.Name.Name,
				Suite:        parsed.Name.Name,
				File:         relativeTo(repoRoot, path),
				Line:         fileSet.Position(fn.Pos()).Line,
				Requirements: requirementsFromDoc(fn.Doc),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tests, nil
}

// isTestFunc reports whether fn is a Go test function: named TestXxx and taking
// *testing.T. Benchmarks, fuzz targets, and helpers are not tests to trace.
func isTestFunc(fn *ast.FuncDecl) bool {
	if !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}

	star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && selector.Sel.Name == "T"
}

// requirementsFromDoc pulls the IDs out of a "Requirements:" line in a doc
// comment. The line may wrap onto following comment lines that are indented or
// begin with a comma.
func requirementsFromDoc(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	var raw []string
	collecting := false

	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")

		if match := goAnnotation.FindStringSubmatch(text); match != nil {
			raw = append(raw, splitCommaList(match[1])...)
			collecting = strings.HasSuffix(strings.TrimSpace(match[1]), ",")
			continue
		}

		if !collecting {
			continue
		}

		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			collecting = false
			continue
		}
		raw = append(raw, splitCommaList(trimmed)...)
		collecting = strings.HasSuffix(trimmed, ",")
	}

	return normalizeIDs(raw)
}

// ParseRobotTests reads Robot suites and returns every test case with the
// requirement IDs from its [Tags] setting.
func ParseRobotTests(paths []string, repoRoot string) ([]TestCase, error) {
	var tests []TestCase

	for _, path := range paths {
		found, err := parseRobotFile(path, repoRoot)
		if err != nil {
			return nil, err
		}
		tests = append(tests, found...)
	}

	return tests, nil
}

func parseRobotFile(path, repoRoot string) ([]TestCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		tests    []TestCase
		current  *TestCase
		inCases  bool
		scanner  = bufio.NewScanner(file)
		lineNo   int
		suite    = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		flushOne = func(t *TestCase) {
			if t != nil {
				tests = append(tests, *t)
			}
		}
	)

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "***") {
			flushOne(current)
			current = nil
			inCases = strings.EqualFold(strings.Trim(trimmed, "* "), "Test Cases")
			continue
		}
		if !inCases || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// A test case name starts in column one; its settings and steps are
		// indented.
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			flushOne(current)
			current = &TestCase{
				Kind:  KindRobot,
				Name:  trimmed,
				Suite: suite,
				File:  relativeTo(repoRoot, path),
				Line:  lineNo,
			}
			continue
		}

		if current == nil {
			continue
		}

		fields := splitRobotCells(trimmed)
		if len(fields) == 0 || !strings.EqualFold(fields[0], "[Tags]") {
			continue
		}
		for _, tag := range fields[1:] {
			if robotRequirementTag.MatchString(strings.ToUpper(tag)) {
				current.Requirements = normalizeIDs(append(current.Requirements, tag))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flushOne(current)

	return tests, nil
}

// splitRobotCells splits a Robot line into cells. Robot separates cells with two
// or more spaces or a tab, so a single space inside a name or tag is preserved.
func splitRobotCells(line string) []string {
	fields := regexp.MustCompile(`\s{2,}|\t`).Split(line, -1)

	var out []string
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
	}
	return out
}

func splitCommaList(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func shouldSkip(path string, skipDirs []string) bool {
	base := filepath.Base(path)
	for _, skip := range skipDirs {
		if base == skip {
			return true
		}
	}
	return false
}

// relativeTo makes a path repo-relative for display, falling back to the
// absolute path when it cannot.
func relativeTo(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
