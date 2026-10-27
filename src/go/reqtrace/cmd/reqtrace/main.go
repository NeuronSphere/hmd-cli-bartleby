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
	repo := flag.String("repo", "", "repository root (default: found by walking up from the working directory)")
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
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		return
	}

	opts := reqtrace.RunOptions{Check: *check, Repo: *repo, Quiet: *quiet, IncludeProposals: *includeProposals}
	if *verify {
		opts.Verify = &reqtrace.VerifyOptions{Needs: *needs, Results: results, Runs: runs, AllRuns: *allRuns, Out: *out, JSON: *asJSON}
	}
	if err := reqtrace.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "reqtrace: %v\n", err)
		os.Exit(1)
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
