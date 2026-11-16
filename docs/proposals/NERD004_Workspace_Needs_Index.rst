.. NERD004 Workspace Needs Index

NERD004 Workspace Needs Index
=============================

.. req:: List every need across a workspace of repositories, with status and coverage, without a build
    :id: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    ``reqtrace`` shall list every sphinx-needs item in one repository or in
    a directory of repositories as structured JSON. Each item carries its
    status and the tests that cover it. The listing needs no Sphinx build.
    It is kept current by an incremental on-disk index, and it can report
    what changed since the last call, so a consumer handles only changed
    needs.

Created 2026-11-16. SPEC001–SPEC006 are implemented; see
``docs/requirements/listing.rst`` for the requirements the tests cite. This is milestone M0 of
:need:`HMD_CLI_BARTLEBY_NERD002`, which also uses this NERD's workspace
discovery.

Motivation
----------

Agents and people keep asking the same question: *what is open, and on
what?* Today the answer needs either a Docker build per repository, to get
``target/bartleby/html/needs.json``, or reading each repository's generated
``traceability.rst``. Neither scales to a workspace: a single directory
holding many repositories checked out side by side, where many sessions
start.

On 2026-11-16, a workspace holding a full checkout of the NeuronSphere
repositories contained:

- 478 directories, of which 91 declare ``.. req::`` or ``.. spec::`` items;
- about 4,067 such directives, and 2,413 candidate ``.rst``, ``.md`` and
  ``.robot`` files under ``docs/`` and ``test/``;
- about 4,000 directories under the scoped trees: 3,001 under ``docs/`` and
  ``test/``, and 983 under ``src/go``.

Indicative timings, measured with ``find`` and ``grep`` over that workspace
on a developer laptop:

- a grep for directives across every repository: 0.5 s;
- a stat of every candidate file: 0.3 s;
- an *unbounded* search for ``*_test.go``: 4.7 s, almost all of it spent
  walking directories that are not tests.

A rescan on every call is therefore too slow. Walking only scoped
directories, and re-parsing only what changed, is fast.

The first consumers are:

- agent memory (``hmd-core-arch-docs`` NEP047): the "what's open on
  <system>?" roll-up, and indexing changed need text into memory;
- business proposals (``hmd-core-arch-docs`` NEP052): a review shows the
  live status and coverage of the needs it touches.

Statuses in use on 2026-11-16, counted over every repository:

- ``implemented`` 1,904, ``proposed`` 1,847, ``in_progress`` 81, ``draft``
  81, ``completed`` 61;
- ``open`` 40, ``partial`` 29, ``amended`` 18, ``withdrawn`` 17, ``fixed``
  13, ``defined`` 13, ``recorded-difference`` 10;
- ``superseded`` 7, ``deferred`` 7, ``future`` 3, ``done`` 2, ``partly`` 2.

No ``conf.py`` declares ``needs_statuses``. Enforcing one vocabulary would
break existing builds, so this NERD *normalises* status and does not
enforce it.

Design
------

.. spec:: reqtrace -format json lists needs with status and coverage
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    ``reqtrace -format json`` shall print one JSON document to standard
    output with this shape::

        {"schema": 1,
         "generated": "<RFC 3339 time>",
         "workspace": "<directory name; workspace mode only>",
         "summary": {...},
         "needs": [...],
         "problems": [...]}

    Each entry in ``needs`` has these fields:

    - ``repo``, ``id``, ``type`` and ``title``;
    - ``status`` (the raw text), and ``stage`` (see
      :need:`HMD_CLI_BARTLEBY_NERD004_SPEC005`);
    - ``tags`` and ``links``;
    - ``source``: the folder under ``docs/`` the need came from, such as
      ``requirements``, ``proposals`` or ``features`` (``docs`` for a file
      directly under ``docs/``);
    - ``file`` (relative to the repository) and ``line``;
    - ``digest``: SHA-256 of the directive's normalised text;
    - ``covered_by``: a list of ``{test, kind, file}``, where ``kind`` is
      ``go`` or ``robot``.

    Coverage uses the matching ``traceability.rst`` already uses: Go
    ``// Requirements:`` comments and Robot ``[Tags]``.

    Every ``.rst`` file under ``docs/`` is read, apart from the generated
    traceability page, so ``docs/proposals`` and ``docs/features`` are always
    included in JSON output. Drafts and NERDs are exactly what consumers ask
    about, and ``source`` tells them apart. MyST Markdown is not read yet. ``-include-proposals`` keeps its present
    meaning for ``traceability.rst``.

    ``-json`` stays as the ``-verify`` output, so ``-format json`` is a new
    flag and no existing caller changes. The listing reads no
    ``needs.json`` and starts no container.

