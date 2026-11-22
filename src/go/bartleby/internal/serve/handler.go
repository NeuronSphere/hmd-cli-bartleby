package serve

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

// Server serves the review hub. Its fields are set before Handler is called.
type Server struct {
	// Workspace is the directory that holds the repositories.
	Workspace string
	// Repos are the directory names to show, in order.
	Repos []string
	// List returns the needs for the repositories. It is injected so tests
	// need no workspace index.
	List func(workspace string, repos []string) (reqtrace.Listing, error)
	// Rebuild builds the HTML for the repository at dir, writing its log to out.
	Rebuild func(ctx context.Context, dir string, out io.Writer) error

	building sync.Mutex
}

// Handler returns the HTTP handler. It dispatches by hand rather than through
// http.ServeMux, which answers a path containing ".." with a redirect instead of
// refusing it.
func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.route) }

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/":
		s.landing(w, r)
	case strings.HasPrefix(p, "/r/"):
		s.serveFile(w, r, strings.TrimPrefix(p, "/r/"))
	case strings.HasPrefix(p, "/rebuild/"):
		s.rebuild(w, r, strings.TrimPrefix(p, "/rebuild/"))
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listed(name string) bool {
	for _, n := range s.Repos {
		if n == name {
			return true
		}
	}
	return false
}

func (s *Server) repoDir(name string) (string, bool) {
	if !s.listed(name) {
		return "", false
	}
	dir := filepath.Join(s.Workspace, name)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return "", false
	}
	return dir, true
}

func readOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

