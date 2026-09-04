package sanitize

import "testing"

// Requirements: REQ_CFG_007
func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"clean title is untouched", "bartleby-docs-1.0", "bartleby-docs-1.0"},
		{"spaces become hyphens", "my doc title", "my-doc-title"},
		{"underscores are subscripts in LaTeX", "my_unsafe_repo_name", "my-unsafe-repo-name"},
		{"ampersand", "docs & diagrams", "docs-diagrams"},
		{"percent starts a LaTeX comment", "100% coverage", "100-coverage"},
		{"hash, dollar, caret, tilde, backslash", `a#b$c^d~e\f`, "a-b-c-d-e-f"},
		{"braces and quotes", `{"quoted"}`, "quoted"},
		{"runs collapse to one hyphen", "a   __  b", "a-b"},
		{"leading and trailing junk is trimmed", " _my-doc_ ", "my-doc"},
		{"parentheses", "release (final)", "release-final"},
		{"nothing usable", "___", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Title(tt.in); got != tt.want {
				t.Errorf("Title(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Requirements: REQ_EXEC_006
func TestContainerName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already valid", "hmd-cli-bartleby", "hmd-cli-bartleby"},
		{"underscores are legal in Docker names", "my_repo", "my_repo"},
		{"spaces", "My Docs", "My-Docs"},
		{"parentheses and dots", "My Docs (v2.1)", "My-Docs-v2.1"},
		{"leading period is rejected by Docker", ".hidden", "hidden"},
		{"leading underscore is rejected by Docker", "_private", "private"},
		{"slashes", "group/subgroup/repo", "group-subgroup-repo"},
		{"nothing usable", "///", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainerName(tt.in); got != tt.want {
				t.Errorf("ContainerName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
