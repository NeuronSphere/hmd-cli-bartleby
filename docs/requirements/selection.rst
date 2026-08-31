.. Builder and root document selection requirements

Builder and Root Document Selection
===================================

Choosing what to build is where the CLI has historically been least trustworthy:
``--shell`` was parsed and never read, so every invocation built everything, and
a builder nobody had configured produced a cheerful exit code 0. These
requirements exist to keep that fixed.

.. req:: Select builders with --shell
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    ``--shell`` shall restrict the run to the named builders. It shall accept a
    single builder, a comma-separated list of builders, or ``all``, and shall
    default to ``all``.

.. spec:: Any builder named in the manifest is valid
    :id: HMD_CLI_BARTLEBY_REQ_SEL_001_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SEL_001
    :status: implemented

    Valid ``--shell`` values shall be whatever the manifest's roots declare,
    rather than a list hard-coded in the CLI. The transform image gains builders
    over time — ``revealjs`` and ``confluence`` arrived after the CLI was
    written — and a hard-coded list would have to be edited to keep up.

.. req:: An unknown builder is an error
    :id: HMD_CLI_BARTLEBY_REQ_SEL_002
    :status: implemented

    When no selected root declares any requested builder, the CLI shall exit
    non-zero with an error listing the builders that *are* declared. It shall not
    report success, because "there was nothing to build" is never what the user
    meant.

.. spec:: A partially unknown builder list still builds
    :id: HMD_CLI_BARTLEBY_REQ_SEL_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SEL_002
    :status: implemented

    When a comma-separated list names some builders that exist and some that do
    not, the run shall proceed with the ones that exist and warn about each one
    that does not, so a typo in a long list neither cancels the build nor passes
    unnoticed.

.. req:: Select root documents with --root-doc
    :id: HMD_CLI_BARTLEBY_REQ_SEL_003
    :status: implemented

    ``--root-doc`` shall restrict the run to the named manifest roots, accepting
    a single name, a comma-separated list, or ``all``, and defaulting to ``all``.
    Surrounding whitespace in a list shall be tolerated. A named root that the
    manifest does not declare shall be skipped with a warning.

.. req:: An entirely unknown root document set is an error
    :id: HMD_CLI_BARTLEBY_REQ_SEL_004
    :status: implemented

    When none of the requested root documents exist, the CLI shall exit non-zero
    with an error listing the roots the manifest declares.

.. req:: Default to a single index root
    :id: HMD_CLI_BARTLEBY_REQ_SEL_005
    :status: implemented

    A repository whose manifest declares no ``bartleby.roots`` shall build the
    root document ``index`` with the ``html`` and ``pdf`` builders. This is the
    common case for a repository that has documentation but no Bartleby
    configuration.

.. req:: Default a root's document name
    :id: HMD_CLI_BARTLEBY_REQ_SEL_006
    :status: implemented

    A manifest root that omits ``root_doc`` shall build the document ``index``.
    The container has no default of its own, so an omitted value must not reach
    it as an empty string.

.. req:: Build in a deterministic order
    :id: HMD_CLI_BARTLEBY_REQ_SEL_007
    :status: implemented

    Builds shall run in a stable order — root document name, then builder name —
    so that repeated runs produce output and logs in the same sequence.
