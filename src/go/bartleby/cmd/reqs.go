package cmd

import (
	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

type reqsOptions struct {
	check bool
	repo  string
	quiet bool
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
		return reqtrace.Run(reqtrace.RunOptions{
			Check: reqsOpts.check,
			Repo:  reqsOpts.repo,
			Quiet: reqsOpts.quiet,
			Out:   cmd.OutOrStdout(),
			Err:   cmd.ErrOrStderr(),
		})
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
}
