.. Serve requirements

Serve
=====

``bartleby serve`` shows the HTML that ``bartleby html`` already built for a
list of repositories, behind one local landing page, so a reviewer does not open
each repository's ``index.html`` by hand. It reads what is on disk, and starts
a build only when asked. NERD005 has the design.

.. req:: Read the repositories to serve from a plain-text list
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_001
    :status: implemented

    The repositories shall be read from a file with one directory name per
    line, ``#`` starting a comment. A missing file shall be an empty list. A
    repeated name shall be kept once. A name that is not a plain directory
    name, such as one containing ``/`` or equal to ``..``, shall be an error,
    so the list cannot name directories outside the workspace.

.. req:: Serve built HTML read-only and only from inside the HTML tree
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_002
    :status: implemented

    ``/r/<repo>/`` shall serve ``target/bartleby/html`` of a listed
    repository. A path that leaves that tree, by ``..`` or by a symbolic link,
    an unlisted repository and a missing file shall each return 404. Only
    ``GET`` and ``HEAD`` shall be accepted. A top-level ``*.pdf`` name shall be
    read from ``target/bartleby``, where PDFs are built.

.. req:: List each repository's NERDs and NEPs, and its build, on a landing page
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_003
    :status: implemented

    ``/`` shall list every repository in the list. For each it shall link to
    the built HTML and the newest PDF, show when it was built, and list the
    ``req`` needs under ``docs/proposals`` with id, title and status as
    text, linked to the rendered page only when that page exists. A directory
    that does not exist shall be shown as missing, a repository without HTML as
    not built, and a failure to list needs shall be shown on the page, which
    shall still render. Titles shall be HTML-escaped.

.. req:: Mark a build stale when its sources are newer than its HTML
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_004
    :status: implemented

    A repository shall be stale when the newest ``.rst`` under ``docs`` is
    modified after ``target/bartleby/html/index.html``. A repository with no
    built HTML shall be not built, not stale.

.. req:: Rebuild one listed repository on an explicit POST
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_005
    :status: implemented

    ``POST /rebuild/<repo>`` shall run ``bartleby html`` in that repository's
    directory and return its output, with 500 when the build fails. It shall
    run one build at a time and answer 409 while one is running. It shall
    answer 404 for a repository not in the list, 405 for any method but
    ``POST``, and 403 when ``Host`` is not a loopback name or ``Origin`` names a
    different host than ``Host``.

.. req:: Listen on the loopback interface only
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_006
    :status: implemented

    ``--addr`` shall accept ``127.0.0.1``, ``::1`` and ``localhost`` and
    refuse every other address, including a bare port and ``0.0.0.0``.

.. req:: Suggest repositories with the user's recent commits
    :id: HMD_CLI_BARTLEBY_REQ_SERVE_007
    :status: implemented

    ``bartleby serve suggest`` shall name the repositories one level under the
    workspace that have a commit by the given author email within the given
    number of days and are not already listed. It shall need an email, and it
    shall not write the list.
