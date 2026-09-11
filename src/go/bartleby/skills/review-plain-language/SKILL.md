---
name: review-plain-language
description: Review writing for plain language — whether the reader can find what they need, understand it first time, and act on it
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - review
  - plain-language
  - accessibility
  - documentation
---

# Review Plain Language

A different review from `review-prose`, and worth running separately. Style asks
"does this follow the rules". Plain language asks one harder question:

> Can the reader find what they need, understand it the first time they read it,
> and use it to do the thing?

A document can pass every style rule and fail that.

## When to use this

- Anything read by someone outside the team that wrote it — customers,
  executives, auditors, a partner's engineers, the person on call at 3am.
- A document people keep asking questions about after reading it.
- Anything with a decision or an obligation in it, where being misread has a
  cost.
- Prose that has been through several hands and accreted hedges.

## When not to use it as written

**Requirements and specifications are the exception**, and this is the tension
worth naming rather than papering over. Plain language pushes toward short,
common, unhedged words. A requirements baseline needs precise modal keywords
that carry a defined meaning — see *Requirements are different* below.

Apply plain language to everything a person reads to understand or act. Apply
the keyword discipline to the sentences a test has to be able to fail.

## Instructions

### Step 1: Name the reader and what they came for

Write it down: *"An SRE at 3am who needs to know whether to roll back."* Every
later judgment appeals to it. "Everyone" is not an audience, and a document
written for everyone tends to serve nobody — most obviously when it opens with
history that only the authors care about.

### Step 2: Put the answer first

Lead with what the reader came for, then the detail. Not background, not
motivation, not a summary of what the document will do.

The two commonest structural failures:

- **Buried answer.** Four paragraphs of context before the instruction. Move the
  instruction up; keep the context if it earns its place, below.
- **Missing bottom line.** A document that describes a situation thoroughly and
  never says what to do or what was decided.

### Step 3: Cut the words that do no work

| Instead of | Write |
|---|---|
| utilize, leverage | use |
| in order to | to |
| at this point in time | now |
| prior to | before |
| in the event that | if |
| has the ability to | can |
| it is recommended that you | we recommend, or just say what to do |
| there is a possibility that | might |

Then the harder pass — **nominalizations**, verbs turned into nouns, which hide
who does what:

- "The **configuration** of the build **is performed** by the CLI" →
  "The CLI **configures** the build."
- "**Verification** of the checksum **occurs** at build time" →
  "The build **verifies** the checksum."

### Step 4: Make it findable, not just readable

Most readers do not read; they scan until something looks like their problem.

- Headings that name the reader's question, not the subject area. "Why is my
  build failing?" beats "Troubleshooting".
- One idea per paragraph, and the idea in the first sentence.
- Lists for anything enumerable — steps, options, conditions.
- Tables when the reader is comparing.
- Short sentences where the content is dense. Twenty-five words is a lot when a
  sentence carries three conditions.

### Step 5: Say who does what, and be concrete

- **Actor first.** "You must rotate the token" or "The pipeline rotates the
  token" — never "the token must be rotated", which leaves the reader wondering
  whose job it is.
- **Concrete over abstract.** "Fails after 30 seconds" beats "may experience
  degraded performance".
- **No unmeasurable adjectives.** "Fast", "robust", "seamless", "significant"
  survive review because nobody can disprove them. Give the number.

### Step 6: Read it as the reader

Read the finished thing once, in their situation, and ask:

1. In the first ten seconds, do they know whether this document is for them?
2. Is there a sentence they would have to read twice?
3. Is there a term used before it is defined?
4. If they act only on the headings and the code blocks, do they get it right?

Question 4 catches the most, because that is what people actually do.

## Requirements are different

In a requirements baseline, the modal verbs are not style — they are defined
vocabulary. :rfc:`2119` gives them meaning, and it is worth quoting exactly:

> "MUST — This word, or the terms 'REQUIRED' or 'SHALL', mean that the
> definition is an absolute requirement of the specification."
>
> "SHOULD — This word, or the adjective 'RECOMMENDED', mean that there may exist
> valid reasons in particular circumstances to ignore a particular item, but the
> full implications must be understood and carefully weighed before choosing a
> different course."
>
> "MAY — This word, or the adjective 'OPTIONAL', mean that an item is truly
> optional."

So `shall` and `must` are the same thing, and plain-language advice to prefer
"must" is compatible with RFC 2119 rather than opposed to it. What is *not*
compatible is softening an obligation into "you may want to", which changes the
meaning of the requirement.

RFC 2119 also asks for restraint, which is the part most often ignored:

> "Imperatives of the type defined in this memo must be used with care and
> sparingly. In particular, they MUST only be used where it is actually required
> for interoperation or to limit behavior which has potential for causing harm."

Everything around the keyword sentence — the prose explaining why a requirement
exists, what breaks without it — is ordinary writing, and plain language applies
to it fully.

## Attribution

The plain-language practice here derives from the US federal plain-language
guidance formerly at `plainlanguage.gov`, now reorganised under
<https://digital.gov/guides/plain-language>. US federal works are generally not
copyrightable, but the reorganised site states no licence, so this skill is
written in our own words with our own examples rather than quoting it.

RFC 2119 is quoted directly: <https://www.rfc-editor.org/rfc/rfc2119.txt>.
IETF RFCs may be reproduced with attribution.
