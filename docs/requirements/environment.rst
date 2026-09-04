.. HMD environment requirements

The HMD Environment
===================

.. req:: Load the HMD environment file
    :id: HMD_CLI_BARTLEBY_REQ_ENV_001
    :status: implemented

    When ``HMD_HOME`` is set, the CLI shall load ``$HMD_HOME/.config/hmd.env``
    before doing anything else, so shared defaults do not have to be exported by
    hand. This is the file the Python CLI read through ``load_hmd_env``.

.. spec:: The process environment wins
    :id: HMD_CLI_BARTLEBY_REQ_ENV_001_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_ENV_001
    :status: implemented

    A variable already present in the environment shall not be overwritten by the
    file, so exporting a value for one command still works. This matches the
    Python call site, which loaded the file with ``override=False``.

.. req:: Treat missing configuration as normal and broken configuration as notable
    :id: HMD_CLI_BARTLEBY_REQ_ENV_002
    :status: implemented

    An unset ``HMD_HOME`` or an absent environment file shall be silent — plenty
    of repositories build without one. A file that exists but cannot be read or
    parsed shall produce a warning and the build shall continue, because the user
    plainly expected its values to apply.

.. req:: Support the usual dotenv syntax
    :id: HMD_CLI_BARTLEBY_REQ_ENV_003
    :status: implemented

    The environment file shall support ``KEY=VALUE`` lines, an optional
    ``export`` prefix, ``#`` comments, blank lines, single- and double-quoted
    values, the common escape sequences inside double quotes, and a trailing
    inline comment on an unquoted value.

.. req:: Write configuration without disturbing the rest of the file
    :id: HMD_CLI_BARTLEBY_REQ_ENV_004
    :status: implemented

    Writing a setting shall replace an existing entry for that key in place and
    append a new one otherwise, preserving every other line including comments.
    Values needing quoting shall be quoted so that they round-trip. Writing shall
    fail with a clear error when ``HMD_HOME`` is unset.

.. req:: Mount global styles from HMD_HOME
    :id: HMD_CLI_BARTLEBY_REQ_ENV_005
    :status: implemented

    When ``HMD_HOME`` is set, ``$HMD_HOME/bartleby/styles`` shall be offered to
    the container as its global styles directory, and omitted when the directory
    does not exist.
