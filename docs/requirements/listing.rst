.. Listing requirements

Listing
=======

Traceability (``TRACE``) writes a page for people to read. The listing
(``LIST``) is the same knowledge as data. It covers every need in one
repository or in a directory of repositories, with its status and the tests
that cover it, without a Sphinx build. Agents, memory indexes and review tools
read it, and it can say what changed since the last time anyone asked. NERD004
has the design.

.. req:: List needs as JSON without a build
    :id: HMD_CLI_BARTLEBY_REQ_LIST_001
    :status: implemented

    ``reqtrace -format json`` shall print one JSON document listing every
    sphinx-needs item found in the ``.rst`` files under the repository's
    ``docs/``, except the generated traceability page.

    Each item shall carry these fields:

    - ``repo``, ``id``, ``type`` and ``title``;
    - ``status`` and ``stage``;
    - ``tags`` and ``links``;
    - ``source``: the folder under ``docs/`` it came from, such as
      ``requirements`` or ``proposals``;
    - the repository-relative ``file`` and ``line``;
    - a ``digest``.

    The listing shall start no container and read no ``needs.json``. The
    directive types are the sphinx-needs defaults together with ``feature``,
    ``question`` and ``decision``, plus any ``needs_types`` a manifest
    declares.

.. req:: Report which tests cover each listed need
    :id: HMD_CLI_BARTLEBY_REQ_LIST_002
    :status: implemented

    Each listed need shall carry ``covered_by``: every Go test whose
    ``Requirements:`` comment names it, and every Robot test whose ``[Tags]``
    name it, found the way traceability finds them. Each entry gives the
    test's title, its kind and its file.

.. req:: Normalise statuses to stages
    :id: HMD_CLI_BARTLEBY_REQ_LIST_003
    :status: implemented

    Every listed need shall keep its raw ``status`` and gain a ``stage``.
    The stage comes from the default table, extended by a manifest's
    ``needs_status_stages``. Matching shall ignore case and surrounding
    white space, and spaces and hyphens shall match underscores.

    A non-empty status the table does not map shall get the stage
    ``unknown`` and an ``unknown-status`` problem. An empty status shall get
    ``unknown`` without a problem.

    When a manifest sets ``needs_status_strict``, traceability's ``-check``
    shall also fail on an unknown status.

.. req:: Discover the repositories in a workspace
    :id: HMD_CLI_BARTLEBY_REQ_LIST_004
    :status: implemented

    ``-workspace DIR`` shall list every repository found up to two levels
    below ``DIR``. A repository is a directory holding
    ``meta-data/manifest.json``, named by the manifest ``name``, or a
    directory holding ``docs/requirements``, ``docs/proposals`` or
    ``docs/features``, named after its folder.

    Hidden directories shall be skipped.

    With ``-format json`` and neither ``-repo`` nor ``-workspace``, run from a
    directory that has no manifest at or above it, ``reqtrace`` shall treat
    that directory as the workspace.

    The summary shall count the directories examined, the repositories with
    needs and those without.

.. req:: Resolve links across repositories and report what does not resolve
    :id: HMD_CLI_BARTLEBY_REQ_LIST_005
    :status: implemented

    Needs shall be keyed by repository and ID. A ``:links:`` target shall
    resolve first within its own repository, then to the one other repository
    that defines it.

    - A target several other repositories define shall be reported as
      ``ambiguous-link``.
    - A target nothing defines shall be reported as ``unknown-link``.
    - An ID defined twice in one repository shall be reported as
      ``duplicate-id``.

    Problems appear in the JSON. They do not change the exit status.

.. req:: Filter the listing and summarise it per repository
    :id: HMD_CLI_BARTLEBY_REQ_LIST_006
    :status: implemented

    These filters shall narrow ``needs``:

    - ``-repo NAME`` (repeatable; in a workspace it matches the repository
      name or its folder);
    - ``-stage STAGE`` (repeatable);
    - ``-type TYPE`` (repeatable);
    - ``-open``: every need whose stage is neither ``implemented`` nor
      ``closed``, and every implemented need with no covering test.

    ``-summary`` shall replace ``needs`` with per-repository counts by stage,
    with counts of covered and open needs.

.. req:: Keep a workspace index current without rescanning
    :id: HMD_CLI_BARTLEBY_REQ_LIST_007
    :status: implemented

    A workspace listing shall be served from an index file. It lives under
    ``$REQTRACE_CACHE_DIR``, or the user cache directory when that is unset,
    and there is one file per workspace path. The index is refreshed on
    every call as follows:

    - the workspace root's modified time, and that of each container
      directory, reveals added and removed repositories;
    - each recorded directory's modified time reveals added, removed and
      renamed files and directories;
    - each recorded file's modified time and size reveals edits;
    - a moved git ``HEAD``, or a changed manifest, causes that repository to
      be walked again.

    Only changed files shall be parsed again.

    The index shall be written atomically. An unreadable index, or one from
    another schema, shall be rebuilt. ``-rebuild`` shall ignore the index.
    If the index cannot be written, the listing shall still be printed, with
    a warning.

.. req:: Report what changed since the previous refresh
    :id: HMD_CLI_BARTLEBY_REQ_LIST_008
    :status: implemented

    ``-changes`` shall add ``changes`` to the listing: ``since``, then
    ``added``, ``changed`` and ``removed`` keys, measured against the index's
    previous refresh.

    A need is changed when its digest, its status or its covering tests
    changed. With no previous refresh, or with ``-rebuild``, every need is
    added.

.. req:: Compare two listings without a repository
    :id: HMD_CLI_BARTLEBY_REQ_LIST_009
    :status: implemented

    ``reqtrace -diff OLD NEW`` shall print the ``changes`` between two
    listings, using the same rule as ``-changes``. It shall refuse two
    listings whose ``schema`` differs.

    A single-repository listing shall contain no absolute path. Apart from
    ``generated``, two listings of the same tree shall be identical.

.. req:: Parse need directives with their bodies and a stable digest
    :id: HMD_CLI_BARTLEBY_REQ_LIST_010
    :status: implemented

    The listing's parser shall read a need directive's options, including
    an option value that wraps onto the next line, and its indented body.

    The digest shall be the SHA-256 of the directive's lines, with line
    endings normalised, trailing white space removed and trailing blank lines
    dropped, so that it does not depend on the machine or the path.

    Directives inside a literal block (after a line ending in ``::``) or
    inside a ``code-block`` shall be ignored.
