.. Autodoc requirements

Autodoc and Private Package Indexes
===================================

.. req:: Require a Python package for autodoc
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_001
    :status: implemented

    ``--autodoc`` shall apply only to a repository with a Python package at
    ``src/python``. Elsewhere the CLI shall warn and build without autodoc rather
    than failing, since the rest of the documentation is still buildable.

.. req:: Generate pip credentials as a private temporary file
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_002
    :status: implemented

    When ``PIP_USERNAME`` and ``PIP_PASSWORD`` are both set, the CLI shall write a
    ``pip.conf`` containing a credentialed ``extra-index-url``, with the password
    URL-escaped, to a temporary file readable only by the current user, and shall
    delete it after the build.

.. spec:: The index host is configurable
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_AUTO_002
    :status: implemented

    The index the generated configuration points at shall default to the HMD
    Artifactory PyPI repository and be overridable with
    ``HMD_PIP_EXTRA_INDEX_HOST``.

.. req:: Fall back to the user's own pip configuration
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_003
    :status: implemented

    Without both credentials, an existing per-user ``pip.conf`` shall be used when
    present. Half-configured credentials — a username with no password — shall not
    produce a generated configuration.

.. spec:: No pip configuration is a valid state
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_003_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_AUTO_003
    :status: implemented

    With no credentials and no user configuration, the build shall proceed with no
    pip configuration at all, which is correct for a package whose dependencies
    are public.

.. req:: Tell the container about pip configuration only when there is some
    :id: HMD_CLI_BARTLEBY_REQ_AUTO_004
    :status: implemented

    ``PIP_CONF`` shall be set in the container environment only for an autodoc
    build that actually has a pip configuration to mount. The image raises when
    the variable points at a file that is not there.
