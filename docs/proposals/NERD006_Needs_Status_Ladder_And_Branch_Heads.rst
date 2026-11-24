.. NERD006 One needs status ladder, and needs on branch heads

NERD006 One Needs Status Ladder, and Needs on Branch Heads
==========================================================

.. req:: The workspace needs index shall normalise every status to one ladder that includes approved, and may index needs on worktree checkouts and on matching branch heads, read through git objects
    :id: HMD_CLI_BARTLEBY_NERD006
    :status: approved
    :tags: nep058

    ``reqtrace``'s workspace needs index (``HMD_CLI_BARTLEBY_NERD004``)
    shall:

    - learn ``approved`` as a stage of its own;
    - normalise statuses to the single ladder NEP058 ("Work Model:
      Systems, Needs and Teams of Agents", section "Needs") sets for all
      tooling: ``draft`` → ``proposed`` → ``approved`` → ``in_progress`` →
      ``implemented``, with ``deferred`` and ``superseded`` closing a need;
    - optionally index needs written on worktree checkouts and on branch
      heads whose names match a pattern (such as ``nsx/*``), reading them
      from git objects and caching them in the index.

Created 2026-11-24, from NEP058 (``hmd-core-arch-docs``).

The workspace needs index is specified by ``HMD_CLI_BARTLEBY_NERD004``
("Workspace Needs Index"), which is implemented on the branch
``feat/reqtrace-workspace-index`` and not yet on ``main``. Its code is
``src/go/reqtrace`` (``stage.go`` for the stage table, ``index.go`` for
discovery and refresh). This NERD changes that code once it has landed; it
is written against ``main`` so that it can be approved on its own.

Motivation
----------

- **No approved stage.** ``stage.go`` maps ``approved`` to nothing: an
  approved need is ``unknown``, and a consumer cannot tell "waiting for
  approval" from "may be worked". NEP058 makes approval the gate between
  refinement and a batch, so the index must report it.
- **Several ladders.** The stages today are ``draft``, ``proposed``,
  ``in_progress``, ``implemented``, ``deferred`` and ``closed``, where
  ``closed`` lumps ``superseded`` with ``withdrawn`` and ``rejected``. NEP058
  sets one ladder for every tool (the nsx dashboard, the refinement
  wizard, the language pack's need noun), and ``superseded`` is a step of
  it.
- **Needs on branches are invisible.** ``index.go`` skips hidden
  directories (``eligibleDir``), so ``.worktrees/`` is never walked, and a
  repository is read only at its checkout. Needs drafted on a worktree
  branch, or on a branch a team of agents works (``nsx/<req-id>``), do not
  appear until merged. The operator's inbox needs them (NEP058, "Operator
  workflow"; ``HMD_CLI_NSX_REQ_DASH_007_SPEC002`` in ``hmd-cli-nsx``).

Specifications
--------------

.. spec:: ``approved`` is a stage, and the stages are one ordered ladder
    :id: HMD_CLI_BARTLEBY_NERD006_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD006
    :status: approved
    :tags: nep058
    :verify: go-test ./... (module src/go/reqtrace)

    The stages become, in ladder order: ``draft``, ``proposed``,
    ``approved``, ``in_progress``, ``implemented``; and the two closing
    stages ``deferred`` and ``superseded``. Plus ``unknown`` for a status
    nothing maps.

    The default table changes as follows; every other entry is unchanged:

    ===============  =================  ==================
    Raw status       Stage before       Stage after
    ===============  =================  ==================
    ``approved``     ``unknown``        ``approved``
    ``superseded``   ``closed``         ``superseded``
    ``withdrawn``    ``closed``         ``superseded``
    ``rejected``     ``closed``         ``superseded``
    ===============  =================  ==================

    ``closed`` is no longer a stage. A repository's
    ``needs_status_stages`` that maps a status to ``closed`` is read as
    ``superseded``, and ``reqtrace`` prints a deprecation notice naming the
    manifest on standard error once per repository, so existing manifests
    keep working. ``StageApproved`` and
    ``StageSuperseded`` are exported; ``StageClosed`` stays as a deprecated
    alias of ``StageSuperseded``. The raw status is always kept beside the
    stage, as today.

.. spec:: The listing reports where a need stands on the ladder, and needs refinement whatever its stage when it has blockers
    :id: HMD_CLI_BARTLEBY_NERD006_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD006_SPEC001
    :status: approved
    :tags: nep058
    :verify: go-test ./... (module src/go/reqtrace)

    Each item in ``reqtrace -format json`` gains ``rung``: the stage's
    position on the ladder (``draft`` 0, ``proposed`` 1, ``approved`` 2,
    ``in_progress`` 3, ``implemented`` 4), absent for ``deferred``,
    ``superseded`` and ``unknown``, so a consumer can ask "at least
    approved" without its own table.

    The readiness blockers are unchanged (no verification, an open
    question, placeholder-marker text, a requirement with no specification), and
    the listing keeps reporting them as it does. A consumer shows a need
    with blockers as needing refinement whatever its stage; the index does
    not change a stage because of a blocker.

    The existing filters follow the ladder: ``-stage`` accepts
    ``approved`` and ``superseded`` (and still accepts ``closed``, read as
    ``superseded``); ``-open`` keeps ``approved`` needs, as it keeps every
    stage below ``implemented``.

.. spec:: Needs on worktree checkouts and matching branch heads are indexed from git objects when asked
    :id: HMD_CLI_BARTLEBY_NERD006_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD006
    :status: approved
    :tags: nep058
    :verify: go-test ./... (module src/go/reqtrace)

    ``reqtrace -format json -heads <spec>`` (and the same option in the
    library call) adds, for each repository, the needs on other heads of
    the same repository. ``<spec>`` is a comma-separated list of:

    - ``worktrees``: the branch checked out in each of the repository's
      linked worktrees (``git worktree list --porcelain``), wherever the
      worktree lives, ``.worktrees/`` included;
    - a branch-name pattern such as ``nsx/*``, matched against
      ``refs/heads/``.

    Without ``-heads`` the listing is exactly as today. A head equal to the
    checkout's own ``HEAD`` is skipped.

    Needs on another head are read **from git objects** at that head's
    commit (``git ls-tree`` of the scoped paths, ``docs/`` and ``test/``,
    then ``git cat-file`` of the candidate files), never from the
    worktree's files, so an uncommitted edit in a worktree is not
    indexed and no worktree is walked. The parser and the needs
    vocabulary are the repository's own, from the manifest at that
    commit.

    Each item from another head carries ``head``: ``{"branch": "<name>",
    "commit": "<sha>", "worktree": "<path or empty>"}``. Items from the
    checkout carry none. An identifier on several heads is listed once per
    head; deciding which text to show is the consumer's.

.. spec:: Branch heads are cached in the index by commit
    :id: HMD_CLI_BARTLEBY_NERD006_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD006_SPEC003
    :status: approved
    :tags: nep058
    :verify: go-test ./... (module src/go/reqtrace)

    The on-disk index (``HMD_CLI_BARTLEBY_NERD004_SPEC003``) keeps, per
    repository and head, the commit it was read at and the needs found,
    keyed by blob id per file. A refresh lists the matching heads (one
    ``git for-each-ref`` and one ``git worktree list`` per repository) and
    re-reads a head only when its commit changed; within a changed head,
    only files whose blob id changed are parsed again. A head that no
    longer exists is dropped, and the change report
    (``HMD_CLI_BARTLEBY_NERD004_SPEC004``) lists its needs as removed. A
    test on a workspace of one repository with 20 branch heads checks that
    a second refresh with no commit changed runs no ``git cat-file``.
