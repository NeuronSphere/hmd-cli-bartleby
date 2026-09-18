.. NERD002 VS Code Needs Graph Extension

NERD002 VS Code Needs Graph Extension
=====================================

.. req:: Show the Sphinx-Needs graph across every repository in a VS Code workspace
    :id: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    A VS Code extension shall present the sphinx-needs objects of every
    NeuronSphere repository open in the workspace as one navigable graph, and
    shall give the editor hover, go-to-definition, find-references,
    diagnostics and CodeLens on need identifiers. The parsing and indexing
    behind both shall be a language server built on the ``reqtrace`` module in
    this repository, so that the command line, ``nsx`` and the editor read one
    parser and disagree about nothing.

This document is the design target. It records the research that shaped it,
the decisions already taken, the architecture, and a staged roadmap. Nothing
in it is implemented; the specifications are ``proposed`` and the first
milestone is a change to ``reqtrace`` that is useful before any editor code
exists.

Motivation
----------

Sphinx-Needs has become the substrate for requirements, specifications and
decisions across NeuronSphere. As of September 2026, 32 of the ~430
repositories under ``projects/`` carry needs — about 2,900 objects (1,857
``spec``, 679 ``req``, 220 ``test``, 29 ``decision``, 16 ``feature``) —
rendered by Bartleby and checked by ``reqtrace``. The ``nsx`` agent teams hand
off to one another by writing needs.

Yet the only ways to *see* the graph are to build the HTML in Docker and read
the ``needtable`` directives, or to query ``nsx_needs search`` from inside a
run. Nothing shows the graph live, in the editor, or across repositories. The
facts below were verified against the working tree on 2026-09-18 and
constrain the design.

**What reqtrace does today.** ``src/go/reqtrace`` is a stdlib-only,
Apache-2.0 Go module with its own ``go.mod``, tagged
``src/go/reqtrace/v0.1.0``, released as its own binary, archive and Homebrew
cask by this repository's GoReleaser configuration. Its parser
(``parse.go``) matches only ``.. req::`` and ``.. spec::`` by line pattern,
reads ``:id:``, ``:status:``, ``:tags:`` and ``:links:``, records a line
number but no byte span and no body, and scans only ``docs/requirements/``
— NERDs in ``docs/proposals/`` and features in ``docs/features/`` are outside
it. ``Scheme.ExpandID`` already tolerates a sibling repository's identifiers,
and ``Requirement.Prefix`` is carried per item so that "a model may hold
items from more than one repository", but nothing loads more than one.
``Validate`` reports six problem kinds; the only output is the generated
``traceability.rst``.

**What nsx has built beside it.** ``neuronsphere/src/go/nsx/internal/needs``
(~2,500 lines) parses a closed seven-type vocabulary — ``feature``, ``req``,
``spec``, ``test``, ``impl``, ``question``, ``decision`` — from both
reStructuredText directives and MyST Markdown fences, records each need's
byte span, preserves options it does not model, and writes needs back without
disturbing the file around them. The decision that adopted ``reqtrace`` for
its scheme, test parsers and validator deliberately kept this parser, pinned
by a test that fails if ``nsx`` ever calls ``reqtrace.ParseRequirements``,
and recorded the intent to contribute it upstream once it settled. A task
opened on 2026-09-17 already plans, *in reqtrace*, a ``needs.json`` loader
and graph helpers (``Neighbourhood``, ``Closure``, ``ByType``, ``Sections``)
with ``nsx`` as first consumer and ``bartleby reqs`` second. An editor is the
natural third.

**How Sphinx is configured.** There is one ``conf.py``, inside the
``hmd-tf-bartleby`` image, with ``sphinx_needs`` 8.5.0 loaded and
``needs_build_json = True``. No repository has a ``docs/conf.py``. Per-repo
``needs_*`` settings live in ``meta-data/manifest.json`` under
``bartleby.roots.<name>.config``; three repositories set any — this one and
``hmd-tf-bartleby`` set ``needs_id_regex`` and ``needs_warnings``, and
``neuronsphere`` declares a full ``needs_types`` list with prefixes and
colours. A Sphinx build writes ``target/bartleby/<builder>/needs.json``; one
repository has such a file today.

