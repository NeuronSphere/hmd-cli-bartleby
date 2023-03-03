=================
HMD CLI Bartleby
=================

Installation
+++++++++++++

The HMD CLI Bartleby tool can be installed using ``pip`` and specifying the HMD pypi server (via command line or using
a pip config file).

.. code-block:: bash

    pip install hmd-cli-bartleby


Running the Bartleby Transform
+++++++++++++++++++++++++++++++

The bartleby CLI uses the ``--repo-name`` and ``--repo-version`` arguments inherited from the base cli app to help build
the rendered documents. However, the CLI is also built with the assumption that the command is being run from the desired
repository root in order to avoid dependencies upon the HMD_REPO_HOME environment variable:

.. code-block:: bash

    hmd --repo-name <repo> bartleby <command>

For the ``<command>``, any combination of the configured options listed under the bartleby transform (see the
"transforms" document under the ``hmd-tf-bartleby`` repo) can be entered as input. If rendered documents in multiple
formats is desired, enter the options as a comma-separated list with *no spaces*:

.. code-block:: bash

    hmd --repo-name <repo> bartleby <option1>,<option2>


Additional Setup
+++++++++++++++++

Ensure the ``hmd-tf-bartleby`` image is built locally using the hmd docker build tool (``hmd docker build`` from the
repository root) prior to running the bartleby CLI. The bartleby CLI will look for a local image under the registry name in
the HMD_CONTAINER_REGISTRY environment variable (defaults to the HMD registry) in order to run the transform.

Development Setup
+++++++++++++++++

After building a new ``hmd-tf-bartleby`` image locally, you need to set the environment variable ``HMD_TF_BARTLEBY_VERSION`` to the new tag created.
By default, the tag will be the contents of ``./meta-data/VERSION`` and ``-linux-<amd64|arm64>`` based on the architecture you are running.
For example on Intel machines with VERSION as 0.1, the tag will be ``0.1-linux-amd64``. 
So, you can run and test your newly built local image by setting ``export HMD_TF_BARTLEBY_VERSION=0.1-linux-amd64``.