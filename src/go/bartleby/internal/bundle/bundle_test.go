package bundle

import "testing"

// Requirements: REQ_SKILL_002, REQ_AGENT_002
func TestDescriptionIsReadFromFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"plain", "---\nname: x\ndescription: Does a thing\n---\n# X", "Does a thing"},
		{"quoted", "---\ndescription: \"Quoted thing\"\n---\n", "Quoted thing"},
		{"single quoted", "---\ndescription: 'Quoted thing'\n---\n", "Quoted thing"},
		{"absent", "---\nname: x\n---\n", ""},
		{"no front matter", "# X\ndescription: not here\n", ""},
		{"only after the fence", "---\nname: x\n---\ndescription: too late\n", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Description(tt.content); got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Requirements: REQ_SKILL_006, REQ_AGENT_005
func TestOutcomesAreDistinguishable(t *testing.T) {
	// The CLI reports these strings, and a user reading "installed" when the
	// file was left alone would be actively misled.
	seen := map[string]bool{}
	for _, o := range []Outcome{Written, Unchanged, Updated, Differs} {
		s := o.String()
		if s == "" || s == "unknown" {
			t.Errorf("outcome %d has no description", o)
		}
		if seen[s] {
			t.Errorf("two outcomes both report %q", s)
		}
		seen[s] = true
	}
}
