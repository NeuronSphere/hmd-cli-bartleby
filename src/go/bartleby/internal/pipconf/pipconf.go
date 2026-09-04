// Package pipconf resolves the pip configuration that autodoc builds need in
// order to install the documented package from a private index.
//
// Two paths, matching the Python CLI: when PIP_USERNAME and PIP_PASSWORD are
// set, a pip.conf is written to a private temporary file and mounted into the
// container; otherwise the user's existing pip config is mounted if it exists.
package pipconf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultIndexPath is the private index the generated pip.conf points at. It can
// be overridden with HMD_PIP_EXTRA_INDEX_HOST for a different Artifactory host
// or repo path.
const DefaultIndexPath = "hmdlabs.jfrog.io/artifactory/api/pypi/hmd_pypi/simple"

// Resolved describes the pip config to use for a build.
type Resolved struct {
	// Path is the host path to mount at /run/secrets/pip_url, or "" when there
	// is no pip config to provide.
	Path string
	// Generated reports whether Path is a temporary file this package created
	// and therefore owns.
	Generated bool
}

// Resolve returns the pip config for an autodoc build plus a cleanup function
// that the caller must always call — it removes the generated credential file.
//
// When credentials are present a pip.conf is generated. When they are not, the
// user's own pip config is used if present. When neither exists the result is
// empty and the build proceeds without a private index, which is correct for a
// package whose dependencies are all public.
func Resolve() (Resolved, func(), error) {
	noop := func() {}

	username := os.Getenv("PIP_USERNAME")
	password := os.Getenv("PIP_PASSWORD")

	if username != "" && password != "" {
		resolved, cleanup, err := generate(username, password)
		if err != nil {
			return Resolved{}, noop, err
		}
		return resolved, cleanup, nil
	}

	path := UserConfigPath()
	if path == "" {
		return Resolved{}, noop, nil
	}
	if _, err := os.Stat(path); err != nil {
		return Resolved{}, noop, nil
	}
	return Resolved{Path: path}, noop, nil
}

// generate writes a pip.conf containing the credentialed extra-index-url into a
// temporary file readable only by the current user.
func generate(username, password string) (Resolved, func(), error) {
	indexPath := os.Getenv("HMD_PIP_EXTRA_INDEX_HOST")
	if indexPath == "" {
		indexPath = DefaultIndexPath
	}

	content := fmt.Sprintf("[global]\nextra-index-url = https://%s:%s@%s\n",
		url.QueryEscape(username), url.QueryEscape(password), strings.TrimPrefix(indexPath, "https://"))

	dir, err := os.MkdirTemp("", "bartleby-pip-")
	if err != nil {
		return Resolved{}, func() {}, fmt.Errorf("creating temporary directory for pip.conf: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not remove temporary pip config %s: %v\n", dir, err)
		}
	}

	path := filepath.Join(dir, "pip.conf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		cleanup()
		return Resolved{}, func() {}, fmt.Errorf("writing %s: %w", path, err)
	}

	return Resolved{Path: path, Generated: true}, cleanup, nil
}

// UserConfigPath returns the conventional per-user pip config location, or ""
// when the home directory cannot be determined.
func UserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "pip", "pip.ini")
	}
	return filepath.Join(home, ".pip", "pip.conf")
}
