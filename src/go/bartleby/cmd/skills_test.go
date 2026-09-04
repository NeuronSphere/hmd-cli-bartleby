package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/neuronsphere/hmd-cli-bartleby/skills"
)

// runSkills drives one of the skills commands and returns what it printed.
func runSkills(t *testing.T, cmd *cobra.Command, args []string) string {
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

// Requirements: REQ_SKILL_002
func TestListShowsEverySkillWithItsDescription(t *testing.T) {
	output := runSkills(t, skillsListCmd, nil)

	for _, s := range skills.All() {
		if !strings.Contains(output, s.Name) {
			t.Errorf("listing omits %s", s.Name)
		}
		if !strings.Contains(output, s.Description) {
			t.Errorf("listing omits the description of %s", s.Name)
		}
	}
}

// Requirements: REQ_SKILL_002
func TestBareSkillsCommandLists(t *testing.T) {
	// `bartleby skills` with no subcommand should be useful rather than print
	// usage, since seeing what is on offer is the common case.
	bare := runSkills(t, skillsCmd, nil)
	listed := runSkills(t, skillsListCmd, nil)

	if bare != listed {
		t.Errorf("`skills` and `skills list` differ:\n%s\n---\n%s", bare, listed)
	}
}

// Requirements: REQ_SKILL_003
func TestShowPrintsTheSkillAndNothingElse(t *testing.T) {
	output := runSkills(t, skillsShowCmd, []string{"check-traceability"})

	skill, err := skills.Get("check-traceability")
	if err != nil {
		t.Fatal(err)
	}
	if output != skill.Content {
		t.Error("show printed something other than the skill verbatim")
	}
}

// Requirements: REQ_SKILL_009
func TestInstallReportsTheDestinationAndEachOutcome(t *testing.T) {
	dir := t.TempDir()

	restore := skillsOpts
	skillsOpts = skillsOptions{dir: dir}
	t.Cleanup(func() { skillsOpts = restore })

	output := runSkills(t, skillsInstallCmd, []string{"add-requirement"})

	if !strings.Contains(output, dir) {
		t.Errorf("output does not say where the skills went:\n%s", output)
	}
	if !strings.Contains(output, "add-requirement") || !strings.Contains(output, "installed") {
		t.Errorf("output does not report the outcome:\n%s", output)
	}

	// Second pass: the report has to distinguish "current" from "installed",
	// or a setup script's output is indistinguishable from a no-op.
	again := runSkills(t, skillsInstallCmd, []string{"add-requirement"})
	if !strings.Contains(again, "already current") {
		t.Errorf("re-install does not report the skill as current:\n%s", again)
	}
}

// Requirements: REQ_SKILL_004
func TestInstallDirDefaultsToTheUserSkillsDirectory(t *testing.T) {
	dir, err := installDir(skillsOptions{})
	if err != nil {
		t.Fatalf("installDir: %v", err)
	}
	if want := filepath.Join(".claude", "skills"); !strings.HasSuffix(dir, want) {
		t.Errorf("installDir() = %q, want it to end in %q", dir, want)
	}
}

// Requirements: REQ_SKILL_007, REQ_SKILL_007_SPEC001
func TestInstallDirResolvesTheDestinationFlags(t *testing.T) {
	explicit, err := installDir(skillsOptions{dir: "/tmp/somewhere"})
	if err != nil {
		t.Fatalf("installDir: %v", err)
	}
	if explicit != "/tmp/somewhere" {
		t.Errorf("--dir = %q", explicit)
	}

	project, err := installDir(skillsOptions{project: true})
	if err != nil {
		t.Fatalf("installDir: %v", err)
	}
	if want := filepath.Join(".claude", "skills"); !strings.HasSuffix(project, want) {
		t.Errorf("--project = %q, want it to end in %q", project, want)
	}

	// Both name a destination, so neither may quietly win.
	if _, err := installDir(skillsOptions{dir: "/tmp/x", project: true}); err == nil {
		t.Error("expected an error when --dir and --project are both given")
	}
}
