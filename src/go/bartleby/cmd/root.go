// Package cmd implements the bartleby command line.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/buildplan"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/gather"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/hmdenv"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/pipconf"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/sanitize"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/sources"
)

const (
	defaultRegistry = "ghcr.io/neuronsphere"
	defaultImageTag = "stable"
	imageRepo       = "hmd-tf-bartleby"
)

// options holds the parsed flags. Resolution functions take it explicitly rather
// than reading globals so they can be tested.
type options struct {
	autodoc          bool
	explain          bool
	image            string
	shell            string
	rootDoc          string
	gather           string
	title            string
	noTimestampTitle bool
	confidential     bool
	defaultLogo      string
	htmlDefaultLogo  string
	pdfDefaultLogo   string
}

var opts options

// envFunc looks up an environment variable; swapped out in tests.
type envFunc func(string) string

var rootCmd = &cobra.Command{
	Use:   "bartleby",
	Short: "Render reStructuredText documentation with Sphinx in Docker",
	Long: `bartleby renders reStructuredText documentation using Sphinx inside a Docker
container: HTML, PDF, RevealJS slides, and PlantUML images.

Run with no subcommand to build every builder configured in
meta-data/manifest.json, or pass --shell to build just one.`,
	// Runtime failures are reported by Execute; usage text is only helpful for
	// flag and argument mistakes, which Cobra reports before RunE is reached.
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// $HMD_HOME/.config/hmd.env supplies registry, customer code, logos, and
		// similar defaults. It is optional: warn, never fail.
		hmdenv.LoadAndWarn(cmd.ErrOrStderr())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuilds(cmd, opts.shell)
	},
}

// Execute runs the CLI. version is injected at build time by GoReleaser.
func Execute(version string) {
	setVersion(version)

	// Interrupt cancels the build context, which lets the runner remove the
	// container it started instead of leaving it behind.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	flags := rootCmd.PersistentFlags()

	flags.BoolVarP(&opts.autodoc, "autodoc", "a", false,
		"Generate Python API docs with autosummary (requires src/python/)")
	flags.BoolVar(&opts.explain, "explain", false,
		"On failure, ask Claude to explain the build log (sends documentation excerpts to the Anthropic API)")
	flags.StringVar(&opts.image, "image", "",
		"Transform image to run, e.g. hmd-tf-bartleby:local — overrides the registry and version variables")
	flags.StringVarP(&opts.shell, "shell", "s", buildplan.AllShells,
		"Builder(s) to run: comma-separated names from the manifest (html, pdf, revealjs, ...), or 'all'")
	flags.StringVarP(&opts.rootDoc, "root-doc", "r", buildplan.AllShells,
		"Root document(s) to build: comma-separated manifest root names, or 'all'")
	flags.StringVarP(&opts.gather, "gather", "g", "",
		"Comma-separated sibling repos whose docs to gather before building (hmd-docs-bartleby only)")
	flags.StringVar(&opts.title, "title", "",
		"Document title (defaults to <repo>-<version>)")
	flags.BoolVar(&opts.noTimestampTitle, "no-timestamp-title", false,
		"Omit the timestamp appended to output document names")
	flags.BoolVar(&opts.confidential, "confidential", false,
		"Stamp documents with HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT")
	flags.StringVar(&opts.defaultLogo, "default-logo", "",
		"URL of the logo used for HTML and PDF unless overridden")
	flags.StringVar(&opts.htmlDefaultLogo, "html-default-logo", "",
		"URL of the HTML logo")
	flags.StringVar(&opts.pdfDefaultLogo, "pdf-default-logo", "",
		"URL of the PDF cover image")

	rootCmd.AddCommand(
		newShellCmd("html", "Render HTML documentation", "html"),
		newShellCmd("pdf", "Render PDF documentation", "pdf"),
		newShellCmd("slides", "Render a RevealJS slideshow", "revealjs"),
		pumlCmd,
		updateImageCmd,
		configureCmd,
		explainCmd,
		skillsCmd,
		versionCmd,
	)
}

// newShellCmd builds a subcommand that pins --shell to one builder.
func newShellCmd(use, short, shell string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reject a contradiction rather than silently ignoring the flag,
			// which is what the previous implementation did.
			if cmd.Flags().Changed("shell") && opts.shell != shell && opts.shell != buildplan.AllShells {
				return fmt.Errorf("%q builds the %s builder; drop --shell=%s or use the bare bartleby command",
					use, shell, opts.shell)
			}
			return runBuilds(cmd, shell)
		},
	}
}

