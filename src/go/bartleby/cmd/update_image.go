package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/runner"
)

var updateImageCmd = &cobra.Command{
	Use:   "update-image",
	Short: "Pull the latest Bartleby Docker image",
	RunE: func(cmd *cobra.Command, args []string) error {
		img := imageName()

		fmt.Printf("Removing old image %s...\n", img)
		if err := runner.RemoveImage(img); err != nil {
			// Non-fatal: image may not exist locally yet.
			fmt.Printf("warning: could not remove image: %v\n", err)
		}

		return runner.PullImage(img)
	},
}
