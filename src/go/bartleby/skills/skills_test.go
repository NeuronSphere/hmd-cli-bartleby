package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neuronsphere/hmd-cli-bartleby/internal/bundle"
)

// Requirements: REQ_SKILL_001
func TestSkillsComeFromTheBinary(t *testing.T) {
	// Nothing here touches the filesystem: if the embed directive stopped
	// matching, All would come back empty and this fails.
	all := Set.All()
	if len(all) == 0 {
		t.Fatal("no skills are embedded; check the go:embed pattern in skills.go")
	}

	for _, s := range all {
		if !strings.HasPrefix(s.Content, "---\n") {
			t.Errorf("%s: content does not start with front matter", s.Name)
		}
		if len(s.Content) < 200 {
			t.Errorf("%s: content is only %d bytes, which is not a skill", s.Name, len(s.Content))
		}
	}
}

// Requirements: REQ_SKILL_001
func TestTheRequirementsSkillsAreBundled(t *testing.T) {
	// Named explicitly: these two are the reason the command exists, and a
	// rename that silently dropped them from the bundle should fail here.
	for _, name := range []string{"add-requirement", "check-traceability"} {
		if _, err := Set.Get(name); err != nil {
			t.Errorf("%s is not bundled: %v", name, err)
		}
	}
}

// Requirements: REQ_SKILL_002
func TestEverySkillHasADescriptionToListIt(t *testing.T) {
	for _, s := range Set.All() {
		if strings.TrimSpace(s.Description) == "" {
			t.Errorf("%s has no description, so listing it says nothing", s.Name)
		}
	}
}

// Requirements: REQ_SKILL_003
func TestGetReturnsTheSkillVerbatim(t *testing.T) {
	skill, err := Set.Get("add-requirement")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if skill.Name != "add-requirement" {
		t.Errorf("Name = %q", skill.Name)
	}
	// Verbatim means the front matter is still there, not stripped for display.
	if !strings.HasPrefix(skill.Content, "---\nname: add-requirement\n") {
		t.Error("content is not the file as written")
	}
}

// Requirements: REQ_SKILL_008
func TestUnknownSkillNamesWhatIsAvailable(t *testing.T) {
	_, err := Set.Get("add-requirements")
	if err == nil {
		t.Fatal("expected an error for a name that is not bundled")
	}
	// The likely cause is a typo, so the error has to carry the real names.
	for _, want := range []string{"skill", "add-requirement", "check-traceability"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s: %v", want, err)
		}
	}
}

// Requirements: REQ_SKILL_005
func TestSelectTakesAllOrASubset(t *testing.T) {
	if got, want := len(mustSelect(t, nil)), len(Set.All()); got != want {
		t.Errorf("Select(nil) returned %d skills, want all %d", got, want)
	}

	selected := mustSelect(t, []string{"check-traceability"})
	if len(selected) != 1 || selected[0].Name != "check-traceability" {
		t.Errorf("Select(one) returned %d skills", len(selected))
	}

	if _, err := Set.Select([]string{"check-traceability", "nope"}); err == nil {
		t.Error("expected an error when one of several names is unknown")
	}
}

// Requirements: REQ_SKILL_004, REQ_SKILL_009
func TestInstallWritesSkillMdUnderTheSkillsName(t *testing.T) {
	dir := t.TempDir()

	results, err := Set.Install(dir, Set.All(), false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != len(Set.All()) {
		t.Fatalf("got %d results for %d skills", len(results), len(Set.All()))
	}

	for _, r := range results {
		if r.Outcome.String() != "installed" {
			t.Errorf("%s: outcome %v, want installed", r.Item.Name, r.Outcome)
		}

		// A skill installs as <name>/SKILL.md, which is what an agent reads.
		want := filepath.Join(dir, r.Item.Name, File)
		if r.Path != want {
			t.Errorf("%s: path %s, want %s", r.Item.Name, r.Path, want)
		}

		onDisk, err := os.ReadFile(r.Path)
		if err != nil {
			t.Fatalf("%s: %v", r.Item.Name, err)
		}
		if string(onDisk) != r.Item.Content {
			t.Errorf("%s: written content differs from the bundled skill", r.Item.Name)
		}
	}
}

// Requirements: REQ_SKILL_004
func TestDefaultDirIsTheUserLevelSkillsDirectory(t *testing.T) {
	dir, err := Set.DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if want := filepath.Join(".claude", "skills"); !strings.HasSuffix(dir, want) {
		t.Errorf("DefaultDir() = %q, want it to end in %q", dir, want)
	}
}

// Requirements: REQ_SKILL_007
func TestProjectDirIsInsideTheRepository(t *testing.T) {
	if got, want := Set.ProjectDir("/repo"), filepath.Join("/repo", ".claude", "skills"); got != want {
		t.Errorf("ProjectDir() = %q, want %q", got, want)
	}
}

// Requirements: REQ_SKILL_006
func TestInstallingTwiceChangesNothing(t *testing.T) {
	dir := t.TempDir()
	one := mustSelect(t, []string{"add-requirement"})

	if _, err := Set.Install(dir, one, false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	results, err := Set.Install(dir, one, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if results[0].Outcome.String() != "already current" {
		t.Errorf("outcome %v, want already current", results[0].Outcome)
	}
}

// Requirements: REQ_SKILL_006
func TestALocallyEditedSkillIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	one := mustSelect(t, []string{"add-requirement"})

	if _, err := Set.Install(dir, one, false); err != nil {
		t.Fatalf("install: %v", err)
	}

	path := filepath.Join(dir, "add-requirement", File)
	edited := "---\nname: add-requirement\n---\n# Mine now\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Set.Install(dir, one, false)
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
}

// Requirements: REQ_SKILL_006
func TestForceOverwritesADifferingSkill(t *testing.T) {
	dir := t.TempDir()
	one := mustSelect(t, []string{"add-requirement"})
	path := filepath.Join(dir, "add-requirement", File)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Set.Install(dir, one, true)
	if err != nil {
		t.Fatalf("forced install: %v", err)
	}
	if results[0].Outcome.String() != "updated" {
		t.Errorf("outcome %v, want updated", results[0].Outcome)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != one[0].Content {
		t.Error("--force did not replace the file with the bundled skill")
	}
}

func mustSelect(t *testing.T, want []string) []bundle.Item {
	t.Helper()
	selected, err := Set.Select(want)
	if err != nil {
		t.Fatalf("Select(%v): %v", want, err)
	}
	return selected
}