// imageName returns the transform image to run.
//
// An explicit reference — --image, or BARTLEBY_IMAGE — is used as given, which is
// what makes a locally built image usable: it needs no registry prefix and no
// version that looks like a release. Otherwise the reference is composed from the
// registry and version variables.
func imageName(o options, env envFunc) string {
	if o.image != "" {
		return o.image
	}
	if explicit := env("BARTLEBY_IMAGE"); explicit != "" {
		return explicit
	}

	registry := env("HMD_CONTAINER_REGISTRY")
	if registry == "" {
		registry = defaultRegistry
	}
	tag := env("HMD_TF_BARTLEBY_VERSION")
	if tag == "" {
		tag = defaultImageTag
	}
	return fmt.Sprintf("%s/%s:%s", registry, imageRepo, tag)
}

// repoPath returns the repository root, which is the working directory.
func repoPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determining the current directory: %w", err)
	}
	return cwd, nil
}

// repoName returns the manifest name, falling back to the directory name.
func repoName(m *manifest.Manifest, path string) string {
	if m.Name != "" {
		return m.Name
	}
	return filepath.Base(path)
}

// resolveLogos applies flag, then environment, then manifest precedence, with
// the HTML and PDF logos falling back to the shared default.
func resolveLogos(o options, m *manifest.Manifest, env envFunc) (defaultLogo, htmlLogo, pdfLogo string) {
	cfg := m.Bartleby.Config

	defaultLogo = firstNonEmpty(o.defaultLogo, env("HMD_BARTLEBY_DEFAULT_LOGO"), cfg.DefaultLogo)
	htmlLogo = firstNonEmpty(o.htmlDefaultLogo, env("HMD_BARTLEBY_HTML_DEFAULT_LOGO"), cfg.HTMLDefaultLogo, defaultLogo)
	pdfLogo = firstNonEmpty(o.pdfDefaultLogo, env("HMD_BARTLEBY_PDF_DEFAULT_LOGO"), cfg.PDFDefaultLogo, defaultLogo)

	return defaultLogo, htmlLogo, pdfLogo
}

// resolveConfidential decides whether to stamp documents and with what text. The
// statement only comes from the environment — it is prose, not a repo setting.
func resolveConfidential(o options, m *manifest.Manifest, env envFunc) (bool, string) {
	confidential := o.confidential
	if !confidential {
		if v := m.Bartleby.Config.Confidential; v != nil {
			confidential = *v
		}
	}
	if !confidential {
		confidential = truthy(env("HMD_BARTLEBY_CONFIDENTIAL"))
	}

	if !confidential {
		return false, ""
	}

	statement := env("HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT")
	if statement == "" {
		return true, ""
	}
	return true, statement
}

// resolveTitle returns the document title to use, plus a note when sanitizing
// changed it. An empty title lets the container derive its own from the repo.
func resolveTitle(o options, repo, version string) (title, note string) {
	title = o.title
	if title == "" {
		title = repo + "-" + version
	}

	clean := sanitize.Title(title)
	if clean == title {
		return clean, ""
	}
	if clean == "" {
		return "", fmt.Sprintf("note: title %q has no characters that are safe for LaTeX; letting the container name the document", title)
	}
	return clean, fmt.Sprintf("note: title %q sanitized to %q for LaTeX", title, clean)
}

