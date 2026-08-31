.. Container execution requirements

Running the Transform Container
===============================

.. req:: Drive the container through the Docker API
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_001
    :status: implemented

    Transforms shall be run by talking to the Docker Engine API directly. The CLI
    shall not shell out to ``docker-compose``, and shall not write a compose file
    into ``target/``.

    Compose was only ever needed to hand pip credentials to the container as a
    secret; a bind-mounted file does that, so the dependency and the generated
    YAML both go away.

.. req:: Pass the transform context the image expects
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_002
    :status: implemented

    Each run shall pass ``TRANSFORM_INSTANCE_CONTEXT`` as a JSON object carrying
    the root name, builder, root document, and merged builder config, along with
    ``BARTLEBY_SHELL``, the ``HMD_*`` deployment variables, and the repository
    name and version.

.. req:: Spell AUTODOC the way the image reads it
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_003
    :status: implemented

    ``AUTODOC`` shall be passed as ``True`` or ``False``. The transform image
    compares it against Python's ``str(True)``, so Go's ``true`` disables autodoc
    without reporting anything.

.. req:: Omit optional variables rather than emptying them
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_004
    :status: implemented

    Optional variables — company name, document title, confidentiality statement,
    the logos, and ``NO_TIMESTAMP_TITLE`` — shall be absent from the container
    environment when they have no value. The image tests several of them for
    presence, so an empty string is not equivalent to unset.

.. req:: Mount the repository, the output directory, and nothing surprising
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_005
    :status: implemented

    Every run shall bind-mount the repository as the container's input and
    ``target/bartleby`` as its output. The global styles directory and a pip
    configuration shall be mounted read-only when they exist, and omitted when
    they do not, so a missing optional path cannot fail container creation.

.. req:: Name containers predictably and legally
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_006
    :status: implemented

    A transform container shall be named from the instance name and the builder,
    sanitized to the character set Docker accepts and always starting with an
    alphanumeric. A repository directory containing spaces or punctuation must not
    produce an opaque container-creation failure.

.. req:: Recover from a leftover container
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_007
    :status: implemented

    When a container of the same name already exists, the CLI shall remove it once
    and retry, reporting that it did so. This is the expected state after a
    previous run was killed.

.. req:: Stream container output as it happens
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_008
    :status: implemented

    The container's standard output and standard error shall be streamed to the
    caller's own streams while the build runs, so a slow Sphinx or LaTeX build
    shows progress rather than silence.

.. req:: Report a failed build as a failure
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_009
    :status: implemented

    A container that exits non-zero shall produce an error naming the build and
    the exit code, and the CLI shall exit non-zero. Remaining builds shall not be
    attempted.

.. req:: Clean up when interrupted
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_010
    :status: implemented

    An interrupt shall cancel the build and still remove the container it started.
    Cleanup must not itself be cancelled, or the next run collides with the
    container the last one abandoned.

.. req:: Find the Docker daemon without being told
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_011
    :status: implemented

    The standard ``DOCKER_*`` environment variables shall be honoured, including
    the TLS settings a remote daemon needs. When ``DOCKER_HOST`` is unset, the
    active docker context shall be consulted, and failing that the socket
    locations Colima, Docker Desktop, and Rancher Desktop use — so the common
    macOS setup works with nothing exported.

.. req:: Create the output directory
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_012
    :status: implemented

    ``target/bartleby`` shall be created before the first build, since the
    container writes into it through a bind mount that must already exist.

.. req:: Resolve the transform image reference from the environment
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_013
    :status: implemented

    The transform image shall be ``<registry>/hmd-tf-bartleby:<tag>``, with the
    registry from ``HMD_CONTAINER_REGISTRY`` defaulting to
    ``ghcr.io/neuronsphere`` and the tag from ``HMD_TF_BARTLEBY_VERSION``
    defaulting to ``stable``.

.. req:: Report pull progress readably
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_014
    :status: implemented

    A pull shall report progress as readable lines rather than the raw JSON
    stream the API returns, collapsing repeated per-layer statuses, and shall
    surface an error the daemon reports inside that stream as a failure.

.. req:: Distinguish an absent image from a removal failure
    :id: HMD_CLI_BARTLEBY_REQ_EXEC_015
    :status: implemented
    :tags: trace-exempt

    Removing an image that is not present locally shall be reported distinctly
    from a removal that genuinely failed, so ``update-image`` can treat the former
    as nothing to do.

    *Verification:* by inspection and manual run — the branch depends on the
    daemon's own not-found response, and reaching it in a test means managing a
    real multi-gigabyte image.
