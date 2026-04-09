package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/manifest"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
)

const defaultRegistry = "ghcr.io/neuronsphere"
const defaultVersion = "stable"
const defaultBuilderConfig = `{}`

// Persistent flags shared across all subcommands.
var (
	flagAutodoc          bool
	flagShell            string
	flagRootDoc          string
	flagDocumentTitle    string
	flagNoTimestampTitle bool
	flagConfidential     bool
	flagDefaultLogo      string
	flagHTMLDefaultLogo  string
	flagPDFDefaultLogo   string
)

var rootCmd = &cobra.Command{
	Use:   "bartleby",
	Short: "Run Bartleby transforms to generate rendered documents",
	Long: `bartleby renders reStructuredText documentation using Sphinx inside a Docker container.

Run with no subcommand to build all configured output formats.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuilds("all")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagAutodoc, "autodoc", "a", false,
		"Enable Python autosummary documentation generation (requires src/python/)")
	rootCmd.PersistentFlags().StringVarP(&flagShell, "shell", "s", "all",
		"Builder(s) to use: html, pdf, revealjs, or all")
	rootCmd.PersistentFlags().StringVarP(&flagRootDoc, "root-doc", "r", "all",
		"Root document(s) to build (comma-separated names, or 'all')")
	rootCmd.PersistentFlags().StringVar(&flagDocumentTitle, "title", "",
		"Custom document title")
	rootCmd.PersistentFlags().BoolVar(&flagNoTimestampTitle, "no-timestamp-title", false,
		"Suppress timestamp in document title")
	rootCmd.PersistentFlags().BoolVar(&flagConfidential, "confidential", false,
		"Include confidentiality statement from HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT")
	rootCmd.PersistentFlags().StringVar(&flagDefaultLogo, "default-logo", "",
		"URL to default logo for HTML and PDF")
	rootCmd.PersistentFlags().StringVar(&flagHTMLDefaultLogo, "html-default-logo", "",
		"URL to default HTML logo")
	rootCmd.PersistentFlags().StringVar(&flagPDFDefaultLogo, "pdf-default-logo", "",
		"URL to default PDF cover image")

	rootCmd.AddCommand(htmlCmd)
	rootCmd.AddCommand(pdfCmd)
	rootCmd.AddCommand(slidesCmd)
	rootCmd.AddCommand(pumlCmd)
	rootCmd.AddCommand(updateImageCmd)
}

// imageName returns the full Docker image reference from environment variables.
func imageName() string {
	registry := os.Getenv("HMD_CONTAINER_REGISTRY")
	if registry == "" {
		registry = defaultRegistry
	}
	version := os.Getenv("HMD_TF_BARTLEBY_VERSION")
	if version == "" {
		version = defaultVersion
	}
	return fmt.Sprintf("%s/hmd-tf-bartleby:%s", registry, version)
}

// repoPath returns the current working directory.
func repoPath() string {
	cwd, _ := os.Getwd()
	return cwd
}

// repoName returns the repo name from the manifest or falls back to the directory name.
func repoName(m *manifest.Manifest) string {
	if m.Name != "" {
		return m.Name
	}
	return filepath.Base(repoPath())
}

// getDocuments returns the roots to build, filtered by the --root-doc flag.
func getDocuments(m *manifest.Manifest, rootDoc string) (map[string]manifest.Root, error) {
	roots := m.Bartleby.Roots
	if roots == nil {
		return map[string]manifest.Root{
			"index": {RootDoc: "index", Builders: []string{"html", "pdf"}},
		}, nil
	}

	if rootDoc == "all" {
		return roots, nil
	}

	requested := strings.Split(rootDoc, ",")
	docs := make(map[string]manifest.Root)
	for _, name := range requested {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if root, ok := roots[name]; ok {
			docs[name] = root
		} else {
			available := make([]string, 0, len(roots))
			for k := range roots {
				available = append(available, k)
			}
			fmt.Fprintf(os.Stderr, "warning: root document %q not found (available: %s)\n",
				name, strings.Join(available, ", "))
		}
	}

	if len(docs) == 0 {
		available := make([]string, 0, len(roots))
		for k := range roots {
			available = append(available, k)
		}
		return nil, fmt.Errorf("none of the requested root documents (%s) found (available: %s)",
			rootDoc, strings.Join(available, ", "))
	}

	return docs, nil
}

type buildSpec struct {
	name    string
	shell   string
	rootDoc string
	config  map[string]interface{}
}

// getBuilds returns the list of (doc, shell) pairs to build, filtered by shell.
func getBuilds(docs map[string]manifest.Root, shellFilter string) []buildSpec {
	var builds []buildSpec

	for rootName, root := range docs {
		for _, b := range root.Builders {
			if shellFilter != "all" && b != shellFilter {
				continue
			}
			config := make(map[string]interface{})
			// Merge root-level config overrides.
			for k, v := range root.Config {
				config[k] = v
			}
			// Merge env-var builder config (HMD_BARTLEBY__SHELL__KEY).
			prefix := fmt.Sprintf("HMD_BARTLEBY__%s__", strings.ToUpper(b))
			for _, kv := range os.Environ() {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) == 2 && strings.HasPrefix(parts[0], prefix) {
					configKey := strings.ToLower(strings.TrimPrefix(parts[0], prefix))
					config[configKey] = parts[1]
				}
			}

			builds = append(builds, buildSpec{
				name:    rootName,
				shell:   b,
				rootDoc: root.RootDoc,
				config:  config,
			})
		}
	}
	return builds
}

// resolveLogos applies flag/env/manifest priority for logo URLs.
func resolveLogos(m *manifest.Manifest) (defaultLogo, htmlLogo, pdfLogo string) {
	manifestConfig := m.Bartleby.Config

	defaultLogo = flagDefaultLogo
	if defaultLogo == "" {
		if v, ok := manifestConfig["default_logo"].(string); ok {
			defaultLogo = v
		}
		if defaultLogo == "" {
			defaultLogo = os.Getenv("HMD_BARTLEBY_DEFAULT_LOGO")
		}
	}

	htmlLogo = flagHTMLDefaultLogo
	if htmlLogo == "" {
		htmlLogo = os.Getenv("HMD_BARTLEBY_HTML_DEFAULT_LOGO")
	}
	if htmlLogo == "" {
		htmlLogo = defaultLogo
	}

	pdfLogo = flagPDFDefaultLogo
	if pdfLogo == "" {
		pdfLogo = os.Getenv("HMD_BARTLEBY_PDF_DEFAULT_LOGO")
	}
	if pdfLogo == "" {
		pdfLogo = defaultLogo
	}

	return
}

// resolveConfidential resolves the confidential flag and its statement.
func resolveConfidential() (bool, string) {
	confidential := flagConfidential
	if !confidential {
		confidential = os.Getenv("HMD_BARTLEBY_CONFIDENTIAL") == "true"
	}
	statement := ""
	if confidential {
		statement = os.Getenv("HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT")
	}
	return confidential, statement
}

// globalStylesPath returns $HMD_HOME/bartleby/styles if HMD_HOME is set, else "".
func globalStylesPath() string {
	hmdHome := os.Getenv("HMD_HOME")
	if hmdHome == "" {
		return ""
	}
	return filepath.Join(hmdHome, "bartleby", "styles")
}

// pipConfigPath returns the path to the user's pip.conf (for autodoc).
func pipConfigPath() string {
	username := os.Getenv("PIP_USERNAME")
	password := os.Getenv("PIP_PASSWORD")
	if username != "" && password != "" {
		// Caller should handle writing a temp pip.conf; return empty to signal that.
		return ""
	}
	return runner.DefaultPipConfigPath()
}

// sanitizeTitle replaces characters that are unsafe in a LaTeX document title or
// filename: spaces become hyphens (spaces break Makefile targets) and underscores
// become hyphens (underscores are subscript operators in LaTeX text mode).
func sanitizeTitle(s string) string {
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// runBuilds is the core orchestration: read manifest, compute builds, run containers.
func runBuilds(shellFilter string) error {
	rp := repoPath()
	m, err := manifest.Read(rp)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	docs, err := getDocuments(m, flagRootDoc)
	if err != nil {
		return err
	}

	builds := getBuilds(docs, shellFilter)
	if len(builds) == 0 {
		fmt.Println("No builds to run.")
		return nil
	}

	outputPath := filepath.Join(rp, "target", "bartleby")
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	img := imageName()
	name := repoName(m)
	version := manifest.ReadVersion(rp)

	hasPython := false
	if _, err := os.Stat(filepath.Join(rp, "src", "python")); err == nil {
		hasPython = true
	}

	autodoc := flagAutodoc && hasPython
	if flagAutodoc && !hasPython {
		fmt.Fprintln(os.Stderr, "warning: --autodoc requires src/python/ — continuing without autodoc")
	}

	pipPath := ""
	if autodoc {
		pipPath = pipConfigPath()
	}

	instanceName := os.Getenv("HMD_INSTANCE_NAME")
	if instanceName == "" {
		instanceName = name
	}

	// Derive an effective document title that is safe for LaTeX (no underscores or
	// spaces). When the user provides --title we sanitize it. When they don't, we
	// build a default from the repo name + version so the container never falls back
	// to the raw manifest name (which may contain underscores).
	effectiveTitle := flagDocumentTitle
	if effectiveTitle == "" {
		effectiveTitle = name + "-" + version
	}
	sanitized := sanitizeTitle(effectiveTitle)
	if sanitized != effectiveTitle {
		fmt.Fprintf(os.Stderr, "note: title %q sanitized to %q for LaTeX compatibility\n",
			effectiveTitle, sanitized)
	}
	effectiveTitle = sanitized

	defaultLogo, htmlLogo, pdfLogo := resolveLogos(m)
	confidential, confidentialStatement := resolveConfidential()

	for _, b := range builds {
		fmt.Printf("Building %s/%s (root: %s)...\n", b.name, b.shell, b.rootDoc)

		cfg := runner.TransformConfig{
			ImageName:    img,
			InstanceName: instanceName,
			Context: runner.TransformInstanceContext{
				Name:    b.name,
				Shell:   b.shell,
				RootDoc: b.rootDoc,
				Config:  b.config,
			},
			Environment:              envOrDefault("HMD_ENVIRONMENT", "local"),
			Region:                   envOrDefault("HMD_REGION", "reg1"),
			Account:                  os.Getenv("HMD_ACCOUNT"),
			CustomerCode:             envOrDefault("HMD_CUSTOMER_CODE", "hmd"),
			DeploymentID:             envOrDefault("HMD_DID", "aaa"),
			Autodoc:                  autodoc,
			DocRepo:                  name,
			DocRepoVersion:           version,
			InputPath:                rp,
			OutputPath:               outputPath,
			PipConfigPath:            pipPath,
			DocumentTitle:            effectiveTitle,
			NoTimestampTitle:         flagNoTimestampTitle,
			Confidential:             confidential,
			ConfidentialityStatement: confidentialStatement,
			DefaultLogo:              defaultLogo,
			HTMLDefaultLogo:          htmlLogo,
			PDFDefaultLogo:           pdfLogo,
			GlobalStylesPath:         globalStylesPath(),
		}

		if err := runner.RunTransform(cfg); err != nil {
			return fmt.Errorf("transform failed for %s/%s: %w", b.name, b.shell, err)
		}
	}

	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
