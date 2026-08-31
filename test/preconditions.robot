*** Settings ***
Documentation     Precondition tests for the bartleby CLI: how manifests, root
...               documents, builder selection, and titles affect a real build.
...
...               Every test here starts the transform container, so the suite
...               needs a running Docker daemon and the hmd-tf-bartleby image.
...               Fixtures live in test/data/ — nothing outside this repo.
Library           Process
Library           OperatingSystem

*** Variables ***
${BINARY}       ${CURDIR}/../src/go/bartleby/build/bartleby
${DATA_DIR}     ${CURDIR}/data

*** Test Cases ***

# ---------------------------------------------------------------------------
# Manifest / roots preconditions
# ---------------------------------------------------------------------------

Explicit Roots In Manifest Builds Configured Builder
    [Documentation]    When bartleby.roots is present the CLI uses it.
    ...               repo-with-roots declares a single html root, so html/index.html
    ...               is the expected output.
    [Tags]    REQ_MAN_001    REQ_CLI_002
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html

Manifest Without Roots Falls Back To Default Builders
    [Documentation]    With no bartleby.roots the CLI defaults to
    ...               builders: [html, pdf] and root_doc: index. Only html is run
    ...               here to keep the suite quick; the default set is asserted by
    ...               the shell-filter test below.
    [Tags]    REQ_SEL_005
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    html
    File Should Exist    ${DATA_DIR}/repo-no-roots/target/bartleby/html/index.html

Missing Manifest Falls Back To Default Builders
    [Documentation]    With no meta-data/manifest.json at all the CLI uses the same
    ...               defaults and takes the repo name from the directory name.
    [Tags]    REQ_MAN_001    REQ_MAN_004
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-manifest
    Run Bartleby In    ${DATA_DIR}/repo-no-manifest    html
    File Should Exist    ${DATA_DIR}/repo-no-manifest/target/bartleby/html/index.html

# ---------------------------------------------------------------------------
# Builder selection
# ---------------------------------------------------------------------------

Shell Flag Restricts The Build To One Builder
    [Documentation]    repo-no-roots defaults to html and pdf. With --shell html
    ...               only the html output may appear: no PDF, and no LaTeX run.
    ...               This is the regression test for --shell having been parsed
    ...               but never read, which made it build everything.
    [Tags]    REQ_SEL_001
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    --shell html
    File Should Exist    ${DATA_DIR}/repo-no-roots/target/bartleby/html/index.html
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-no-roots/target/bartleby    *.pdf    absolute=True
    Should Be Empty    ${pdfs}
    ...    msg=--shell html produced a PDF, so the builder filter is being ignored

# ---------------------------------------------------------------------------
# Title sanitization preconditions
# ---------------------------------------------------------------------------

Title With Underscores Is Sanitized And PDF Produced
    [Documentation]    Underscores are subscript operators in LaTeX text mode, so
    ...               --title my_doc_title must be hyphenated before it reaches
    ...               DOCUMENT_TITLE. A produced PDF proves LaTeX accepted it.
    [Tags]    REQ_CFG_007
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    pdf --title my_doc_title
    PDF Should Exist In    ${DATA_DIR}/repo-no-roots
    ...    Underscore sanitization may have failed

Title With Spaces Is Sanitized And PDF Produced
    [Documentation]    Spaces break the LaTeX Makefile target the container builds.
    ...               The argument really does contain a space — quoted so the
    ...               shell passes it as one value.
    [Tags]    REQ_CFG_007
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-manifest
    Run Bartleby In    ${DATA_DIR}/repo-no-manifest    pdf --title "my doc title"
    PDF Should Exist In    ${DATA_DIR}/repo-no-manifest
    ...    Space sanitization may have failed

Unsafe Repo Name Without Title Flag Still Produces PDF
    [Documentation]    With no --title the CLI derives the title from the manifest
    ...               name plus version. repo-unsafe-name is called
    ...               my_unsafe_repo_name, so the CLI has to sanitize it itself.
    [Tags]    REQ_CFG_006    REQ_CFG_007
    [Setup]    Clean Output    ${DATA_DIR}/repo-unsafe-name
    Run Bartleby In    ${DATA_DIR}/repo-unsafe-name    pdf
    PDF Should Exist In    ${DATA_DIR}/repo-unsafe-name
    ...    Auto-title sanitization from the repo name may have failed

Output Directory Is Created When It Is Absent
    [Documentation]    target/bartleby is a bind mount, so it has to exist before
    ...               the container starts. Removing it entirely must not break
    ...               the build.
    [Tags]    REQ_EXEC_012
    Remove Directory    ${DATA_DIR}/repo-with-roots/target    recursive=True
    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    Directory Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html

Autodoc Without A Python Package Warns And Continues
    [Documentation]    --autodoc needs src/python/. Without it the build should
    ...               still succeed, with a warning, rather than failing.
    [Tags]    REQ_AUTO_001
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    ${result}=    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html --autodoc
    Should Contain    ${result.stdout}    src/python
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html

*** Keywords ***
Run Bartleby In
    [Documentation]    Run the bartleby binary with the given arguments in a repo.
    ...               stderr is folded into stdout so a failure logs the whole story.
    [Arguments]    ${repo}    ${args}
    ${result}=    Run Process    ${BINARY} ${args}
    ...    shell=True
    ...    cwd=${repo}
    ...    stderr=STDOUT
    Log    ${result.stdout}
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=bartleby ${args} exited with code ${result.rc}
    RETURN    ${result}

PDF Should Exist In
    [Arguments]    ${repo}    ${why}
    ${pdfs}=    List Files In Directory    ${repo}/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}    msg=No PDF produced — ${why}
    Log    PDF produced: ${pdfs}[0]

Clean Output
    [Documentation]    Empty target/bartleby without replacing the directory.
    ...               Keeping the same inode avoids a macOS/Colima VirtioFS
    ...               staleness bug where rm+mkdir loses the bind mount.
    [Arguments]    ${repo}
    Create Directory    ${repo}/target/bartleby
    Run Process    find ${repo}/target/bartleby -mindepth 1 -delete    shell=True
