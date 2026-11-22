package serve

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// GitEmail returns the configured git user.email.
func GitEmail() (string, error) {
	out, err := exec.Command("git", "config", "--get", "user.email").Output()
	if err != nil {
		return "", fmt.Errorf("no git user.email is configured: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Suggest names the repositories under workspace, one level down, that have a
// commit by email within the last days days and are not already in listed. It
// only reports: the list is the user's to edit.
func Suggest(workspace string, listed []string, email string, days int) ([]string, error) {
	if email == "" {
		return nil, fmt.Errorf("an author email is required to suggest repositories")
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, n := range listed {
		have[n] = true
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || have[name] || strings.HasPrefix(name, ".") {
			continue
		}
		dir := workspace + string(os.PathSeparator) + name
		if _, err := os.Stat(dir + string(os.PathSeparator) + ".git"); err != nil {
			continue
		}
		// --fixed-strings: the address is data, not a pattern. Anchoring on the
		// angle brackets keeps "me@x.com" from matching "notme@x.com".
		log, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%H",
			fmt.Sprintf("--since=%d days ago", days), "--fixed-strings", "--author=<"+email+">").Output()
		if err == nil && len(strings.TrimSpace(string(log))) > 0 {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}
