.. Gather mode requirements

Gather Mode
===========

``--gather`` rebuilds ``docs/`` from the demo library's index plus the
documentation of named sibling repositories. It is destructive by nature, which
is what the preconditions are for.

.. req:: Only gather where gathering makes sense
    :id: HMD_CLI_BARTLEBY_REQ_GATH_001
    :status: implemented

    Gather mode shall run only from a ``hmd-docs-bartleby`` checkout that has a
    ``docs`` directory and sits alongside ``hmd-lib-bartleby-demos``. Anywhere
    else it shall refuse with an error naming what is missing, before touching
    anything.

.. req:: Validate every repository before deleting anything
    :id: HMD_CLI_BARTLEBY_REQ_GATH_002
    :status: implemented

    Every named repository shall be checked for a ``docs`` directory before
    ``docs/`` is emptied. A typo in the list shall leave the working tree
    untouched, rather than clearing the documentation and then failing partway
    through rebuilding it.

.. req:: Rebuild the docs tree from the demo index
    :id: HMD_CLI_BARTLEBY_REQ_GATH_003
    :status: implemented

    Gathering shall clear ``docs/`` except for the index, install the demo
    library's ``index.rst`` in its place, copy each gathered repository's ``docs``
    to ``docs/<repo>``, and add a toctree entry for each — before a trailing
    "Indexes and tables" section when there is one, and at the end otherwise.

.. req:: Do nothing when nothing was asked for
    :id: HMD_CLI_BARTLEBY_REQ_GATH_004
    :status: implemented

    An empty ``--gather`` value shall be a no-op, leaving the working tree alone
    and letting the build proceed normally.
