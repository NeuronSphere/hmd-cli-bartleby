---
name: review-vale
description: Set up and run Vale to enforce prose style mechanically — configuration, style packages, vocabularies, and CI — so human review is spent on judgment
version: "1.0"
author: HMD Labs
requires:
  - vale
tags:
  - review
  - linting
  - vale
  - ci
  - style
---

# Review Vale

The third flavour of review, and the only one a machine can do. `review-prose`
applies a style guide by judgment; **Vale applies the mechanical part of the
same guide on every commit**, which is what stops a reviewer spending their
attention on a missing serial comma.

Vale ships an MIT implementation of the Google developer documentation style
guide, so the two skills enforce the same rules from different ends.

## When to use this

- A repository whose documentation gets style comments in review, repeatedly.
- Setting up a new docs repository, before habits set.
- Prose review that keeps finding the same five things.
- A house vocabulary that people keep spelling three ways.

## What it can and cannot do

| Vale catches | Vale cannot catch |
|---|---|
| Banned and inconsistent words | Whether the document is the right *kind* (`plan-docs`) |
| Passive voice, weak modals, hedges | Whether the answer is buried (`review-plain-language`) |
| Heading case, list punctuation | Whether an instruction is correct |
| Spelling, including a house vocabulary | Whether a claim is true |
| "simply", "easy", "just" | Whether the reader can do the task |

Run it first, then review what is left. A reviewer arriving after Vale has a
better conversation.

## Instructions

### Step 1: Install it

```bash
brew install vale
```

**reStructuredText needs `docutils`**, because Vale parses `.rst` through
`rst2html`:

```bash
pip install docutils
```

Without it, Vale silently reports nothing for `.rst` files, which looks exactly
like a clean run. Check that it is actually reading the files:
`vale ls-metrics docs/index.rst`.

### Step 2: Configure it

`.vale.ini` at the repository root:

```ini
StylesPath = styles
MinAlertLevel = suggestion

Packages = Google, write-good

Vocab = House

[*.{md,rst}]
BasedOnStyles = Vale, Google

# Sphinx roles and directives are not prose; do not lint their contents.
TokenIgnores = (:\w+:`[^`]+`)
```

Then:

```bash
vale sync     # downloads the packages into styles/
vale docs/
```

`vale sync` is required after any change to `Packages`, and `styles/` is
generated — gitignore it and commit `.vale.ini` instead.

Style packages worth knowing, all MIT:

| Package | What it is |
|---|---|
| `Google` | Implementation of the Google developer documentation style guide |
| `Microsoft` | Implementation of the Microsoft style guide — a rules implementation, distinct from the proprietary guide text |
| `write-good` | General weak-writing heuristics |
| `proselint` | Broad prose checks, opinionated |
| `alex` | Insensitive or inconsiderate wording |

Pick **one** house guide. `Google` and `Microsoft` together contradict each
other and produce noise that trains people to ignore the tool.

### Step 3: Teach it your vocabulary

The first run on a real repository is mostly false positives on your own nouns.
Fix that once, in `styles/config/vocabularies/House/`:

`accept.txt` — words Vale should stop flagging. Regex, one per line:

```
[Bb]artleby
NeuronSphere
reqtrace
sphinx-needs
PlantUML
Diátaxis
```

`reject.txt` — words that are always wrong here:

```
[Ss]imply
[Ee]asy
[Jj]ust use
```

A vocabulary is not a workaround; it is the part that makes the tool yours.

### Step 4: Set the level that will actually be enforced

Vale has three severities, and the choice is a policy decision:

- `suggestion` — shown, never blocks.
- `warning` — shown, blocks only if you say so.
- `error` — blocks.

Start with `MinAlertLevel = suggestion` locally so people see everything, and
gate CI at `error` only:

```bash
vale --minAlertLevel=error docs/
```

Then promote individual rules to `error` as the team agrees to them. Turning
everything on at once on an existing repository produces thousands of alerts and
one decision: to disable Vale.

To change a single rule's severity, in `.vale.ini`:

```ini
Google.Passive = warning
Google.Headings = error
```

### Step 5: Wire it into CI

```yaml
lint-prose:
  script:
    - pip install docutils
    - vale sync
    - vale --minAlertLevel=error --output=line docs/
```

`--output=line` gives `file:line:col: message`, which most CI systems turn into
annotations on the diff.

Two things worth doing in the same change:

- **Lint only what changed**, on large existing repositories:
  `git diff --name-only origin/main... -- '*.rst' '*.md' | xargs -r vale`
- **Commit the config, not the styles.** `styles/` comes from `vale sync`.

### Step 6: Write a rule when a review comment repeats

If the same comment appears three times, it is a rule. `styles/House/`:

```yaml
# styles/House/NoUnmeasurable.yml
extends: existence
message: "'%s' cannot be verified — give the number or the behaviour."
level: error
ignorecase: true
tokens:
  - seamless(ly)?
  - robust
  - blazing(ly)? fast
  - significant(ly)?
```

Rule types you will use most: `existence` (banned words), `substitution`
(prefer X over Y), `occurrence` (limit per file), `consistency` (pick one of a
pair), `capitalization` (heading case).

Keep house rules few and mean something. A rule nobody agreed to is a rule
somebody will disable.

## Attribution

**Vale** — <https://vale.sh>, MIT, by errata-ai. The `Google` and `Microsoft`
style packages are also MIT. RST support requires `docutils`, per
<https://docs.vale.sh/formats/restructuredtext>.
