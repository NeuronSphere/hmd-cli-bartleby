.. Traceability process requirements

The Traceability Process
========================

The traceability tooling is part of the product, so it carries requirements of
its own and its tests are traced the same way everything else is.

.. req:: Parse requirements from the documentation
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_001
    :status: implemented

    The tooling shall read every ``req`` and ``spec`` item under
    ``docs/requirements``, capturing each item's ID, type, title, status, tags,
    links, and source location. The documents are the source of truth; nothing
    else may declare a requirement.

.. req:: Read coverage from the test sources
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_002
    :status: implemented

    Coverage shall be read where the tests are: a ``Requirements:`` line in the
    doc comment of a Go test function, and requirement tags in a Robot test's
    ``[Tags]``. A short ID shall be expanded with the repository prefix so that
    annotations stay readable.

.. spec:: Only real test functions count
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_TRACE_002
    :status: implemented

    Go tests shall be identified through the syntax tree — a function named
    ``TestXxx`` taking ``*testing.T`` — so that a helper, a benchmark, or a
    comment mentioning a requirement inside a function body is not mistaken for a
    coverage claim.

.. req:: Fail on a requirement no test verifies
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_003
    :status: implemented

    A requirement with no covering test shall fail the check, unless it is tagged
    ``trace-exempt``. This is the check the whole exercise exists for: NERD001's
    specification was marked implemented while the behaviour it describes was
    broken, and nothing was watching.

.. req:: Fail on a test that verifies nothing
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_004
    :status: implemented

    A test that declares no requirement shall fail the check, so that new tests
    either trace to a requirement or prompt one to be written.

.. req:: Fail on a reference that does not resolve
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_005
    :status: implemented

    A test claiming a requirement that does not exist, a duplicate requirement
    ID, and a ``:links:`` target that does not exist shall each fail the check.
    Without this, renaming a requirement would quietly drop its coverage.

.. req:: Generate the matrix reproducibly and detect staleness
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_006
    :status: implemented

    The traceability page shall be generated deterministically from the parsed
    model, and the check shall fail when the committed page does not match what
    the current requirements and annotations would produce. The page is committed
    because the documentation has to build inside the transform container, which
    has no Go toolchain.

.. spec:: Test item identifiers are stable
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_006_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_TRACE_006
    :status: implemented

    A generated test item's ID shall be derived from its suite and name rather
    than its position, so adding one test does not renumber the rest and produce a
    diff across the whole page.

.. req:: Report problems with a location and a fix
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_007
    :status: implemented

    Every reported problem shall name the file and line it concerns and say what
    to do about it, and problems shall be reported in a stable order so the output
    is comparable between runs.

.. req:: Be usable without taking the CLI on
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_008
    :status: implemented

    The traceability tool shall be a self-contained module — its own ``go.mod``,
    licensed Apache-2.0, with no dependency outside the Go standard library — so
    that a project can adopt it without adopting Bartleby or a BSL dependency.

    This is why it lives in ``src/go/reqtrace`` rather than inside the CLI. All
    three conditions have to hold for the carve-out to mean anything, so the
    tests assert the module path, the absence of ``require`` directives, and the
    licence itself rather than trusting review to notice.

.. req:: Report which build is running
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_009
    :status: implemented

    ``reqtrace -version`` shall report the version, injected at build time. A
    binary built without the injection shall report ``dev``, so a local build is
    distinguishable from a released one.

.. spec:: Distributed as its own Homebrew cask
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_008_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_TRACE_008
    :status: implemented
    :tags: trace-exempt

    ``reqtrace`` shall be installable as ``brew install neuronsphere/tap/reqtrace``
    — its own cask, not a second binary inside Bartleby's, since two casks cannot
    both link the same binary name and bundling it would foreclose installing it
    alone.

    *Verification:* by inspection of ``.goreleaser.yaml`` and a rehearsed release
    (``goreleaser release --snapshot --skip=publish``, which writes both casks),
    plus installing from the tap. Exercising it in a unit test would mean
    publishing a release.

.. req:: Reachable from the CLI as well
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_010
    :status: implemented

    ``bartleby reqs`` shall generate the matrix and ``bartleby reqs --check``
    shall check it, with the same ``--repo`` and ``--quiet`` behaviour as the
    standalone tool. Someone who already has the CLI shall not need to install a
    second binary.

    Both entry points shall call one function rather than each orchestrating the
    steps, so that they cannot drift into disagreeing about what a check means.

.. req:: Work in a repository that has no Go in it
    :id: HMD_CLI_BARTLEBY_REQ_TRACE_011
    :status: implemented

    A repository with no Go tree shall be read as having no Go tests, not as an
    error. Coverage shall come from whatever test sources the repository does
    have.

    Found by adopting the tool in ``hmd-tf-bartleby``, which is Python and Robot:
    it failed outright with ``lstat src/go: no such file or directory``, so the
    tool was unusable in precisely the kind of repository ``REQ_TRACE_008``
    exists to serve.
