// Package serve implements `bartleby serve`: a loopback web server that shows,
// from one landing page, the HTML Bartleby already built for a chosen list of
// repositories. It reads what is on disk. The one thing it can start is a
// rebuild of a listed repository, on an explicit POST.
package serve

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// ReposFileRel is the default list location, relative to $HMD_HOME.
var ReposFileRel = filepath.Join(".config", "bartleby-serve.repos")

// DefaultReposFile returns the default list path, or "" when HMD_HOME is unset.
func DefaultReposFile() string {
	home := os.Getenv("HMD_HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(os.ExpandEnv(home), ReposFileRel)
}

// ParseRepos reads one repository directory name per line. '#' starts a
// comment, blank lines are ignored, and a repeated name is kept once. A name
// that is not a plain directory name is an error: the list is what decides which
// directories the server will ever open, so it must not be able to name others.
func ParseRepos(r io.Reader) ([]string, error) {
	var repos []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if !plainName(name) {
			return nil, fmt.Errorf("line %d: %q is not a plain repository directory name", n, name)
		}
		if !seen[name] {
			seen[name] = true
			repos = append(repos, name)
		}
	}
	return repos, sc.Err()
}

// LoadRepos reads the list at path. A missing file is an empty list, not an
// error, so a first run can say what to do instead of failing.
func LoadRepos(path string) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	repos, err := ParseRepos(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return repos, nil
}

func plainName(s string) bool {
	return s != "." && s != ".." && !strings.ContainsAny(s, `/\`+"\x00")
}

// CheckLoopback refuses a listen address that is not on the loopback
// interface, so a flag cannot expose the server to the network.
func CheckLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if !loopbackHost(host) {
		return fmt.Errorf("address %q is not a loopback address; bartleby serve only listens on 127.0.0.1, ::1 or localhost", addr)
	}
	return nil
}

func loopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
