package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
)

// fakeEnv builds an envFunc over a map so resolution can be tested without
// touching the process environment.
func fakeEnv(values map[string]string) envFunc {
	return func(key string) string { return values[key] }
}

// Requirements: REQ_EXEC_013
func TestImageName(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"defaults", nil, "ghcr.io/neuronsphere/hmd-tf-bartleby:stable"},
		{
			"registry override",
			map[string]string{"HMD_CONTAINER_REGISTRY": "registry.internal.test/ns"},
			"registry.internal.test/ns/hmd-tf-bartleby:stable",
		},
		{
			"tag override",
			map[string]string{"HMD_TF_BARTLEBY_VERSION": "1.2.3"},
			"ghcr.io/neuronsphere/hmd-tf-bartleby:1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageName(fakeEnv(tt.env)); got != tt.want {
				t.Errorf("imageName = %q, want %q", got, tt.want)
			}
		})
	}
}

// Requirements: REQ_CFG_004
func TestResolveLogosPrecedence(t *testing.T) {
	m := &manifest.Manifest{}
	m.Bartleby.Config = manifest.Config{
		DefaultLogo:     "manifest-default",
		HTMLDefaultLogo: "manifest-html",
	}
	env := fakeEnv(map[string]string{
		"HMD_BARTLEBY_DEFAULT_LOGO":     "env-default",
		"HMD_BARTLEBY_PDF_DEFAULT_LOGO": "env-pdf",
	})

	t.Run("flags win", func(t *testing.T) {
		o := options{defaultLogo: "flag-default", htmlDefaultLogo: "flag-html", pdfDefaultLogo: "flag-pdf"}
		def, html, pdf := resolveLogos(o, m, env)

		if def != "flag-default" || html != "flag-html" || pdf != "flag-pdf" {
			t.Errorf("got %q/%q/%q, want the flag values", def, html, pdf)
		}
	})

	t.Run("environment beats manifest", func(t *testing.T) {
		def, _, pdf := resolveLogos(options{}, m, env)

		if def != "env-default" {
			t.Errorf("default = %q, want env-default", def)
		}
		if pdf != "env-pdf" {
			t.Errorf("pdf = %q, want env-pdf", pdf)
		}
	})

	t.Run("manifest is used when nothing else is set", func(t *testing.T) {
		_, html, _ := resolveLogos(options{}, m, fakeEnv(nil))

		if html != "manifest-html" {
			t.Errorf("html = %q, want manifest-html", html)
		}
	})

	t.Run("html and pdf fall back to the shared default", func(t *testing.T) {
		bare := &manifest.Manifest{}
		def, html, pdf := resolveLogos(options{defaultLogo: "shared"}, bare, fakeEnv(nil))

		if def != "shared" || html != "shared" || pdf != "shared" {
			t.Errorf("got %q/%q/%q, want all three to be shared", def, html, pdf)
		}
	})
}

// Requirements: REQ_CFG_005, REQ_CFG_005_SPEC001
func TestResolveConfidential(t *testing.T) {
	statement := "Internal use only"
	env := fakeEnv(map[string]string{"HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT": statement})

	t.Run("off by default", func(t *testing.T) {
		on, got := resolveConfidential(options{}, &manifest.Manifest{}, env)
		if on {
			t.Error("should be off without a flag, manifest value, or env var")
		}
		if got != "" {
			t.Errorf("statement = %q, want empty when off", got)
		}
	})

	t.Run("flag", func(t *testing.T) {
		on, got := resolveConfidential(options{confidential: true}, &manifest.Manifest{}, env)
		if !on || got != statement {
			t.Errorf("on = %v, statement = %q", on, got)
		}
	})

	t.Run("manifest", func(t *testing.T) {
		yes := true
		m := &manifest.Manifest{}
		m.Bartleby.Config.Confidential = &yes

		if on, _ := resolveConfidential(options{}, m, env); !on {
			t.Error("manifest confidential: true should enable it")
		}
	})

	// HMD_BARTLEBY_CONFIDENTIAL=True did nothing before, because the check was
	// an exact comparison against "true".
	t.Run("environment accepts the usual spellings", func(t *testing.T) {
		for _, value := range []string{"true", "True", "TRUE", "1", "yes", "on"} {
			on, _ := resolveConfidential(options{}, &manifest.Manifest{},
				fakeEnv(map[string]string{"HMD_BARTLEBY_CONFIDENTIAL": value}))
			if !on {
				t.Errorf("HMD_BARTLEBY_CONFIDENTIAL=%q should enable confidentiality", value)
			}
		}
		for _, value := range []string{"", "false", "no", "0", "maybe"} {
			on, _ := resolveConfidential(options{}, &manifest.Manifest{},
				fakeEnv(map[string]string{"HMD_BARTLEBY_CONFIDENTIAL": value}))
			if on {
				t.Errorf("HMD_BARTLEBY_CONFIDENTIAL=%q should not enable confidentiality", value)
			}
		}
	})

	t.Run("on with no statement", func(t *testing.T) {
		on, got := resolveConfidential(options{confidential: true}, &manifest.Manifest{}, fakeEnv(nil))
		if !on {
			t.Error("the flag alone should still turn it on")
		}
		if got != "" {
			t.Errorf("statement = %q, want empty", got)
		}
	})
}

