---
name: diagram-c4
description: Draw architecture diagrams with the C4 model in PlantUML — context, container, component — that render offline in the Bartleby pipeline
version: "1.0"
author: HMD Labs
requires:
  - hmd-cli-bartleby
tags:
  - architecture
  - c4
  - plantuml
  - diagrams
  - rst
---

# Diagram C4

Draw an architecture diagram someone can actually read, using the C4 model, in
PlantUML, rendered by the same pipeline that builds the documents.

## When to use this

- A design needs a picture and there isn't one.
- The picture that exists is a box-and-line drawing with no stated level, so
  nobody can tell whether a box is a process, a library, or a team.
- Onboarding: the fastest thing you can hand a new person is one context diagram
  and one container diagram.
- Reviewing a proposal that asserts a shape without showing it.

## The levels, and picking one

C4 is four nested levels of zoom. Each diagram is at **exactly one** level, and
saying which is most of the value.

| Level | One diagram shows | Audience | Boxes are |
|-------|-------------------|----------|-----------|
| 1. **Context** | one system and who/what it talks to | anyone, including non-technical | people and systems |
| 2. **Container** | the deployable/runnable pieces inside that system | technical | applications, services, data stores |
| 3. **Component** | the pieces inside one container | developers on that container | groupings of code |
| 4. **Code** | classes, functions | rarely worth drawing | code |

**Draw levels 1 and 2. Stop.** Level 3 earns its keep for one complicated
container; level 4 goes stale before it is merged and a reader can get it from
the code. A diagram set that stops at container and is *current* beats a
complete set that is wrong.

"Container" means a separately runnable or deployable thing — a service, a CLI,
a database, a browser app. It is not a Docker container, though it is often
shipped as one. This is the most misread word in C4.

## Instructions

### Step 1: State the level and the scope in one sentence

Before drawing: *"Container diagram for the documentation pipeline."* If you
cannot write that sentence, the diagram will mix levels — the usual failure,
where a box called "Postgres" sits beside a box called "the retry helper".

### Step 2: Write it with the bundled C4 macros

PlantUML ships C4 in its standard library, so `!include <C4/...>` needs **no
network at render time** — verified against the transform image with networking
disabled. Do not use the `!include https://raw.githubusercontent.com/...` form
you will find in most examples online: it makes every render depend on GitHub
being reachable, which fails in a locked-down CI runner.

```
@startuml
!include <C4/C4_Container>
LAYOUT_WITH_LEGEND()
title Container diagram — Bartleby documentation pipeline

Person(author, "Engineer", "Writes documentation")
System_Boundary(bartleby, "Bartleby") {
  Container(cli, "bartleby CLI", "Go", "Resolves config, runs the transform")
  Container(tf, "transform image", "Python, Sphinx, LaTeX", "Renders documents")
}
System_Ext(registry, "ghcr.io", "Image registry")

Rel(author, cli, "runs")
Rel(cli, tf, "starts a container", "Docker API")
Rel(cli, registry, "pulls the image", "HTTPS")
@enduml
```

The includes for each level:

| Level | Include | Main macros |
|-------|---------|-------------|
| Context | `<C4/C4_Context>` | `Person`, `System`, `System_Ext`, `Rel` |
| Container | `<C4/C4_Container>` | `Container`, `ContainerDb`, `ContainerQueue`, `System_Boundary` |
| Component | `<C4/C4_Component>` | `Component`, `ComponentDb`, `Container_Boundary` |
| Deployment | `<C4/C4_Deployment>` | `Deployment_Node`, `Node` |

`_Ext` variants mark things outside your control, and they matter: the boundary
between what you own and what you merely call is usually the most interesting
thing on the page.

### Step 3: Label every box and every line

The macros take a description for a reason. Fill it in.

- **Box**: name, technology, and one line of what it does. `Container(db,
  "Metering store", "Postgres 16", "Holds raw usage events")` — not
  `Container(db, "Database")`.
- **Line**: what the relationship *is*, and how. `Rel(cli, tf, "starts a
  container", "Docker API")`. An unlabelled arrow is a claim that two things are
  connected somehow, which the reader already assumed.

Direction matters too: `Rel(a, b, ...)` reads "a does something to b". Use
`Rel_Back` rather than reversing the arguments, so the sentence still reads
correctly.

### Step 4: Render it through the pipeline

Two ways, and the choice is about where the diagram belongs:

**Inline in a document** — the diagram is part of the prose:

```rst
.. uml::

   @startuml
   !include <C4/C4_Container>
   ...
   @enduml
```

**A standalone `.puml` file** — the diagram is an artifact in its own right,
reused across documents or handed to someone:

```bash
bartleby puml     # renders docs/**/*.puml to target/bartleby/puml_images/
```

Inline is right for one diagram explaining one passage. Standalone is right when
the same picture belongs in several places, or when someone will want the PNG.

### Step 5: Keep it honest

- **One level per diagram.** If a box needs opening up, that is a second
  diagram, not a bigger first one.
- **Nine boxes is a lot.** Past that, either the scope is too wide or there is a
  boundary you have not drawn.
- **Date it or link it.** A diagram in a repository that renders on every build
  stays current because it is next to the code; a diagram exported to a slide
  deck does not.
- **Say what is not shown.** "Authentication is omitted; see the identity
  context diagram" is worth a line, because a reader cannot tell the difference
  between "absent because irrelevant" and "absent because forgotten".

## Beyond the four levels

C4 has supplementary diagrams worth knowing:

- **System Landscape** — several systems and how they relate; useful when your
  work spans more than one system.
- **Dynamic** — a numbered sequence over the same boxes, for explaining a flow.
- **Deployment** — which containers run where. `<C4/C4_Deployment>`.

Use them when the question is "how does this flow" or "where does this run",
which a static container diagram answers badly.

## Attribution

The **C4 model** is Simon Brown's — <https://c4model.com>. The PlantUML macros
are **C4-PlantUML** (<https://github.com/plantuml-stdlib/C4-PlantUML>), MIT
licensed, and bundled in PlantUML's standard library, which is why the
`!include <C4/...>` form works offline.

This skill is a restatement in our own words. Read c4model.com; it is short.
