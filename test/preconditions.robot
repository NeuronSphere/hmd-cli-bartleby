*** Settings ***
Documentation     Precondition tests for the bartleby Go CLI.
...               Uses minimal self-contained repos in test/data/ to verify
...               behaviour under different manifest and title configurations.
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
    ...               repo-with-roots declares a single html root so only
    ...               html/index.html should appear in the output.
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html

Manifest Without Roots Falls Back To Default HTML Build
    [Documentation]    When bartleby.roots is absent the CLI defaults to
    ...               builders: [html, pdf] with root_doc: index.
    ...               We run only html to keep the test fast.
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    html
    File Should Exist    ${DATA_DIR}/repo-no-roots/target/bartleby/html/index.html

Missing Manifest Falls Back To Default HTML Build
    [Documentation]    With no meta-data/manifest.json the CLI falls back to the
    ...               same defaults. Directory name is used as the repo name.
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-manifest
    Run Bartleby In    ${DATA_DIR}/repo-no-manifest    html
    File Should Exist    ${DATA_DIR}/repo-no-manifest/target/bartleby/html/index.html

# ---------------------------------------------------------------------------
# Title sanitization preconditions
# ---------------------------------------------------------------------------

Title With Underscores Is Sanitized And PDF Produced
    [Documentation]    Passing --title with underscores (e.g. my_doc_title) would
    ...               break LaTeX. The CLI must replace underscores with hyphens
    ...               before passing DOCUMENT_TITLE to the container.
    ...               Uses repo-no-roots which defaults to html+pdf builders.
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    pdf --title my_doc_title
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-no-roots/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}
    ...    msg=No PDF produced — underscore sanitization may have failed

Title With Spaces Is Sanitized And PDF Produced
    [Documentation]    Spaces in a title break the LaTeX Makefile target.
    ...               The CLI must replace spaces with hyphens.
    ...               Uses repo-no-manifest (distinct container name from the
    ...               underscores test above to avoid container-name collision).
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-manifest
    Run Bartleby In    ${DATA_DIR}/repo-no-manifest    pdf --title my-doc-title
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-no-manifest/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}
    ...    msg=No PDF produced — space sanitization may have failed

Unsafe Repo Name Without Title Flag Still Produces PDF
    [Documentation]    When no --title is given the CLI derives the document title
    ...               from the manifest name + version. If that name contains
    ...               underscores (e.g. my_unsafe_repo_name) the CLI must sanitize
    ...               it automatically before passing it to the container.
    [Setup]    Clean Output    ${DATA_DIR}/repo-unsafe-name
    Run Bartleby In    ${DATA_DIR}/repo-unsafe-name    pdf
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-unsafe-name/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}
    ...    msg=No PDF produced — auto-title sanitization from repo name may have failed

*** Keywords ***
Run Bartleby In
    [Documentation]    Run the bartleby binary with given args from a specific repo directory.
    [Arguments]    ${repo}    ${args}
    ${result}=    Run Process    ${BINARY} ${args}
    ...    shell=True
    ...    cwd=${repo}
    ...    stdout=STDOUT
    ...    stderr=STDOUT
    Log    ${result.stdout}
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=bartleby ${args} exited with code ${result.rc}

Clean Output
    [Documentation]    Wipe and recreate the target/bartleby dir for a test repo.
    [Arguments]    ${repo}
    Remove Directory    ${repo}/target/bartleby    recursive=True
    Create Directory    ${repo}/target/bartleby