**How needs are actually written.** ``:links:`` is the only link type in use
— no repository configures ``needs_extra_links`` — and the inline
``:need:`` role is the other reference form. Tags are occasionally
semicolon-separated. Identifier conventions vary by era:
``HMD_CLI_BARTLEBY_REQ_AREA_001`` (current), ``NEURONSPHERE_NSX_REQ_WS_001``,
``NERD0004_HMD_MS_DEPLOYMENT_SPEC0001`` (legacy, repository as suffix), and
bare ``NERD001`` through ``NERD005``, which are defined independently in four
repositories. There are **no cross-repository** ``:links:`` anywhere. So the
graph will at first be a forest of per-repository components, and the
cross-repository value is seeing them side by side, catching collisions, and
making future cross-repository links possible and checked. Scans must skip
``target/``, ``build/``, the vendored ``src/go/nsctl/.artifacts/`` tree and
``hmd-tf-bartleby``'s ``test/output_files*`` fixtures.

**What the organisation has for editors.** No ``.vscode/`` directory,
``.code-workspace`` file, or VS Code extension exists in any repository.
``hmd-ui-graphs`` is built on GoJS, which is commercially licensed and cannot
ship inside an Apache-2.0 extension.

Decisions taken
---------------

Three questions were put to the maintainer and settled before this design
was written.

.. list-table::
   :header-rows: 1
   :widths: 20 80

   * - Question
     - Decision
   * - Where does parsing live?
     - In a **Go language server built on** ``reqtrace``. A parser written in
       TypeScript would be a third one, drifting from the two that exist; a
       ``needs.json``-only approach is Sphinx-accurate but stale until the
       next Docker build and available for one repository.
   * - What is in the first version?
     - The **graph webview and the editor intelligence** together — hover,
       definition, references, diagnostics, CodeLens. The language server
       provides them almost for free once it exists; leaving them out would
       be leaving the cheap half of the value on the table.
   * - Which repositories form the workspace?
     - **VS Code workspace folders.** Each folder that carries
       ``meta-data/manifest.json`` is one repository. A multi-root
       ``.code-workspace`` file selects the set; a recursive scan is an
       option for later.

Architecture
------------

Three tiers, one parser. The extension host speaks the Language Server
Protocol to ``reqtrace lsp`` over stdio; the graph webview talks to the
extension host over ``postMessage``; every question about a need is answered
by the server.

