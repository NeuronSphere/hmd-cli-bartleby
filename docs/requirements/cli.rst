.. Command surface requirements

Command Surface
===============

.. req:: Build every configured output by default
    :id: HMD_CLI_BARTLEBY_REQ_CLI_001
    :status: implemented

    Run with no subcommand, ``bartleby`` shall build every builder configured for
    every selected root document, from the repository in the working directory.

.. req:: Provide a subcommand per common builder
    :id: HMD_CLI_BARTLEBY_REQ_CLI_002
    :status: implemented

    ``bartleby html``, ``bartleby pdf``, and ``bartleby slides`` shall build the
    ``html``, ``pdf``, and ``revealjs`` builders respectively, and shall be
    equivalent to passing that builder to ``--shell``.

.. spec:: A subcommand and a contradictory --shell is an error
    :id: HMD_CLI_BARTLEBY_REQ_CLI_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_CLI_002
    :status: implemented

    Because a subcommand pins one builder, ``bartleby html --shell pdf`` is a
    contradiction. It shall be reported as an error naming the flag, rather than
    resolved silently in favour of either side.

.. req:: Render PlantUML files on request
    :id: HMD_CLI_BARTLEBY_REQ_PUML_001
    :status: implemented

    ``bartleby puml`` shall render every ``.puml`` file under ``docs/`` to an
    image in ``target/bartleby/puml_images``.

.. spec:: PlantUML file discovery
    :id: HMD_CLI_BARTLEBY_REQ_PUML_001_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_PUML_001
    :status: implemented

    Discovery shall be recursive, match the extension case-insensitively, pass
    paths to the container relative to ``docs/`` with forward slashes, and be
    sorted so that repeated runs are identical.

.. req:: Report clearly when there is nothing to render
    :id: HMD_CLI_BARTLEBY_REQ_PUML_002
    :status: implemented

    ``bartleby puml`` shall fail with an error when there is no ``docs``
    directory, and shall report that there is nothing to do and succeed when the
    directory exists but holds no ``.puml`` files.

.. req:: Refresh the transform image on request
    :id: HMD_CLI_BARTLEBY_REQ_CLI_004
    :status: implemented
    :tags: trace-exempt

    ``bartleby update-image`` shall remove the local copy of the transform image
    and pull it again, so that a moving tag such as ``:stable`` is genuinely
    re-fetched. An image that is not present locally shall be reported as
    nothing to remove rather than as a failure.

    *Verification:* by inspection and manual run. Automating it means pulling a
    6 GB image, which is not a reasonable thing to do in a test suite. The image
    reference it operates on is covered by :need:`HMD_CLI_BARTLEBY_REQ_EXEC_013`
    and the progress reporting by :need:`HMD_CLI_BARTLEBY_REQ_EXEC_014`.

.. req:: Configure defaults interactively
    :id: HMD_CLI_BARTLEBY_REQ_CLI_005
    :status: implemented

    ``bartleby configure`` shall prompt for the Bartleby defaults — logo,
    confidentiality statement, container registry — showing the current value of
    each, and write the answers to the ``$HMD_HOME`` environment file. An empty
    answer shall keep the value shown.

.. req:: Report the version
    :id: HMD_CLI_BARTLEBY_REQ_CLI_006
    :status: implemented

    ``bartleby --version`` and ``bartleby version`` shall both report the version
    injected at build time, and shall agree with each other. A binary built
    without the version linker flag shall report ``dev``.

.. req:: Fail loudly and usefully
    :id: HMD_CLI_BARTLEBY_REQ_CLI_007
    :status: implemented

    A runtime failure shall exit non-zero and print a single ``Error:`` line to
    standard error. Usage text shall not be printed for runtime failures — only
    for a malformed command line — because a wall of usage buries the error.

.. req:: Reject unrecognised input
    :id: HMD_CLI_BARTLEBY_REQ_CLI_008
    :status: implemented

    Unknown flags and unexpected positional arguments shall be rejected rather
    than ignored.

.. req:: Document the command surface in help
    :id: HMD_CLI_BARTLEBY_REQ_CLI_009
    :status: implemented

    ``bartleby --help`` shall list every subcommand and every persistent flag,
    so the help output is a complete description of what the CLI can do.
