package cmd

import (
	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

type reqsOptions struct {
	check            bool
	repo             string
	quiet            bool
	includeProposals bool
	verify           bool
	needs            string
	results          []string
	runs             []string
	allRuns          bool
	out              string
	json             bool
}

var reqsOpts reqsOptions

// reqsCmd is a front door onto the same function the standalone reqtrace binary
// calls. It exists because someone who already has bartleby should not have to
// install a second tool to keep their requirements honest — but reqtrace stays
// separately installable, since it is Apache-2.0 and useful without bartleby.
var reqsCmd = &cobra.Command{
	Use:   "reqs",
	Short: "Generate the requirements traceability matrix, or check it",
	Long: `Generate the requirements traceability matrix, or check it.

Reads the requirements in docs/requirements/ and the annotations on the tests
that verify them, writes docs/requirements/traceability.rst, and reports a
requirement no test covers, a test that declares nothing, a reference that does
not resolve, a duplicate ID, or a stale matrix.

Needs neither Docker nor Sphinx. The same tool installs on its own as
` + "`brew install neuronsphere/tap/reqtrace`" + `, which is the better choice for a
repository that does not otherwise use bartleby.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := reqtrace.RunOptions{
			Check:            reqsOpts.check,
			Repo:             reqsOpts.repo,
			Quiet:            reqsOpts.quiet,
			IncludeProposals: reqsOpts.includeProposals,
			Out:              cmd.OutOrStdout(),
			Err:              cmd.ErrOrStderr(),
		}
		if reqsOpts.verify {
			opts.Verify = &reqtrace.VerifyOptions{
				Needs: reqsOpts.needs, Results: reqsOpts.results, Runs: reqsOpts.runs, AllRuns: reqsOpts.allRuns, Out: reqsOpts.out, JSON: reqsOpts.json,
			}
		}
		return reqtrace.Run(opts)
	},
}

func init() {
	flags := reqsCmd.Flags()
	flags.BoolVar(&reqsOpts.check, "check", false,
		"report problems and stale output instead of writing; exit non-zero on either")
	flags.StringVar(&reqsOpts.repo, "repo", "",
		"repository root (default: found by walking up from the working directory)")
	flags.BoolVar(&reqsOpts.quiet, "quiet", false,
		"print nothing on success")
	flags.BoolVar(&reqsOpts.includeProposals, "include-proposals", false,
		"also scan docs/proposals (NERD documents) for req/spec items")
	flags.BoolVar(&reqsOpts.verify, "verify", false,
		"join needs.json with test results instead of generating the matrix; exit non-zero if a requirement fails or has no result")
	flags.StringVar(&reqsOpts.needs, "needs", "",
		"needs.json from a bartleby build, for --verify (default target/bartleby/html/needs.json)")
	flags.StringArrayVar(&reqsOpts.results, "results", nil,
		"for --verify: LABEL[@VERSION]=PATH of a Robot output.xml, go test -json output or Bender run folder; repeatable")
	flags.StringArrayVar(&reqsOpts.runs, "runs", nil,
		"for --verify: a directory of Bender run folders, such as bender/runs; uses each label's latest run; repeatable")
	flags.BoolVar(&reqsOpts.allRuns, "all-runs", false,
		"for --verify with --runs: use every run, each as its own column, not each label's latest")
	flags.StringVar(&reqsOpts.out, "out", "",
		"for --verify: page to write (default docs/requirements/verification_matrix.rst; - for none)")
	flags.BoolVar(&reqsOpts.json, "json", false,
		"for --verify: print the verification as JSON")
}