.. uml::

   @startuml
   skinparam componentStyle rectangle
   skinparam defaultFontName Helvetica

   package "VS Code" {
     [Extension host\n(TypeScript)] as host
     [Graph webview\n(Cytoscape.js)] as web
     [Editor surfaces\nhover · definition · references\ndiagnostics · CodeLens · tree · status bar] as editor
     host -down-> web : postMessage
     host -up-> editor : vscode API
   }

   package "reqtrace lsp (Go, stdlib-only, Apache-2.0)" {
     [lsp] as lsp
     [workspace] as ws
     [graph] as graph
     [needs] as needs
     [needsjson] as nj
     lsp --> ws
     ws --> graph
     graph --> needs
     ws ..> nj : optional overlay
   }

   host -right-> lsp : JSON-RPC over stdio\nLSP 3.17 + needs/*

   database "Workspace folders" as repos {
     [repo A\ndocs/**/*.rst *.md\n*_test.go  test/*.robot\nmeta-data/manifest.json]
     [repo B ...]
   }
   ws --> repos : read + file watch

   [nsx] as nsx
   [reqtrace CLI\nbartleby reqs] as cli
   nsx --> needs : go.mod
   nsx --> graph
   cli --> needs
   @enduml

Dependencies point one way — ``lsp → workspace → graph → needs`` — and the
existing root package of ``reqtrace`` is not changed by any of them, so a
consumer pinned to ``v0.1.0`` keeps working through every milestone.

.. spec:: The language server is a subcommand of the reqtrace binary
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The server shall be invoked as ``reqtrace lsp --stdio`` (with
    ``--socket`` for debugging) and shall live in new sub-packages of the
    existing nested module at ``src/go/reqtrace``: ``needs/``, ``needsjson/``,
    ``graph/``, ``workspace/`` and ``lsp/``. It shall keep the module
    stdlib-only and Apache-2.0.

    *Why here.* The module already ships as ``reqtrace_<version>_<os>_<arch>``
    archives and a Homebrew cask from this repository's GoReleaser
    configuration, so the server needs no new release plumbing, and one
    binary serves the command line, ``nsx`` (through ``go.mod``) and the
    editor. A ``bartleby lsp`` subcommand was rejected: Bartleby is
    BSL-licensed — the GoReleaser file says so in as many words when it
    explains why ``reqtrace`` is installable on its own — and its binary
    carries the Docker client. A separate third module or repository was
    rejected because it would version the server apart from the parser it
    depends on.

    The JSON-RPC framing and the roughly twenty LSP 3.17 structures the server
    uses shall be written by hand rather than imported: the available Go LSP
    libraries are either stale or bring logging and encoding dependencies
    that would end the module's stdlib-only promise, and the surface needed
    here is small and stable.

.. spec:: nsx's span-recording parser is lifted into reqtrace/needs
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    ``reqtrace/needs`` shall be created by lifting ``nsx/internal/needs``
    (``needs.go``, ``parse.go``, and in a second step ``write.go``) and
    generalising it: the closed type vocabulary becomes a ``Parser.Types``
    field, the ``exitcode`` wrapping is dropped in favour of plain errors,
    and a ``Need.AsRequirement(scheme)`` conversion keeps ``Validate``
    working unchanged. The parser shall record, per need, the span of the
    whole directive, of the ``:id:`` value and of the title, one ``Ref`` per
    ``:links:`` token, the enclosing section headings, and the body; and per
    document, every inline ``:need:`` role. A golden corpus of tricky files
    copied from real repositories shall pin its behaviour.

    *Why now.* It is the only parser with byte spans, Markdown support,
    wrapped-option handling and a test suite, and lifting it was already
    ``nsx``'s stated intent. ``nsx``'s dependency test forbids the *old*
    ``reqtrace`` functions by name, so a new package is not blocked; once the
    lift lands that test flips to *require* ``reqtrace/needs`` and ``nsx``
    deletes its copy. Three consumers on one parser is the point.

    When a repository's manifest declares ``needs_types``, that list replaces
    the default vocabulary, as it does in Sphinx, and a directive outside it
    is diagnosed. When it declares none, the default is the union of the
    sphinx-needs defaults (``req``, ``spec``, ``impl``, ``test``) and the
    ``nsx`` vocabulary (``feature``, ``question``, ``decision``); a strict
    mode limited to the Sphinx defaults is an opt-in setting.

.. spec:: Needs are keyed by repository and identifier in a multi-repository graph
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    ``reqtrace/graph`` shall key every node as ``{repo, id}``, where
    ``repo`` is the manifest ``name`` and ``id`` is the literal text in the
    file. Link resolution shall try the linking repository first, then a
    unique sibling, and otherwise report the target as *ambiguous* (defined
    in several other repositories) or *unknown*. The four repositories that
    each define a bare ``NERD001`` therefore see four unrelated nodes and no
    problem; a fifth repository linking ``NERD001`` receives an
    ``ambiguous-link`` diagnostic naming the candidates. Edges resolved to a
    sibling are marked as cross-repository so the webview can style them.

    The package shall offer the queries the vault task already names —
    ``Neighbourhood(key, hops, direction, filter)``, ``Closure``, ``ByType``,
    ``Backlinks``, ``Sections`` — plus ``Whole(filter)`` with a node cap and
    a ``truncated`` flag, and a JSON-tagged ``Subgraph{nodes, edges}`` that is
    the exact payload the webview receives. Generated ``test`` nodes are
    synthesised from the existing Go and Robot test parsers through
    ``TestCase.NeedID()``, so the committed ``traceability.rst`` is skipped
    rather than parsed twice.

    ``reqtrace/workspace`` shall discover repositories from workspace folders
    (reading ``name``, ``bartleby.roots.*.config.needs_types`` and
    ``needs_id_regex`` from the manifest), treat one file as the unit of
    re-parse, let open editor buffers override disk, and rebuild the indexes
    in full on every change — at three thousand needs that is well under a
    millisecond, so no incremental graph algebra is warranted.

.. spec:: The server implements the standard LSP features on need identifiers
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    Over a document selector of reStructuredText, Markdown, ``*_test.go``
    and ``*.robot`` files, the server shall provide:

    .. list-table::
       :header-rows: 1
       :widths: 25 75

       * - Method
         - Behaviour
       * - ``textDocument/hover``
         - Type, title, status, tags, repository, the first paragraph of the
           body, link/backlink/test counts, and a "Show in graph" command
           link — on a definition, a ``:links:`` token, a ``:need:`` role, a
           Go ``// Requirements:`` annotation or a Robot ``[Tags]`` cell.
       * - ``textDocument/definition``
         - Identifier → the ``:id:`` span of its need; a generated
           ``TEST_GO_*`` / ``TEST_ROBOT_*`` identifier → the test function or
           case.
       * - ``textDocument/references``
         - Every reference across every repository, all kinds.
       * - ``textDocument/documentSymbol``, ``workspace/symbol``
         - Needs nested under their sections; fuzzy search over identifier
           and title, labelled ``ID (repo)``.
       * - ``textDocument/codeLens``
         - Above each directive: links, backlinks, covering tests, "Show in
           graph"; above a test: the requirements it covers.
       * - ``textDocument/completion``
         - Inside ``:links:`` and ``:need:``, candidates from the workspace,
           same repository first.
       * - ``textDocument/publishDiagnostics``
         - The six existing ``reqtrace`` problem kinds, anchored to spans
           (``Problem`` gains ``File``, ``Line`` and ``Span``), plus
           ``ambiguous-link``, ``unknown-need-type``, ``invalid-id`` (only
           where the manifest declares ``needs_id_regex``), ``missing-id`` and
           ``unresolved-role``. ``requirement-without-test`` is a warning only
           in repositories that have at least one annotated test, and off
           elsewhere, so the 2,900 needs written before traceability was
           adopted do not light up the Problems panel.

    Rename is deferred: it edits Go and Robot files across repositories and
    deserves its own review. The server never writes a file; ``needs/export``
    returns JSON for the client to save.

.. spec:: Custom needs/* requests feed the graph webview
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC005
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    Beyond the standard protocol the server shall answer ``needs/repos``,
    ``needs/graph``, ``needs/neighbourhood``, ``needs/need``,
    ``needs/needAt`` (identifier under a cursor), ``needs/search``,
    ``needs/reindex``, ``needs/export`` and, later, ``needs/compare``; and
    shall send ``needs/graphChanged`` (changed and removed keys, so the
    webview re-queries only what it holds) and ``needs/status`` (indexing
    state and problem counts, for the status bar). It shall register file
    watchers for ``**/*.{rst,md}``, ``**/meta-data/manifest.json``,
    ``**/*_test.go`` and ``**/test/**/*.robot``.

