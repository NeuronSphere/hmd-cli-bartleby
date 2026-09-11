---
name: review-prose
description: Review documentation against the Google developer documentation style guide — voice, tense, person, headings, and the words to stop using
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - review
  - style
  - documentation
  - google-style
---

# Review Prose

Review documentation the way a style guide would: consistently, on the things
that actually affect whether a reader understands it, and without rewriting
someone's voice for the sake of it.

This applies the **Google developer documentation style guide**, which is
licensed CC BY 4.0 — permissively, so its rules can be quoted here directly.
See the attribution at the end.

## When to use this

- Reviewing a documentation PR, and "looks fine" is not a review.
- Prose that is accurate and still hard to read.
- Documentation written by several people that reads like it.
- Before publishing anything outward-facing.

## When not to use this

This is a **style** review. It does not check whether the document is the right
*kind* of document — that is `plan-docs`. It does not check plain-language
readability for a non-expert audience — that is `review-plain-language`. And it
does not replace a linter: `review-vale` enforces much of this mechanically, and
should run first so that a human review is spent on judgment rather than on
catching "e.g." without a comma.

## Instructions

### Step 1: Fix the frame before the sentences

In order, because each one changes the sentences below it:

1. **Audience and their state** — is the reader learning, or working with a goal?
2. **Person** — second person. Address the reader as "you". Avoid "we" for
   instructions; reserve it for the authoring team's decisions.
3. **Tense** — present. "The build fails", not "the build will fail".
4. **Voice** — active, with the actor named. Passive is acceptable when the
   actor is genuinely unknown or irrelevant.
5. **Mood** — imperative for instructions. "Run the build", not "You should now
   run the build".

### Step 2: Work the sentence-level rules

The ones that recur most:

| Rule | Instead of | Write |
|------|-----------|-------|
| Second person | "Users must configure…" | "Configure…" |
| Active voice, named actor | "The container is started" | "The CLI starts the container" |
| Present tense | "This will return an error" | "This returns an error" |
| Sentence case headings | "Configuring The Build Options" | "Configuring the build options" |
| No wishful adverbs | "simply run", "just add", "it's easy to" | "run", "add" |
| Precise modal verbs | "you can't do X" (when you *may not*) | "must not" for prohibition, "can" for ability, "might" for possibility |
| Serial comma | "html, pdf and slides" | "html, pdf, and slides" |
| No Latin abbreviations in prose | "e.g.", "i.e.", "etc." | "for example", "that is", finish the list |
| Descriptive link text | "click [here]" | "see [the manifest reference]" |
| Define on first use | "the NID" | "the node identifier (NID)" |

**"Simply", "easy", "just", "obviously" are the highest-value deletions in any
review.** They add nothing when the reader succeeds, and when the reader is
stuck they say the difficulty is theirs. Delete them without asking.

### Step 3: Check the things that are not sentences

- **Headings** describe content, are sentence case, and are unique in the page.
  A reader scanning only the headings should be able to navigate.
- **Lists** are parallel: every item the same grammatical shape, punctuated the
  same way. Numbered for sequence, bulleted otherwise.
- **Code** in code blocks with a language, so it is highlighted and copyable.
  Inline code for identifiers, flags, and paths.
- **Placeholders** are consistent and obviously placeholders —
  `PROJECT_ID`, not `your-project`.
- **Tables** have a header row and no sentences long enough to need one.

### Step 4: Review the diff, not the document

Comment on what changed, at the line. Two rules that keep a style review from
becoming an argument:

- **Distinguish "wrong" from "different".** A rule broken is a comment; a
  sentence you would have phrased differently is not. Say which you mean.
- **Suggest the replacement.** "Passive voice" is work for the author;
  "The CLI starts the container" is a review.

### Step 5: Do not launder the voice out

The point is a document a reader can follow, not one that sounds machine-made.
Leave a well-turned sentence alone even if you would not have written it. If
every review makes every document sound the same, the reviews have started
costing more than they return.

## Attribution

Rules and examples above are drawn from the **Google developer documentation
style guide**, <https://developers.google.com/style>, licensed **CC BY 4.0**
(code samples Apache 2.0). CC BY permits reuse and adaptation with attribution
and has **no share-alike obligation**, which is why this skill can quote its
rules directly where `plan-docs` could not quote Diátaxis.

Where this skill and the guide disagree, the guide is right. It is far more
detailed — the word list alone is worth reading before writing any reference
documentation.
