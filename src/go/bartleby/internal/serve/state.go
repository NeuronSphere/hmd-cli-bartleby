package serve

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// State describes one repository's built output relative to its sources.
type State struct {
	Built   bool
	BuiltAt time.Time
	Stale   bool
	// PDF is the file name, inside target/bartleby, of the newest PDF, or "".
	PDF string
}

func htmlDir(repoDir string) string { return filepath.Join(repoDir, "target", "bartleby", "html") }

// BuildState compares the newest .rst under docs with index.html. A repository
// with no built HTML is "not built", not "stale": there is nothing to be out of
// date with.
func BuildState(repoDir string) State {
	var st State
	idx, err := os.Stat(filepath.Join(htmlDir(repoDir), "index.html"))
	if err != nil {
		return st
	}
	st.Built, st.BuiltAt = true, idx.ModTime()

	var newest time.Time
	_ = filepath.WalkDir(filepath.Join(repoDir, "docs"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".rst") {
			return nil
		}
		if info, ierr := d.Info(); ierr == nil && info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	st.Stale = newest.After(st.BuiltAt)

	out := filepath.Join(repoDir, "target", "bartleby")
	entries, _ := os.ReadDir(out)
	var pdfAt time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pdf") {
			continue
		}
		if info, ierr := e.Info(); ierr == nil && (st.PDF == "" || info.ModTime().After(pdfAt)) {
			st.PDF, pdfAt = e.Name(), info.ModTime()
		}
	}
	return st
}
