.. NERD005 Serve Review Hub

NERD005 Serve Review Hub
========================

.. req:: Review rendered NERDs and NEPs for many repositories from one local URL
    :id: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    ``bartleby serve`` shall serve the HTML that ``bartleby html`` already
    built for a chosen set of repositories, behind one landing page that
    lists each repository's NERDs and NEPs with their status. It shall only
    read what is on disk, except where the reader asks for one repository to
    be rebuilt. It shall be reachable from the local machine only.

Created 2026-11-22. The testable requirements are ``REQ_SERVE_001`` to
``REQ_SERVE_007`` in ``docs/requirements/serve.rst``, one for each spec below, and
the tests cite those.

Motivation
----------

NERDs and NEPs are written in one repository each and rendered to
``target/bartleby/html/`` in that repository. Reviewing them means asking
for a render, then opening each repository's ``index.html`` by hand.
Bartleby has no ``serve``, ``watch`` or ``index`` command, so there is no
single place that shows what is in review.

The pieces already exist. ``reqtrace`` lists every need in a workspace
without a Sphinx build (:need:`HMD_CLI_BARTLEBY_NERD004`), and the built
HTML has a stable layout. This NERD joins them with a small web server.

.. spec:: The set of repositories is an explicit list
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    The repositories served shall be read from a plain-text file, one
    repository directory name per line, with ``#`` starting a comment and
    blank lines ignored. The file defaults to
    ``$HMD_HOME/.config/bartleby-serve.repos`` and ``--repos-file`` overrides
    it. A name that has no directory under the workspace shall be reported on
    the landing page as missing and shall not stop the server. A name that
    is not a plain directory name, such as one holding ``/`` or ``..``, shall
    be rejected when the file is read. The workspace defaults to
    ``$HMD_REPO_HOME`` and ``--workspace`` overrides it.

.. spec:: Built HTML is served read-only under a per-repository prefix
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    Each listed repository's ``target/bartleby/html`` shall be served at
    ``/r/<repo>/``. A request shall resolve to a file inside that directory
    only. A path that leaves it, by ``..`` or by a symbolic link whose target
    is outside it, shall return 404. A repository not in the list shall
    return 404. Only ``GET`` and ``HEAD`` shall be accepted on these paths.
    The one exception to the HTML tree is a top-level ``*.pdf`` name, which
    is read from ``target/bartleby``, where PDFs are built.

.. spec:: The landing page lists each repository's needs and build state
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC003
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    ``/`` shall list every repository in the list. For each it shall show a
    link to its HTML, a link to its newest PDF when one exists, the time the
    HTML was built, and whether the build is stale. It shall list the
    repository's NERDs and NEPs from ``reqtrace.ListWorkspace``: the needs of
    type ``req`` declared under ``docs/proposals``, each with its id, title
    and status shown as text, linking to the rendered page and the ``#<id>``
    anchor when that page has been built. A repository with no built HTML shall say so rather
    than show a dead link. Opening the page shall never start a build.

.. spec:: A build is stale when a source file is newer than the HTML
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC004
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    A repository's build shall be stale when the newest modification time
    of any ``.rst`` file under its ``docs`` directory is later than the
    modification time of ``target/bartleby/html/index.html``. A repository
    without built HTML shall be reported as not built, not as stale.

.. spec:: One repository can be rebuilt on request
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC005
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    ``POST /rebuild/<repo>`` shall run ``bartleby html`` with that
    repository as its working directory, in a child process of the running
    binary, and return its combined output. It shall run at most one build
    at a time and answer 409 while one is running. It shall refuse a
    repository not in the list with 404, and any method other than ``POST``
    with 405. A request whose ``Host`` is not a loopback name, or whose
    ``Origin`` header names a different host than ``Host``, shall be refused
    with 403, so that a web page open in the same browser, or a rebinding
    DNS name, cannot start a build.

.. spec:: The server binds to the loopback interface
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC006
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    The default address shall be ``127.0.0.1:8099``. ``--addr`` shall
    refuse an address that is not a loopback address, so the server cannot
    be exposed to the network by a flag.

.. spec:: Repositories with recent local commits can be suggested
    :id: HMD_CLI_BARTLEBY_NERD005_SPEC007
    :links: HMD_CLI_BARTLEBY_NERD005
    :status: implemented

    ``bartleby serve suggest`` shall print the names of repositories in the
    workspace that have commits by the configured ``git user.email`` in the
    last 30 days and are not yet in the list. It shall print them only. It
    shall not write the list.

Out of scope
------------

Discovering every repository without a list, changing a need's status from
the page, authentication beyond the loopback bind, and a file watcher for
live reload. Each can follow once this is in use.
