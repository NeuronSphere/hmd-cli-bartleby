.. NERD001 Specify Root Document

NERD001 Specify Root Document
===================================

.. req:: Specify Root Document
    :id: HMD_CLI_BARTLEBY_NERD001

    The Bartleby CLI should read from a NeuronSphere project manifest configuration for which root documents to render, 
    and what shell builders can be used with them.
    It should default to the root document ``'index'`` and all shell builders.

The Bartleby CLI already reads some configuration from the project's manifest file. 
With this proposal, it will read an additional optional property called ``roots``.
It will be a dictionary with the keys being the name of the root document to render, and the value being a list of valid ``shell`` values to use.
When some runs ``hmd bartleby``, it will run for each combination of root document and valid ``shell`` value.

.. spec:: Allow rendering subset of roots
    :id: HMD_CLI_BARTLEBY_NERD001_SPEC001
    :links: HMD_CLI_BARTLEBY_NERD001
    :status: implemented

    An additional command line option will be added to allow only rendering a specific root document.
    Also if a shell builder is passed via command line, e.g. ``hmd bartelby pdf``, then only roots that specify ``pdf`` in the configuration will be rendered.

.. spec:: Limit default 'index' root to certain builders
    :id: HMD_CLI_BARTLEBY_NERD001_SPEC002
    :links: HMD_CLI_BARTLEBY_NERD001
    :status: implemented

    A special root document shall be reserved for the default 'index'. Specifying this in the manifest is optional but will allow limiting which shell builders run for the default root.