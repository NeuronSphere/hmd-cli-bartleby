---
name: write-from-template
description: Pick the right document template for what needs writing and fill it in — a rubric over The Good Docs Project templates, keyed to the Diátaxis modes
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - documentation
  - templates
  - good-docs
  - diataxis
  - rst
---

# Write From Template

`plan-docs` decides **what kind** of document you need. This skill decides
**which template** to write it from, and what that template is going to make you
answer.

The templates are The Good Docs Project's, licensed **MIT No Attribution** — the
most permissive licence there is, so they can be copied, adapted, and shipped
with no attribution obligation. We credit them anyway.

## When to use this

- You know the document type and want a starting structure rather than a blank
  file.
- A document exists but is missing the sections its type always needs.
- Standardising a doc set so every how-to has the same shape.
- Someone asks "what should this page contain?"

## The rubric

Work down the first matching row. The reader's situation decides, not the
subject matter.

| The reader… | Mode | Template | The template forces you to state |
|---|---|---|---|
| has never used this and needs to succeed once, guided | Tutorial | `tutorial` | Overview, Background, **Before you start**, the tasks, Summary, Next steps |
| wants the shortest path to a first result, unguided | Tutorial | `quickstart` | The single happy path, prerequisites, one verification |
| has a specific goal and knows the vocabulary | How-to | `how-to` | Overview, **Before you start**, the task broken into sub-tasks, See also |
| is installing or upgrading | How-to | `installation-guide` | Requirements, per-platform steps, verification, uninstall |
| has an error in front of them | How-to | `troubleshooting` | Symptom → cause → resolution, per problem |
| needs a fact and will scan for it | Reference | `reference` | Description, then structured entries — tables, not prose |
| needs a fact about an API | Reference | `api-reference` | Endpoint, parameters, responses, errors, example |
| does not know what a word means here | Reference | `glossary` | One entry per term, defined without using the term |
| wants to understand how it fits together, or why | Explanation | `concept` | Background, **use cases**, comparison with alternatives, related reading |
| is evaluating whether to adopt it | Explanation | `sdk-overview` | What it is, what it is for, what it is not for |
| landed in the repository from a link | — | `readme` | What it is, install, minimal use, where the real docs are |
| wants to know what changed | — | `release-notes` / `changelog` | Version, date, added/changed/fixed, upgrade notes |
| wants to contribute | — | `contributing-guide` | Setup, workflow, review expectations, tests |

The last four rows sit outside the four Diátaxis modes on purpose. A changelog
is not documentation in that sense, and forcing it into a quadrant makes it
worse.

## Instructions

### Step 1: Confirm the mode before choosing a template

If you are unsure, `plan-docs` first. Picking a template for the wrong mode
produces a document with the right headings and the wrong content — a "tutorial"
full of options, or a "reference" that tells the reader what to do.

### Step 2: Fetch the template

The templates live at <https://gitlab.com/tgdp/templates>, one directory per
type, each with `template_<type>.md` plus a guide explaining every section.

```bash
curl -fsSL "https://gitlab.com/tgdp/templates/-/raw/main/how-to/template_how-to.md" -o docs/how-to/new-thing.md
```

Read the accompanying `guide_<type>.md` once per type — it explains *why* each
section exists, which is the difference between filling a template in and
writing to it.

### Step 3: Convert to RST if the repository is RST

These templates are Markdown. This pipeline builds both, but a repository is
usually one or the other — do not mix formats for the sake of a template. The
sections are what matter, not the syntax; retype the headings in RST and keep
the structure.

### Step 4: Fill in the sections that people skip

Three sections carry most of the template's value and are the first to be
deleted:

- **Before you start** (tutorial, how-to). Prerequisites, permissions, and
  assumed state. Its absence is the single commonest reason a how-to does not
  work for the reader.
- **Use cases** (concept). *When* you would want this thing. An explanation
  without them describes a mechanism and leaves the reader unable to tell
  whether it is for them.
- **See also / Next steps.** Where to go next. A document that ends without one
  is a dead end, and a doc set of dead ends is why people say they cannot find
  anything.

### Step 5: Delete what does not apply, and say nothing else

Templates carry optional sections — the Good Docs ones mark them `(Optional)`.
Delete the ones that do not apply. Do **not** leave a heading with "N/A" or
"TBD" under it: a reader cannot tell an empty section from an unfinished one,
and it makes the document look abandoned.

### Step 6: Check it against its own type

Before opening the PR, re-read it as the reader in the rubric row you picked.
For a how-to: could someone with your stated prerequisites follow it without
asking a question? For reference: can they find one fact in ten seconds without
reading prose? If not, the type is wrong or the content is in the wrong document.

## Attribution

Templates and their section structures are **The Good Docs Project** —
<https://www.thegooddocsproject.dev>, repository <https://gitlab.com/tgdp/templates>,
licensed **MIT No Attribution**. MIT-0 imposes no attribution requirement; the
credit here is courtesy, and it means these templates can be vendored into a
repository or a product without a licence obligation following them.

The four-mode column is **Diátaxis** (Daniele Procida, CC BY-SA 4.0) — see
`plan-docs`, which explains why we restate it rather than quote it.