.. spec:: A directory of repositories is discovered as a workspace
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    ``-workspace DIR`` shall treat each directory under ``DIR``, to a depth
    of 2, that holds ``meta-data/manifest.json`` as a repository. Its name is
    the manifest ``name``. A directory with ``docs/requirements``,
    ``docs/proposals`` or ``docs/features`` but no manifest is also a
    repository, named after its folder.

    When ``reqtrace`` runs with neither ``-repo`` nor ``-workspace``, from a
    directory that has no manifest above it but has repositories below it,
    it behaves as ``-workspace .``. Opening a session in a workspace
    directory and asking for the listing then just works.

    Needs are keyed ``{repo, id}`` and links resolve as in
    :need:`HMD_CLI_BARTLEBY_NERD002_SPEC003`.

    When several folders share one manifest ``name``, for example a copy or
    a second worktree, each is listed as ``name@folder``, so their needs are
    never merged, and a ``duplicate-repo`` problem names the folders.

    Scans skip ``target/``, ``build/``, ``node_modules/``, ``vendor/``,
    ``.git/`` and the vendored and fixture trees listed in NERD002.

    ``summary`` counts the directories examined, the repositories with
    needs, and the repositories without, so that "none found" is never
    confused with "not scanned".

    The filters are ``-repo NAME`` (repeatable; inside a workspace it
    selects repositories, by name or folder), ``-stage STAGE`` (repeatable), ``-type TYPE``
    (repeatable) and ``-open``, which is shorthand for every stage that is
    not ``implemented`` or ``closed``, plus implemented needs with no
    coverage.

    ``-summary`` prints only per-repository counts by stage and coverage, so
    a caller can start small and then drill down.

.. spec:: An on-disk index is refreshed incrementally, without a watcher
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    Each workspace root has its own index file. It lives under
    ``$REQTRACE_CACHE_DIR``, or under ``os.UserCacheDir()/reqtrace`` when that
    is unset, and is named after a hash of the absolute workspace path.
    ``reqtrace`` assumes no particular directory layout. Wrapping tools may
    set ``REQTRACE_CACHE_DIR`` to place the index wherever their own
    configuration says.

    The index records:

    - for every repository: its ``name``, its git ``HEAD``, and the modified
      time of every directory in its scoped trees (``docs/``, ``test/`` and
      ``src/go/``), including scoped roots that do not exist yet, recorded as
      absent;
    - for every candidate file: its modified time, size, digest, and parsed
      needs and tests;
    - the modified time of the workspace root itself.

    A refresh, run on every call, proceeds in these steps:

    1. If the workspace root's modified time changed, list the top level
       again. A new repository gets a full scoped walk. A missing one is
       dropped.
    2. If a repository's ``HEAD`` moved (a branch switch, pull or rebase),
       walk its scoped trees in full again.
    3. Otherwise, stat every recorded directory. Read only the directories
       whose modified time changed: a new entry is parsed, or walked if it is
       a directory, and a vanished entry is dropped.

       This is how *new* files are found. On APFS and other POSIX file
       systems, a directory's modified time changes when an entry is added,
       removed or renamed directly inside it. The change does not reach its
       ancestors, so every directory is recorded, not only the roots.
    4. Stat every recorded file. Re-parse only those whose modified time or
       size changed. Editing a file does not change its directory's modified
       time, so this step catches edits.
    5. Rebuild the in-memory graph in full. At this scale that takes less
       than a millisecond, as NERD002 SPEC003 already notes.

    The index is written atomically, to a temporary file that is then
    renamed, so concurrent callers never see a torn file. An index that is
    unreadable, or from another schema version, is rebuilt without comment.
    ``-rebuild`` forces a cold build.

    The targets are a warm refresh under 0.5 s over a workspace the size of
    the one measured above, and a cold build of a few seconds.

    *Why no file watcher.* On macOS, ``fsnotify`` uses ``kqueue``, which
    holds one file descriptor for each watched directory. A workspace the
    size of the one measured above would need about 4,000. Descriptor
    exhaustion at that scale degrades other processes on the machine,
    including container runtimes. A watcher would also need a resident daemon and a dependency outside the
    standard library. The language server of NERD002 can keep this index
    warm in memory and take its change notices from the editor's
    ``workspace/didChangeWatchedFiles``, because the editor is already
    watching those files.

.. spec:: A refresh can report what changed since the previous one
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    ``-changes`` shall add a ``changes`` object to the JSON output:
    ``{since, added, changed, removed}``.

    - ``since`` is the time of the previous refresh of that index.
    - ``added``, ``changed`` and ``removed`` each list ``{repo, id}`` keys.
    - A need counts as ``changed`` when its digest, its status or its
      coverage changed.

    Consumers use this to process only what changed. The memory index
    re-embeds only changed need text, and a proposal review refreshes only
    the needs it touches.

    ``-changes`` with ``-rebuild`` reports every need as ``added``.

