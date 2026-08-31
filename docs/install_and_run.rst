.. bartleby installation and development

Bartleby Install and Run
==========================

Installation
-------------

Install via Homebrew (recommended)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

    brew tap neuronsphere/tap
    brew install bartleby

This installs a self-contained Go binary with no Python runtime required.
Docker (or Colima) must be running when you execute builds.

To upgrade to the latest release:

.. code-block:: bash

    brew update
    brew upgrade bartleby

Build from Source
~~~~~~~~~~~~~~~~~~

Clone the repository and build with Make:

.. code-block:: bash

    git clone https://github.com/neuronsphere/hmd-cli-bartleby.git
    cd hmd-cli-bartleby
    make build

The binary is written to ``src/go/bartleby/build/bartleby``.

Legacy Installation (Python CLI)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. note::

   The Python CLI has been superseded by the Go binary above, which now covers
   everything it did — including ``bartleby.sources``, ``--gather``,
   ``configure``, ``$HMD_HOME`` environment loading, and per-builder config.
   Unlike the Python version it talks to the Docker API directly: no
   ``docker-compose`` binary, and no ``docker-compose-<shell>.yaml`` written into
   ``target/``. Compose was only ever needed to pass pip credentials as a secret,
   and a bind-mounted file does that job.

   These instructions are retained for anyone still on the ``hmd bartleby``
   plugin.

The HMD CLI Bartleby tool can be installed using ``pip`` and specifying the HMD pypi server (via command line or using
a pip config file).

.. code-block:: bash

    pip install hmd-cli-bartleby


Running the Bartleby Transform
--------------------------------

``bartleby`` builds the repository it is run from, so run it from the repository
root. It reads ``meta-data/manifest.json`` and ``meta-data/VERSION`` there, mounts
the repository into the transform container, and writes output to
``target/bartleby/``.

.. code-block:: bash

    bartleby                  # every builder configured for every root document
    bartleby html             # HTML only
    bartleby pdf              # PDF only
    bartleby slides           # RevealJS slideshow
    bartleby puml             # render docs/**/*.puml to images
    bartleby update-image     # re-pull the transform image
    bartleby configure        # write defaults to $HMD_HOME/.config/hmd.env
    bartleby version          # print the version

Selecting builders and root documents
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``--shell`` selects builders and ``--root-doc`` selects root documents. Both take
a comma-separated list, or ``all``:

.. code-block:: bash

    bartleby --shell html,pdf
    bartleby --root-doc guide,api
    bartleby html --root-doc guide

Anything named in a root's ``builders`` array is a valid ``--shell`` value. A
builder no root declares is an error that lists what is available, rather than a
silent success. The subcommands are shorthand for a single builder, so combining
one with a contradictory ``--shell`` is also an error.

Options
~~~~~~~

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Flag
     - Effect
   * - ``-s, --shell``
     - Builder(s) to run. Comma-separated, or ``all`` (default).
   * - ``-r, --root-doc``
     - Root document(s) to build, by manifest key. Comma-separated, or ``all``.
   * - ``-a, --autodoc``
     - Generate Python API docs with autosummary. Requires ``src/python/``; the
       CLI warns and continues without it when there is no Python package.
   * - ``-g, --gather``
     - Gather sibling repositories' docs before building. Only valid from an
       ``hmd-docs-bartleby`` checkout that sits next to ``hmd-lib-bartleby-demos``.
   * - ``--title``
     - Document title. Defaults to ``<repo>-<version>``. Sanitized for LaTeX.
   * - ``--no-timestamp-title``
     - Omit the timestamp the container appends to output document names.
   * - ``--confidential``
     - Stamp documents with ``HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT``.
   * - ``--default-logo``
     - Logo URL for HTML and PDF unless one of the two below overrides it.
   * - ``--html-default-logo``
     - HTML logo URL.
   * - ``--pdf-default-logo``
     - PDF cover image URL.
   * - ``--version``
     - Print the version and exit.

Titles are sanitized automatically: spaces, underscores, and the other characters
that break LaTeX text mode or a Makefile target are replaced with hyphens, and the
CLI prints a note when it changes what you passed.

Environment
~~~~~~~~~~~

