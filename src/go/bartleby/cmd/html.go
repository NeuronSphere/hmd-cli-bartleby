package cmd

import "github.com/spf13/cobra"

var htmlCmd = &cobra.Command{
	Use:   "html",
	Short: "Render HTML documentation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuilds("html")
	},
}
