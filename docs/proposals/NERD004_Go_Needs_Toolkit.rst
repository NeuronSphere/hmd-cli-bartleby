.. NERD004 Go Needs Toolkit

NERD004 Go Needs Toolkit
========================

.. req:: Bartleby shall own the sphinx-needs format in Go: reading, writing, graph, digest and checks
    :id: HMD_CLI_BARTLEBY_NERD004
    :status: proposed

    Bartleby shall provide one Go module that reads and writes sphinx-needs in
    RST and MyST, builds their link graph, computes a need's approval digest,
    and checks approvals. ``reqtrace`` and every other Go consumer, nsx first,
    shall use it rather than parsing needs themselves. All of this shall run
    with no Sphinx, no container, no credentials and no network.

Created 2026-11-06, alongside nsx's ``HMD_CLI_NSX_NERD005``, which makes nsx a
consumer of this module and moves its needs code here.

Motivation
----------

There are two needs parsers today.

- **reqtrace** (``src/go/reqtrace/parse.go``) reads ``.. req::`` and ``..
  spec::`` from RST only, with ``:id:``, ``:status:``, ``:tags:`` and
  ``:links:``. It cannot write.
- **nsx** (``hmd-cli-nsx/src/go/nsx/internal/needs``, about 1,500 lines) reads
  all seven types (feature, req, spec, test, impl, question, decision), in RST
  and MyST. It keeps body and byte spans, writes amendments back, and builds
  the graph. It does this because reqtrace could not (see the header of its
  ``internal/reqs/reqs.go``).

The second one already imports reqtrace for the ID scheme and test
annotations, and pins a published tag, so the two drift. The format of a
repository's requirements is Bartleby's concern. Bartleby builds the documents,
generates ``traceability.rst`` and judges verification. Its Go tooling should
be the one place that knows the format.

nsx is also becoming the approval ledger (``HMD_CLI_NSX_NERD005``). An approval
is fields in a need plus a digest of its text, and the check that the digest
still matches belongs with the traceability check in CI, not in nsx.

Design
------

.. spec:: A ``needs`` module beside reqtrace
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: proposed

    ``src/go/needs`` is a new Go module,
    ``github.com/neuronsphere/hmd-cli-bartleby/src/go/needs``, standard library
    only, Apache-2.0 like reqtrace. It starts as nsx's ``internal/needs``
    moved here with its tests (``parse``, ``write``, ``graph``, ``render``,
    ``untrusted``). It provides:

    - **Parse:** RST directives and MyST fences, for every type in the
      repository's ``needs_types`` (the seven above by default), with options,
      body, file, line and byte span.
    - **Amend:** change a need's title, status, tags, links, body or arbitrary
      options in place, preserving everything else byte for byte, in either
      markup.
    - **Graph:** links in both directions, and what traces to a need.
    - **Vocabulary:** need types and statuses as data a caller can extend.

    nsx's own policy (who may set which status, the role words) stays in nsx.
    The module knows the format, not the workflow.

.. spec:: reqtrace parses through the needs module
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD004, HMD_CLI_BARTLEBY_NERD004_SPEC001
    :status: proposed

    ``reqtrace``'s ``ParseRequirements`` is reimplemented over the ``needs``
    module, so traceability reads MyST too and there is one parser. Its public
    API and the output of ``traceability.rst`` stay the same, and the existing
    reqtrace tests pass unchanged. The release is ``src/go/reqtrace/v0.2.0``
    and ``src/go/needs/v0.1.0``. Bartleby's own ``go.mod`` keeps its ``replace``
    to the sibling directories, as it does for reqtrace today.

.. spec:: The approval digest
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD004, HMD_CLI_BARTLEBY_NERD004_SPEC001
    :status: proposed

    ``needs.Digest(need)`` returns ``sha256:<hex>``, computed over a canonical
    form of what states the obligation:

    - the need's type, identifier, sorted links and body;
    - the body with trailing whitespace stripped per line and line endings
      normalised;
    - not its ``:status:``, its tags, or any ``approved-*``, ``approval-*``,
      ``accepted-*`` or ``acceptance-*`` field.

    The same text gives the same digest in RST and in MyST. Marking a need
    ``implemented`` leaves its digest unchanged, and editing its body or links
    changes it.

.. spec:: ``reqs --check`` reports lapsed approvals
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD004, HMD_CLI_BARTLEBY_NERD004_SPEC003
    :status: proposed

    For every need with ``:status: approved``, ``reqtrace -check`` (and
    ``bartleby reqs --check``) recomputes the digest and adds a problem with
    file and line:

    - ``approval-lapsed`` when it differs from ``:approval-digest:``;
    - ``approval-incomplete`` when ``:approved-by:``, ``:approved-at:`` or
      ``:approval-digest:`` is missing.

    ``superseded`` becomes a status that needs no test, like ``trace-exempt``.
    These are new problem kinds. Repositories that never write ``approved``
    see no change.

.. spec:: ``bartleby needs`` exposes the module on the command line
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC005
    :links: HMD_CLI_BARTLEBY_NERD004, HMD_CLI_BARTLEBY_NERD004_SPEC001
    :status: proposed

    ``bartleby needs`` works from the source tree, with no Sphinx build:

    - ``list [--type t] [--status s] [--json]``
    - ``show <id> [--json]``: options, body, location, links in and out, and
      the digest.
    - ``set <id> --status s [--option key=value ...]``: amends in place. It
      refuses the approval and acceptance fields, which only nsx writes.
    - ``graph <id> [--json]``: what the need links to and what traces to it.

    This gives agents and scripts a structured way to read and change needs
    without parsing RST, and nsx uses the same module in-process.

.. spec:: Signature verification is designed in nsx NERD006, not built here yet
    :id: HMD_CLI_BARTLEBY_NERD004_SPEC006
    :links: HMD_CLI_BARTLEBY_NERD004
    :status: proposed

    ``HMD_CLI_NSX_NERD006``, which is under review, proposes signed approvals:

    - a ``docs/requirements/approvers`` roster in OpenSSH ``allowed_signers``
      format;
    - an ``:approval-sig:`` field;
    - verification in pure Go in ``reqs --check``, with verdicts ``approved``,
      ``lapsed``, ``unverified`` and ``revoked``.

    If it is accepted, the verifier lands in this module as ``needs/approval``.
    Nothing in this NERD depends on it.

Migration
---------

1. Move ``internal/needs`` from nsx with its tests, under a new package path.
   No behaviour changes.
2. Point reqtrace's parser at it, and keep reqtrace's tests green.
3. Add the digest, the new ``--check`` problems and ``bartleby needs``.
4. Tag the modules. nsx drops ``internal/needs`` and imports ``needs``. During
   development it uses an uncommitted ``go.work``.

The requirements that describe the format today live in nsx's
``docs/requirements/process.rst`` and ``docprotocol.rst``. They move here
under ``HMD_CLI_BARTLEBY_REQ_NEEDS_*``, and nsx marks its copies
``superseded``.

Open questions
--------------

- Whether ``needs`` and ``reqtrace`` should become one module. Two modules keep
  reqtrace's existing tag and Homebrew release untouched.
- Whether ``bartleby needs set`` should also allocate identifiers, which nsx
  does today (``Stem`` in ``internal/reqs/convention.go``).
