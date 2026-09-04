// Package sources implements the bartleby.sources feature: stitching external
// documentation trees into a repo's docs before a build, then putting the docs
// directory back exactly as it was.
//
// A source either points at an artifact checked out somewhere under the repo
// (artifact_path, staged into docs/_sources/<key>) or at a directory that is
// already present at docs/<key>. Either way a toctree entry is injected into the
// root document so Sphinx picks it up.
package sources

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

// Marker is the explicit placement directive. When a root document contains it,
// the injected toctree replaces that line instead of being guessed into place.
const Marker = ".. bartleby-sources::"

// StagingDir is the directory under docs/ that staged artifact sources are
// copied into. It is created before a build and removed afterwards.
const StagingDir = "_sources"

// indexMarkers are the trailing sections a Sphinx index conventionally ends
// with; a toctree has to be inserted before them, not after.
var indexMarkers = []string{"Indexes and tables", "Indices and tables"}

// Validate drops sources whose documentation directory does not exist, warning
// about each one. Building with a missing source is a warning rather than an
// error because a source is often an optional artifact that CI has not fetched.
func Validate(repoPath, docsPath string, all map[string]manifest.Source, warnTo io.Writer) map[string]manifest.Source {
	valid := make(map[string]manifest.Source)

	for _, key := range keys(all) {
		source := all[key]
		path, kind := sourcePath(repoPath, docsPath, key, source)

		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			if warnTo != nil {
				fmt.Fprintf(warnTo, "warning: source %q skipped: %s %q not found\n", key, kind, path)
			}
			continue
		}
		valid[key] = source
	}

	return valid
}

// sourcePath returns where a source's docs live on disk, plus a label for
// messages.
func sourcePath(repoPath, docsPath, key string, source manifest.Source) (path, kind string) {
	if source.ArtifactPath != "" {
		return filepath.Join(repoPath, source.ArtifactPath, source.Docs()), "artifact docs path"
	}
	return filepath.Join(docsPath, key), "docs path"
}

// Stage copies every artifact-backed source into docs/_sources/<key>. Sources
// that are already in docs/ need no staging. It returns the staged directories.
func Stage(repoPath, docsPath string, valid map[string]manifest.Source) ([]string, error) {
	var staged []string

	for _, key := range keys(valid) {
		source := valid[key]
		if source.ArtifactPath == "" {
			continue
		}

		src := filepath.Join(repoPath, source.ArtifactPath, source.Docs())
		dest := filepath.Join(docsPath, StagingDir, key)

		if err := os.RemoveAll(dest); err != nil {
			return staged, fmt.Errorf("clearing staging directory %s: %w", dest, err)
		}
		if err := copyDir(src, dest); err != nil {
			return staged, fmt.Errorf("staging source %q from %s: %w", key, src, err)
		}
		staged = append(staged, dest)
	}

	return staged, nil
}

// Cleanup removes the staging directory. It is safe to call when nothing was
// staged.
func Cleanup(docsPath string) error {
	return os.RemoveAll(filepath.Join(docsPath, StagingDir))
}

// Toctree renders the toctree blocks for the given sources, in key order.
func Toctree(valid map[string]manifest.Source) string {
	if len(valid) == 0 {
		return ""
	}

	var b strings.Builder
	for i, key := range keys(valid) {
		source := valid[key]

		title := source.Title
		if title == "" {
			title = key
		}

		path := key + "/index"
		if source.ArtifactPath != "" {
			path = StagingDir + "/" + key + "/index"
		}

		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, ".. toctree::\n   :maxdepth: 2\n   :caption: %s\n\n   %s\n", title, path)
	}
	b.WriteString("\n")

	return b.String()
}

// Inject writes source toctrees into a root document and returns its original
// content so the caller can restore it. Three placements are tried in order:
// the explicit Marker line, immediately before an "Indexes and tables" section,
// and finally appended to the end.
func Inject(indexPath string, valid map[string]manifest.Source) (original string, err error) {
	if len(valid) == 0 {
		return "", nil
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", indexPath, err)
	}
	original = string(data)

	toctree := Toctree(valid)
	lines := splitKeepEnds(original)

	for i, line := range lines {
		if strings.Contains(line, Marker) {
			lines[i] = toctree
			return original, writeLines(indexPath, lines)
		}
	}

	for i, line := range lines {
		if isIndexMarker(line) {
			lines = append(lines[:i], append([]string{toctree + "\n"}, lines[i:]...)...)
			return original, writeLines(indexPath, lines)
		}
	}

	appended := original
	if !strings.HasSuffix(appended, "\n") {
		appended += "\n"
	}
	appended += "\n" + toctree

	if err := writeFilePreservingMode(indexPath, []byte(appended)); err != nil {
		return "", err
	}
	return original, nil
}

// Restore puts a root document back to the content Inject returned.
func Restore(indexPath, original string) error {
	if original == "" {
		return nil
	}
	if err := os.WriteFile(indexPath, []byte(original), 0o644); err != nil {
		return fmt.Errorf("restoring %s: %w", indexPath, err)
	}
	return nil
}

// isIndexMarker reports whether a line is one of the conventional trailing
// section headings. The comparison ignores surrounding whitespace; the Python
// implementation required an exact "Indexes and tables\n" match and silently
// fell through to appending when a file used different spacing.
func isIndexMarker(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, marker := range indexMarkers {
		if strings.EqualFold(trimmed, marker) {
			return true
		}
	}
	return false
}

func writeLines(path string, lines []string) error {
	return writeFilePreservingMode(path, []byte(strings.Join(lines, "")))
}

// writeFilePreservingMode writes data to path, keeping the file's current
// permissions when it can determine them.
func writeFilePreservingMode(path string, data []byte) error {
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// splitKeepEnds splits s into lines, keeping the trailing newline on each.
func splitKeepEnds(s string) []string {
	var lines []string
	for len(s) > 0 {
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			lines = append(lines, s)
			break
		}
		lines = append(lines, s[:idx+1])
		s = s[idx+1:]
	}
	return lines
}

// copyDir recursively copies src to dest. Symlinks are skipped rather than
// followed or recreated: the transform container copies the docs tree again on
// its side, and a dangling link there fails the build.
func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		switch {
		case d.IsDir():
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		case d.Type()&fs.ModeSymlink != 0:
			return nil
		case !d.Type().IsRegular():
			return nil
		default:
			return copyFile(path, target, d)
		}
	})
}

func copyFile(src, dest string, d fs.DirEntry) error {
	info, err := d.Info()
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		return errors.Join(err, out.Close())
	}
	return out.Close()
}

// keys returns the sorted keys of a source map so staging, toctree order, and
// warnings are deterministic.
func keys(m map[string]manifest.Source) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
