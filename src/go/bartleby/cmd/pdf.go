package cmd

import "github.com/spf13/cobra"

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Render PDF documentation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuilds("pdf")
	},
}
