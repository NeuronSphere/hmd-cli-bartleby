package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set by Execute from the value GoReleaser injects into main. The
// zero value means someone built the binary directly with `go build`.
var version = "dev"

func setVersion(v string) {
	if v != "" {
		version = v
	}
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("bartleby {{.Version}}\n")
}

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Print the bartleby version",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "bartleby %s\n", version)
		return err
	},
}
