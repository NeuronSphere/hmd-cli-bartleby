.. NERD003 Verification Matrix

NERD003 Verification Matrix
===========================

.. req:: Report which requirements passed, against which implementation, and when
    :id: HMD_CLI_BARTLEBY_NERD003
    :status: implemented

    ``reqtrace -verify`` (and ``bartleby reqs --verify``) shall join the
    sphinx-needs ``needs.json`` from a bartleby build with the results of one or
    more test runs, and give each requirement one verdict per implementation,
    so a port from Python to Go is judged against the same requirements as the
    code it replaces.

Created 2026-10-26. The first slice is implemented beside this document; the
later items are proposed.

Motivation
----------

``reqtrace`` gives **traceability**: from the source tree, which tests cover
each requirement, checked in both directions by ``make check``. It says nothing
about whether those tests passed. The Go port of the NeuronSphere persistence
stack (``hmd-entity-storage`` NERD005) is judged by running the same tagged
suites against the Python and the Go implementation, and for regulated
customers the evidence that matters is **verification**: requirement, tests,
verdict, implementation, version, time.

Both halves already exist as build artefacts. ``hmd-tf-bartleby`` exports
``needs.json`` from every build since ``4a8bceb`` (2026-09-15), including the
test items ``reqtrace`` generates into ``traceability.rst`` with their
``links`` to requirements. Bender writes a timestamped ``output.xml`` for every
run, and ``go test -json`` gives the same for Go tests.

Design
------

.. spec:: Requirements and links come from needs.json
    :id: HMD_CLI_BARTLEBY_NERD003_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD003
    :status: implemented

    The requirement side is read from ``needs.json``, not the RST sources, so the
    verified set is what the published documentation declares. A result is
    matched to its test item by title: ``<suite>: <name>`` for Robot (the suite
    file's stem), ``<package>.<TestName>`` for Go, with a unique-name fallback for
    ``package main`` and external ``_test`` packages. Items declared in
    ``docs/proposals`` are left out unless ``-include-proposals`` is given, as
    for traceability.

.. spec:: Results are labelled with the implementation they ran against
    :id: HMD_CLI_BARTLEBY_NERD003_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD003
    :status: implemented

    ``-results LABEL[@VERSION]=PATH`` is repeatable. Files with the same label
    are one implementation, typically a Robot ``output.xml`` and a
    ``go test -json`` stream. Each run keeps its own timestamp (``generated`` in
    ``output.xml``, the first event time in Go output).

.. spec:: Verdicts, baseline counts and the exit code
    :id: HMD_CLI_BARTLEBY_NERD003_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD003
    :status: implemented

    Per requirement and implementation: *pass*, *fail*, *skip*, *missing* (no
    linked test ran) or *exempt* (``trace-exempt`` with no linked test). Tests
    tagged ``baseline`` — the ones that existed before a port began — are counted
    apart from new ones. The command writes
    ``docs/requirements/verification_matrix.rst`` (``-out`` to change, ``-`` for
    none), prints JSON with ``-json``, reports results for tests ``needs.json``
    does not declare, and exits 1 when any cell is *fail* or *missing*.

Usage
-----

.. code-block:: bash

   bartleby html                       # writes target/bartleby/html/needs.json
   hmd-bender                          # writes bender/output.xml
   go test -json ./... > go-test.json

   reqtrace -verify \
     -results python=bender-python/output.xml \
     -results go@0.4.0=bender-go/output.xml \
     -results go@0.4.0=go-test.json

Verified on ``hmd-cli-bender`` (2026-10-26): 27 requirements, all *pass* for
``go``, from 7 Robot and 43 Go results.

Proposed next
-------------

.. spec:: Keep each run's results as a dated artefact
    :id: HMD_CLI_BARTLEBY_NERD003_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD003
    :status: proposed

    A conformance run stores ``needs.json``, each ``output.xml`` and the JSON
    verification under one dated directory named for the implementation, so a
    verdict can be reproduced from what was kept rather than re-run.

.. spec:: Engines as a second axis
    :id: HMD_CLI_BARTLEBY_NERD003_SPEC005
    :links: HMD_CLI_BARTLEBY_NERD003
    :status: proposed

    For ``hmd-entity-storage`` NERD005 the matrix needs requirement ×
    implementation × engine. Until a dedicated field exists, the label carries
    both (``go-postgres``, ``python-dynamo``).

Open questions
--------------

1. **The published ``stable`` bartleby image predates needs.json.**
   ``ghcr.io/neuronsphere/hmd-tf-bartleby:stable`` is ``1.0.73``, built
   2026-09-11, before ``4a8bceb``; ``ghcr.io/hmdlabs/hmd-tf-bartleby:1.0.79``
   (2026-10-19) exports it. Until ``stable`` moves, set
   ``HMD_CONTAINER_REGISTRY=ghcr.io/hmdlabs HMD_TF_BARTLEBY_VERSION=1.0.79``.
2. Should the generated matrix be committed (like ``traceability.rst``) or kept
   only as a run artefact? Recommended: an artefact, because it describes runs,
   not the source tree; commit it only for a release's evidence.
