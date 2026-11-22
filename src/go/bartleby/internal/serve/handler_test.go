package serve

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

func fixture(t *testing.T) (ws string, s *Server) {
	t.Helper()
	ws = t.TempDir()
	now := time.Now()
	write(t, filepath.Join(ws, "alpha", "target", "bartleby", "html", "index.html"), now)
	write(t, filepath.Join(ws, "alpha", "target", "bartleby", "html", "proposals", "NERD001_X.html"), now)
	write(t, filepath.Join(ws, "alpha", "target", "bartleby", "alpha-1.0.pdf"), now)
	write(t, filepath.Join(ws, "alpha", "docs", "a.rst"), now.Add(-time.Hour))
	if err := os.MkdirAll(filepath.Join(ws, "beta", "docs"), 0o755); err != nil { // listed, never built
		t.Fatal(err)
	}
	write(t, filepath.Join(ws, "secret.txt"), now)
	write(t, filepath.Join(ws, "unlisted", "target", "bartleby", "html", "index.html"), now)

	s = &Server{
		Workspace: ws,
		Repos:     []string{"alpha", "beta", "ghost"},
		List: func(string, []string) (reqtrace.Listing, error) {
			return reqtrace.Listing{Needs: []reqtrace.ListedNeed{
				{Repo: "alpha", ID: "ALPHA_NERD001", Type: "req", Title: "First <proposal>", Status: "proposed", File: "docs/proposals/NERD001_X.rst"},
				{Repo: "alpha", ID: "ALPHA_NERD001_SPEC001", Type: "spec", Title: "A spec", Status: "proposed", File: "docs/proposals/NERD001_X.rst"},
				{Repo: "alpha", ID: "ALPHA_REQ_1", Type: "req", Title: "Not a proposal", Status: "implemented", File: "docs/requirements/cli.rst"},
			}}, nil
		},
	}
	return ws, s
}

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Requirements: REQ_SERVE_002
func TestServesBuiltHTML(t *testing.T) {
	_, s := fixture(t)
	h := s.Handler()
	if rec := get(t, h, "GET", "/r/alpha/proposals/NERD001_X.html"); rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	if rec := get(t, h, "HEAD", "/r/alpha/index.html"); rec.Code != 200 {
		t.Fatalf("HEAD code %d", rec.Code)
	}
}

// Requirements: REQ_SERVE_002
func TestRefusesUnlistedMissingAndEscapes(t *testing.T) {
	ws, s := fixture(t)
	h := s.Handler()
	for _, p := range []string{
		"/r/unlisted/index.html",
		"/r/beta/index.html",
		"/r/alpha/../../secret.txt",
		"/r/alpha/%2e%2e/%2e%2e/secret.txt",
		"/r/..%2fsecret.txt",
		"/r/alpha/nope.html",
	} {
		if rec := get(t, h, "GET", p); rec.Code != 404 {
			t.Errorf("%s: code %d, want 404", p, rec.Code)
		}
	}
	// A symlink inside the HTML tree that points outside it must not be followed.
	link := filepath.Join(ws, "alpha", "target", "bartleby", "html", "leak.txt")
	if err := os.Symlink(filepath.Join(ws, "secret.txt"), link); err != nil {
		t.Fatal(err)
	}
	if rec := get(t, h, "GET", "/r/alpha/leak.txt"); rec.Code != 404 {
		t.Errorf("symlink escape: code %d, want 404", rec.Code)
	}
	if rec := get(t, h, "PUT", "/r/alpha/index.html"); rec.Code != 405 {
		t.Errorf("PUT: code %d, want 405", rec.Code)
	}
}

