package reqtrace

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// runList prints a listing of one repository or of a workspace.
//
// Which one: -workspace names a workspace; otherwise -repo, or the repository
// found by walking up from the working directory. Run from a directory that
// is in no repository but has repositories below it, it lists that directory
// as a workspace, so a session opened over a checkout of many repositories
// just works. In a workspace, -repo values select repositories; a single
// -repo that is itself a repository directory lists that repository.
func runList(o RunOptions, out, errOut io.Writer) error {
	lo := *o.List
	var (
		l   Listing
		err error
	)

	switch {
	case lo.Workspace != "":
		l, err = ListWorkspace(lo.Workspace, lo)
	case o.Repo != "":
		l, err = ListRepo(o.Repo, lo)
	case len(lo.Repos) == 1 && isRepoDir(lo.Repos[0]):
		dir := lo.Repos[0]
		lo.Repos = nil
		l, err = ListRepo(dir, lo)
	default:
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return cwdErr
		}
		if root, findErr := FindRepoRoot(cwd); findErr == nil {
			l, err = ListRepo(root, lo)
		} else if hasReposBelow(cwd) {
			l, err = ListWorkspace(cwd, lo)
		} else {
			return findErr
		}
	}
	if err != nil {
		return err
	}

	for _, w := range l.Warnings {
		fmt.Fprintf(errOut, "reqtrace: warning: %s\n", w)
	}
	return writeJSON(out, l)
}

// runDiff prints the changes between two listing files.
func runDiff(paths []string, out io.Writer) error {
	if len(paths) != 2 {
		return fmt.Errorf("-diff takes two listing files, OLD and NEW; got %d", len(paths))
	}
	var listings [2]Listing
	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &listings[i]); err != nil {
			return fmt.Errorf("%s is not a reqtrace listing: %w", p, err)
		}
	}
	changes, err := DiffListings(listings[0], listings[1])
	if err != nil {
		return err
	}
	return writeJSON(out, changes)
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
