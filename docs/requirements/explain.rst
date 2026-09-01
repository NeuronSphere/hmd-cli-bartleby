.. Log explanation requirements

Explaining a Failed Build
=========================

Sphinx and LaTeX report failures in their own terms, several inferences away from
the change a person needs to make, and they do it inside thousands of lines of
log. ``bartleby explain`` makes one request to Claude with the evidence that
matters and asks for the answer.

One request, no tools, no agent loop: the aim is a single well-informed attempt at
the question "what do I do about this", not an autonomous debugging session.

.. req:: Explain the last build on request
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_001
    :status: implemented

    ``bartleby explain`` shall gather the evidence from the most recent build in
    the working directory and report an explanation. ``--builder`` shall select a
    particular builder's log and ``--log`` a file directly. With no logs at all,
    it shall say so and say to run a build first.

.. req:: Offer an explanation when a build fails
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_002
    :status: implemented

    With ``--explain``, or ``BARTLEBY_EXPLAIN`` set to a truthy value, a failed
    build shall attempt an explanation immediately.

.. spec:: The explanation never changes the outcome
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_002_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_EXPL_002
    :status: implemented

    The build's own error and exit code shall be what the caller sees, whatever
    happens in the explanation attempt — no credentials, no network, a refusal, a
    timeout. A failure to explain shall be a warning on standard error. The
    attempt shall be bounded by a timeout so a broken build is not held open.

.. req:: Send nothing without being asked
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_003
    :status: implemented

    The evidence includes excerpts of the user's documentation, so nothing shall
    be sent to the API unless the user runs ``explain`` or opts in for the build.
    ``--dry-run`` shall assemble and print exactly what would be sent, and send
    nothing.

.. req:: Send the evidence that explains a build failure
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_004
    :status: implemented

    One request shall carry: the Sphinx warnings and errors in full; the tail of
    the combined build log; the LaTeX error slice when the PDF builder ran; the
    repository name, version, builder, and manifest; and the source lines every
    warning refers to.

    The source excerpts are what turn a warning into an answerable question — a
    citation of ``index.rst:8`` means little without the eight lines around it.

.. spec:: Overlapping excerpts are merged
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_004_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_EXPL_004
    :status: implemented

    Several warnings about neighbouring lines of one document shall produce one
    excerpt with each cited line marked, rather than near-identical copies of the
    same passage.

.. spec:: A build with no cited lines still shows the document
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_004_SPEC002
    :links: HMD_CLI_BARTLEBY_REQ_EXPL_004
    :status: implemented

    A LaTeX failure cites a line of the generated ``.tex`` file, so no warning
    resolves to a source file. In that case the root documents named in the
    manifest shall be attached instead, capped, and the substitution stated in the
    request — the alternative is asking the model to explain an error with none of
    the markup that caused it in view.

.. req:: Cap the request and say what was left out
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_005
    :status: implemented

    The assembled evidence shall be held under a size cap, giving up the LaTeX
    slice first, then the log tail, then — only if they alone exceed the cap — the
    warnings. Whatever is dropped shall be listed in the request itself, so the
    model knows the evidence is partial rather than being quietly misled.

.. req:: Resolve the paths in a warning to files on this machine
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_006
    :status: implemented

    Sphinx runs against a copy of the documentation inside the container, so it
    cites paths such as ``/tmp/tmpyzfqp5i3/source/index.rst`` that exist nowhere
    on the host. Those shall be mapped back to the repository's own files —
    ``docs/index.rst`` — so the excerpt comes from the source the user can edit. A
    path that cannot be resolved shall be skipped rather than guessed at.

.. req:: Find credentials the way the SDK does
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_007
    :status: implemented

    Credentials shall come from the Anthropic SDK's own resolution —
    ``ANTHROPIC_API_KEY``, ``ANTHROPIC_AUTH_TOKEN``, a profile stored by ``ant
    auth login``, or workload identity — rather than from one variable. When none
    is available, the CLI shall name the ways to provide one and mention
    ``--dry-run``, and shall not attempt a request.

.. req:: Let the prompt be replaced
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_008
    :status: implemented

    The instruction sent with the evidence shall be overridable, in precedence
    order: ``--prompt-file``, ``BARTLEBY_EXPLAIN_PROMPT_FILE``,
    ``BARTLEBY_EXPLAIN_PROMPT``, a repository's ``.bartleby/explain-prompt.md``,
    then the built-in. The CLI shall report which one it used. A named prompt file
    that is missing or empty shall be an error, not a silent fall back to the
    built-in.

.. req:: Ask a capable model by default, and allow a cheaper one
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_009
    :status: implemented

    The default model shall be ``claude-opus-5``: reading a build log is a
    reasoning problem and the request is infrequent. ``--model`` and
    ``BARTLEBY_EXPLAIN_MODEL`` shall override it.

.. spec:: The answer is streamed
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_009_SPEC001
    :links: HMD_CLI_BARTLEBY_REQ_EXPL_009
    :status: implemented
    :tags: trace-exempt

    The response shall be streamed to the terminal as it arrives, because a
    thinking model can take a while and a CLI that prints nothing looks broken.

    *Verification:* by inspection and manual run. Exercising it needs live API
    credentials, which the test environment deliberately does not have; the
    request-assembly and response-handling paths around it are covered by tests
    with a stub requester.

.. req:: Keep the explanation with the logs
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_010
    :status: implemented

    The explanation shall be written to ``target/bartleby/logs/<builder>-explain.md``
    alongside the logs it explains, so it can be reread, shared, or attached to an
    issue without asking again.

.. req:: Ask once
    :id: HMD_CLI_BARTLEBY_REQ_EXPL_011
    :status: implemented

    One invocation shall make exactly one request. No retry loop that quietly
    multiplies cost, and no tool-use loop.
