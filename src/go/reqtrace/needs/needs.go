// Package needs reads sphinx-needs directives out of reStructuredText: every
// need of the configured types, with its options, its body and a digest of its
// text.
//
// It is the parser behind reqtrace's listing. The traceability page keeps its
// own narrower req/spec parser in the parent package, so that adding types
// here cannot change what "reqtrace -check" judges.
//
// Like that parser it is line-oriented rather than a full docutils parse. A
// need is a directive line, indented option lines and an indented body, which
// is unambiguous enough in practice. Directives inside literal blocks and
// code blocks are skipped, because documentation about needs quotes them.
package needs

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"regexp"
	"strings"
)

// Need is one sphinx-needs item.
type Need struct {
	Type   string   `json:"type"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Status string   `json:"status,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Links  []string `json:"links,omitempty"`
	// Line is the 1-based line of the directive.
	Line int `json:"line"`
	// Body is the indented content after the options, dedented. It is not
	// kept in reqtrace's index.
	Body string `json:"body,omitempty"`
	// Digest is the hex SHA-256 of the directive's text: its lines with
	// line endings normalised, trailing white space removed and trailing
	// blank lines dropped. It changes when the need's text changes and for
	// no other reason.
	Digest string `json:"digest"`
}

// DefaultTypes is the vocabulary used when a repository declares none: the
// sphinx-needs defaults plus the feature, question and decision types in use
// across NeuronSphere repositories.
func DefaultTypes() map[string]bool {
	return map[string]bool{
		"req": true, "spec": true, "impl": true, "test": true,
		"feature": true, "question": true, "decision": true,
	}
}

var (
	directive = regexp.MustCompile(`^(\s*)\.\.\s+([A-Za-z][\w-]*)::(.*)$`)
	option    = regexp.MustCompile(`^\s*:([A-Za-z_][\w-]*):\s*(.*)$`)
)

// codeDirectives hold literal content whose lines must not be read as needs.
var codeDirectives = map[string]bool{
	"code-block": true, "code": true, "sourcecode": true, "literalinclude": true,
}

// ParseFile parses the file at path.
func ParseFile(path string, types map[string]bool) ([]Need, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, types)
}

// Parse reads every need of the given types from r. A need without an :id:
// is dropped, as sphinx-needs would give it a generated one nobody can cite.
func Parse(r io.Reader, types map[string]bool) ([]Need, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), " \t\r"))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var found []Need
	// skipIndent >= 0 while inside a literal or code block: every non-blank
	// line indented deeper than it belongs to the block.
	skipIndent := -1
	// pendingLiteral is the indent of a paragraph line ending in "::", whose
	// following indented block is literal.
	pendingLiteral := -1

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := indentOf(line)

		if skipIndent >= 0 {
			if indent > skipIndent {
				continue
			}
			skipIndent = -1
		}
		if pendingLiteral >= 0 {
			if indent > pendingLiteral {
				skipIndent = pendingLiteral
				pendingLiteral = -1
				continue
			}
			pendingLiteral = -1
		}

		match := directive.FindStringSubmatch(line)
		if match == nil {
			if strings.HasSuffix(line, "::") {
				pendingLiteral = indent
			}
			continue
		}

		name := strings.ToLower(match[2])
		if codeDirectives[name] {
			skipIndent = indent
			continue
		}
		if !types[name] {
			continue
		}

		need, end := parseNeed(lines, i, indent, name, strings.TrimSpace(match[3]))
		if need.ID != "" {
			found = append(found, need)
		}
		i = end
	}
	return found, nil
}

// parseNeed reads the need whose directive is lines[start] and returns it with
// the index of its last line.
func parseNeed(lines []string, start, indent int, typ, title string) (Need, int) {
	need := Need{Type: typ, Title: title, Line: start + 1}

	// The directive extends over every following line that is blank or
	// indented deeper than the directive itself.
	end := start
	for j := start + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) != "" && indentOf(lines[j]) <= indent {
			break
		}
		end = j
	}
	for end > start && strings.TrimSpace(lines[end]) == "" {
		end--
	}

	// Options come first, each possibly wrapped onto deeper-indented lines;
	// the first blank or non-option line starts the body.
	var (
		lastOption  string
		optionDepth int
		bodyStart   = end + 1
	)
	for j := start + 1; j <= end; j++ {
		line := lines[j]
		if strings.TrimSpace(line) == "" {
			bodyStart = j + 1
			break
		}
		if m := option.FindStringSubmatch(line); m != nil {
			lastOption, optionDepth = m[1], indentOf(line)
			setOption(&need, lastOption, strings.TrimSpace(m[2]), false)
			continue
		}
		if lastOption != "" && indentOf(line) > optionDepth {
			setOption(&need, lastOption, strings.TrimSpace(line), true)
			continue
		}
		bodyStart = j
		break
	}

	if bodyStart <= end {
		need.Body = dedent(lines[bodyStart : end+1])
	}

	h := sha256.Sum256([]byte(strings.Join(lines[start:end+1], "\n")))
	need.Digest = hex.EncodeToString(h[:])
	return need, end
}

func setOption(n *Need, name, value string, continued bool) {
	switch name {
	case "id":
		if !continued {
			n.ID = value
		}
	case "status":
		if !continued {
			n.Status = value
		}
	case "tags":
		n.Tags = append(n.Tags, splitList(value)...)
	case "links":
		n.Links = append(n.Links, splitList(value)...)
	}
}

func splitList(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func indentOf(line string) int {
	n := 0
	for _, r := range line {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// trimColumns removes up to n columns of leading white space, counting a tab
// as four, the way indentOf measures it.
func trimColumns(line string, n int) string {
	col := 0
	for i, r := range line {
		if col >= n || (r != ' ' && r != '\t') {
			return line[i:]
		}
		if r == '\t' {
			col += 4
		} else {
			col++
		}
	}
	return ""
}

func dedent(lines []string) string {
	least := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if d := indentOf(l); least < 0 || d < least {
			least = d
		}
	}
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(trimColumns(l, least))
	}
	return b.String()
}
