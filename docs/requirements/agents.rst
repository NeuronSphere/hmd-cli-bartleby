.. Bundled agent requirements

Bundled Agents
==============

Bartleby bundles agent definitions as well as skills — an agent is a persona and
a standing brief, where a skill is a procedure. They are distributed the same
way and for the same reason: the Go binary is the product, so it has to carry
them.

The one substantive difference is the shape on disk. A skill installs as a
directory containing ``SKILL.md``; an agent installs as a single ``<name>.md``,
because that is what an agent runtime reads out of its agents directory.

.. req:: Carry the agents in the binary
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_001
    :status: implemented

    The bundled agents shall be embedded in the binary, so that installing them
    needs no network access, no repository checkout, and no Python runtime.

.. req:: List and print the bundled agents
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_002
    :status: implemented

    ``bartleby agents`` and ``bartleby agents list`` shall print each bundled
    agent's name and description. ``bartleby agents show <name>`` shall print
    that agent's definition verbatim.

.. req:: Install an agent as a single file
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_003
    :status: implemented

    ``bartleby agents install`` shall write each agent to
    ``<destination>/<name>.md``, defaulting to ``~/.claude/agents``, with
    ``--dir`` overriding the destination. The bundled layout — a directory
    containing ``AGENT.md`` — shall not be reproduced at the destination.

.. req:: Install everything, or just what was asked for
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_004
    :status: implemented

    ``bartleby agents install`` with no arguments shall install every bundled
    agent; named arguments shall install only those. A name that is not bundled
    shall be an error that lists the names that are.

.. req:: Do not overwrite someone's edits
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_005
    :status: implemented

    Where an agent is already installed, an identical copy shall be reported as
    current and left alone, and a differing copy shall be left alone as well and
    reported. ``--force`` shall overwrite it.

.. req:: Install into a repository instead
    :id: HMD_CLI_BARTLEBY_REQ_AGENT_006
    :status: implemented

    ``--project`` shall install into ``.claude/agents`` in the working
    directory, so a repository can carry its agents for everyone who checks it
    out.
