// Command reqtrace generates and checks the requirements traceability matrix.
//
//	reqtrace            # regenerate docs/requirements/traceability.rst
//	reqtrace -check     # fail if it is stale or coverage has a gap
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
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		return
	}

	if err := reqtrace.Run(reqtrace.RunOptions{Check: *check, Repo: *repo, Quiet: *quiet}); err != nil {
		fmt.Fprintf(os.Stderr, "reqtrace: %v\n", err)
		os.Exit(1)
	}
}

// versionString is what -version prints.
func versionString() string {
	return "reqtrace " + version
}