// globalStylesPath returns $HMD_HOME/bartleby/styles, or "" when HMD_HOME is unset.
func globalStylesPath(env envFunc) string {
	home := env("HMD_HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(os.ExpandEnv(home), "bartleby", "styles")
}

// runBuilds is the orchestration shared by the root command and every builder
// subcommand: read the manifest, work out what to build, prepare the docs tree,
// and run one container per build.
func runBuilds(cmd *cobra.Command, shellFilter string) error {
	ctx := cmd.Context()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	env := envFunc(os.Getenv)

	rp, err := repoPath()
	if err != nil {
		return err
	}

	m, err := manifest.Read(rp)
	if err != nil && !errors.Is(err, manifest.ErrNotFound) {
		return err
	}
	if errors.Is(err, manifest.ErrNotFound) {
		fmt.Fprintf(stderr, "note: no %s — building %s.rst with %s\n",
			manifest.Path(rp), manifest.DefaultRootDoc, strings.Join(manifest.DefaultBuilders, " and "))
	}

	version, err := manifest.ReadVersion(rp)
	if err != nil {
		fmt.Fprintf(stderr, "warning: %v — using version %q\n", err, version)
	}

	if opts.gather != "" {
		if err := gather.Repos(rp, opts.gather, stdout); err != nil {
			return err
		}
	}

	docs, err := buildplan.Documents(m, opts.rootDoc, os.Stderr)
	if err != nil {
		return err
	}

	builds, err := buildplan.Builds(docs, shellFilter, m.Bartleby.Config)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(rp, "target", "bartleby")
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputPath, err)
	}

	name := repoName(m, rp)

	autodoc := opts.autodoc
	if autodoc && !isDir(filepath.Join(rp, "src", "python")) {
		fmt.Fprintln(stderr, "warning: --autodoc needs a Python package at src/python/ — building without it")
		autodoc = false
	}

	pipConfig := pipconf.Resolved{}
	if autodoc {
		resolved, cleanup, err := pipconf.Resolve()
		if err != nil {
			return err
		}
		defer cleanup()
		pipConfig = resolved
	}

	instanceName := firstNonEmpty(env("HMD_INSTANCE_NAME"), name)
	title, note := resolveTitle(opts, name, version)
	if note != "" {
		fmt.Fprintln(stderr, note)
	}

	defaultLogo, htmlLogo, pdfLogo := resolveLogos(opts, m, env)
	_, statement := resolveConfidential(opts, m, env)

	restore, err := prepareSources(rp, m, builds, stderr)
	if err != nil {
		return err
	}
	defer restore()

	img := imageName(opts, env)

	for _, b := range builds {
		fmt.Fprintf(stdout, "Building %s with the %s builder (root document: %s)...\n", b.Name, b.Shell, b.RootDoc)

		cfg := runner.TransformConfig{
			ImageName:    img,
			InstanceName: instanceName,
			Context: runner.TransformInstanceContext{
				Name:    b.Name,
				Shell:   b.Shell,
				RootDoc: b.RootDoc,
				Config:  b.Config,
			},
			Environment:              firstNonEmpty(env("HMD_ENVIRONMENT"), "local"),
			Region:                   firstNonEmpty(env("HMD_REGION"), "reg1"),
			Account:                  env("HMD_ACCOUNT"),
			CustomerCode:             firstNonEmpty(env("HMD_CUSTOMER_CODE"), "hmd"),
			DeploymentID:             firstNonEmpty(env("HMD_DID"), "aaa"),
			CompanyName:              env("HMD_DOC_COMPANY_NAME"),
			DocRepo:                  name,
			DocRepoVersion:           version,
			InputPath:                rp,
			OutputPath:               outputPath,
			Autodoc:                  autodoc,
			PipConfigPath:            pipConfig.Path,
			DocumentTitle:            title,
			NoTimestampTitle:         opts.noTimestampTitle,
			ConfidentialityStatement: statement,
			DefaultLogo:              defaultLogo,
			HTMLDefaultLogo:          htmlLogo,
			PDFDefaultLogo:           pdfLogo,
			GlobalStylesPath:         globalStylesPath(env),
		}

		if err := runner.RunTransform(ctx, cfg); err != nil {
			buildErr := fmt.Errorf("building %s with the %s builder: %w", b.Name, b.Shell, err)

			// Advisory only: whatever the explanation attempt does, the build
			// failure is what the caller gets back and what sets the exit code.
			if explainEnabled(opts, env) {
				explainFailure(cmd, rp, b.Shell)
			}
			return buildErr
		}
	}

	fmt.Fprintf(stdout, "Done. Output is in %s\n", outputPath)
	return nil
}

// prepareSources stages any bartleby.sources trees and injects their toctrees
// into the root documents being built. The returned function undoes both and is
// safe to call when there was nothing to do.
func prepareSources(rp string, m *manifest.Manifest, builds []buildplan.Build, stderr io.Writer) (func(), error) {
	noop := func() {}

	declared := m.Bartleby.Sources
	if len(declared) == 0 {
		return noop, nil
	}

	docsPath := filepath.Join(rp, "docs")
	valid := sources.Validate(rp, docsPath, declared, stderr)
	if len(valid) == 0 {
		return noop, nil
	}

	if _, err := sources.Stage(rp, docsPath, valid); err != nil {
		// Roll the staging directory back before surfacing the failure.
		if cleanupErr := sources.Cleanup(docsPath); cleanupErr != nil {
			fmt.Fprintf(stderr, "warning: %v\n", cleanupErr)
		}
		return noop, err
	}

	originals := make(map[string]string)
	rollback := func() {
		for path, original := range originals {
			if err := sources.Restore(path, original); err != nil {
				fmt.Fprintf(stderr, "warning: %v\n", err)
			}
		}
		if err := sources.Cleanup(docsPath); err != nil {
			fmt.Fprintf(stderr, "warning: %v\n", err)
		}
	}

	seen := make(map[string]bool)
	for _, b := range builds {
		if seen[b.RootDoc] {
			continue
		}
		seen[b.RootDoc] = true

		indexPath := filepath.Join(docsPath, b.RootDoc+".rst")
		if !exists(indexPath) {
			fmt.Fprintf(stderr, "warning: root document %s not found; sources not injected for %s\n", indexPath, b.Name)
			continue
		}

		original, err := sources.Inject(indexPath, valid)
		if err != nil {
			rollback()
			return noop, err
		}
		originals[indexPath] = original
	}

	return rollback, nil
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// truthy interprets an environment variable as a boolean the way a shell user
// expects. The previous implementation compared against "true" exactly, so
// HMD_BARTLEBY_CONFIDENTIAL=True did nothing.
func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
