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

The bartleby CLI uses the ``--repo-name`` and ``--repo-version`` arguments inherited from the base cli app to identify
the repository that the rendered documents should be generated for. For example, if the command is being run from
somewhere other than the repository root, the repository can be specified as follows:

.. code-block:: bash

    hmd --repo-name <repo> bartleby <command>

For the ``<command>``, any combination of the configured options listed under the bartleby transform (see the
"transforms" document under the ``hmd-tf-bartleby`` repo) can be entered as input. If rendered documents in multiple
formats is desired, enter the options as a comma-separated list with *no spaces*:

.. code-block:: bash

    hmd --repo-name <repo> bartleby <option1>,<option2>


Additional Setup
+++++++++++++++++

Ensure the ``hmd-tf-bartleby`` image is built locally using the hmd docker build tool prior to running the bartleby CLI.
The bartleby CLI will look for a local image with the standard HMD naming convention to run the transform.