package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// runBundle drives one of the bundle commands and returns what it printed.
func runBundle(t *testing.T, cmd *cobra.Command, args []string) string {
	t.Helper()

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	t.Cleanup(func() {
		cmd.SetOut(nil)
		cmd.SetErr(nil)
	})

	if err := cmd.RunE(cmd, args); err != nil {
		t.Fatalf("%s: %v", cmd.Name(), err)
	}
	return out.String()
}

// Requirements: REQ_SKILL_002, REQ_AGENT_002
func TestListShowsEveryItemWithItsDescription(t *testing.T) {
	for _, b := range []*bundleCmds{skillCmds, agentCmds} {
		output := runBundle(t, b.list, nil)

		for _, item := range b.set.All() {
			if !strings.Contains(output, item.Name) {
				t.Errorf("%s listing omits %s", b.set.Kind, item.Name)
			}
			if !strings.Contains(output, item.Description) {
				t.Errorf("%s listing omits the description of %s", b.set.Kind, item.Name)
			}
		}

		// The listing has to say how to install, or it is a dead end.
		if !strings.Contains(output, "bartleby "+b.set.Home+" install") {
			t.Errorf("%s listing does not say how to install:\n%s", b.set.Kind, output)
		}
	}
}

// Requirements: REQ_SKILL_002, REQ_AGENT_002
func TestBareCommandLists(t *testing.T) {
	// `bartleby skills` with no subcommand should be useful rather than print
	// usage, since seeing what is on offer is the common case.
	for _, b := range []*bundleCmds{skillCmds, agentCmds} {
		if bare, listed := runBundle(t, b.root, nil), runBundle(t, b.list, nil); bare != listed {
			t.Errorf("`%s` and `%s list` differ:\n%s\n---\n%s", b.set.Home, b.set.Home, bare, listed)
		}
	}
}

// Requirements: REQ_SKILL_003, REQ_AGENT_002
func TestShowPrintsTheItemAndNothingElse(t *testing.T) {
	cases := []struct {
		cmds *bundleCmds
		name string
	}{
		{skillCmds, "check-traceability"},
		{agentCmds, "rst-doc-expert"},
	}

	for _, c := range cases {
		item, err := c.cmds.set.Get(c.name)
		if err != nil {
			t.Fatal(err)
		}
		if output := runBundle(t, c.cmds.show, []string{c.name}); output != item.Content {
			t.Errorf("show %s printed something other than the %s verbatim", c.name, c.cmds.set.Kind)
		}
	}
}

// Requirements: REQ_SKILL_009
func TestInstallReportsTheDestinationAndEachOutcome(t *testing.T) {
	dir := t.TempDir()

	restore := skillCmds.opts
	skillCmds.opts = bundleOptions{dir: dir}
	t.Cleanup(func() { skillCmds.opts = restore })

	output := runBundle(t, skillCmds.instal, []string{"add-requirement"})

	if !strings.Contains(output, dir) {
		t.Errorf("output does not say where the skills went:\n%s", output)
	}
	if !strings.Contains(output, "add-requirement") || !strings.Contains(output, "installed") {
		t.Errorf("output does not report the outcome:\n%s", output)
	}

	// Second pass: the report has to distinguish "current" from "installed",
	// or a setup script's output is indistinguishable from a no-op.
	again := runBundle(t, skillCmds.instal, []string{"add-requirement"})
	if !strings.Contains(again, "already current") {
		t.Errorf("re-install does not report the skill as current:\n%s", again)
	}
}

// Requirements: REQ_SKILL_004, REQ_AGENT_003
func TestInstallDirDefaultsToTheUserLevelDirectory(t *testing.T) {
	for _, b := range []*bundleCmds{skillCmds, agentCmds} {
		restore := b.opts
		b.opts = bundleOptions{}
		dir, err := b.installDir()
		b.opts = restore

		if err != nil {
			t.Fatalf("installDir: %v", err)
		}
		if want := filepath.Join(".claude", b.set.Home); !strings.HasSuffix(dir, want) {
			t.Errorf("installDir() = %q, want it to end in %q", dir, want)
		}
	}
}

// Requirements: REQ_SKILL_007, REQ_SKILL_007_SPEC001, REQ_AGENT_006
func TestInstallDirResolvesTheDestinationFlags(t *testing.T) {
	for _, b := range []*bundleCmds{skillCmds, agentCmds} {
		restore := b.opts
		t.Cleanup(func() { b.opts = restore })

		b.opts = bundleOptions{dir: "/tmp/somewhere"}
		if dir, err := b.installDir(); err != nil || dir != "/tmp/somewhere" {
			t.Errorf("--dir = %q, %v", dir, err)
		}

		b.opts = bundleOptions{project: true}
		dir, err := b.installDir()
		if err != nil {
			t.Fatalf("--project: %v", err)
		}
		if want := filepath.Join(".claude", b.set.Home); !strings.HasSuffix(dir, want) {
			t.Errorf("--project = %q, want it to end in %q", dir, want)
		}

		// Both name a destination, so neither may quietly win.
		b.opts = bundleOptions{dir: "/tmp/x", project: true}
		if _, err := b.installDir(); err == nil {
			t.Errorf("%s: expected an error when --dir and --project are both given", b.set.Kind)
		}

		b.opts = restore
	}
}
