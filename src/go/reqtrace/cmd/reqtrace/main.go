// Command reqtrace generates and checks the requirements traceability matrix.
//
//	go run ./tools/reqtrace            # regenerate docs/requirements/traceability.rst
//	go run ./tools/reqtrace -check     # fail if it is stale or coverage has a gap
//
// The Makefile wraps both as "make reqs" and "make reqs-check", and "make check"
// runs the latter, so traceability breaks the build without needing Docker or
// Sphinx.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

func main() {
	check := flag.Bool("check", false, "report problems and stale output instead of writing; exit non-zero on either")
	repo := flag.String("repo", "", "repository root (default: found by walking up from the working directory)")
	quiet := flag.Bool("quiet", false, "print nothing on success")
	flag.Parse()

	if err := run(*check, *repo, *quiet); err != nil {
		fmt.Fprintf(os.Stderr, "reqtrace: %v\n", err)
		os.Exit(1)
	}
}

func run(check bool, repo string, quiet bool) error {
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repo, err = reqtrace.FindRepoRoot(cwd)
		if err != nil {
			return err
		}
	}

	layout := reqtrace.DefaultLayout(repo)

	model, err := reqtrace.Load(layout)
	if err != nil {
		return err
	}

	problems := reqtrace.Validate(model)
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, problem)
	}

	if check {
		staleErr := reqtrace.CheckFresh(layout, model)
		if staleErr != nil {
			fmt.Fprintln(os.Stderr, staleErr)
		}
		if len(problems) > 0 || staleErr != nil {
			return fmt.Errorf("traceability check failed: %d problem(s)%s",
				len(problems), staleSuffix(staleErr))
		}
		if !quiet {
			fmt.Printf("traceability ok: %s\n", reqtrace.Summary(model))
		}
		return nil
	}

	if err := reqtrace.Write(layout, model); err != nil {
		return err
	}
	if !quiet {
		fmt.Printf("wrote %s (%s)\n", layout.GeneratedPath(), reqtrace.Summary(model))
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
