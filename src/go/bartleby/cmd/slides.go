package cmd

import "github.com/spf13/cobra"

var slidesCmd = &cobra.Command{
	Use:   "slides",
	Short: "Render RevealJS slideshow",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuilds("revealjs")
	},
}
