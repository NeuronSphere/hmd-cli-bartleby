.. Verification requirements

Verification
============

Traceability (``TRACE``) says which tests cover a requirement, from the source
tree. Verification says whether those tests passed, against which
implementation, and when, from the artefacts a build and a test run leave
behind. It is how a port is judged: the same requirements, one verdict per
implementation. NERD003 has the design.

.. req:: Read the requirements and their test links from needs.json
    :id: HMD_CLI_BARTLEBY_REQ_VERIFY_001
    :status: implemented

    Verification shall take its requirements, and the links from each generated
    test item to the requirements it covers, from the ``needs.json`` that
    sphinx-needs exports from a bartleby build, not from the RST sources. What is
    verified is then exactly what the published documentation claims. As with
    traceability, items declared in ``docs/proposals`` are left out unless
    ``-include-proposals`` is given.

.. req:: Read test results from Robot and Go runs
    :id: HMD_CLI_BARTLEBY_REQ_VERIFY_002
    :status: implemented

    Verification shall read a Robot Framework ``output.xml`` (schema 4 and 5,
    including nested pabot suites) and the event stream of ``go test -json``,
    recording each top-level test's suite, name, outcome, tags and the run's own
    timestamp, and match each result to its test item in ``needs.json``.

.. req:: Give one verdict per requirement and implementation
    :id: HMD_CLI_BARTLEBY_REQ_VERIFY_003
    :status: implemented

    Each result file shall be labelled with the implementation it ran against
    (``LABEL[@VERSION]=PATH``), files with the same label shall form one
    implementation, and each requirement shall get one verdict per
    implementation: *pass* when every linked test that ran passed, *fail* when
    any failed, *skip* when all were skipped, *missing* when none ran, and
    *exempt* for a ``trace-exempt`` requirement with no linked test.

.. req:: Count baseline tests apart from new ones
    :id: HMD_CLI_BARTLEBY_REQ_VERIFY_004
    :status: implemented

    Results tagged ``baseline`` — the tests that existed before a port began —
    shall be counted separately from the rest, per implementation, so the
    conformance floor and the coverage a port adds are both visible.

.. req:: Write the matrix and fail on a gap
    :id: HMD_CLI_BARTLEBY_REQ_VERIFY_005
    :status: implemented

    Verification shall write ``docs/requirements/verification_matrix.rst`` (or a
    page named by ``-out``), print JSON with ``-json``, report results for tests
    ``needs.json`` does not declare, and exit non-zero when any requirement is
    failing or missing for any implementation.