.. spec:: The webview renders neighbourhoods first and the whole graph on request
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC006
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The graph panel shall use Cytoscape.js (MIT) with the ``fcose`` and
    ``dagre`` layouts: its canvas renderer handles thousands of nodes,
    compound nodes give repository clusters for free, and its selector
    styling maps directly onto type, status and repository. D3-force,
    vis-network and ELK were weighed and rejected for, respectively, the
    amount to hand-build, weak hierarchical layout, and speed beyond a few
    hundred nodes.

    Opening the graph from a cursor, CodeLens or tree item shall show that
    need's two-hop neighbourhood in both directions. Opening it with no
    focus shall show a repository overview — one compound node per
    repository with type counts, since today there are no cross-repository
    links to draw between them. A *ladder* layout (``dagre``, ranked
    ``feature → req → spec → test/impl``, with ``decision`` and ``question``
    beside what they govern) serves filtered views; a *cluster* layout
    (``fcose`` with compound repositories) serves cross-repository views.
    The whole graph sits behind a configurable node cap, and the server's
    ``truncated`` flag prompts the user to filter.

    Nodes shall be shaped by type and coloured from the repository's
    ``needs_types`` colour when declared, else from a fixed theme-aware
    palette; bordered by status; ringed and badged when they carry problems.
    ``:links:`` edges are solid arrows, cross-repository edges dashed,
    test-coverage edges thin and grey, and generated test nodes hidden by
    default. Click opens a details panel; double-click opens the file at the
    ``:id:`` span; a context menu offers focus, expand one hop, collapse,
    hide and find-references; a filter bar covers repository, type, status,
    tag, area, tests, cross-repository-only and hop count; a "follow editor"
    toggle re-focuses on the need under the cursor. Filter, focus and layout
    persist in ``workspaceState``; the panel retains its context when hidden
    and restores after a window reload.

