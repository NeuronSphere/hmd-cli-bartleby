package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/explain"
)

// explainTimeout bounds the request. One pass over a build log does not need
// long, and a build that has already failed should not be held up.
const explainTimeout = 3 * time.Minute

var explainOpts struct {
	log        string
	builder    string
	promptFile string
	model      string
	dryRun     bool
	maxBytes   int
}

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Ask Claude what went wrong in the last build",
	Long: `explain gathers the evidence from the last build — the Sphinx warnings, the end
of the build log, the LaTeX log if the PDF builder ran, and the source lines the
warnings point at — and asks Claude to interpret it in a single request.

It sends that evidence, which includes excerpts of your documentation, to the
Anthropic API. Nothing is sent unless you run this command or pass --explain to a
build. Use --dry-run to see exactly what would be sent.

Credentials come from the Anthropic SDK's usual sources: ANTHROPIC_API_KEY,
ANTHROPIC_AUTH_TOKEN, a profile from "ant auth login", or workload identity.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		rp, err := repoPath()
		if err != nil {
			return err
		}

		answer, err := runExplain(cmd, rp, explainOpts.builder)
		if err != nil {
			return err
		}
		if answer == "" {
			return errors.New("no explanation was produced")
		}
		return nil
	},
}

func init() {
	flags := explainCmd.Flags()
	flags.StringVar(&explainOpts.log, "log", "", "Log file to explain (default: the most recent build log)")
	flags.StringVar(&explainOpts.builder, "builder", "", "Builder whose log to explain, e.g. html or pdf")
	flags.StringVar(&explainOpts.promptFile, "prompt-file", "", "File holding the prompt to use instead of the built-in one")
	flags.StringVar(&explainOpts.model, "model", "", "Model to ask (default "+explain.DefaultModel+")")
	flags.BoolVar(&explainOpts.dryRun, "dry-run", false, "Print what would be sent and stop")
	flags.IntVar(&explainOpts.maxBytes, "max-bytes", explain.DefaultMaxBytes, "Cap on the evidence sent")
}

// runExplain performs one explanation attempt for a repository.
func runExplain(cmd *cobra.Command, repo, builder string) (string, error) {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	model := explainOpts.model
	if model == "" {
		model = os.Getenv(explain.EnvModel)
	}

	options := explain.RunOptions{
		Collect: explain.Options{
			RepoPath: repo,
			Builder:  builder,
			LogPath:  explainOpts.log,
			MaxBytes: explainOpts.maxBytes,
		},
		Prompt: explain.PromptOptions{
			File:     explainOpts.promptFile,
			RepoPath: repo,
		},
		Model:  model,
		Out:    stdout,
		Status: stderr,
		DryRun: explainOpts.dryRun,
	}

	if !options.DryRun {
		if !explain.HasCredentials(os.Getenv, anthropicProfileExists) {
			return "", errors.New(credentialsHelp())
		}
		options.Requester = explain.Claude{Model: model, Stream: stdout}
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), explainTimeout)
	defer cancel()

	return explain.Run(ctx, options)
}

// explainFailure is the opt-in hook on a failed build: it offers an explanation
// and never changes the outcome. The build's error is what the user gets back,
// whatever happens here.
func explainFailure(cmd *cobra.Command, repo, builder string) {
	stderr := cmd.ErrOrStderr()

	fmt.Fprintln(stderr, "\nAsking Claude what went wrong (--explain)...")

	if _, err := runExplain(cmd, repo, builder); err != nil {
		fmt.Fprintf(stderr, "warning: could not explain the failure: %v\n", err)
	}
}

// explainEnabled reports whether a failed build should try to explain itself.
func explainEnabled(o options, env envFunc) bool {
	return o.explain || truthy(env(explain.EnvEnabled))
}

// anthropicProfileExists reports whether `ant auth login` has stored a profile.
func anthropicProfileExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(home, ".config", "anthropic"))
	return err == nil && info.IsDir()
}

// credentialsHelp explains the options rather than just reporting absence.
func credentialsHelp() string {
	var b strings.Builder
	b.WriteString("no Anthropic credentials found. Either:\n")
	b.WriteString("  export ANTHROPIC_API_KEY=...      (also works in CI)\n")
	b.WriteString("  ant auth login                    (stores a profile the SDK reads)\n")
	b.WriteString("Run with --dry-run to see what would be sent without sending it.")
	return b.String()
}
