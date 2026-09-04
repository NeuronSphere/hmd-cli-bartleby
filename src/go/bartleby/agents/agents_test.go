package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Requirements: REQ_AGENT_001
func TestAgentsComeFromTheBinary(t *testing.T) {
	all := Set.All()
	if len(all) == 0 {
		t.Fatal("no agents are embedded; check the go:embed pattern in agents.go")
	}

	for _, a := range all {
		if !strings.HasPrefix(a.Content, "---\n") {
			t.Errorf("%s: content does not start with front matter", a.Name)
		}
	}
}

// Requirements: REQ_AGENT_002
func TestEveryAgentHasADescriptionToListIt(t *testing.T) {
	for _, a := range Set.All() {
		if strings.TrimSpace(a.Description) == "" {
			t.Errorf("%s has no description, so listing it says nothing", a.Name)
		}
	}
}

// Requirements: REQ_AGENT_002
func TestGetReturnsTheAgentVerbatim(t *testing.T) {
	agent, err := Set.Get("rst-doc-expert")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.HasPrefix(agent.Content, "---\nname: rst-doc-expert\n") {
		t.Error("content is not the file as written")
	}
}

// Requirements: REQ_AGENT_003
func TestAnAgentInstallsAsOneFileNotADirectory(t *testing.T) {
	dir := t.TempDir()

	results, err := Set.Install(dir, Set.All(), false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, r := range results {
		// The destination shape is the whole difference from a skill: an agent
		// runtime reads <name>.md out of its agents directory, so the bundled
		// <name>/AGENT.md layout must not survive the install.
		want := filepath.Join(dir, r.Item.Name+".md")
		if r.Path != want {
			t.Errorf("%s: path %s, want %s", r.Item.Name, r.Path, want)
		}

		info, err := os.Stat(r.Path)
		if err != nil {
			t.Fatalf("%s: %v", r.Item.Name, err)
		}
		if info.IsDir() {
			t.Errorf("%s installed as a directory", r.Item.Name)
		}

		if _, err := os.Stat(filepath.Join(dir, r.Item.Name, File)); err == nil {
			t.Errorf("%s: the bundled AGENT.md layout was reproduced at the destination", r.Item.Name)
		}

		onDisk, err := os.ReadFile(r.Path)
		if err != nil {
			t.Fatal(err)
		}
		if string(onDisk) != r.Item.Content {
			t.Errorf("%s: written content differs from the bundled agent", r.Item.Name)
		}
	}
}

// Requirements: REQ_AGENT_003
func TestDefaultDirIsTheUserLevelAgentsDirectory(t *testing.T) {
	dir, err := Set.DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if want := filepath.Join(".claude", "agents"); !strings.HasSuffix(dir, want) {
		t.Errorf("DefaultDir() = %q, want it to end in %q", dir, want)
	}
}

// Requirements: REQ_AGENT_004
func TestSelectTakesAllOrASubsetAndRejectsUnknown(t *testing.T) {
	all, err := Set.Select(nil)
	if err != nil {
		t.Fatalf("Select(nil): %v", err)
	}
	if len(all) != len(Set.All()) {
		t.Errorf("Select(nil) returned %d agents, want all %d", len(all), len(Set.All()))
	}

	one, err := Set.Select([]string{"rst-doc-expert"})
	if err != nil {
		t.Fatalf("Select(one): %v", err)
	}
	if len(one) != 1 {
		t.Errorf("Select(one) returned %d agents", len(one))
	}

	_, err = Set.Get("rst-doc-experts")
	if err == nil {
		t.Fatal("expected an error for a name that is not bundled")
	}
	for _, want := range []string{"agent", "rst-doc-expert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

// Requirements: REQ_AGENT_005
func TestALocallyEditedAgentIsLeftAloneUnlessForced(t *testing.T) {
	dir := t.TempDir()
	all := Set.All()

	if _, err := Set.Install(dir, all, false); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Installing again over an untouched copy changes nothing.
	results, err := Set.Install(dir, all, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if results[0].Outcome.String() != "already current" {
		t.Errorf("outcome %v, want already current", results[0].Outcome)
	}

	path := filepath.Join(dir, all[0].Name+".md")
	edited := "---\nname: mine\n---\n# Mine now\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err = Set.Install(dir, all, false)
	if err != nil {
		t.Fatalf("install over an edit: %v", err)
	}
	if results[0].Outcome.String() != "left alone (differs)" {
		t.Errorf("outcome %v, want left alone", results[0].Outcome)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != edited {
		t.Error("the local edit was overwritten without --force")
	}

	results, err = Set.Install(dir, all, true)
	if err != nil {
		t.Fatalf("forced install: %v", err)
	}
	if results[0].Outcome.String() != "updated" {
		t.Errorf("outcome %v, want updated", results[0].Outcome)
	}
	after, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != all[0].Content {
		t.Error("--force did not replace the file with the bundled agent")
	}
}

// Requirements: REQ_AGENT_006
func TestProjectDirIsInsideTheRepository(t *testing.T) {
	if got, want := Set.ProjectDir("/repo"), filepath.Join("/repo", ".claude", "agents"); got != want {
		t.Errorf("ProjectDir() = %q, want %q", got, want)
	}
}