.. spec:: The extension contributes a tree, a status bar item and commands
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC007
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The extension shall add an activity-bar container with a tree of
    *repository → type (count) → need*, decorated red where problems exist,
    with open / show-in-graph / copy-identifier / find-references actions; a
    status bar item reading, for example, ``2,914 needs · 12 problems``,
    spinning while indexing and red while errors exist; and the commands
    ``needs.showGraph``, ``showGraphAtCursor``, ``showGraphForRepo``,
    ``reindex``, ``check`` (a task running ``reqtrace -check`` per repository
    with a problem matcher for its ``file:line: kind: message`` output),
    ``generateTraceability``, ``exportNeedsJson`` (a ``needs.json`` without
    Docker), ``openNeed``, ``copyId``, ``restartServer`` and
    ``showServerLog``. It shall contribute the ``restructuredtext`` language
    identifier for ``.rst`` under the same name the widely installed
    reStructuredText extension uses, so the two coexist.

    Settings live under ``needs.*``: ``server.path``, ``server.version``,
    ``server.download``, ``docsDirs``, ``excludeGlobs`` (defaulting to the
    directories listed under *Motivation*), ``types.fallback``,
    ``types.strict``, ``diagnostics.uncovered``, ``codeLens.enabled``,
    ``graph.defaultHops``, ``graph.maxNodes``, ``graph.layout``,
    ``graph.showTests``, ``needsJson.overlay``, ``discovery`` and
    ``trace.server``.

.. spec:: The extension lives in a new repository and obtains the server binary in a fixed order
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC008
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    The extension shall be a new repository, ``hmd-vscode-needs``, with the
    standard layout (``meta-data/manifest.json``, ``src/typescript/``,
    ``docs/requirements/`` traced by ``reqtrace`` itself), published as
    ``neuronsphere.needs``. Placing it under this repository's
    ``src/typescript/`` was rejected because the tree is BSL-licensed and a
    Marketplace extension with a Node and ``vsce`` toolchain does not belong
    in a Go and Docker CLI's release.

    On activation the extension shall locate the server in this order: the
    ``needs.server.path`` setting; ``reqtrace`` on ``PATH`` if
    ``reqtrace -version`` satisfies the range the extension was tested
    against; a cached download under the extension's global storage; a fresh
    download of the matching ``reqtrace_<version>_<os>_<arch>.tar.gz`` from
    this repository's GitHub release. Failing all four it shall show one
    actionable message pointing at the Homebrew cask. A single small VSIX
    thus serves every platform, Homebrew users download nothing, and remote
    and container workspaces work; per-platform VSIX packaging is a later
    option. Whether a file downloaded by Node escapes macOS quarantine, as
    expected, is to be verified in the first extension milestone.

.. spec:: needs.json is loaded now and overlaid later
    :id: HMD_CLI_BARTLEBY_NERD002_SPEC009
    :links: HMD_CLI_BARTLEBY_NERD002
    :status: proposed

    ``reqtrace/needsjson`` shall load the sphinx-needs export
    (``versions[""].needs``) in the first milestone, because ``nsx`` needs it
    for its run graph. The editor shall *not* depend on it: the live parse
    already yields links, backlinks, sections and bodies, and the data only
    Sphinx has — nested ``parent_need``, ``parts``, toctree-resolved
    docnames — is rare. Its value to the editor is fidelity: a later
    ``needs/compare`` request shall diff Sphinx's view against the live parse
    and report what is missing, extra or different, the same signal ``nsx``
    plans as ``docs.graph_mismatch``; a "Sphinx view" toggle in the graph and
    ``sphinx-mismatch`` diagnostics follow from it.

Roadmap
-------

Milestones are ordered so that the ``reqtrace`` work — which ``nsx`` and
``bartleby reqs`` benefit from on its own — lands first, and so that the
server is demonstrable from any LSP client before a line of TypeScript
exists.

