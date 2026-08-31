// Package sanitize cleans user- and manifest-supplied strings that end up in
// places with stricter rules than Go strings: LaTeX document titles, output
// filenames, and Docker container names.
package sanitize

import (
	"regexp"
	"strings"
)

// latexHostile matches characters that either have special meaning in LaTeX text
// mode or break Makefile targets and filenames. Underscore is a subscript
// operator, ampersand an alignment tab, percent a comment, and so on; spaces
// break the Makefile target the container builds.
var latexHostile = regexp.MustCompile(`[\s_&%#$^\\~{}"'` + "`" + `:;,/|<>*?!()\[\]=+]+`)

var repeatedHyphen = regexp.MustCompile(`-{2,}`)

// Title makes s safe to use as a LaTeX document title and output filename. It
// collapses every run of hostile characters into a single hyphen and trims
// hyphens from the ends. An input that sanitizes down to nothing returns "".
func Title(s string) string {
	s = latexHostile.ReplaceAllString(s, "-")
	s = repeatedHyphen.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// dockerNameIllegal matches anything outside Docker's allowed container-name
// character set, which is [a-zA-Z0-9][a-zA-Z0-9_.-]*.
var dockerNameIllegal = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// leadingIllegal matches the characters Docker rejects in first position.
var leadingIllegal = regexp.MustCompile(`^[^a-zA-Z0-9]+`)

// ContainerName makes s usable as a Docker container name. Docker requires the
// first character to be alphanumeric and the rest to be alphanumeric, underscore,
// period, or hyphen; a repo directory called "My Docs (v2)" would otherwise fail
// container creation with an opaque API error.
//
// An input with no usable characters returns "" and the caller should fall back
// to something known-good.
func ContainerName(s string) string {
	s = dockerNameIllegal.ReplaceAllString(s, "-")
	s = repeatedHyphen.ReplaceAllString(s, "-")
	s = leadingIllegal.ReplaceAllString(s, "")
	return strings.Trim(s, "-")
}
