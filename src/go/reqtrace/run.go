package reqtrace

import (
	"fmt"
	"io"
	"os"
)

// RunOptions drives one generate-or-check pass.
type RunOptions struct {
	// Check reports problems and staleness instead of writing, and fails on
	// either.
	Check bool
	// Repo is the repository root. Empty means walk up from the working
	// directory.
	Repo string
	// Quiet suppresses the success summary.
	Quiet bool
	// Out receives the summary, Err the problems. Nil means stdout and stderr.
	Out, Err io.Writer
}

// Run generates the traceability matrix, or checks it.
//
// This is the whole of what the command line does, so that the standalone
// `reqtrace` binary and `bartleby reqs` cannot drift apart: they are two
// front doors onto this function.
func Run(o RunOptions) error {
	out, errOut := o.Out, o.Err
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}

	repo := o.Repo
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repo, err = FindRepoRoot(cwd)
		if err != nil {
			return err
		}
	}

	layout := DefaultLayout(repo)

	model, err := Load(layout)
	if err != nil {
		return err
	}

	problems := Validate(model)
	for _, problem := range problems {
		fmt.Fprintln(errOut, problem)
	}

	if o.Check {
		staleErr := CheckFresh(layout, model)
		if staleErr != nil {
			fmt.Fprintln(errOut, staleErr)
		}
		if len(problems) > 0 || staleErr != nil {
			return fmt.Errorf("traceability check failed: %d problem(s)%s",
				len(problems), staleSuffix(staleErr))
		}
		if !o.Quiet {
			fmt.Fprintf(out, "traceability ok: %s\n", Summary(model))
		}
		return nil
	}

	if err := Write(layout, model); err != nil {
		return err
	}
	if !o.Quiet {
		fmt.Fprintf(out, "wrote %s (%s)\n", layout.GeneratedPath(), Summary(model))
	}

	// Writing still reports problems: regenerating a matrix with a gap in it is
	// not success, it just means the page now shows the gap.
	if len(problems) > 0 {
		return fmt.Errorf("%d traceability problem(s)", len(problems))
	}
	return nil
}

func staleSuffix(staleErr error) string {
	if staleErr == nil {
		return ""
	}
	return ", plus stale generated output"
}
