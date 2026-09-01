package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
)

var updateImageCmd = &cobra.Command{
	Use:           "update-image",
	Short:         "Pull the latest Bartleby transform image",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		img := imageName(opts, os.Getenv)
		out := cmd.OutOrStdout()

		// The local image is removed first so that a moving tag such as :stable
		// is genuinely re-fetched rather than reported as up to date.
		fmt.Fprintf(out, "Removing local %s...\n", img)
		switch err := runner.RemoveImage(ctx, img); {
		case err == nil:
		case errors.Is(err, runner.ErrImageNotFound):
			fmt.Fprintln(out, "  not present locally — nothing to remove")
		default:
			return err
		}

		return runner.PullImage(ctx, img)
	},
}
