// Package hmdenv loads the NeuronSphere environment file that the legacy Python
// CLI read through hmd_cli_tools.load_hmd_env: $HMD_HOME/.config/hmd.env.
//
// Loading is best effort by design. A missing HMD_HOME or a missing file is not
// an error — plenty of repos build fine without one — but a file that exists and
// cannot be read or parsed is worth a warning, because the user almost certainly
// expected its values to apply.
//
// Values already present in the process environment always win, matching the
// Python call site (load_hmd_env(override=False)): a variable exported in the
// shell beats the same variable in hmd.env.
package hmdenv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RelPath is the location of the env file relative to $HMD_HOME.
var RelPath = filepath.Join(".config", "hmd.env")

// Path returns the hmd.env path, or "" when HMD_HOME is unset.
func Path() string {
	home := os.Getenv("HMD_HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(os.ExpandEnv(home), RelPath)
}

// Load reads $HMD_HOME/.config/hmd.env and sets any variable not already
// present in the process environment.
//
// It returns the number of variables applied. A missing HMD_HOME or missing file
// yields (0, nil). A file that exists but cannot be read or contains malformed
// lines yields a non-nil error; the caller is expected to warn and continue.
func Load() (int, error) {
	path := Path()
	if path == "" {
		return 0, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}
	defer f.Close()

	vars, err := Parse(f)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", path, err)
	}

	applied := 0
	for k, v := range vars {
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return applied, fmt.Errorf("setting %s from %s: %w", k, path, err)
		}
		applied++
	}
	return applied, nil
}

// LoadAndWarn calls Load and writes a warning to w on failure. It never returns
// an error: HMD_HOME config is an enhancement, not a prerequisite.
func LoadAndWarn(w io.Writer) {
	if _, err := Load(); err != nil {
		fmt.Fprintf(w, "warning: could not load HMD environment: %v\n", err)
	}
}

// Set writes key=value into $HMD_HOME/.config/hmd.env, replacing an existing
// entry for that key and preserving everything else in the file, comments
// included. It is the write half of the config that Load reads.
func Set(key, value string) error {
	path := Path()
	if path == "" {
		return fmt.Errorf("HMD_HOME is not set, so there is no %s to write", RelPath)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	entry := fmt.Sprintf("%s=%s", key, quote(value))
	lines := strings.Split(strings.TrimRight(string(existing), "\n"), "\n")

	replaced := false
	for i, line := range lines {
		candidate := strings.TrimPrefix(strings.TrimSpace(line), "export ")
		name, _, found := strings.Cut(candidate, "=")
		if found && strings.TrimSpace(name) == key {
			lines[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		if len(lines) == 1 && lines[0] == "" {
			lines = []string{entry}
		} else {
			lines = append(lines, entry)
		}
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// quote wraps a value in double quotes when it contains characters that would
// not survive a round trip through Parse.
func quote(v string) string {
	if v == "" || strings.ContainsAny(v, " \t\"'#\\") {
		escaped := strings.ReplaceAll(v, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return v
}

// Parse reads dotenv-style content: KEY=VALUE, one per line, with optional
// `export ` prefix, # comments, blank lines, and single- or double-quoted
// values. Escape sequences inside double quotes (\n, \t, \\, \") are expanded,
// single-quoted values are literal.
func Parse(r io.Reader) (map[string]string, error) {
	vars := make(map[string]string)
	scanner := bufio.NewScanner(r)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, rawValue, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE, got %q", lineNo, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo)
		}

		vars[key] = unquote(strings.TrimSpace(rawValue))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

// unquote strips matching surrounding quotes and, for double quotes, expands the
// common escape sequences. Unquoted values have trailing inline comments removed.
func unquote(v string) string {
	if len(v) >= 2 {
		switch {
		case v[0] == '\'' && v[len(v)-1] == '\'':
			return v[1 : len(v)-1]
		case v[0] == '"' && v[len(v)-1] == '"':
			inner := v[1 : len(v)-1]
			inner = strings.ReplaceAll(inner, `\n`, "\n")
			inner = strings.ReplaceAll(inner, `\t`, "\t")
			inner = strings.ReplaceAll(inner, `\"`, `"`)
			inner = strings.ReplaceAll(inner, `\\`, `\`)
			return inner
		}
	}
	// An unquoted value ends at the first " #" — dotenv convention for inline comments.
	if idx := strings.Index(v, " #"); idx >= 0 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}