.. spec:: Raw statuses are normalised to a small set of stages
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC005
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    Every need keeps its raw ``status``. It also gets a ``stage``, taken
    from a default table. A repository can extend that table with
    ``needs_status_stages`` under ``bartleby.roots.<name>.config`` in its
    manifest.

    .. list-table::
       :header-rows: 1
       :widths: 20 80

       * - Stage
         - Raw statuses mapped to it by default
       * - ``draft``
         - ``draft``. Written by an agent or a business proposal and not yet
           approved by a person.
       * - ``proposed``
         - ``proposed``, ``open``, ``defined``, ``future``
       * - ``in_progress``
         - ``in_progress``, ``partial``, ``partly``, ``amended``
       * - ``implemented``
         - ``implemented``, ``completed``, ``done``, ``fixed``
       * - ``deferred``
         - ``deferred``
       * - ``closed``
         - ``withdrawn``, ``superseded``, ``rejected``

    Matching ignores case and surrounding white space. A status the table
    does not map gets the stage ``unknown`` and an ``unknown-status``
    problem. That problem fails ``-check`` only when the repository sets
    ``needs_status_strict: true``.

    The meaning of ``draft → proposed`` (a person approved it) belongs to
    this vocabulary and to the approval semantics in NEP052. It is not owned
    by any one tool that writes needs. Declaring ``needs_statuses`` in the
    shared ``conf.py`` is deferred until ``unknown`` is empty across the
    workspace.

.. spec:: Two listings can be compared without an index
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC006
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: implemented

    ``reqtrace -diff OLD.json NEW.json`` shall print the same ``changes``
    object as ``-changes`` (:need:`HMD_CLI_BARTLEBY_NERD004_SPEC004`),
    computed only from two listings produced by ``-format json``. It reads
    no repository and no index.

    This is the change feed for a consumer that has no working tree. For
    example, a service may receive one listing per built version as a build
    artifact and compare consecutive versions.

    For that to work, the listing's ``schema`` number and the meaning of
    ``digest`` are a contract:

    - The digest is computed from the directive's text with line endings
      normalised and trailing white space removed. It does not depend on
      file paths or on the machine.
    - A change to either rule increments ``schema``.
    - ``-diff`` refuses two listings with different ``schema`` numbers and
      says why.

    A single-repository listing (``-repo DIR -format json``) is
    self-contained. It holds no absolute paths and no timestamps other than
    a ``generated`` field, so a build can publish it as an artifact and two
    builds of the same commit produce the same needs.

Verification
------------

- A fixture workspace, built in a temporary directory by
  ``listing_test.go`` so that modified times behave as on a real disk. It
  holds:

  - a repository with a manifest, and one with ``docs`` and no manifest;
  - a repository with no needs, and one nested in a container folder;
  - a hidden folder;
  - a link from one repository to another, and an ambiguous ``NERD001``;
  - a duplicate ID, and two folders sharing a manifest name;
  - a Go cover and a Robot cover.
- Index tests: add, delete and rename a file; create a new
  ``docs/proposals/`` directory; add a Go test deep under ``src/go``; add and
  remove a repository, including one inside a container; move a
  repository's ``HEAD``; a corrupt index, ``-rebuild``, and an unwritable
  cache. Each must appear correctly in ``-changes`` and must re-parse only
  what changed.
- A Robot case in ``test/cli.robot`` runs ``bartleby reqs --format json``
  over a two-repository workspace, then checks that ``--changes`` reports
  exactly the one need edited.
- Against the real tree, on 2026-11-16:

  - **Against Sphinx's output.** Each of the 38 repositories with a
    ``needs.json`` from an earlier build was compared. 16 match exactly.
    Every other difference is a need whose source file changed after that
    build, or one whose source file has since been deleted. No difference
    pointed to a parser miss.
  - **Against ``grep``.** Over a full workspace, 4,083 ``req`` and ``spec``
    needs were found in 90 repositories. ``grep`` finds 4,084 directives in
    91; the one extra is a quoted example inside a ``code-block``, which is
    correctly skipped.
  - **Timings.** A cold build took 0.3–0.5 s and a warm refresh 0.05–0.07 s,
    with an index of about 4 MB.
- ``src/go/reqtrace/go.mod`` stays standard-library only.

Out of scope
------------

- Writing needs back to files: ``needs/write.go``, NERD002 M0.5.
- Pass and fail verdicts, which remain ``-verify``'s job
  (:need:`HMD_CLI_BARTLEBY_NERD003`). The listing reports coverage, not
  results.
- Any particular consumer, whether a memory store, a proposal app or an
  agent framework. They read the JSON.
