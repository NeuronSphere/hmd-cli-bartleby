.. bartleby installation and development

Bartleby Install and Run
==========================

Installation
-------------

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