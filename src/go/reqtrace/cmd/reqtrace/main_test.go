package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Requirements: REQ_TRACE_009
func TestVersionIsInjectedAtBuildTime(t *testing.T) {
	// Built and run rather than inspected: the failure this guards against is
	// the ldflag silently ceasing to apply, which no in-process check can see.
	dir := t.TempDir()
	binary := filepath.Join(dir, "reqtrace")

	build := exec.Command("go", "build", "-ldflags", "-X main.version=9.9.9-test", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	out, err := exec.Command(binary, "-version").CombinedOutput()
	if err != nil {
		t.Fatalf("reqtrace -version: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "reqtrace 9.9.9-test" {
		t.Errorf("-version printed %q, want the injected version", got)
	}
}

// Requirements: REQ_TRACE_009
func TestVersionDefaultsToDev(t *testing.T) {
	// A binary built without the ldflag has to say so, or a local build is
	// indistinguishable from a release.
	if version != "dev" {
		t.Errorf("version = %q, want dev as the built-in default", version)
	}
	if !strings.HasPrefix(versionString(), "reqtrace ") {
		t.Errorf("versionString() = %q, want it to name the tool", versionString())
	}
}

// Requirements: REQ_TRACE_008
func TestReqtraceHasNoDependenciesToInherit(t *testing.T) {
	// The point of the carve-out is that a project can take reqtrace without
	// taking anything else. A single `require` here would undo that, so this
	// fails the build rather than letting one in quietly.
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	text := string(data)
	if requires := regexp.MustCompile(`(?m)^\s*require`).FindAllString(text, -1); len(requires) > 0 {
		t.Errorf("go.mod has %d require directive(s); reqtrace is meant to need nothing "+
			"outside the standard library:\n%s", len(requires), text)
	}
}

// Requirements: REQ_TRACE_008
func TestReqtraceIsItsOwnModule(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	want := "module github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
	if !strings.Contains(string(data), want) {
		t.Errorf("go.mod does not declare %q; `go install` of the documented path would break", want)
	}
}

// Requirements: REQ_TRACE_008
func TestLicenceIsApacheNotBSL(t *testing.T) {
	// The licence is the reason the module exists separately: a BSL dependency
	// is what consumers cannot take.
	data, err := os.ReadFile("../../LICENSE")
	if err != nil {
		t.Fatalf("reading LICENSE: %v", err)
	}
	if !strings.Contains(string(data), "Apache License") {
		t.Error("LICENSE is not the Apache License")
	}
}