If ``HMD_HOME`` is set, ``bartleby`` loads ``$HMD_HOME/.config/hmd.env`` before it
does anything else, so shared defaults do not have to be exported by hand. Values
already present in the environment win over the file. A missing ``HMD_HOME`` or a
missing file is not an error; a file that exists but cannot be read or parsed
produces a warning and the build continues.

``bartleby configure`` writes to that same file.

.. list-table::
   :header-rows: 1
   :widths: 40 60

   * - Variable
     - Effect
   * - ``HMD_CONTAINER_REGISTRY``
     - Registry holding the transform image. Defaults to ``ghcr.io/neuronsphere``.
   * - ``HMD_TF_BARTLEBY_VERSION``
     - Transform image tag. Defaults to ``stable``.
   * - ``HMD_BARTLEBY_DEFAULT_LOGO``
     - Default logo URL. Also ``_HTML_`` and ``_PDF_`` variants.
   * - ``HMD_BARTLEBY_CONFIDENTIAL``
     - Turn on the confidentiality stamp without the flag. Accepts ``true``,
       ``1``, ``yes``, ``on``, in any case.
   * - ``HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT``
     - The text that stamp uses.
   * - ``HMD_BARTLEBY_<SHELL>_CONFIG``
     - Per-builder Sphinx config as a JSON object, e.g.
       ``HMD_BARTLEBY_PDF_CONFIG='{"papersize": "a4paper"}'``. The manifest's
       ``bartleby.config.builders.<shell>`` takes precedence over it.
   * - ``HMD_BARTLEBY__<SHELL>__<KEY>``
     - A single per-builder config value. Highest precedence of the config layers.
   * - ``HMD_HOME``
     - Root of the NeuronSphere config directory: supplies ``.config/hmd.env``
       and ``bartleby/styles/`` (see *Custom Style Overrides*).
   * - ``PIP_USERNAME`` / ``PIP_PASSWORD``
     - Credentials for autodoc installs from a private index. When both are set
       the CLI writes a temporary ``pip.conf`` (mode 600), mounts it as the
       container's pip secret, and deletes it afterwards. Otherwise ``~/.pip/pip.conf``
       is used if present.
   * - ``DOCKER_HOST`` and the other ``DOCKER_*`` variables
     - Honoured as usual. With no ``DOCKER_HOST`` the CLI asks the active docker
       context, then falls back to the Colima, Docker Desktop, and Rancher socket
       locations — so Colima works without exporting anything.

Interrupting a build
~~~~~~~~~~~~~~~~~~~~

Ctrl-C cancels the build and removes the container it started, so the next run
does not trip over a leftover container with the same name.

Configuring Multiple Root Documents
-----------------------------------

Bartleby can render multiple different root documents with different builders available to each. For example, you might want to render one toctree 
for PDF outputs and another for HTML. The below config enables that. It should be put in the ``meta-data/manifest.json`` file of the project.

Alongside the manifest's other top-level keys:

.. code-block:: json

    {
        "bartleby": {
            "roots": {
                "html_doc": {
                    "root_doc": "index",
                    "builders": ["html"]
                },
                "pdf_doc": {
                    "root_doc": "pdf_index",
                    "builders": ["pdf"]
                }
            }
        }
    }

Each entry should contain a ``root_doc`` property equal to the name of the root RST file to use, without the ``.rst`` extension.
The paths are relative to the ``docs/`` directory.
The entry should also have an array of ``builders`` that can render this document. The values in the array should be valid options sent to the ``--shell`` flag, i.e. html, pdf, revealjs, confluence.

A builder may instead be given as an object, which lets it carry its own Sphinx
config:

.. code-block:: json

    {
        "bartleby": {
            "roots": {
                "html_doc": {
                    "root_doc": "index",
                    "builders": [
                        "html",
                        {"shell": "pdf", "config": {"papersize": "a4paper"}}
                    ]
                }
            }
        }
    }

Builder config is merged from four layers, lowest priority first: the root's own
``config``, then ``bartleby.config.builders.<shell>`` (or the
``HMD_BARTLEBY_<SHELL>_CONFIG`` JSON environment variable when the manifest has
none), then a builder object's inline ``config``, then any
``HMD_BARTLEBY__<SHELL>__<KEY>`` environment variables.

