package serve

import (
	"net"
	"strings"
	"testing"
)

// Requirements: REQ_SERVE_001
func TestParseReposIgnoresCommentsAndBlanks(t *testing.T) {
	in := "# my repos\n\nhmd-cli-bartleby\n  hmd-core-arch-docs  # NEPs\n\n"
	got, err := ParseRepos(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"hmd-cli-bartleby", "hmd-core-arch-docs"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Requirements: REQ_SERVE_001
func TestParseReposRejectsPathsAndDuplicates(t *testing.T) {
	for _, bad := range []string{"../etc", "a/b", "/abs", "..", "."} {
		if _, err := ParseRepos(strings.NewReader(bad + "\n")); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
	got, err := ParseRepos(strings.NewReader("a\nb\na\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("duplicates should collapse, got %v", got)
	}
}

// Requirements: REQ_SERVE_001
func TestLoadReposMissingFileIsEmpty(t *testing.T) {
	got, err := LoadRepos(t.TempDir() + "/nope.repos")
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want empty list and no error", got, err)
	}
}

// Requirements: REQ_SERVE_006
func TestCheckLoopback(t *testing.T) {
	for _, ok := range []string{"127.0.0.1:8099", "localhost:8099", "[::1]:8099", "127.0.0.1:0"} {
		if err := CheckLoopback(ok); err != nil {
			t.Errorf("%s: unexpected error %v", ok, err)
		}
	}
	for _, bad := range []string{"0.0.0.0:8099", ":8099", "192.168.1.5:8099", "example.com:80", "nonsense"} {
		if err := CheckLoopback(bad); err == nil {
			t.Errorf("%s: expected an error", bad)
		}
	}
	_ = net.IPv4zero
}
