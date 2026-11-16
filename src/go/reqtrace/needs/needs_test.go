package needs

import (
	"strings"
	"testing"
)

const sample = `Title
=====

.. req:: First requirement
    :id: X_REQ_001
    :status: proposed
    :tags: a, b
    :links: X_NERD001,
        X_NERD002

    Body line one.

    Body line two.

.. spec:: A spec
    :id: X_SPEC_001
    :status: implemented

Prose that ends the spec.

.. note:: not a need

.. decision:: A decision
    :id: X_DEC_001
`

// Requirements: REQ_LIST_010
func TestParseReadsOptionsBodiesAndWrappedValues(t *testing.T) {
	got, err := Parse(strings.NewReader(sample), DefaultTypes())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("parsed %d needs, want 3: %+v", len(got), got)
	}

	req := got[0]
	if req.Type != "req" || req.ID != "X_REQ_001" || req.Title != "First requirement" || req.Status != "proposed" {
		t.Errorf("req = %+v", req)
	}
	if req.Line != 4 {
		t.Errorf("req.Line = %d, want 4", req.Line)
	}
	if strings.Join(req.Tags, "|") != "a|b" {
		t.Errorf("tags = %v", req.Tags)
	}
	if strings.Join(req.Links, "|") != "X_NERD001|X_NERD002" {
		t.Errorf("wrapped :links: = %v, want both targets", req.Links)
	}
	if !strings.Contains(req.Body, "Body line one.") || !strings.Contains(req.Body, "Body line two.") {
		t.Errorf("body = %q", req.Body)
	}

	if got[1].ID != "X_SPEC_001" || strings.Contains(got[1].Body, "Prose") {
		t.Errorf("spec = %+v; unindented prose must end the directive", got[1])
	}
	if got[2].Type != "decision" || got[2].Status != "" {
		t.Errorf("decision = %+v", got[2])
	}
}

// Requirements: REQ_LIST_010
func TestParseSkipsLiteralAndCodeBlocks(t *testing.T) {
	const text = `Example::

    .. req:: Inside a literal block
        :id: X_LIT_001

.. code-block:: rst

    .. req:: Inside a code block
        :id: X_CODE_001

.. req:: Real
    :id: X_REAL_001
`
	got, err := Parse(strings.NewReader(text), DefaultTypes())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "X_REAL_001" {
		t.Fatalf("got %+v, want only X_REAL_001", got)
	}
}

// Requirements: REQ_LIST_010
func TestParseIgnoresNeedsWithoutAnID(t *testing.T) {
	got, err := Parse(strings.NewReader(".. req:: No id\n    :status: draft\n"), DefaultTypes())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want nothing", got)
	}
}

// Requirements: REQ_LIST_010
func TestDigestIgnoresLineEndingsAndTrailingSpace(t *testing.T) {
	unix := ".. req:: T\n    :id: X_1\n\n    Body.\n\n\n"
	windows := ".. req:: T  \r\n    :id: X_1\r\n\r\n    Body.   \r\n"
	edited := ".. req:: T\n    :id: X_1\n\n    Body changed.\n"

	digest := func(text string) string {
		t.Helper()
		got, err := Parse(strings.NewReader(text), DefaultTypes())
		if err != nil || len(got) != 1 {
			t.Fatalf("parse %q: %v %+v", text, err, got)
		}
		return got[0].Digest
	}

	if digest(unix) != digest(windows) {
		t.Error("digest differs between LF and CRLF with trailing space")
	}
	if digest(unix) == digest(edited) {
		t.Error("digest did not change when the body changed")
	}
	if len(digest(unix)) != 64 {
		t.Errorf("digest %q is not hex SHA-256", digest(unix))
	}
}

// Requirements: REQ_LIST_001
func TestTypesAreConfigurable(t *testing.T) {
	text := ".. req:: R\n    :id: X_R\n\n.. story:: S\n    :id: X_S\n"

	got, _ := Parse(strings.NewReader(text), DefaultTypes())
	if len(got) != 1 {
		t.Fatalf("default types parsed %d needs, want 1 (story is not a default type)", len(got))
	}

	types := DefaultTypes()
	types["story"] = true
	got, _ = Parse(strings.NewReader(text), types)
	if len(got) != 2 {
		t.Fatalf("with story declared, parsed %d needs, want 2", len(got))
	}
}
