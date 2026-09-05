---
name: plan-docs
description: Decide what kind of document a piece of content is and where it belongs, using the Diátaxis four-mode model — and audit an existing doc set against it
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - documentation
  - diataxis
  - information-architecture
  - rst
  - review
---

# Plan Docs

Work out what kind of document something is before writing it, and put it where a
reader looking for that kind of thing will find it.

This applies the **Diátaxis** model: four kinds of documentation serving four
different needs. See the attribution note at the end — the ideas here are
Daniele Procida's, the wording is ours.

## When to use this

- Starting documentation for a project and deciding what files to create.
- Holding a page that has grown into several documents wearing one hat.
- A reader complains they "can't find" something that is, in fact, documented.
- Reviewing a doc set that accreted rather than was designed.
- Before `add-nerd` or `add-requirement`, when it isn't clear whether what you
  have is a proposal, a requirement, or documentation.

## The four modes

Each answers a different question, for a reader in a different state.

| Mode | Reader's need | Reader is | Serves |
|------|---------------|-----------|--------|
| **Tutorial** | "teach me" | learning, no context yet | acquisition of skill |
| **How-to guide** | "help me do this" | working, has a goal | application of skill |
| **Reference** | "tell me the facts" | working, needs a detail | information |
| **Explanation** | "help me understand" | studying, wants context | understanding |

The split that matters most is **learning vs working** (tutorial/explanation vs
how-to/reference) crossed with **practical vs theoretical** (tutorial/how-to vs
reference/explanation). Two documents can cover the same subject and belong in
different modes — "install the CLI" as a tutorial step, as a how-to, and as a
reference table of flags are three legitimate documents, not one document
written three times.

## Instructions

### Step 1: Name the reader's situation, not the subject

The most common mistake is organising by subject ("Logging", "Authentication")
and letting each page become all four modes at once. Ask instead:

- Is the reader **doing** something right now, or **learning/understanding**?
- Do they already know what they want, or are they being shown?

Those two answers pick the mode. Write the answer down before writing prose.

### Step 2: Apply the test for each mode

**Tutorial** — a lesson. It takes a beginner through a meaningful, working result
by a route you chose. It must work end to end, every time, with no decisions
left to the reader.

- Right: "Render your first document."
- Wrong: options, alternatives, explanations of why, or anything the reader must
  decide. A tutorial with a decision in it is a how-to guide.

**How-to guide** — a recipe for a goal the reader already has. It assumes
competence and starts from a realistic situation.

- Right: "Publish documentation from a CI pipeline."
- Wrong: teaching concepts, or completeness for its own sake. If it explains
  what a pipeline *is*, that part is explanation.

**Reference** — a description of the machinery. Austere, complete, structured to
mirror the thing it describes.

- Right: every flag, every manifest key, every environment variable.
- Wrong: instruction. The moment reference tells the reader what to *do*, it has
  become a how-to and stopped being scannable.

**Explanation** — the discussion. Why it works this way, what was rejected, how
it relates to other things, where it came from.

- Right: "Why the transform runs in a container."
- Wrong: steps. Explanation the reader is expected to follow along with is a
  tutorial in disguise.

### Step 3: Split what is wearing more than one hat

The signal is a document that changes mode partway down: a "Getting started"
that begins as a lesson, turns into a flag table, and ends with three paragraphs
on design rationale.

Split it. Cross-link the pieces rather than merging them — a tutorial *may* link
to reference; it must not *become* reference.

### Step 4: Lay it out so the mode is visible

In a Sphinx repository, make the four modes structural rather than implied:

```
docs/
  index.rst              # the map: what each section is for
  tutorials/
  how-to/
  reference/
  explanation/
```

Then say in `index.rst` what each section is for, in the reader's terms. A doc
set the reader cannot navigate by need is the problem Diátaxis exists to fix,
and folders alone do not fix it.

Where a repository has an established structure that does not match — this one's
own `docs/` does not — do not restructure it as a side effect of some other
task. Note the mismatch and raise it as its own piece of work.

### Step 5: Check what a mode is missing

Audit by mode, not by page count. The usual pattern:

- **Reference** is usually the most complete, because it is generated or
  mechanical.
- **How-to** is usually thin, because the authors already know the recipes.
- **Explanation** is usually missing entirely, and its absence is what makes
  people say documentation is "hard to follow" — the facts are there and the
  reasoning is not.
- **Tutorial** is usually stale, because nobody re-runs it. **Re-run it.** A
  tutorial that no longer works is worse than no tutorial: it teaches the reader
  the project is broken.

### Step 6: Do not force it

Diátaxis is a tool for thinking, not a compliance regime. Some documents are
legitimately none of the four — a changelog, a release note, a licence, a
requirements baseline, an architecture decision record. Do not file them into a
quadrant to be tidy. This repository's `docs/requirements/` and
`docs/proposals/` are examples: a requirement is a contract and a NERD is a
proposal, and neither is documentation in the Diátaxis sense.

## Relationship to the other skills

| If the content is… | Use |
|---|---|
| A proposal to change something | `add-nerd` |
| An obligation the software must meet | `add-requirement` |
| An API's surface (reference) | `document-api` |
| Documentation for a reader | this skill, then write it |

## Attribution

The four-mode model is **Diátaxis**, by Daniele Procida —
<https://diataxis.fr>. Diátaxis is licensed CC BY-SA 4.0.

This skill is a restatement in our own words, with our own examples, so that it
can ship under this repository's licence; it quotes nothing. If you want to
include Diátaxis text verbatim, that makes this file adapted material and it
must then carry CC BY-SA 4.0 — put it in its own file with its own licence
notice rather than relicensing anything else.

Read the source. It is short, well written, and makes the case better than a
summary can.
