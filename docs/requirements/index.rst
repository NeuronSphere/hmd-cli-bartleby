.. Bartleby requirements

Requirements
============

What the Bartleby CLI must do, written as `sphinx-needs
<https://sphinx-needs.readthedocs.io/>`_ items so that each one can be linked to
the tests that verify it. The generated :doc:`traceability` page carries the
matrix.

This is the standing baseline: it describes the CLI as it is meant to behave,
not the history of how it got there. Proposals for *changes* remain NERDs under
:doc:`../proposals/index`, and a NERD links to the requirements it adds or
amends.

How this works
--------------

**Requirements** are ``.. req::`` items. Where the way a requirement is met is
itself worth pinning down, a ``.. spec::`` item links to it.

**IDs** are area-coded::

    HMD_CLI_BARTLEBY_REQ_<AREA>_<NNN>

The ``HMD_<REPO_TYPE>_<NAME>_`` prefix matches the convention the NERDs use. The
area segment keeps the ID readable where it matters most — on a test:

.. list-table::
   :header-rows: 1
   :widths: 20 80

   * - Area
     - Covers
   * - ``CLI``
     - The command surface: subcommands, exit behaviour, version reporting.
   * - ``PUML``
     - PlantUML rendering.
   * - ``SEL``
     - Choosing which builders and root documents run.
   * - ``MAN``
     - Reading ``meta-data/manifest.json`` and ``meta-data/VERSION``.
   * - ``CFG``
     - Resolving builder config, logos, confidentiality, and titles.
   * - ``EXEC``
     - Running the transform container.
   * - ``SRC``
     - ``bartleby.sources``: stitching in external documentation.
   * - ``GATH``
     - ``--gather``: assembling sibling repositories' documentation.
   * - ``ENV``
     - ``$HMD_HOME`` configuration and ``bartleby configure``.
   * - ``AUTO``
     - Autodoc builds and the pip credentials they need.
   * - ``TRACE``
     - This traceability process itself.

**Tests declare their own coverage**, in the test source rather than in a
separate list that would drift. A Go test declares it in the doc comment on the
test function:

.. code-block:: go

    // Requirements: REQ_SEL_001, REQ_SEL_006
    func TestBuildsAcceptsAShellList(t *testing.T) {

A Robot test declares it in ``[Tags]``:

.. code-block:: robotframework

    Unknown Builder Is An Error Not A Silent No-Op
        [Tags]    REQ_SEL_002    REQ_CLI_007

The short form is used in annotations; the tooling expands it to the full ID.

**Both directions are enforced.** ``make check`` runs ``reqtrace -check``, which
fails on:

- a requirement no test verifies,
- a test that declares no requirement,
- a reference to a requirement that does not exist,
- a duplicate requirement ID,
- a ``:links:`` target that does not exist,
- a stale :doc:`traceability` page.

None of that needs Docker or Sphinx, so it holds on a laptop and in CI.

**Exemptions** exist for requirements no automated test can reasonably cover.
Such a requirement is tagged ``trace-exempt`` and has to say in its own text how
it *is* verified. There are two, both about pulling a multi-gigabyte image.

.. toctree::
   :maxdepth: 2

   cli
   selection
   manifest
   config
   execution
   sources
   gather
   environment
   autodoc
   process
   traceability
