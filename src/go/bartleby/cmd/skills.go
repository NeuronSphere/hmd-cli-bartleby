package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/skills"
)

type skillsOptions struct {
	dir     string
	project bool
	force   bool
}

var skillsOpts skillsOptions

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Agent skills bundled with bartleby",
	Long: `Agent skills bundled with bartleby.

The skills are built into the binary, so they can be installed on a machine with
no network and no Python runtime. With no subcommand, they are listed.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSkills(cmd)
	},
}

var skillsListCmd = &cobra.Command{
	Use:           "list",
	Short:         "List the bundled skills",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSkills(cmd)
	},
}

var skillsShowCmd = &cobra.Command{
	Use:           "show <skill>",
	Short:         "Print a bundled skill",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		skill, err := skills.Get(args[0])
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), skill.Content)
		return nil
	},
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install [skill...]",
	Short: "Install bundled skills where an agent will find them",
	Long: `Install bundled skills where an agent will find them.

With no names, every bundled skill is installed. The default destination is
~/.claude/skills; --project installs into .claude/skills in the repository
instead, so the skills travel with the project.

A skill already installed with identical content is left alone. One that differs
is reported and left alone as well, since it may carry local edits; --force
overwrites it.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		selected, err := skills.Select(args)
		if err != nil {
			return err
		}

		dir, err := installDir(skillsOpts)
		if err != nil {
			return err
		}

		results, err := skills.Install(dir, selected, skillsOpts.force)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Installing into %s\n", dir)

		differs := 0
		for _, r := range results {
			fmt.Fprintf(out, "  %-22s %s\n", r.Skill.Name, r.Outcome)
			if r.Outcome == skills.Differs {
				differs++
			}
		}

		if differs > 0 {
			fmt.Fprintf(out, "\n%d skill(s) already there and different, so nothing was written to them.\n"+
				"Pass --force to overwrite, or diff them first if the changes might be yours.\n", differs)
		}
		return nil
	},
}

// installDir resolves where to install to, from the flags.
func installDir(o skillsOptions) (string, error) {
	if o.dir != "" && o.project {
		return "", fmt.Errorf("--dir and --project both name a destination; use one")
	}
	if o.dir != "" {
		return o.dir, nil
	}
	if o.project {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return skills.ProjectDir(cwd), nil
	}
	return skills.DefaultDir()
}

func listSkills(cmd *cobra.Command) error {
	all := skills.All()
	out := cmd.OutOrStdout()

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, s := range all {
		fmt.Fprintf(w, "%s\t%s\n", s.Name, s.Description)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	dir, err := skills.DefaultDir()
	if err == nil {
		fmt.Fprintf(out, "\n%d bundled skills. Install them with:\n  bartleby skills install\nDestination: %s\n",
			len(all), filepath.Clean(dir))
	}
	return nil
}

func init() {
	flags := skillsInstallCmd.Flags()
	flags.StringVar(&skillsOpts.dir, "dir", "",
		"directory to install into (default ~/.claude/skills)")
	flags.BoolVar(&skillsOpts.project, "project", false,
		"install into .claude/skills in the current repository")
	flags.BoolVar(&skillsOpts.force, "force", false,
		"overwrite a skill that is already installed and different")

	skillsCmd.AddCommand(skillsListCmd, skillsShowCmd, skillsInstallCmd)
}
