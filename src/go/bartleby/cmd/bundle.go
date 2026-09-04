package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/agents"
	"github.com/neuronsphere/hmd-cli-bartleby/internal/bundle"
	"github.com/neuronsphere/hmd-cli-bartleby/skills"
)

// bundleOptions are the flags for installing a bundled set.
type bundleOptions struct {
	dir     string
	project bool
	force   bool
}

// bundleCmds is the command group for one bundled set. Skills and agents get
// the same surface, because they are the same job with a different destination.
type bundleCmds struct {
	set                      bundle.Set
	opts                     bundleOptions
	root, list, show, instal *cobra.Command
}

var (
	skillCmds = newBundleCmds(skills.Set, "Agent skills bundled with bartleby")
	agentCmds = newBundleCmds(agents.Set, "Agent definitions bundled with bartleby")
)

// newBundleCmds builds `bartleby <home>` with list, show, and install under it.
func newBundleCmds(set bundle.Set, short string) *bundleCmds {
	b := &bundleCmds{set: set}
	name := set.Home

	b.root = &cobra.Command{
		Use:   name,
		Short: short,
		Long: fmt.Sprintf(`%s.

They are built into the binary, so installing them needs no network access and
no repository checkout. With no subcommand, they are listed.`, short),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, args []string) error { return b.listItems(cmd) },
	}

	b.list = &cobra.Command{
		Use:           "list",
		Short:         fmt.Sprintf("List the bundled %ss", set.Kind),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, args []string) error { return b.listItems(cmd) },
	}

	b.show = &cobra.Command{
		Use:           fmt.Sprintf("show <%s>", set.Kind),
		Short:         fmt.Sprintf("Print a bundled %s", set.Kind),
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			item, err := b.set.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), item.Content)
			return nil
		},
	}

	b.instal = &cobra.Command{
		Use:   fmt.Sprintf("install [%s...]", set.Kind),
		Short: fmt.Sprintf("Install bundled %ss where an agent will find them", set.Kind),
		Long: fmt.Sprintf(`Install bundled %ss where an agent will find them.

With no names, every bundled %s is installed. The default destination is
~/.claude/%s; --project installs into .claude/%s in the current directory
instead, so they travel with the repository.

One already installed with identical content is left alone. One that differs is
reported and left alone as well, since it may carry local edits; --force
overwrites it.`, set.Kind, set.Kind, set.Home, set.Home),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          b.runInstall,
	}

	flags := b.instal.Flags()
	flags.StringVar(&b.opts.dir, "dir", "",
		fmt.Sprintf("directory to install into (default ~/.claude/%s)", set.Home))
	flags.BoolVar(&b.opts.project, "project", false,
		fmt.Sprintf("install into .claude/%s in the current directory", set.Home))
	flags.BoolVar(&b.opts.force, "force", false,
		fmt.Sprintf("overwrite a %s that is already installed and different", set.Kind))

	b.root.AddCommand(b.list, b.show, b.instal)
	return b
}

func (b *bundleCmds) runInstall(cmd *cobra.Command, args []string) error {
	selected, err := b.set.Select(args)
	if err != nil {
		return err
	}

	dir, err := b.installDir()
	if err != nil {
		return err
	}

	results, err := b.set.Install(dir, selected, b.opts.force)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Installing into %s\n", dir)

	differs := 0
	for _, r := range results {
		fmt.Fprintf(out, "  %-22s %s\n", r.Item.Name, r.Outcome)
		if r.Outcome == bundle.Differs {
			differs++
		}
	}

	if differs > 0 {
		fmt.Fprintf(out, "\n%s already there and different, so nothing was written.\n"+
			"Pass --force to overwrite, or diff them first if the changes might be yours.\n",
			plural(differs, b.set.Kind))
	}
	return nil
}

// installDir resolves where to install to, from the flags.
func (b *bundleCmds) installDir() (string, error) {
	if b.opts.dir != "" && b.opts.project {
		return "", fmt.Errorf("--dir and --project both name a destination; use one")
	}
	if b.opts.dir != "" {
		return b.opts.dir, nil
	}
	if b.opts.project {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return b.set.ProjectDir(cwd), nil
	}
	return b.set.DefaultDir()
}

func (b *bundleCmds) listItems(cmd *cobra.Command) error {
	all := b.set.All()
	out := cmd.OutOrStdout()

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, item := range all {
		fmt.Fprintf(w, "%s\t%s\n", item.Name, item.Description)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	dir, err := b.set.DefaultDir()
	if err != nil {
		return nil
	}
	fmt.Fprintf(out, "\n%s bundled. Install with:\n  bartleby %s install\nDestination: %s\n",
		plural(len(all), b.set.Kind), b.set.Home, dir)
	return nil
}

// plural counts a thing without the "1 items" wart.
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