When a specific shell is specified on the command line, only documents with that value in their ``builders`` array will be rendered. For example, running ``bartleby --shell html`` or the shortcut ``bartleby html`` will only render the ``html_doc`` document.


Rendering RevealJS Slideshows
------------------------------

Bartleby supports rendering RST documents as RevealJS slideshows via the ``slides`` subcommand.
The ``hmd-tf-bartleby`` transform image includes the ``sphinx-revealjs`` extension, so no additional
installation is needed.

To use this feature, add ``"revealjs"`` to the ``builders`` array for the desired root document in
``meta-data/manifest.json``:

.. code-block:: json

    {
        "bartleby": {
            "roots": {
                "presentation": {
                    "root_doc": "slides_index",
                    "builders": ["revealjs"]
                }
            }
        }
    }

Then render the slideshow with:

.. code-block:: bash

    bartleby slides

You can also target a specific root document:

.. code-block:: bash

    bartleby slides --root-doc presentation

Alternatively, the ``--shell`` flag still works:

.. code-block:: bash

    bartleby --shell revealjs

Combining External Documentation Sources
-----------------------------------------

Bartleby can pull documentation from multiple external repositories into a single combined site. This uses
``pre_build_artifacts`` to download build artifacts from other repos and ``bartleby.sources`` to configure
how docs are staged and injected into the toctree.

**Step 1: Configure pre_build_artifacts**

In ``meta-data/manifest.json``, declare the build artifacts to download:

.. code-block:: json

    {
        "build": {
            "pre_build_artifacts": [
                ["hmd-ms-transform@1.0:build", "target/artifacts/transform"],
                ["hmd-ms-deployment@0.1:build", "target/artifacts/deployment"]
            ]
        }
    }

**Step 2: Configure bartleby.sources**

Add a ``bartleby.sources`` section to the manifest. Each key becomes a staging directory under ``docs/_sources/``:

.. code-block:: json

    {
        "bartleby": {
            "sources": {
                "transform": {
                    "artifact_path": "target/artifacts/transform",
                    "docs_root": "docs",
                    "title": "Transform Service API"
                },
                "deployment": {
                    "artifact_path": "target/artifacts/deployment",
                    "docs_root": "docs",
                    "title": "Deployment Service API"
                }
            },
            "roots": {
                "index": {
                    "root_doc": "index",
                    "builders": ["html", "pdf"]
                }
            }
        }
    }

Source configuration fields:

- **key** (e.g., ``"transform"``): Name used as the staging directory under ``docs/_sources/``
- **artifact_path** (optional): Where ``pre_build_artifacts`` downloaded the build artifact, relative to the repo root
- **docs_root** (optional, default ``"docs"``): Subdirectory within the artifact containing RST files
- **title**: Display name used as the toctree caption

If ``artifact_path`` is omitted, the key is treated as a path relative to ``docs/`` and docs must already be in place.

**Step 3: Control toctree placement (optional)**

Add the ``.. bartleby-sources::`` marker directive in your ``index.rst`` to control where the external
toctree entries are inserted:

.. code-block:: rst

    Welcome
    =======

    .. toctree::
       :maxdepth: 2
       :caption: Local Docs

       local/overview

    .. bartleby-sources::

    Indices and tables
    ==================

If no marker is present, Bartleby inserts entries before the "Indexes and tables" or "Indices and tables"
heading. If neither is found, entries are appended to the end of the file.

**Step 4: Build**

Run a full build (downloads artifacts then runs Bartleby):

.. code-block:: bash

    hmd build

Or run pre-build artifacts separately, then Bartleby:

.. code-block:: bash

    hmd build -pdo
    bartleby

Bartleby will automatically stage the external docs, inject toctree entries, run the Sphinx transform,
and clean up staging files and restore ``index.rst`` afterwards (even if the build fails).

Custom Style Overrides
-----------------------

Bartleby supports custom style overrides at two levels:

1. **Global** — organisation-wide defaults stored at ``$HMD_HOME/bartleby/styles/``
2. **Per-repo** — project-specific overrides in the repo's ``docs/`` directory

