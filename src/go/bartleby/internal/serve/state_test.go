package serve

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// Requirements: REQ_SERVE_004
func TestBuildStateNotBuilt(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "docs", "a.rst"), time.Now())
	st := BuildState(repo)
	if st.Built || st.Stale {
		t.Fatalf("got %+v; a repo with no HTML is not built and not stale", st)
	}
}

// Requirements: REQ_SERVE_004
func TestBuildStateFreshAndStale(t *testing.T) {
	repo := t.TempDir()
	base := time.Now().Add(-time.Hour)
	write(t, filepath.Join(repo, "target", "bartleby", "html", "index.html"), base)
	write(t, filepath.Join(repo, "docs", "a.rst"), base.Add(-time.Minute))
	if st := BuildState(repo); !st.Built || st.Stale {
		t.Fatalf("source older than HTML should be fresh, got %+v", st)
	}
	write(t, filepath.Join(repo, "docs", "proposals", "N.rst"), base.Add(time.Minute))
	st := BuildState(repo)
	if !st.Built || !st.Stale {
		t.Fatalf("source newer than HTML should be stale, got %+v", st)
	}
	if !st.BuiltAt.Equal(base.Truncate(time.Second)) && st.BuiltAt.Sub(base) > time.Second {
		t.Fatalf("BuiltAt = %v, want about %v", st.BuiltAt, base)
	}
}

// Requirements: REQ_SERVE_003
func TestBuildStateNewestPDF(t *testing.T) {
	repo := t.TempDir()
	out := filepath.Join(repo, "target", "bartleby")
	now := time.Now()
	write(t, filepath.Join(out, "html", "index.html"), now)
	write(t, filepath.Join(out, "old-1.0.pdf"), now.Add(-time.Hour))
	write(t, filepath.Join(out, "new-1.0.pdf"), now)
	if got := BuildState(repo).PDF; got != "new-1.0.pdf" {
		t.Fatalf("PDF = %q, want new-1.0.pdf", got)
	}
}
