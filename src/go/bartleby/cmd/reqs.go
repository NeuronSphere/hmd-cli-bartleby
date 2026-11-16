package cmd

import (
	"fmt"

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
	// repos holds every --repo given; repo is the single root the tests set.
	repos     []string
	format    string
	workspace string
	stages    []string
	types     []string
	open      bool
	summary   bool
	changes   bool
	rebuild   bool
	diff      []string
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

With --format json it instead lists every need, in this repository or, run
over a directory of repositories, in all of them, with its stage and the tests
that cover it. --diff OLD.json NEW.json compares two such listings.

Needs neither Docker nor Sphinx. The same tool installs on its own as
` + "`brew install neuronsphere/tap/reqtrace`" + `, which is the better choice for a
repository that does not otherwise use bartleby.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos := reqsOpts.repos
		if reqsOpts.repo != "" {
			repos = append([]string{reqsOpts.repo}, repos...)
		}
		opts := reqtrace.RunOptions{
			Check:            reqsOpts.check,
			Quiet:            reqsOpts.quiet,
			IncludeProposals: reqsOpts.includeProposals,
			Out:              cmd.OutOrStdout(),
			Err:              cmd.ErrOrStderr(),
		}
		switch reqsOpts.format {
		case "":
			if len(repos) > 1 {
				return fmt.Errorf("--repo names one repository root unless --format json is given")
			}
			if len(repos) == 1 {
				opts.Repo = repos[0]
			}
		case "json":
			opts.List = &reqtrace.ListOptions{
				Repos: repos, Workspace: reqsOpts.workspace, Stages: reqsOpts.stages, Types: reqsOpts.types,
				Open: reqsOpts.open, Summary: reqsOpts.summary, Changes: reqsOpts.changes, Rebuild: reqsOpts.rebuild,
			}
		default:
			return fmt.Errorf("--format %q is not supported; use json", reqsOpts.format)
		}
		if len(reqsOpts.diff) > 0 {
			if len(reqsOpts.diff) != 2 {
				return fmt.Errorf("--diff takes two listing files: --diff OLD.json --diff NEW.json")
			}
			opts.Diff = reqsOpts.diff
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
	flags.StringArrayVar(&reqsOpts.repos, "repo", nil,
		"repository root (default: found by walking up from the working directory); with --format json in a workspace, a repository to keep, by name or folder; repeatable")
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
	flags.StringVar(&reqsOpts.format, "format", "",
		`"json" lists every need with its stage and covering tests instead of generating the matrix`)
	flags.StringVar(&reqsOpts.workspace, "workspace", "",
		"for --format json: list every repository under this directory")
	flags.StringArrayVar(&reqsOpts.stages, "stage", nil,
		"for --format json: keep needs in this stage; repeatable")
	flags.StringArrayVar(&reqsOpts.types, "type", nil,
		"for --format json: keep needs of this directive type; repeatable")
	flags.BoolVar(&reqsOpts.open, "open", false,
		"for --format json: keep needs not implemented or closed, and implemented needs no test covers")
	flags.BoolVar(&reqsOpts.summary, "summary", false,
		"for --format json: print per-repository counts instead of the needs")
	flags.BoolVar(&reqsOpts.changes, "changes", false,
		"for --format json in a workspace: add what changed since the previous call")
	flags.BoolVar(&reqsOpts.rebuild, "rebuild", false,
		"for --format json in a workspace: ignore the index and build it again")
	flags.StringArrayVar(&reqsOpts.diff, "diff", nil,
		"print the changes between two listings: --diff OLD.json --diff NEW.json")
}