**Precedence:** Built-in defaults < global (``$HMD_HOME``) < per-repo (``docs/``). The repo always wins.

Global Style Directory
~~~~~~~~~~~~~~~~~~~~~~

Create subdirectories under ``$HMD_HOME/bartleby/styles/`` for each output format:

.. code-block:: text

    $HMD_HOME/bartleby/styles/
      revealjs/
        _static/
          corporate-theme.css
          logo.png
        _templates/
          revealjs/section.html
        conf_overrides.json
      html/
        _static/
          custom.css
        _templates/
          layout.html
        conf_overrides.json
      pdf/
        _static/
          cover-logo.png
        conf_overrides.json

Each format subdirectory can contain:

- ``_static/`` — Static assets (CSS, images, JS) copied into the Sphinx ``_static`` directory
- ``_templates/`` — Jinja2 templates that override Sphinx defaults
- ``conf_overrides.json`` — Sphinx configuration values applied at build time

Per-Repo Style Overrides
~~~~~~~~~~~~~~~~~~~~~~~~~

Place overrides directly in the repo's ``docs/`` directory:

- ``docs/_static/`` — Static assets (these already work with Bartleby)
- ``docs/_templates/`` — Jinja2 template overrides
- Sphinx config overrides via the ``config`` key in ``meta-data/manifest.json``

Because per-repo files are copied after global files, they take precedence. Any file with the
same name in both global and per-repo will use the per-repo version.

conf_overrides.json
~~~~~~~~~~~~~~~~~~~~

The ``conf_overrides.json`` file sets Sphinx configuration variables. Each key-value pair is
injected into ``conf.py`` at build time using ``globals()[key] = value``.

**RevealJS example:**

.. code-block:: json

    {
        "revealjs_theme": "night",
        "revealjs_css_files": ["_static/corporate-theme.css"]
    }

**HTML example:**

.. code-block:: json

    {
        "html_theme": "furo",
        "html_theme_options": {
            "sidebar_hide_name": true
        }
    }

**PDF example:**

.. code-block:: json

    {
        "latex_theme": "manual",
        "latex_elements": {
            "preamble": "\\usepackage{charter}"
        }
    }

Disabling Default Styles
~~~~~~~~~~~~~~~~~~~~~~~~~~

To completely replace the built-in ``styles.css``, set ``disable_default_styles`` to ``true``
in either ``conf_overrides.json`` or the manifest ``config``:

.. code-block:: json

    {
        "disable_default_styles": true
    }

Additional Setup
-----------------

Ensure the ``hmd-tf-bartleby`` image is built locally using the hmd docker build tool (``hmd docker build`` from the
repository root) prior to running the bartleby CLI. The bartleby CLI will look for a local image under the registry name in
the HMD_CONTAINER_REGISTRY environment variable (defaults to the HMD registry) in order to run the transform.

Requirements and Traceability
------------------------------

What the CLI must do is written down as sphinx-needs requirements under
:doc:`requirements/index`, and every requirement is linked to the tests that
verify it. Coverage is declared in the test source — a ``// Requirements:`` doc
comment on a Go test, ``[Tags]`` on a Robot test — and the matrix is generated
from those declarations:

.. code-block:: bash

    make reqs          # regenerate docs/requirements/traceability.rst
    make reqs-check    # fail on a gap, a bad reference, or a stale matrix
    make check         # fmt + vet + unit tests + reqs-check

``make check`` needs neither Docker nor Sphinx, so the traceability rules hold on
a laptop and in CI. Adding a requirement without a test, or a test without a
requirement, fails the check.

Development Setup
-------------------

After building a new ``hmd-tf-bartleby`` image locally, you need to set the environment variable ``HMD_TF_BARTLEBY_VERSION`` to the new tag created.
By default, the tag will be the contents of ``./meta-data/VERSION`` and ``-linux-<amd64|arm64>`` based on the architecture you are running.
For example on Intel machines with VERSION as 0.1, the tag will be ``0.1-linux-amd64``. 
So, you can run and test your newly built local image by setting ``export HMD_TF_BARTLEBY_VERSION=0.1-linux-amd64``.