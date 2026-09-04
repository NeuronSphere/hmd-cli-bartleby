package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Requirements: REQ_SKILL_001
func TestSkillsComeFromTheBinary(t *testing.T) {
	// Nothing here touches the filesystem: if the embed directive stopped
	// matching, All would come back empty and this fails.
	all := All()
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
	want := []string{"add-requirement", "check-traceability"}
	for _, name := range want {
		if _, err := Get(name); err != nil {
			t.Errorf("%s is not bundled: %v", name, err)
		}
	}
}

// Requirements: REQ_SKILL_002
func TestEverySkillHasADescriptionToListIt(t *testing.T) {
	for _, s := range All() {
		if strings.TrimSpace(s.Description) == "" {
			t.Errorf("%s has no description, so listing it says nothing", s.Name)
		}
	}
}

// Requirements: REQ_SKILL_002
func TestDescriptionIsReadFromFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"plain", "---\nname: x\ndescription: Does a thing\n---\n# X", "Does a thing"},
		{"quoted", "---\ndescription: \"Quoted thing\"\n---\n", "Quoted thing"},
		{"absent", "---\nname: x\n---\n", ""},
		{"no front matter", "# X\ndescription: not here\n", ""},
		{"only after the fence", "---\nname: x\n---\ndescription: too late\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := description(tt.content); got != tt.want {
				t.Errorf("description() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Requirements: REQ_SKILL_003
func TestGetReturnsTheSkillVerbatim(t *testing.T) {
	skill, err := Get("add-requirement")
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
	_, err := Get("add-requirements")
	if err == nil {
		t.Fatal("expected an error for a name that is not bundled")
	}
	// The likely cause is a typo, so the error has to carry the real names.
	for _, want := range []string{"add-requirement", "check-traceability"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not list %s: %v", want, err)
		}
	}
}

// Requirements: REQ_SKILL_005
func TestSelectTakesAllOrASubset(t *testing.T) {
	if got, want := len(mustSelect(t, nil)), len(All()); got != want {
		t.Errorf("Select(nil) returned %d skills, want all %d", got, want)
	}

	selected := mustSelect(t, []string{"check-traceability"})
	if len(selected) != 1 || selected[0].Name != "check-traceability" {
		t.Errorf("Select(one) = %v", names(selected))
	}

	if _, err := Select([]string{"check-traceability", "nope"}); err == nil {
		t.Error("expected an error when one of several names is unknown")
	}
}

// Requirements: REQ_SKILL_004, REQ_SKILL_009
func TestInstallWritesSkillMdUnderTheSkillsName(t *testing.T) {
	dir := t.TempDir()

	results, err := Install(dir, All(), false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != len(All()) {
		t.Fatalf("got %d results for %d skills", len(results), len(All()))
	}

	for _, r := range results {
		if r.Outcome != Written {
			t.Errorf("%s: outcome %v, want installed", r.Skill.Name, r.Outcome)
		}

		want := filepath.Join(dir, r.Skill.Name, File)
		if r.Path != want {
			t.Errorf("%s: path %s, want %s", r.Skill.Name, r.Path, want)
		}

		// The reported path is the point of the report: it has to be the file.
		onDisk, err := os.ReadFile(r.Path)
		if err != nil {
			t.Fatalf("%s: %v", r.Skill.Name, err)
		}
		if string(onDisk) != r.Skill.Content {
			t.Errorf("%s: written content differs from the bundled skill", r.Skill.Name)
		}
	}
}

// Requirements: REQ_SKILL_004
func TestDefaultDirIsTheUserLevelSkillsDirectory(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if want := filepath.Join(".claude", "skills"); !strings.HasSuffix(dir, want) {
		t.Errorf("DefaultDir() = %q, want it to end in %q", dir, want)
	}
}

// Requirements: REQ_SKILL_007
func TestProjectDirIsInsideTheRepository(t *testing.T) {
	if got, want := ProjectDir("/repo"), filepath.Join("/repo", ".claude", "skills"); got != want {
		t.Errorf("ProjectDir() = %q, want %q", got, want)
	}
}

// Requirements: REQ_SKILL_006
func TestInstallingTwiceChangesNothing(t *testing.T) {
	dir := t.TempDir()
	one := mustSelect(t, []string{"add-requirement"})

	if _, err := Install(dir, one, false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	results, err := Install(dir, one, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if results[0].Outcome != Unchanged {
		t.Errorf("outcome %v, want already current", results[0].Outcome)
	}
}

// Requirements: REQ_SKILL_006
func TestALocallyEditedSkillIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	one := mustSelect(t, []string{"add-requirement"})

	if _, err := Install(dir, one, false); err != nil {
		t.Fatalf("install: %v", err)
	}

	path := filepath.Join(dir, "add-requirement", File)
	edited := "---\nname: add-requirement\n---\n# Mine now\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(dir, one, false)
	if err != nil {
		t.Fatalf("install over an edit: %v", err)
	}
	if results[0].Outcome != Differs {
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

	results, err := Install(dir, one, true)
	if err != nil {
		t.Fatalf("forced install: %v", err)
	}
	if results[0].Outcome != Updated {
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

func mustSelect(t *testing.T, want []string) []Skill {
	t.Helper()
	selected, err := Select(want)
	if err != nil {
		t.Fatalf("Select(%v): %v", want, err)
	}
	return selected
}

func names(skills []Skill) []string {
	var out []string
	for _, s := range skills {
		out = append(out, s.Name)
	}
	return out
}
