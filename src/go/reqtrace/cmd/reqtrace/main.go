// Command reqtrace generates and checks the requirements traceability matrix.
//
//	reqtrace            # regenerate docs/requirements/traceability.rst
//	reqtrace -check     # fail if it is stale or coverage has a gap
//	reqtrace -verify -results python=bender/output.xml -results go=go-output.xml
//	                    # join needs.json with test results: one verdict per
//	                    # requirement and implementation
//	reqtrace -verify -runs bender/runs
//	                    # the same, from Bender's archived run folders: each
//	                    # label's latest run (-all-runs for every run)
//	reqtrace -format json
//	                    # list every need with its stage and covering tests;
//	                    # run over a directory of repositories, lists them all
//	reqtrace -format json -workspace DIR -open -changes
//	                    # open needs across DIR, and what changed since last time
//	reqtrace -diff OLD.json NEW.json
//	                    # what changed between two listings
//
// It needs neither Docker nor Sphinx, so it runs on a laptop and in CI. In this
// repository the Makefile wraps both as "make reqs" and "make reqs-check", and
// "make check" runs the latter, so traceability breaks the build. The same work
// is reachable as "bartleby reqs" for anyone who already has the CLI.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
)

// version is set at build time with -ldflags "-X main.version=<version>".
// A binary built without it reports "dev", which is how you tell a local build
// from a released one.
var version = "dev"

func main() {
	check := flag.Bool("check", false, "report problems and stale output instead of writing; exit non-zero on either")
	var repos stringList
	flag.Var(&repos, "repo", "repository root (default: found by walking up from the working directory); with -format json in a workspace, a repository to keep, by name or folder; repeatable")
	quiet := flag.Bool("quiet", false, "print nothing on success")
	includeProposals := flag.Bool("include-proposals", false, "also scan docs/proposals (NERD documents) for req/spec items")
	showVersion := flag.Bool("version", false, "print the version and exit")
	verify := flag.Bool("verify", false, "join needs.json with test results instead of generating the matrix; exit non-zero if a requirement fails or has no result")
	needs := flag.String("needs", "", "needs.json from a bartleby build, for -verify (default target/bartleby/html/needs.json)")
	var results stringList
	flag.Var(&results, "results", "for -verify: LABEL[@VERSION]=PATH of a Robot output.xml, go test -json output or Bender run folder; repeatable")
	var runs stringList
	flag.Var(&runs, "runs", "for -verify: a directory of Bender run folders, such as bender/runs; uses each label's latest run; repeatable")
	allRuns := flag.Bool("all-runs", false, "for -verify with -runs: use every run, each as its own column, not each label's latest")
	out := flag.String("out", "", "for -verify: page to write (default docs/requirements/verification_matrix.rst; - for none)")
	asJSON := flag.Bool("json", false, "for -verify: print the verification as JSON")
	format := flag.String("format", "", `"json" lists every need with its stage and covering tests instead of generating the matrix`)
	workspace := flag.String("workspace", "", "for -format json: list every repository under this directory")
	var stages, types stringList
	flag.Var(&stages, "stage", "for -format json: keep needs in this stage (draft, proposed, in_progress, implemented, deferred, closed, unknown); repeatable")
	flag.Var(&types, "type", "for -format json: keep needs of this directive type, such as req or spec; repeatable")
	open := flag.Bool("open", false, "for -format json: keep needs not implemented or closed, and implemented needs no test covers")
	summary := flag.Bool("summary", false, "for -format json: print per-repository counts instead of the needs")
	changes := flag.Bool("changes", false, "for -format json in a workspace: add what changed since the previous call")
	rebuild := flag.Bool("rebuild", false, "for -format json in a workspace: ignore the index and build it again")
	diff := flag.Bool("diff", false, "print the changes between two listings: reqtrace -diff OLD.json NEW.json")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		return
	}

	opts, err := options(*format, repos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reqtrace: %v\n", err)
		os.Exit(2)
	}
	opts.Check, opts.Quiet, opts.IncludeProposals = *check, *quiet, *includeProposals
	if *verify {
		opts.Verify = &reqtrace.VerifyOptions{Needs: *needs, Results: results, Runs: runs, AllRuns: *allRuns, Out: *out, JSON: *asJSON}
	}
	if opts.List != nil {
		opts.List.Workspace, opts.List.Stages, opts.List.Types = *workspace, stages, types
		opts.List.Open, opts.List.Summary, opts.List.Changes, opts.List.Rebuild = *open, *summary, *changes, *rebuild
	}
	if *diff {
		opts.Diff = flag.Args()
		if len(opts.Diff) != 2 {
			fmt.Fprintln(os.Stderr, "reqtrace: -diff takes two listing files: reqtrace -diff OLD.json NEW.json")
			os.Exit(2)
		}
	}
	if err := reqtrace.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "reqtrace: %v\n", err)
		os.Exit(1)
	}
}

// options turns -format and -repo into RunOptions: for a listing, -repo values
// select repositories; otherwise there is at most one, the repository root.
func options(format string, repos []string) (reqtrace.RunOptions, error) {
	switch format {
	case "":
		if len(repos) > 1 {
			return reqtrace.RunOptions{}, fmt.Errorf("-repo names one repository root unless -format json is given")
		}
		var opts reqtrace.RunOptions
		if len(repos) == 1 {
			opts.Repo = repos[0]
		}
		return opts, nil
	case "json":
		return reqtrace.RunOptions{List: &reqtrace.ListOptions{Repos: repos}}, nil
	default:
		return reqtrace.RunOptions{}, fmt.Errorf("-format %q is not supported; use json", format)
	}
}

// versionString is what -version prints.
func versionString() string {
	return "reqtrace " + version
}

// stringList is a repeatable string flag.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}
