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

   The Python-based CLI has been replaced by the Go binary above.
   These instructions are retained for reference only.

The HMD CLI Bartleby tool can be installed using ``pip`` and specifying the HMD pypi server (via command line or using
a pip config file).

.. code-block:: bash

    pip install hmd-cli-bartleby


Running the Bartleby Transform
--------------------------------

The bartleby CLI uses the ``--repo-name`` and ``--repo-version`` arguments inherited from the base cli app to help build
the rendered documents. However, the CLI is also built with the assumption that the command is being run from the desired
repository root in order to avoid dependencies upon the HMD_REPO_HOME environment variable:

.. code-block:: bash

    hmd bartleby <command>

For the ``<command>``, any combination of the configured options listed under the bartleby transform (see the
"transforms" document under the ``hmd-tf-bartleby`` repo) can be entered as input. If rendered documents in multiple
formats is desired, enter the options as a comma-separated list with *no spaces*:

.. code-block:: bash

    hmd bartleby --shell <option1>,<option2>

Configuring Multiple Root Documents
-----------------------------------

Bartleby can render multiple different root documents with different builders available to each. For example, you might want to render one toctree 
for PDF outputs and another for HTML. The below config enables that. It should be put in the ``meta-data/manifest.json`` file of the project.

.. code-block:: json

    {
        ...
        "bartleby": {
            "roots": {
                "html_doc": {
                    "root_doc": "index",
                    "builders": ["html"],
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

When a specific shell is specified on the command line, only documents with that value in their ``builders`` array will be rendered. For example, running ``hmd bartleby --shell html`` or the shortcut ``hmd bartleby html`` will only render the ``html_doc`` document.


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

    hmd bartleby slides

You can also target a specific root document:

.. code-block:: bash

    hmd bartleby slides -rd presentation

Alternatively, the ``--shell`` flag still works:

.. code-block:: bash

    hmd bartleby --shell revealjs

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
    hmd bartleby

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

Development Setup
-------------------

After building a new ``hmd-tf-bartleby`` image locally, you need to set the environment variable ``HMD_TF_BARTLEBY_VERSION`` to the new tag created.
By default, the tag will be the contents of ``./meta-data/VERSION`` and ``-linux-<amd64|arm64>`` based on the architecture you are running.
For example on Intel machines with VERSION as 0.1, the tag will be ``0.1-linux-amd64``. 
So, you can run and test your newly built local image by setting ``export HMD_TF_BARTLEBY_VERSION=0.1-linux-amd64``.