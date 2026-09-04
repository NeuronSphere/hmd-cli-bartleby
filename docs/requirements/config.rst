.. Configuration resolution requirements

Configuration Resolution
========================

.. req:: Merge builder configuration in a defined order
    :id: HMD_CLI_BARTLEBY_REQ_CFG_001
    :status: implemented

    The Sphinx configuration handed to a builder shall be merged from four
    layers, each overriding the ones before it:

    1. the root's own ``config`` object;
    2. ``bartleby.config.builders.<shell>`` from the manifest;
    3. an object-form builder's inline ``config``;
    4. ``HMD_BARTLEBY__<SHELL>__<KEY>`` environment variables.

    The order runs from least to most specific, so a value set for one run in the
    environment beats a value committed in the manifest.

.. req:: Accept per-builder configuration from the environment
    :id: HMD_CLI_BARTLEBY_REQ_CFG_002
    :status: implemented

    ``HMD_BARTLEBY_<SHELL>_CONFIG`` shall be read as a JSON object and used where
    the manifest declares no configuration for that builder. Because the manifest
    is the more specific source, a manifest entry shall win over this variable.

.. spec:: Malformed configuration JSON is skipped, not fatal
    :id: HMD_CLI_BARTLEBY_REQ_CFG_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_CFG_002
    :status: implemented

    A ``HMD_BARTLEBY_<SHELL>_CONFIG`` value that is not a JSON object shall be
    ignored with a warning, leaving the other configuration layers intact.

.. req:: Accept single configuration values from the environment
    :id: HMD_CLI_BARTLEBY_REQ_CFG_003
    :status: implemented

    ``HMD_BARTLEBY__<SHELL>__<KEY>`` shall set one configuration key for one
    builder, with the key lower-cased. This is the most specific layer and shall
    override all others.

.. req:: Resolve logos from flag, environment, then manifest
    :id: HMD_CLI_BARTLEBY_REQ_CFG_004
    :status: implemented

    The default, HTML, and PDF logos shall each be resolved from the
    corresponding flag, then the corresponding environment variable, then the
    manifest. The HTML and PDF logos shall fall back to the resolved default logo
    when nothing sets them specifically.

.. req:: Resolve the confidentiality stamp from any of three sources
    :id: HMD_CLI_BARTLEBY_REQ_CFG_005
    :status: implemented

    Documents shall be stamped confidential when ``--confidential`` is passed,
    when the manifest sets ``bartleby.config.confidential``, or when
    ``HMD_BARTLEBY_CONFIDENTIAL`` is set to a truthy value. The stamp text shall
    come from ``HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT``.

.. spec:: Truthy environment values are interpreted generously
    :id: HMD_CLI_BARTLEBY_REQ_CFG_005_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_CFG_005
    :status: implemented

    ``true``, ``1``, ``yes``, ``on``, ``t``, and ``y`` shall all count as true,
    in any case. An exact match against ``"true"`` is a trap: the Python CLI
    wrote ``True`` into the same variable, and that combination silently did
    nothing.

.. req:: Derive a document title when none is given
    :id: HMD_CLI_BARTLEBY_REQ_CFG_006
    :status: implemented

    With no ``--title``, the document title shall be the repository name and
    version joined by a hyphen.

.. req:: Make document titles safe for LaTeX
    :id: HMD_CLI_BARTLEBY_REQ_CFG_007
    :status: implemented

    The title shall have every character that breaks LaTeX text mode or a
    Makefile target replaced with a hyphen — whitespace, underscores, and
    ``& % # $ ^ \\ ~ { } " '`` among others — with runs collapsed and the ends
    trimmed. When sanitizing changes the title, the CLI shall say so, because the
    output filename will not be what the user typed.

.. spec:: A title with nothing usable is omitted
    :id: HMD_CLI_BARTLEBY_REQ_CFG_007_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_CFG_007
    :status: implemented

    A title that sanitizes down to nothing shall be omitted entirely, letting the
    container derive its own, rather than passing an empty title.
