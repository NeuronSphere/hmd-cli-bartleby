---
name: add-requirement
description: Add or amend a requirement in a repository's standing requirements baseline, and cover it with a test
version: "1.0"
author: HMD Labs
requires:
  - reqtrace
tags:
  - requirements
  - sphinx-needs
  - traceability
  - testing
  - rst
---

# Add Requirement

Write a requirement into the standing baseline in `docs/requirements/`, give it a
correct area-coded ID, and connect it to a test that verifies it — in one pass,
so the repository is never left with a requirement nothing checks.

## When to use this

- Behaviour is being added or changed, and the baseline should say so.
- A NERD was accepted, and its proposal needs to become a standing requirement.
- Review asks "which requirement covers this?" and the answer is none.
- A bug was fixed that the baseline never required not happening.

## When not to use this

**A proposal for a change is a NERD, not a requirement.** Use the `add-nerd`
skill for that. NERDs live in `docs/proposals/` and record what someone wants to
change and why; requirements in `docs/requirements/` describe how the software is
meant to behave now. Coverage is enforced on the baseline, not on proposals, so a
requirement parked in a NERD is a requirement nothing verifies.

The two connect: an accepted NERD's spec carries `:links:` to the requirement it
adds or amends.

## Instructions

### Step 1: Read the baseline before adding to it

```bash
cat docs/requirements/index.rst
ls docs/requirements/
```

`index.rst` is authoritative for this repository: it lists the area codes, the ID
scheme, and the annotation conventions. **Do not invent a convention that
contradicts it.** Note in particular:

- the ID prefix, e.g. `HMD_CLI_BARTLEBY_REQ_`
- the area table — which area your new requirement belongs to
- whether the repository uses `spec::` items, and for what

If `docs/requirements/` does not exist, see *Bootstrapping a repository* at the
end of this skill.

### Step 2: Place it in an area

Requirements are grouped one file per area, and the area code is part of the ID.
Pick the area from the table in `index.rst`, by what the requirement is *about*:

```bash
grep -rn '\.\. req::' docs/requirements/<area>.rst | tail -5
```

If nothing fits, adding an area is legitimate but costs three edits: a new
`docs/requirements/<area>.rst`, a row in the area table in `index.rst`, and an
entry in that file's `toctree`. Prefer an existing area unless the new subject is
genuinely separate — an area with one requirement in it is a smell.

### Step 3: Allocate the ID

IDs are `<PREFIX>REQ_<AREA>_<NNN>`, three digits, sequential within the area:

```bash
grep -rhoE 'REQ_<AREA>_[0-9]{3}' docs/requirements/ | sort -u | tail -3
```

Take the next number. **Never reuse or renumber an ID** — tests, NERDs, commit
messages, and review comments all reference them, and renumbering silently
re-points every one of those. A withdrawn requirement is deleted and its number
retired, not backfilled.

A `spec::` that pins down *how* a requirement is met appends `_SPEC<NNN>` to the
requirement's own ID and links back to it:

```
HMD_CLI_BARTLEBY_REQ_EXPL_002_SPEC001  links to  HMD_CLI_BARTLEBY_REQ_EXPL_002
```

### Step 4: Write it

```rst
.. req:: Short imperative title
    :id: <PREFIX>REQ_<AREA>_<NNN>
    :status: implemented

    The CLI shall <one obligation, stated as observable behaviour>.

<Optional prose: why this is required, what breaks without it. This is where the
reasoning goes — not in the requirement body.>
```

What makes a requirement worth having:

- **One obligation.** Two "shall"s joined by "and" is two requirements, and a
  test can pass for one while the other is broken.
- **Observable behaviour, not implementation.** "shall exit non-zero and name the
  log file" can be tested and will outlive a rewrite; "shall call
  `checkExitCode()`" cannot and will not.
- **Say what happens in the bad case.** Most defects live there. "An unknown
  builder shall be an error that lists the valid ones" is a requirement; "shall
  support builders" is a wish.
