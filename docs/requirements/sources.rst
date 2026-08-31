.. External documentation source requirements

External Documentation Sources
==============================

``bartleby.sources`` stitches documentation that lives elsewhere into this
repository's docs for the duration of a build. Since it edits the working tree,
the requirements about putting it back are as important as the ones about
assembling it.

.. req:: Stage artifact-backed sources into the docs tree
    :id: HMD_CLI_BARTLEBY_REQ_SRC_001
    :status: implemented

    A source declaring an ``artifact_path`` shall have its documentation directory
    copied to ``docs/_sources/<key>`` before the build. A source without one is
    expected to be present at ``docs/<key>`` already and shall be used in place.

.. spec:: Honour docs_root
    :id: HMD_CLI_BARTLEBY_REQ_SRC_001_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SRC_001
    :status: implemented

    The documentation directory within an artifact shall be ``docs_root``,
    defaulting to ``docs``.

.. spec:: Replace stale staging rather than merging into it
    :id: HMD_CLI_BARTLEBY_REQ_SRC_001_SPEC002
    :links: HMD_CLI_BARTLEBY_REQ_SRC_001
    :status: implemented

    An existing staging directory for a source shall be removed before the copy,
    so a file deleted upstream does not survive in the build.

.. req:: Skip a source that is not there
    :id: HMD_CLI_BARTLEBY_REQ_SRC_002
    :status: implemented

    A source whose documentation directory does not exist shall be skipped with a
    warning naming the path, and the build shall continue. A source is often an
    optional artifact that the current build did not fetch.

.. req:: Inject a toctree for each source
    :id: HMD_CLI_BARTLEBY_REQ_SRC_003
    :status: implemented

    Each valid source shall be added to the root document as a toctree entry, with
    the caption taken from its ``title`` and falling back to its key.

.. spec:: Toctree placement
    :id: HMD_CLI_BARTLEBY_REQ_SRC_003_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SRC_003
    :status: implemented

    Placement shall be tried in three ways, in order: replacing a
    ``.. bartleby-sources::`` marker line; inserting before a trailing "Indexes
    and tables" or "Indices and tables" section, matched ignoring case and
    surrounding whitespace; appending to the end of the document.

.. req:: Leave the docs tree as it was found
    :id: HMD_CLI_BARTLEBY_REQ_SRC_004
    :status: implemented

    After the build, every edited root document shall be restored byte for byte
    and the staging directory removed — whether the build succeeded, failed, or
    could not start. The repository must not be left dirty by a build.

.. req:: Order sources deterministically
    :id: HMD_CLI_BARTLEBY_REQ_SRC_005
    :status: implemented

    Staging, injection, and warnings shall process sources in a stable order, so
    the generated toctree does not change between runs.
