package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/hmdenv"
)

// DefaultLogoURL is the logo offered when configuring a fresh environment.
const DefaultLogoURL = "https://neuronsphere.io/hubfs/bartleby_assets/NeuronSphereSwoosh.jpg"

// setting is one prompt in the configure flow.
type setting struct {
	key         string
	prompt      string
	defaultVal  string
	description string
}

var settings = []setting{
	{
		key:         "HMD_BARTLEBY_DEFAULT_LOGO",
		prompt:      "Default logo URL",
		defaultVal:  DefaultLogoURL,
		description: "Used for the HTML logo and the PDF cover unless a build overrides it.",
	},
	{
		key:         "HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT",
		prompt:      "Confidentiality statement",
		description: "Stamped on documents built with --confidential. Leave blank for none.",
	},
	{
		key:         "HMD_CONTAINER_REGISTRY",
		prompt:      "Container registry",
		defaultVal:  defaultRegistry,
		description: "Where the hmd-tf-bartleby transform image is pulled from.",
	},
}

var configureCmd = &cobra.Command{
	Use:           "configure",
	Short:         "Set Bartleby defaults in $HMD_HOME/.config/hmd.env",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := hmdenv.Path()
		if path == "" {
			return fmt.Errorf("HMD_HOME is not set; export it to the directory that should hold %s", hmdenv.RelPath)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Writing to %s\n", path)
		fmt.Fprintln(out, "Press Enter to keep the value shown in brackets.")

		reader := bufio.NewReader(cmd.InOrStdin())
		written := 0

		for _, s := range settings {
			current := firstNonEmpty(os.Getenv(s.key), s.defaultVal)

			fmt.Fprintf(out, "\n%s\n  %s\n", s.prompt, s.description)
			if current != "" {
				fmt.Fprintf(out, "  [%s]: ", current)
			} else {
				fmt.Fprint(out, "  []: ")
			}

			answer, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return fmt.Errorf("reading input: %w", err)
			}
			answer = strings.TrimSpace(answer)

			value := answer
			if value == "" {
				value = current
			}
			if value == "" {
				continue
			}

			if err := hmdenv.Set(s.key, value); err != nil {
				return err
			}
			written++

			if err == io.EOF {
				break
			}
		}

		fmt.Fprintf(out, "\nSaved %d setting(s) to %s\n", written, path)
		return nil
	},
}
