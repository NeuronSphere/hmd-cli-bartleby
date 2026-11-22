package serve

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func commit(t *testing.T, dir, email string, when time.Time) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL="+email,
			"GIT_AUTHOR_DATE="+when.Format(time.RFC3339), "GIT_COMMITTER_DATE="+when.Format(time.RFC3339))
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("commit", "-q", "--allow-empty", "-m", "x")
}

// Requirements: REQ_SERVE_007
func TestSuggest(t *testing.T) {
	ws := t.TempDir()
	now := time.Now()
	commit(t, filepath.Join(ws, "mine-recent"), "me@example.com", now.Add(-24*time.Hour))
	commit(t, filepath.Join(ws, "mine-listed"), "me@example.com", now.Add(-24*time.Hour))
	commit(t, filepath.Join(ws, "mine-old"), "me@example.com", now.Add(-90*24*time.Hour))
	commit(t, filepath.Join(ws, "theirs"), "other@example.com", now.Add(-24*time.Hour))
	if err := os.MkdirAll(filepath.Join(ws, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Suggest(ws, []string{"mine-listed"}, "me@example.com", 30)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"mine-recent"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Requirements: REQ_SERVE_007
func TestSuggestNeedsAnEmail(t *testing.T) {
	if _, err := Suggest(t.TempDir(), nil, "", 30); err == nil {
		t.Fatal("expected an error when no email is configured")
	}
}
