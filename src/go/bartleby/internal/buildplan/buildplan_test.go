package buildplan

import (
	"strings"
	"testing"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

func roots() map[string]manifest.Root {
	return map[string]manifest.Root{
		"guide": {
			RootDoc:  "guide",
			Builders: []manifest.Builder{{Shell: "html"}, {Shell: "pdf"}},
		},
		"api": {
			Builders: []manifest.Builder{{Shell: "html"}},
		},
	}
}

// Requirements: REQ_SEL_003
func TestDocumentsAll(t *testing.T) {
	m := &manifest.Manifest{}
	m.Bartleby.Roots = roots()

	docs, err := Documents(m, AllShells, nil)
	if err != nil {
		t.Fatalf("Documents: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("got %d docs, want 2", len(docs))
	}
}

// Requirements: REQ_SEL_003
func TestDocumentsSubset(t *testing.T) {
	m := &manifest.Manifest{}
	m.Bartleby.Roots = roots()

	docs, err := Documents(m, " api , guide ", nil)
	if err != nil {
		t.Fatalf("Documents: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("whitespace around names should be tolerated, got %d docs", len(docs))
	}
}

// Requirements: REQ_SEL_003
func TestDocumentsUnknownNameIsSkipped(t *testing.T) {
	m := &manifest.Manifest{}
	m.Bartleby.Roots = roots()

	docs, err := Documents(m, "guide,nope", nil)
	if err != nil {
		t.Fatalf("Documents: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("got %d docs, want just guide", len(docs))
	}
}

// Requirements: REQ_SEL_004
func TestDocumentsAllUnknownIsAnError(t *testing.T) {
	m := &manifest.Manifest{}
	m.Bartleby.Roots = roots()

	_, err := Documents(m, "nope,also-nope", nil)
	if err == nil {
		t.Fatal("expected an error when no requested root exists")
	}
	if !strings.Contains(err.Error(), "api") || !strings.Contains(err.Error(), "guide") {
		t.Errorf("error should list the available roots, got: %v", err)
	}
}

// Requirements: REQ_SEL_005
func TestDocumentsNoRootsFallsBackToDefault(t *testing.T) {
	docs, err := Documents(&manifest.Manifest{}, AllShells, nil)
	if err != nil {
		t.Fatalf("Documents: %v", err)
	}
	if _, ok := docs[manifest.DefaultRootDoc]; !ok {
		t.Errorf("expected the default root, got %+v", docs)
	}
}

// Requirements: REQ_SEL_001
func TestBuildsFiltersByShell(t *testing.T) {
	builds, err := Builds(roots(), "pdf", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 1 {
		t.Fatalf("got %d builds, want 1", len(builds))
	}
	if builds[0].Name != "guide" || builds[0].Shell != "pdf" {
		t.Errorf("build = %+v, want guide/pdf", builds[0])
	}
}

// The documentation has always described a comma-separated --shell list; now it
// works.
// Requirements: REQ_SEL_001
func TestBuildsAcceptsAShellList(t *testing.T) {
	builds, err := Builds(roots(), "pdf,html", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 3 {
		t.Fatalf("got %d builds, want 3 (api/html, guide/html, guide/pdf)", len(builds))
	}
}

// Requirements: REQ_SEL_001
func TestBuildsShellListWithAllMeansAll(t *testing.T) {
	builds, err := Builds(roots(), "html,all", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 3 {
		t.Errorf("got %d builds, want every builder", len(builds))
	}
}

// One good name in the list should still build, so a typo cannot silently cancel
// the whole run.
// Requirements: REQ_SEL_002_SPEC001
func TestBuildsShellListWithOneUnknownName(t *testing.T) {
	builds, err := Builds(roots(), "html,typo", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 2 {
		t.Errorf("got %d builds, want the two html builds", len(builds))
	}
}

// Requirements: REQ_SEL_001
func TestBuildsEmptyShellFilterMeansAll(t *testing.T) {
	builds, err := Builds(roots(), "", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}
	if len(builds) != 3 {
		t.Errorf("got %d builds, want every builder", len(builds))
	}
}

// A --shell value no root declares used to print "No builds to run." and exit 0.
// Requirements: REQ_SEL_002, REQ_SEL_001_SPEC001
func TestBuildsUnknownShellIsAnError(t *testing.T) {
	_, err := Builds(roots(), "confluence", manifest.Config{})
	if err == nil {
		t.Fatal("expected an error for a builder no root declares")
	}
	if !strings.Contains(err.Error(), "html") || !strings.Contains(err.Error(), "pdf") {
		t.Errorf("error should list the declared builders, got: %v", err)
	}
}

// Requirements: REQ_SEL_007
func TestBuildsAreDeterministic(t *testing.T) {
	want := []struct{ name, shell string }{
		{"api", "html"},
		{"guide", "html"},
		{"guide", "pdf"},
	}

	for range 5 {
		builds, err := Builds(roots(), AllShells, manifest.Config{})
		if err != nil {
			t.Fatalf("Builds: %v", err)
		}
		if len(builds) != len(want) {
			t.Fatalf("got %d builds, want %d", len(builds), len(want))
		}
		for i, w := range want {
			if builds[i].Name != w.name || builds[i].Shell != w.shell {
				t.Fatalf("build %d = %s/%s, want %s/%s", i, builds[i].Name, builds[i].Shell, w.name, w.shell)
			}
		}
	}
}

// A root that omits root_doc must reach the container as "index", not "".
// Requirements: REQ_SEL_006
func TestBuildsDefaultRootDoc(t *testing.T) {
	builds, err := Builds(roots(), "html", manifest.Config{})
	if err != nil {
		t.Fatalf("Builds: %v", err)
	}

	for _, b := range builds {
		if b.RootDoc == "" {
			t.Fatalf("build %+v has an empty root document", b)
		}
		if b.Name == "api" && b.RootDoc != manifest.DefaultRootDoc {
			t.Errorf("api root doc = %q, want %q", b.RootDoc, manifest.DefaultRootDoc)
		}
	}
}

// Requirements: REQ_CFG_001, REQ_CFG_003
func TestBuilderConfigPrecedence(t *testing.T) {
	t.Setenv("HMD_BARTLEBY__HTML__FROM_ENV_SCALAR", "scalar")
	t.Setenv("HMD_BARTLEBY__HTML__SHARED", "env-scalar-wins")

	root := manifest.Root{
		Config: map[string]any{"shared": "root-config", "only_root": "root"},
	}
	builder := manifest.Builder{
		Shell:  "html",
		Config: map[string]any{"shared": "builder-inline", "only_builder": "builder"},
	}
	cfg := manifest.Config{Builders: map[string]any{
		"html": map[string]any{"shared": "manifest-builders", "only_manifest": "manifest"},
	}}

	got := BuilderConfig(root, builder, cfg)

	checks := map[string]any{
		"only_root":       "root",
		"only_manifest":   "manifest",
		"only_builder":    "builder",
		"from_env_scalar": "scalar",
		"shared":          "env-scalar-wins",
	}
	for key, want := range checks {
		if got[key] != want {
			t.Errorf("config[%q] = %v, want %v", key, got[key], want)
		}
	}
}

// When the manifest declares no per-builder config, the JSON env var supplies it.
// Requirements: REQ_CFG_002
func TestBuilderConfigFromJSONEnv(t *testing.T) {
	t.Setenv("HMD_BARTLEBY_PDF_CONFIG", `{"papersize": "a4paper", "pointsize": 11}`)

	got := BuilderConfig(manifest.Root{}, manifest.Builder{Shell: "pdf"}, manifest.Config{})

	if got["papersize"] != "a4paper" {
		t.Errorf("papersize = %v, want a4paper", got["papersize"])
	}
	if got["pointsize"] != float64(11) {
		t.Errorf("pointsize = %v (%T), want 11", got["pointsize"], got["pointsize"])
	}
}

// The manifest is more specific than an environment default, so it wins.
// Requirements: REQ_CFG_002
func TestBuilderConfigManifestBeatsJSONEnv(t *testing.T) {
	t.Setenv("HMD_BARTLEBY_HTML_CONFIG", `{"theme": "from-env"}`)

	cfg := manifest.Config{Builders: map[string]any{
		"html": map[string]any{"theme": "from-manifest"},
	}}
	got := BuilderConfig(manifest.Root{}, manifest.Builder{Shell: "html"}, cfg)

	if got["theme"] != "from-manifest" {
		t.Errorf("theme = %v, want from-manifest", got["theme"])
	}
}

// Requirements: REQ_CFG_002_SPEC001
func TestBuilderConfigIgnoresMalformedJSONEnv(t *testing.T) {
	t.Setenv("HMD_BARTLEBY_HTML_CONFIG", `{"theme": `)

	got := BuilderConfig(manifest.Root{Config: map[string]any{"keep": "me"}},
		manifest.Builder{Shell: "html"}, manifest.Config{})

	if got["keep"] != "me" {
		t.Errorf("malformed JSON must not discard the rest of the config: %+v", got)
	}
}

// Requirements: REQ_CFG_003
func TestEnvScalarConfig(t *testing.T) {
	environ := []string{
		"HMD_BARTLEBY__HTML__THEME=furo",
		"HMD_BARTLEBY__HTML__NAV_DEPTH=3",
		"HMD_BARTLEBY__PDF__THEME=classic",
		"HMD_BARTLEBY__HTML__=empty-key",
		"UNRELATED=1",
		"malformed-entry",
	}

	got := EnvScalarConfig("html", environ)

	if len(got) != 2 {
		t.Fatalf("got %d keys, want 2: %+v", len(got), got)
	}
	if got["theme"] != "furo" || got["nav_depth"] != "3" {
		t.Errorf("config = %+v", got)
	}
}

// Requirements: REQ_CFG_003
func TestEnvScalarConfigEmpty(t *testing.T) {
	if got := EnvScalarConfig("html", []string{"UNRELATED=1"}); got != nil {
		t.Errorf("want nil for no matches, got %+v", got)
	}
}