// Requirements: REQ_CFG_006, REQ_CFG_007, REQ_CFG_007_SPEC001
func TestResolveTitle(t *testing.T) {
	tests := []struct {
		name     string
		opts     options
		repo     string
		version  string
		want     string
		wantNote bool
	}{
		{"defaults to repo-version", options{}, "hmd-docs-example", "1.4", "hmd-docs-example-1.4", false},
		{"explicit title", options{title: "release-notes"}, "repo", "1.0", "release-notes", false},
		{"unsafe explicit title", options{title: "My Doc_Title"}, "repo", "1.0", "My-Doc-Title", true},
		{"unsafe repo name", options{}, "my_unsafe_repo_name", "1.0", "my-unsafe-repo-name-1.0", true},
		{"nothing usable", options{title: "___"}, "repo", "1.0", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, note := resolveTitle(tt.opts, tt.repo, tt.version)
			if got != tt.want {
				t.Errorf("title = %q, want %q", got, tt.want)
			}
			if tt.wantNote && note == "" {
				t.Error("expected a note explaining the change")
			}
			if !tt.wantNote && note != "" {
				t.Errorf("unexpected note: %s", note)
			}
		})
	}
}

// Requirements: REQ_MAN_004
func TestRepoName(t *testing.T) {
	m := &manifest.Manifest{Name: "from-manifest"}
	if got := repoName(m, "/tmp/from-directory"); got != "from-manifest" {
		t.Errorf("name = %q, want from-manifest", got)
	}
	if got := repoName(&manifest.Manifest{}, "/tmp/from-directory"); got != "from-directory" {
		t.Errorf("name = %q, want from-directory", got)
	}
}

// Requirements: REQ_ENV_005
func TestGlobalStylesPath(t *testing.T) {
	if got := globalStylesPath(fakeEnv(nil)); got != "" {
		t.Errorf("want empty without HMD_HOME, got %q", got)
	}

	got := globalStylesPath(fakeEnv(map[string]string{"HMD_HOME": "/opt/hmd"}))
	want := filepath.Join("/opt/hmd", "bartleby", "styles")
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

// Requirements: REQ_CFG_005_SPEC001
func TestTruthy(t *testing.T) {
	for _, v := range []string{"1", "t", "true", "TRUE", " yes ", "on", "Y"} {
		if !truthy(v) {
			t.Errorf("truthy(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "off", "banana"} {
		if truthy(v) {
			t.Errorf("truthy(%q) = true, want false", v)
		}
	}
}

// Requirements: REQ_CFG_004
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "third"); got != "third" {
		t.Errorf("got %q, want third", got)
	}
	if got := firstNonEmpty("first", "second"); got != "first" {
		t.Errorf("got %q, want first", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// The subcommands pin a builder; contradicting that with --shell should be
// reported rather than ignored.
// Requirements: REQ_CLI_002_SPEC001
func TestShellSubcommandRejectsConflictingFlag(t *testing.T) {
	defer func() { opts = options{} }()

	cmd := newShellCmd("html", "Render HTML documentation", "html")
	cmd.SetOut(new(strings.Builder))
	cmd.SetErr(new(strings.Builder))
	cmd.Flags().StringVarP(&opts.shell, "shell", "s", "all", "")

	if err := cmd.Flags().Set("shell", "pdf"); err != nil {
		t.Fatal(err)
	}

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected an error for `bartleby html --shell pdf`")
	}
	if !strings.Contains(err.Error(), "--shell") {
		t.Errorf("error should mention the flag, got: %v", err)
	}
}

// Requirements: REQ_PUML_001_SPEC001
func TestFindPumlFiles(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"b.puml",
		"a.puml",
		filepath.Join("nested", "deep", "c.PUML"),
		"notes.rst",
		filepath.Join("nested", "diagram.txt"),
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("@startuml\n@enduml\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := findPumlFiles(root)
	if err != nil {
		t.Fatalf("findPumlFiles: %v", err)
	}

	want := []string{"a.puml", "b.puml", "nested/deep/c.PUML"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i, w := range want {
		if files[i] != w {
			t.Errorf("file %d = %q, want %q (results must be sorted, relative, and slash-separated)", i, files[i], w)
		}
	}
}

// Requirements: REQ_PUML_002
func TestFindPumlFilesEmptyTree(t *testing.T) {
	files, err := findPumlFiles(t.TempDir())
	if err != nil {
		t.Fatalf("findPumlFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("want no files, got %v", files)
	}
}