- **`shall` for the obligation.** It marks the testable sentence apart from the
  prose around it.
- **No unmeasurable adjectives.** "fast", "robust", "user-friendly" cannot fail a
  test. Give the number, or the specific behaviour that stands in for the quality.

`:status:` is `implemented` once the code is in. Use `proposed` only while the
requirement is genuinely ahead of the code, and expect the traceability check to
fail until a test exists.

### Step 5: Cover it in the same change

A requirement with no test is a gap the check will fail on, so do this now
rather than later. Either find the test that already exercises the behaviour, or
write one, then annotate it with the **short form** of the ID (the tooling
expands it):

Go — a doc comment line on the test function:

```go
// Requirements: REQ_<AREA>_<NNN>
func TestTheBehaviour(t *testing.T) {
```

Robot — in `[Tags]`, separated by two or more spaces:

```robotframework
The Behaviour Holds
    [Tags]    REQ_<AREA>_<NNN>
```

One test may carry several requirements, and one requirement may be covered by
several tests. Tag only what the test actually exercises — see the
`check-traceability` skill for why a decorative tag is worse than a visible gap.

### Step 6: Regenerate and check

```bash
make reqs && make check      # in repositories with the Makefile wrappers
reqtrace && reqtrace -check  # anywhere else
```

`reqtrace` rewrites the generated traceability page and **exits non-zero if the
result still has a gap** — regenerating is not the same as passing. Commit the
regenerated page with your change; CI compares against it and a stale page fails.

### Step 7: Only if no test can reasonably exist

Some requirements cannot be tested automatically — pulling a multi-gigabyte
image, or behaviour that needs live credentials CI deliberately lacks. Tag the
requirement `trace-exempt` **and say in its own text how it is verified
instead**:

```rst
.. req:: Pull the transform image on request
    :id: <PREFIX>REQ_<AREA>_<NNN>
    :status: implemented
    :tags: trace-exempt

    ``bartleby update-image`` shall re-pull the transform image.

    *Verification:* by manual run. Exercising it needs a multi-gigabyte pull that
    CI will not do.
```

An exemption without that sentence is just an untested requirement with the
alarm switched off. Exemptions should be countable on one hand; if they are
accumulating, the tests are in the wrong place, not the requirements.

## Bootstrapping a repository

Where `docs/requirements/` does not exist yet:

1. Create `docs/requirements/index.rst` describing the scheme — prefix, area
   table, annotation conventions, and what the check enforces. Copy the structure
   from `hmd-cli-bartleby`, which is the reference implementation.
2. Add one area file and one real requirement, tagged onto one real test. Prove
   the loop end to end before writing thirty requirements.
3. Add `requirements/index` to `docs/index.rst`'s toctree.
4. Wire the check in:

   ```bash
   brew install neuronsphere/tap/reqtrace   # macOS
   go install github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/cmd/reqtrace@latest
   ```

   then add `reqtrace -check` to the repository's `check` target and to CI. It
   needs neither Docker nor Sphinx, so it runs on a laptop.
5. Configure sphinx-needs in `meta-data/manifest.json` so the IDs are validated
   at build time too:

   ```json
   "needs_id_regex": "^HMD_[A-Z0-9_]+$"
   ```

Start from the behaviour that already has tests. Writing requirements against
tests that exist is fast and produces a baseline that is green from day one;
writing thirty aspirational requirements produces thirty gaps and a check
everybody learns to ignore.

## Anti-patterns

| Don't | Because |
|-------|---------|
| Park requirements in a NERD | Coverage is enforced on the baseline; a NERD requirement is unverified |
| Renumber or reuse an ID | Every test, NERD, and review comment referencing it silently re-points |
| Restate the implementation | The requirement dies at the next refactor and tests nothing meaningful |
| Add a requirement without a test | Leaves the check red for whoever commits next |
| `trace-exempt` to get to green | An exemption with no stated verification method is a hidden gap |
| One requirement per function | The baseline becomes a second, worse copy of the code |