.. list-table::
   :header-rows: 1
   :widths: 12 18 45 25

   * - Milestone
     - Repositories
     - Delivers
     - Demonstrated by
   * - M0 — reqtrace v0.2.0
     - hmd-cli-bartleby, neuronsphere
     - ``needs/``, ``needsjson/``, ``graph/``; ``Problem`` gains
       ``File``/``Line``/``Span``; ``reqtrace -format json``; golden corpus;
       ``nsx`` switches to ``reqtrace/needs``.
     - ``reqtrace -format json`` over this repository, ``neuronsphere`` and
       ``hmd-ms-deployment``, identifiers diffed against the one existing
       ``needs.json``; ``nsx`` tests green.
   * - M0.5 — v0.2.1
     - both
     - ``needs/write.go`` lifted; ``nsx`` deletes its copy.
     - ``nsx`` tests green on the upstream writer.
   * - M1 — v0.3.0
     - hmd-cli-bartleby
     - ``workspace/``, ``lsp/``, ``reqtrace lsp --stdio``.
     - Neovim or Helix attached to a two-repository workspace: hover and
       definition on ``NEURONSPHERE_NSX_REQ_WS_001``; a broken ``:links:``
       diagnosed; editing ``needs_types`` in a manifest toggles
       ``unknown-need-type``.
   * - M2 — extension scaffold
     - hmd-vscode-needs (new)
     - Contributions, server acquisition, language client, status bar,
       reindex/check/restart, CI packaging a VSIX on tag.
     - A three-repository ``.code-workspace``: hover, definition,
       references, Problems panel.
   * - M3 — graph
     - both
     - The ``needs/*`` requests; the Cytoscape webview with neighbourhood
       and overview modes, filters, details, open-at-span, follow-editor.
     - "Show graph at cursor" on a spec, expand a hop, double-click into a
       requirement in another repository; the four bare ``NERD001`` render as
       four nodes.
   * - M4 — navigation
     - both
     - Tree view, CodeLens, ladder layout, completion,
       ``generateTraceability``, ``exportNeedsJson``, ``openNeed``, the
       check task and problem matcher.
     - Tree → graph → editor round trip; a ``needs.json`` without Docker.
   * - M5 — fidelity and polish
     - both
     - ``needs/compare`` and the Sphinx overlay, rename, recursive
       discovery, per-platform VSIX, Marketplace and Open VSX publishing, a
       ``use-needs-graph`` skill shipped through ``bartleby skills``.
     - ``needs/compare`` against a fresh Docker build of ``neuronsphere``
       (379+ needs) reports nothing missing or extra.

Risks
-----

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - Risk
     - Mitigation
   * - Parser fidelity against docutils: directives inside literal blocks or
       comments, ``.. only::``, nested needs, ``:id:`` on a continuation
       line, MyST fences in ``.md`` files Sphinx is not configured to parse.
     - The golden corpus drawn from the 32 repositories; ``needs/compare``
       in M5 as the honest measure; skip the deeper-indented block after a
       line ending in ``::``; document the known gaps.
   * - Webview layouts stall above ~1,500 nodes.
     - Neighbourhood-first defaults, the node cap and ``truncated`` flag,
       asynchronous layouts, a Web Worker for ``fcose`` if needed.
   * - Binary distribution: unsigned macOS binaries, corporate proxies,
       offline machines, no Windows build.
     - The four-step acquisition order; verify the quarantine behaviour;
       Windows deferred until someone in the organisation runs it.
   * - Three consumers — the CLI, ``nsx``, the server — on one parser.
     - Contract tests in ``reqtrace``; ``nsx``'s dependency test inverted to
       *require* the shared package; semantic versioning on the nested
       module; ``bartleby reqs`` keeps calling ``reqtrace.Run``.
   * - Breaking ``v0.1.0`` users.
     - ``Load``, ``Validate`` and ``Run`` untouched through v0.3;
       ``ParseRequirements`` deprecated with a pointer, not removed.
   * - LSP positions are UTF-16 code units, spans are bytes.
     - One conversion routine, tested with non-ASCII titles.
   * - One repository with several ``bartleby.roots`` declaring different
       ``needs_types``.
     - Union them per repository; revisit if a real case appears.

Open questions
--------------

Each carries the recommended answer; none blocks M0.

1. **Discovery.** Ship ``needs.discovery: "recursive"`` — a depth-limited
   scan for ``meta-data/manifest.json`` so opening ``projects/`` alone works
   — in M1 rather than M5? *Recommended: yes; it is a few dozen lines and the
   organisation's layout invites it.*
2. **Type fallback.** Lenient union by default with strict as an opt-in, or
   strict by default so that a ``.. decision::`` in a repository that never
   declared it is flagged (a real bug finder, since the shared ``conf.py``
   would refuse to build it)? *Recommended: lenient default, ``types.strict``
   opt-in.*
3. **Windows.** Add ``windows/amd64`` to the ``reqtrace`` GoReleaser build
   now, or declare it unsupported? *Recommended: defer.*
4. **Naming.** Keep ``reqtrace`` as the name of the binary that is now also a
   language server, or rename it? *Recommended: keep it; a rename touches the
   cask, Makefiles, skills and ``nsx`` documentation for no functional gain.*