// serveFile serves rest, "<repo>/<path>", from that repository's built HTML.
// Every failure to open the file is a 404, whether the file is missing or the
// path tries to leave the HTML tree: os.Root refuses ".." and symbolic links
// that point outside it.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, rest string) {
	if !readOnly(w, r) {
		return
	}
	name, rel, _ := strings.Cut(rest, "/")
	dir, ok := s.repoDir(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			http.NotFound(w, r)
			return
		}
	}
	// The HTML tree is the whole of what is served, with one exception: a PDF is
	// built beside it, in target/bartleby, so a top-level *.pdf name is read from
	// there. Nothing else in that directory is reachable.
	base := htmlDir(dir)
	if !strings.Contains(rel, "/") && strings.HasSuffix(rel, ".pdf") {
		base = filepath.Dir(base)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer root.Close()

	if rel == "" || strings.HasSuffix(rel, "/") {
		rel += "index.html"
	}
	f, err := root.Open(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if fi.IsDir() {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

// ---- landing page ----------------------------------------------------------

type needView struct {
	ID, Title, Status, Href string
}

type repoView struct {
	Name    string
	Missing bool
	State   State
	Needs   []needView
}

type landingView struct {
	Repos     []repoView
	ListError string
	Empty     bool
}

func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	if !readOnly(w, r) {
		return
	}
	v := landingView{Empty: len(s.Repos) == 0}
	var listing reqtrace.Listing
	if len(s.Repos) > 0 && s.List != nil {
		var err error
		if listing, err = s.List(s.Workspace, s.Repos); err != nil {
			v.ListError = err.Error()
		}
	}
	for _, name := range s.Repos {
		rv := repoView{Name: name}
		dir, ok := s.repoDir(name)
		if !ok {
			rv.Missing = true
			v.Repos = append(v.Repos, rv)
			continue
		}
		rv.State = BuildState(dir)
		for _, n := range listing.Needs {
			if n.Repo != name || n.Type != "req" || !strings.HasPrefix(n.File, "docs/proposals/") {
				continue
			}
			nv := needView{ID: n.ID, Title: n.Title, Status: n.Status}
			page := strings.TrimSuffix(strings.TrimPrefix(n.File, "docs/"), ".rst") + ".html"
			if _, err := os.Stat(filepath.Join(htmlDir(dir), filepath.FromSlash(page))); err == nil && rv.State.Built {
				nv.Href = "/r/" + name + "/" + path.Clean(page) + "#" + url.PathEscape(n.ID)
			}
			rv.Needs = append(rv.Needs, nv)
		}
		sort.SliceStable(rv.Needs, func(i, j int) bool { return rv.Needs[i].ID < rv.Needs[j].ID })
		v.Repos = append(v.Repos, rv)
	}

	var buf bytes.Buffer
	if err := landingTmpl.Execute(&buf, v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// ---- rebuild ---------------------------------------------------------------

func (s *Server) rebuild(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// A rebuild runs a process, so a page open in the same browser must not be
	// able to start one: refuse a Host that is not loopback (DNS rebinding) and
	// an Origin that is not this server.
	host := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		host = h
	}
	if !loopbackHost(strings.Trim(host, "[]")) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		if u, err := url.Parse(origin); err != nil || u.Host != r.Host {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	dir, ok := s.repoDir(name)
	if !ok || s.Rebuild == nil {
		http.NotFound(w, r)
		return
	}
	if !s.building.TryLock() {
		http.Error(w, "a build is already running", http.StatusConflict)
		return
	}
	defer s.building.Unlock()

	var out bytes.Buffer
	err := s.Rebuild(r.Context(), dir, &out)
	code := http.StatusOK
	if err != nil {
		code = http.StatusInternalServerError
	}
	var buf bytes.Buffer
	_ = rebuildTmpl.Execute(&buf, map[string]any{"Name": name, "Log": out.String(), "Err": errString(err)})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	w.Write(buf.Bytes())
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ExecRebuild returns a Rebuild function that runs "<exe> html" in the
// repository's directory. A child process keeps the build's working directory
// and options out of the server's own, and is what `bartleby html` does anyway.
func ExecRebuild(exe string) func(context.Context, string, io.Writer) error {
	return func(ctx context.Context, dir string, out io.Writer) error {
		cmd := exec.CommandContext(ctx, exe, "html")
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = out, out
		return cmd.Run()
	}
}

// ---- templates -------------------------------------------------------------

var landingTmpl = template.Must(template.New("landing").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Review hub</title>
<style>
:root{--bg:#fff;--fg:#1b1f24;--mut:#59636e;--line:#d1d9e0;--card:#f6f8fa;--warn:#9a6700;--bad:#cf222e}
@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3;--mut:#9198a1;--line:#30363d;--card:#151b23;--warn:#d29922;--bad:#f85149}}
body{background:var(--bg);color:var(--fg);font:16px/1.5 system-ui,sans-serif;margin:0 auto;max-width:60rem;padding:1rem}
a{color:inherit}.card{background:var(--card);border:1px solid var(--line);border-radius:8px;margin:1rem 0;padding:.75rem 1rem}
.card h2{font-size:1.1rem;margin:0 0 .25rem}.mut{color:var(--mut)}.warn{color:var(--warn)}.bad{color:var(--bad)}
table{border-collapse:collapse;width:100%}td{border-top:1px solid var(--line);padding:.25rem .5rem .25rem 0;vertical-align:top}
td.s{white-space:nowrap;color:var(--mut)}input[type=search]{width:100%;padding:.5rem;box-sizing:border-box;margin:.5rem 0}
button{font:inherit;cursor:pointer}form{display:inline;margin-left:.75rem}
</style></head><body>
<h1>Review hub</h1>
<input type="search" id="q" placeholder="Filter needs…" aria-label="Filter needs">
{{if .ListError}}<p class="bad">Needs could not be listed: {{.ListError}}</p>{{end}}
{{if .Empty}}<p>No repositories are listed. Add one directory name per line to the repos file, or run <code>bartleby serve suggest</code>.</p>{{end}}
{{range .Repos}}<section class="card" data-repo="{{.Name}}">
<h2>{{.Name}}</h2>
{{if .Missing}}<p class="bad">missing: no directory with this name in the workspace</p>
{{else if not .State.Built}}<p class="warn">not built — use Rebuild to render it
<form method="post" action="/rebuild/{{.Name}}"><button type="submit">Rebuild</button></form></p>
{{else}}<p>
<a href="/r/{{.Name}}/index.html">Open HTML</a>
{{if .State.PDF}} · <a href="/r/{{.Name}}/{{.State.PDF}}">PDF</a>{{end}}
<span class="mut"> · built {{.State.BuiltAt.Format "2006-01-02 15:04"}}</span>
{{if .State.Stale}}<span class="warn"> · stale: sources are newer than the HTML</span>{{else}}<span class="mut"> · up to date</span>{{end}}
<form method="post" action="/rebuild/{{.Name}}"><button type="submit">Rebuild</button></form></p>{{end}}
{{if .Needs}}<table>{{range .Needs}}<tr class="need"><td class="s">{{.Status}}</td>
<td>{{if .Href}}<a href="{{.Href}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}<br><span class="mut">{{.ID}}</span></td></tr>{{end}}</table>{{end}}
</section>{{end}}
<script>
document.getElementById('q').addEventListener('input',function(e){var q=e.target.value.toLowerCase();
document.querySelectorAll('tr.need').forEach(function(r){r.hidden=q&&r.textContent.toLowerCase().indexOf(q)<0});});
</script></body></html>
`))

var rebuildTmpl = template.Must(template.New("rebuild").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Rebuild {{.Name}}</title>
<style>:root{--bg:#fff;--fg:#1b1f24}@media (prefers-color-scheme:dark){:root{--bg:#0d1117;--fg:#e6edf3}}
body{background:var(--bg);color:var(--fg);font:16px/1.5 system-ui,sans-serif;margin:0 auto;max-width:60rem;padding:1rem}pre{overflow:auto;white-space:pre-wrap}</style>
</head><body><h1>Rebuild {{.Name}}</h1>
{{if .Err}}<p><strong>Failed:</strong> {{.Err}}</p>{{else}}<p><strong>Built.</strong></p>{{end}}
<pre>{{.Log}}</pre><p><a href="/">Back to the review hub</a></p></body></html>
`))
