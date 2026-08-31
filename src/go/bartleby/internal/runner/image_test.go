package runner

import (
	"strings"
	"testing"
)

// Requirements: REQ_EXEC_014
func TestReportProgressCollapsesRepeatedStatuses(t *testing.T) {
	stream := strings.Join([]string{
		`{"status":"Pulling from neuronsphere/hmd-tf-bartleby"}`,
		`{"status":"Downloading","id":"a1b2","progressDetail":{"current":1,"total":10}}`,
		`{"status":"Downloading","id":"a1b2","progressDetail":{"current":5,"total":10}}`,
		`{"status":"Download complete","id":"a1b2"}`,
		`{"status":"Status: Downloaded newer image"}`,
	}, "\n")

	var out strings.Builder
	if err := reportProgress(strings.NewReader(stream), &out); err != nil {
		t.Fatalf("reportProgress: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4 (the repeated Downloading is collapsed):\n%s", len(lines), out.String())
	}
	if !strings.Contains(out.String(), "a1b2: Download complete") {
		t.Errorf("layer status missing:\n%s", out.String())
	}
}

// Requirements: REQ_EXEC_014
func TestReportProgressSurfacesStreamErrors(t *testing.T) {
	stream := `{"status":"Pulling"}` + "\n" + `{"error":"manifest unknown"}`

	var out strings.Builder
	err := reportProgress(strings.NewReader(stream), &out)
	if err == nil {
		t.Fatal("expected an error from the progress stream")
	}
	if !strings.Contains(err.Error(), "manifest unknown") {
		t.Errorf("err = %v, want the daemon's message", err)
	}
}

// Requirements: REQ_EXEC_014
func TestReportProgressEmptyStream(t *testing.T) {
	var out strings.Builder
	if err := reportProgress(strings.NewReader(""), &out); err != nil {
		t.Errorf("an empty stream is not an error, got %v", err)
	}
}