// Requirements: REQ_SERVE_003
func TestLandingPage(t *testing.T) {
	_, s := fixture(t)
	rec := get(t, s.Handler(), "GET", "/")
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	page := string(body)
	for _, want := range []string{
		`href="/r/alpha/index.html"`,
		`href="/r/alpha/alpha-1.0.pdf"`,
		`href="/r/alpha/proposals/NERD001_X.html#ALPHA_NERD001"`,
		"First &lt;proposal&gt;", // escaped
		"proposed",               // status as text
		"not built",              // beta
		"missing",                // ghost has no directory
	} {
		if !strings.Contains(page, want) {
			t.Errorf("landing page lacks %q", want)
		}
	}
	for _, unwanted := range []string{"A spec", "Not a proposal", "First <proposal>", "unlisted"} {
		if strings.Contains(page, unwanted) {
			t.Errorf("landing page should not contain %q", unwanted)
		}
	}
}

// Requirements: REQ_SERVE_003
func TestLandingPageSurvivesListerError(t *testing.T) {
	_, s := fixture(t)
	s.List = func(string, []string) (reqtrace.Listing, error) { return reqtrace.Listing{}, errors.New("boom") }
	rec := get(t, s.Handler(), "GET", "/")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("code %d; page should still render and report the error", rec.Code)
	}
}

func post(t *testing.T, h http.Handler, path string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, nil)
	req.Host = "127.0.0.1:8099"
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Requirements: REQ_SERVE_005
func TestRebuildRunsInRepoDirectory(t *testing.T) {
	ws, s := fixture(t)
	var gotDir string
	s.Rebuild = func(ctx context.Context, dir string, out io.Writer) error {
		gotDir = dir
		io.WriteString(out, "built ok")
		return nil
	}
	rec := post(t, s.Handler(), "/rebuild/alpha", map[string]string{"Origin": "http://127.0.0.1:8099"})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "built ok") {
		t.Fatalf("code %d body %q", rec.Code, rec.Body.String())
	}
	if gotDir != filepath.Join(ws, "alpha") {
		t.Fatalf("ran in %q", gotDir)
	}
}

// Requirements: REQ_SERVE_005
func TestRebuildRefusals(t *testing.T) {
	_, s := fixture(t)
	s.Rebuild = func(context.Context, string, io.Writer) error { return nil }
	h := s.Handler()

	if rec := post(t, h, "/rebuild/unlisted", nil); rec.Code != 404 {
		t.Errorf("unlisted: %d, want 404", rec.Code)
	}
	if rec := post(t, h, "/rebuild/ghost", nil); rec.Code != 404 {
		t.Errorf("missing dir: %d, want 404", rec.Code)
	}
	if rec := get(t, h, "GET", "/rebuild/alpha"); rec.Code != 405 {
		t.Errorf("GET: %d, want 405", rec.Code)
	}
	if rec := post(t, h, "/rebuild/alpha", map[string]string{"Origin": "http://evil.example"}); rec.Code != 403 {
		t.Errorf("foreign Origin: %d, want 403", rec.Code)
	}
	if rec := post(t, h, "/rebuild/alpha", map[string]string{"Host": "evil.example"}); rec.Code != 403 {
		t.Errorf("foreign Host (DNS rebinding): %d, want 403", rec.Code)
	}
}

// Requirements: REQ_SERVE_005
func TestRebuildOneAtATime(t *testing.T) {
	_, s := fixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	s.Rebuild = func(context.Context, string, io.Writer) error {
		close(started)
		<-release
		return nil
	}
	h := s.Handler()
	done := make(chan int)
	go func() { done <- post(t, h, "/rebuild/alpha", nil).Code }()
	<-started
	if rec := post(t, h, "/rebuild/alpha", nil); rec.Code != 409 {
		t.Errorf("second build: %d, want 409", rec.Code)
	}
	close(release)
	if code := <-done; code != 200 {
		t.Errorf("first build: %d, want 200", code)
	}
}

// Requirements: REQ_SERVE_005
func TestRebuildFailureIsReported(t *testing.T) {
	_, s := fixture(t)
	s.Rebuild = func(_ context.Context, _ string, out io.Writer) error {
		io.WriteString(out, "sphinx exploded")
		return errors.New("exit status 2")
	}
	rec := post(t, s.Handler(), "/rebuild/alpha", nil)
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), "sphinx exploded") {
		t.Fatalf("code %d body %q", rec.Code, rec.Body.String())
	}
}
