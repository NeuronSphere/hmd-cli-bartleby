---
name: add-nerd
description: Add a NERD (NeuronSphere Engineering Requirements Document) using sphinx-needs
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - nerd
  - requirements
  - sphinx-needs
  - proposals
  - rst
---

# Add NERD

Create a new NERD (NeuronSphere Engineering Requirements Document) using sphinx-needs directives.

## Instructions

When the user invokes this skill, follow these steps:

### Step 1: Determine NERD Number

1. **Check existing NERDs** in `docs/proposals/` or `docs/nerd/`:
   - List existing NERD files (e.g., `NERD001_Title.rst`, `NERD002_Title.rst`)
   - Determine the next sequential number (e.g., if NERD001 exists, next is NERD002)
   - If no proposals directory exists, the first NERD will be NERD001

2. **Identify the repository context**:
   - Read repository name from `meta-data/manifest.json`
   - Determine repo type from the naming convention (e.g., `hmd-cli-bartleby` → type `CLI`, name `BARTLEBY`)

### Step 2: Ask the User

If not already specified, ask the user for:
- **Title**: A concise title for the requirement (e.g., "Support Multiple Build Targets")
- **Description**: What the requirement is about
- **Specifications**: Any specific implementation details or acceptance criteria

### Step 3: Generate the NERD File

Create `docs/proposals/NERD<NNN>_<Title_Underscored>.rst` with this structure:

```rst
.. NERD<NNN> <Title>

NERD<NNN> <Title>
===================================

.. req:: <Title>
    :id: HMD_<REPO_TYPE>_<NAME>_NERD<NNN>
    :status: proposed

    <Requirement description written by the user or derived from discussion.>

<Prose context explaining the motivation, background, and approach for this requirement.
Reference existing behavior, explain what changes, and why.>

.. spec:: <Specification Title>
    :id: HMD_<REPO_TYPE>_<NAME>_NERD<NNN>_SPEC001
    :links: HMD_<REPO_TYPE>_<NAME>_NERD<NNN>
    :status: proposed

    <Detailed specification of how this requirement will be implemented.>
```

#### ID Convention

IDs follow the pattern: `HMD_<REPO_TYPE>_<NAME>_NERD<NNN>`

Examples:
- `hmd-cli-bartleby` → `HMD_CLI_BARTLEBY_NERD002`
- `hmd-ms-gozer` → `HMD_MS_GOZER_NERD001`
- `hmd-lib-robot-shared` → `HMD_LIB_ROBOT_SHARED_NERD001`
- `hmd-tf-bartleby` → `HMD_TF_BARTLEBY_NERD001`

Spec IDs append `_SPEC<NNN>`:
- `HMD_CLI_BARTLEBY_NERD002_SPEC001`
- `HMD_CLI_BARTLEBY_NERD002_SPEC002`

#### Status Values

- `proposed` — Initial state for new requirements and specs
- `in_progress` — Actively being implemented
- `completed` — Implementation finished
- `done` — Equivalent to completed
- `implemented` — Specification has been implemented

### Step 4: Ensure Proposals Index Exists

If `docs/proposals/index.rst` does not exist, create it:

```rst
.. NERD Proposals

NERD Proposals
====================

.. toctree::
   :maxdepth: 2
   :caption: Contents:
   :glob:

   ./*
```

The `:glob:` directive automatically picks up new NERD files without manual toctree edits.

### Step 5: Update Parent index.rst

Check `docs/index.rst` and ensure `proposals/index` is in the toctree. If not, add it:

```rst
.. toctree::

   install_and_run
   proposals/index
```

### Step 6: Offer to Build

After creating the NERD, offer to build:

```bash
hmd bartleby html
```

## Example

### Input

User: "Add a NERD for supporting RevealJS slideshows"

### Existing State

- Repository: `hmd-cli-bartleby`
- Existing NERDs: `NERD001_Specify_Root_Document.rst`
- Next number: NERD002

### Generated File: `docs/proposals/NERD002_Support_RevealJS_Slideshows.rst`

```rst
.. NERD002 Support RevealJS Slideshows

NERD002 Support RevealJS Slideshows
=========================================

.. req:: Support RevealJS Slideshows
    :id: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The Bartleby CLI should support rendering RST documents as RevealJS
    slideshows via a dedicated ``slides`` subcommand and ``revealjs`` builder
    configuration in the manifest.

RevealJS is a popular HTML presentation framework. Adding support for it in
Bartleby allows NeuronSphere teams to author slide decks alongside their
documentation using familiar RST syntax. The ``sphinx_revealjs`` extension
is already available in the Bartleby transform image.

.. spec:: Add slides subcommand
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    A new ``hmd bartleby slides`` subcommand will invoke the ``revealjs``
    builder. It will follow the same pattern as the existing ``html`` and
    ``pdf`` subcommands, filtering roots that have ``revealjs`` in their
    builders list.

.. spec:: Support revealjs builder in manifest roots
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The ``bartleby.roots`` manifest configuration will accept ``revealjs``
    as a valid builder name. Roots that include this builder will be
    rendered as slideshows when ``hmd bartleby slides`` is invoked.
```
