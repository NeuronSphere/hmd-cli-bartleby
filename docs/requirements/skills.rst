.. Bundled skill requirements

Bundled Agent Skills
====================

Bartleby ships the agent skills that describe how to do documentation and
requirements work with it. They are only useful once they are on a machine where
an agent will read them, so distributing them is part of the CLI rather than a
separate step.

They used to reach users only through the Python package. The Go binary is now
the product — installed from Homebrew, with no Python runtime — so it carries
them itself.

.. req:: Carry the skills in the binary
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_001
    :status: implemented

    The bundled skills shall be embedded in the binary, so that installing them
    needs no network access, no repository checkout, and no Python runtime.

.. req:: List the bundled skills
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_002
    :status: implemented

    ``bartleby skills`` and ``bartleby skills list`` shall print each bundled
    skill's name and its description, so a user can see what is on offer without
    installing anything.

.. req:: Print one skill
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_003
    :status: implemented

    ``bartleby skills show <name>`` shall print that skill's instructions
    verbatim, so it can be read, diffed against an installed copy, or piped into
    a tool that takes instructions on standard input.

.. req:: Install where an agent will find them
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_004
    :status: implemented

    ``bartleby skills install`` shall write each skill to
    ``<destination>/<name>/SKILL.md``, defaulting to the user-level skills
    directory ``~/.claude/skills``. ``--dir`` shall override the destination.

.. req:: Install everything, or just what was asked for
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_005
    :status: implemented

    ``bartleby skills install`` with no arguments shall install every bundled
    skill; named arguments shall install only those.

.. req:: Do not overwrite someone's edits
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_006
    :status: implemented

    Where a skill is already installed, an identical copy shall be reported as
    current and left alone, and a **differing** copy shall be left alone as well
    and reported — it may carry local edits, and losing those silently is worse
    than declining to act. ``--force`` shall overwrite it.

    Installing twice therefore changes nothing the second time, which is what
    makes the command safe to put in a setup script.

.. req:: Install into a repository instead
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_007
    :status: implemented

    ``--project`` shall install into ``.claude/skills`` in the working
    directory, so a repository can carry the skills for everyone who checks it
    out rather than each person installing them per machine.

.. spec:: Two destinations is a contradiction, not a precedence
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_007_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_SKILL_007
    :status: implemented

    ``--dir`` and ``--project`` both name a destination, so giving both shall be
    an error rather than one quietly winning.

.. req:: Name what is available when a skill is not
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_008
    :status: implemented

    A skill name that is not bundled shall be an error that lists the names that
    are, since the likely cause is a typo or a half-remembered name.

.. req:: Say what was done, and where
    :id: HMD_CLI_BARTLEBY_REQ_SKILL_009
    :status: implemented

    Installing shall report the destination directory and, for each skill,
    whether it was installed, updated, already current, or left alone. A user
    who does not know where the files went cannot edit or remove them.
