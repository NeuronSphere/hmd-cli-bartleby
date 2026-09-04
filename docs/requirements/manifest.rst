.. Manifest reading requirements

Reading the Manifest
====================

.. req:: Read the project manifest
    :id: HMD_CLI_BARTLEBY_REQ_MAN_001
    :status: implemented

    The CLI shall read ``meta-data/manifest.json`` from the repository in the
    working directory and take its Bartleby configuration from the ``bartleby``
    object. A repository with no manifest shall build with the defaults and say
    so, since that is a legitimate shape rather than a mistake.

.. req:: Distinguish a broken manifest from a missing one
    :id: HMD_CLI_BARTLEBY_REQ_MAN_002
    :status: implemented

    A manifest that exists but cannot be read or parsed shall be reported as an
    error naming the file. It shall not be treated as absent — collapsing the two
    turns a JSON typo or a permissions problem into a build that silently ignores
    the whole configuration.

.. req:: Accept both builder spellings
    :id: HMD_CLI_BARTLEBY_REQ_MAN_003
    :status: implemented

    A root's ``builders`` array shall accept a bare builder name and an object of
    the form ``{"shell": ..., "config": ...}``, in any mixture. The object form
    carries builder-specific Sphinx configuration.

.. spec:: A builder object requires a shell
    :id: HMD_CLI_BARTLEBY_REQ_MAN_003_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_MAN_003
    :status: implemented

    An object-form builder with no ``shell`` shall be an error, rather than
    producing a build with an empty builder name.

.. req:: Name the documented repository
    :id: HMD_CLI_BARTLEBY_REQ_MAN_004
    :status: implemented

    The repository name passed to the transform shall come from the manifest's
    ``name``, falling back to the name of the working directory.

.. req:: Read the repository version
    :id: HMD_CLI_BARTLEBY_REQ_MAN_005
    :status: implemented

    The version shall be read from ``meta-data/VERSION``. A missing file shall
    fall back to ``stable`` silently; a file that exists but is empty or
    unreadable shall fall back with a warning, because in that case the fallback
    is probably not what the user wanted in their document title.
