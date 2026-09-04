---
name: check-traceability
description: Diagnose and fix a failing requirements traceability check — uncovered requirements, untagged tests, bad references, stale matrix
version: "1.0"
author: HMD Labs
requires:
  - reqtrace
tags:
  - requirements
  - traceability
  - testing
  - ci
  - sphinx-needs
---

# Check Traceability

Get the requirements traceability matrix back to green, without papering over
what it is reporting.

## When to use this

- `make check`, `make reqs-check`, or `reqtrace -check` failed, in CI or locally.
- Tests were just added, renamed, or deleted.
- Before opening a PR that touched `docs/requirements/` or any test.
- Reviewing whether coverage is real rather than nominal.

## The one judgment that matters

Every failure below has a fix that makes the check pass and a fix that makes the
software right, and they are not always the same. A tag can be added to any test
in seconds and the gap disappears from the report — while nothing verifies the
requirement.

**Tag a test with a requirement only if that test would fail when that
requirement is broken.** If it wouldn't, the honest move is to write the test,
or to leave the gap visible. A green matrix full of decorative tags is worse
than a red one, because it stops anyone looking.

## Instructions

### Step 1: Run it and read the whole output

```bash
make check           # repositories with the Makefile wrappers
make reqs-check      # traceability alone
reqtrace -check      # anywhere; -repo <path> if not run from the repo
```

If `reqtrace` is not installed:

```bash
go install github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/cmd/reqtrace@latest
```

Problems are printed one per line with `file:line`, grouped by kind, and the
final line counts them. Fix them by kind rather than top to bottom — one cause
often produces several lines, and a renamed requirement will show up as both a
bad link and a batch of unknown references.

Success looks like:

```
traceability ok: 102 requirements, 99 covered, 3 exempt; 181 tests
```

### Step 2: Diagnose by kind

| Message | What it means | The fix |
|---------|---------------|---------|
| `... has no test — cover it, or tag it "trace-exempt"` | A requirement nothing verifies | Write or find a test and annotate it. Only exempt it if no automated test can reasonably exist, and then say in the requirement's text how it *is* verified |
| `... declares no requirement` | A test claims no coverage | Annotate it with what it exercises. If it genuinely tests something the baseline never required, that is a missing requirement — use the `add-requirement` skill |
| `... claims X, which is not defined` | A test references an ID that no longer exists | The requirement was renumbered, renamed, or deleted. Point the test at the current ID; do not delete the annotation to silence it |
| `... links to X, which is not defined` | A `:links:` target is gone | Usually a `spec::` pointing at a renamed `req::`. Fix the target |
| `... is already defined at file:line` | Duplicate ID | Two requirements share an ID, so coverage of one credits the other. Renumber the *newer* one |
| `no req or spec items were found` | The requirements directory is empty or unparsed | Wrong working directory, or a repository with no baseline yet — see `add-requirement` |
| stale generated output | `traceability.rst` no longer matches the sources | Run `reqtrace` (or `make reqs`) and commit the result |

### Step 3: Annotate correctly

The annotation has to be where the parser looks.

**Go** — a line in the **doc comment on the test function**, not inside the body:

```go
// Requirements: REQ_SEL_001, REQ_SEL_006
func TestBuildsAcceptsAShellList(t *testing.T) {
```

Correct: directly above `func`, part of the doc comment, comma-separated.
Wrong: a comment inside the function body, or separated from `func` by a blank
line — the doc comment ends at the blank line and the annotation is invisible.

**Robot** — in `[Tags]`, separated by **two or more spaces**:

```robotframework
Unknown Builder Is An Error Not A Silent No-Op
    [Tags]    REQ_SEL_002    REQ_CLI_007
```

Single-space separation makes one long tag, and the requirement stays uncovered
while the test looks annotated.

**Use the short form.** Write `REQ_<AREA>_<NNN>`; the tooling expands it to the
full `HMD_<TYPE>_<NAME>_REQ_<AREA>_<NNN>`. The full form is accepted, but the
short one is what keeps a test header readable.

### Step 4: Regenerate, re-check, commit the page

```bash
make reqs && make check
```

Note that `reqtrace` without `-check` **also exits non-zero when problems
remain** — it regenerates the page and then reports, because a matrix that now
displays a gap is not a fixed gap. So a clean run means both that the page is
current and that nothing is outstanding.

`traceability.rst` is generated but **committed**: it is what makes the matrix
visible in the rendered documentation and in review, and CI fails when it is
stale. Never hand-edit it — the next run overwrites it.

### Step 5: Sanity-check coverage rather than count it

Before calling it done, read the coverage the report claims for the requirements
you touched:

- Does each named test actually assert the required behaviour, or does it merely
  execute the same code path?
- Is a requirement covered only by a test of the happy path, when the
  requirement is about the failure?
- Did a test get tagged with five requirements because it happens to run a lot
  of code?

The check can verify that a link exists. Only a person can verify that the link
means something.

## Common situations

**A requirement was deleted.** Remove its annotation from the tests too, or every
one of them reports an unknown reference. If the tests still have value, tag them
with the requirement that replaced it.

**A test was deleted.** Its requirements may now be uncovered. Either cover them
elsewhere or reconsider whether the requirement still holds — a requirement whose
only test was deleted deliberately is often a requirement that was withdrawn and
never removed.

**Renaming an area.** The area is part of every ID in it, so this is a
renumbering of the whole area: `index.rst` table, the file, every requirement ID,
every `:links:`, and every annotation. Doing it with `sed` across the repository
is reasonable; doing it by hand is not. Then run `reqtrace` and confirm the
counts match what they were before.

**CI fails and local passes.** Almost always an uncommitted regenerated page, or
a test file CI sees and the local layout does not. Check `git status`.

**A new repository layout.** `reqtrace` defaults to `docs/requirements/`,
`src/go/**` for Go tests, and `test/*.robot` for Robot suites. A repository that
puts them elsewhere needs the layout set explicitly — the tool is usable as a
library for that, or the directories can be aligned to the defaults, which is
usually less work.

## Anti-patterns

| Don't | Because |
|-------|---------|
| Tag whichever test is nearest to get to green | Produces coverage that verifies nothing and stops anyone looking |
| Delete a requirement to close a gap | Withdrawing a requirement is a decision, not a build fix |
| Hand-edit `traceability.rst` | Generated; the next run overwrites it and CI will fail on the difference |
| Remove a test's annotation to clear an unknown reference | The reference is stale, not wrong — repoint it |
| `trace-exempt` for anything inconvenient to test | Exemptions are for what cannot be tested, and each one owes a stated verification method |
| Trust the summary counts alone | They prove links exist, not that the tests mean anything |
