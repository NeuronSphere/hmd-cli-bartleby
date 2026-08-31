*** Settings ***
Documentation     End-to-end tests for the bartleby CLI: every builder, the
...               bartleby.sources feature that rewrites docs/ around a build,
...               and how the CLI behaves when a container fails or is
...               interrupted.
...
...               These run the real transform container, so a Docker daemon and
...               the hmd-tf-bartleby image are required. Fixtures are in
...               test/data/ — unlike the first version of this suite, nothing
...               here depends on a sibling checkout outside the repo.
Library           Process
Library           OperatingSystem
Library           String

*** Variables ***
${BINARY}       ${CURDIR}/../src/go/bartleby/build/bartleby
${DATA_DIR}     ${CURDIR}/data
# A title must survive LaTeX: no spaces (they break the Makefile target) and no
# underscores (subscript operators in text mode). The CLI sanitizes anyway; this
# passes something already clean so the test isolates the build itself.
${PDF_TITLE}    bartleby-docs
${REGISTRY}     ghcr.io/neuronsphere
${FAIL_TAG}     bartleby-test-fail

*** Test Cases ***
HTML Build Produces An Index File
    [Tags]    REQ_CLI_002
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html

PDF Build Produces A PDF File
    [Documentation]    The container appends a %Y-%m-%d_%H_%M_%S timestamp to the
    ...               document name, so the file name is not predictable — assert
    ...               on the glob instead.
    [Tags]    REQ_CLI_002
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    pdf --title ${PDF_TITLE}
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-no-roots/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}    msg=No PDF produced
    Log    PDF produced: ${pdfs}[0]

Default Build Produces Every Default Builder
    [Documentation]    With no subcommand and no --shell, a repo without roots
    ...               builds both html and pdf.
    [Tags]    REQ_CLI_001
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    Run Bartleby In    ${DATA_DIR}/repo-no-roots    --title ${PDF_TITLE}
    File Should Exist    ${DATA_DIR}/repo-no-roots/target/bartleby/html/index.html
    ${pdfs}=    List Files In Directory
    ...    ${DATA_DIR}/repo-no-roots/target/bartleby    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}    msg=No PDF produced by the default build

Sources Are Staged Into The Build
    [Documentation]    An artifact-backed source is copied into docs/_sources and
    ...               added to the root document's toctree, so its page and its
    ...               caption appear in the rendered output.
    [Tags]    REQ_SRC_001    REQ_SRC_003
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-sources
    Run Bartleby In    ${DATA_DIR}/repo-with-sources    html
    ${output}=    Set Variable    ${DATA_DIR}/repo-with-sources/target/bartleby/html
    File Should Exist    ${output}/index.html
    File Should Exist    ${output}/_sources/widget/index.html
    ${index}=    Get File    ${output}/index.html
    Should Contain    ${index}    Widget Guide

Sources Leave The Docs Tree Exactly As They Found It
    [Documentation]    Injection edits docs/index.rst and staging writes into
    ...               docs/_sources; both must be undone whether the build passes
    ...               or fails, or the repo is left dirty.
    [Tags]    REQ_SRC_004
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-sources
    ${repo}=      Set Variable    ${DATA_DIR}/repo-with-sources
    ${before}=    Get File    ${repo}/docs/index.rst
    Run Bartleby In    ${repo}    html
    ${after}=     Get File    ${repo}/docs/index.rst
    Should Be Equal    ${before}    ${after}
    ...    msg=docs/index.rst was not restored after the build
    Directory Should Not Exist    ${repo}/docs/_sources
    ...    msg=Staged sources were left behind in docs/

PlantUML Files Are Rendered To Images
    [Tags]    REQ_PUML_001
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-puml
    Run Bartleby In    ${DATA_DIR}/repo-with-puml    puml
    File Should Exist    ${DATA_DIR}/repo-with-puml/target/bartleby/puml_images/sequence.png

A Repo With No PlantUML Files Is Not An Error
    [Tags]    REQ_PUML_002
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    ${result}=    Run Bartleby In    ${DATA_DIR}/repo-with-roots    puml
    Should Contain    ${result.stdout}    No .puml files

Container Output Is Streamed To The Caller
    [Documentation]    The transform logs its progress; those lines have to reach
    ...               the user while the build runs, not be swallowed.
    [Tags]    REQ_EXEC_008
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    ${result}=    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    Should Contain    ${result.stdout}    Transform complete.

A Container That Exits Non-Zero Fails The Build
    [Documentation]    Uses a local image tagged from the real one with its
    ...               entrypoint replaced by /bin/false, so the container fails
    ...               without needing a broken document or a network pull.
    ...
    ...               Note: the transform image itself does not currently fail on
    ...               a failed Sphinx build — it logs the exit code and exits 0 —
    ...               so a genuinely broken document would not exercise this path.
    [Tags]    REQ_EXEC_009
    [Setup]    Create Failing Image
    ${result}=    Run Process    ${BINARY} html
    ...    shell=True
    ...    cwd=${DATA_DIR}/repo-with-roots
    ...    env:HMD_TF_BARTLEBY_VERSION=${FAIL_TAG}
    Log    ${result.stdout}
    Log    ${result.stderr}
    Should Not Be Equal As Integers    ${result.rc}    0
    ...    msg=A failing container must fail the build
    Should Contain    ${result.stderr}    exited with code
    [Teardown]    Remove Failing Image

A Leftover Container Is Removed And The Build Retried
    [Documentation]    A container of the same name is what an interrupted run
    ...               leaves behind, so the next run has to clear it rather than
    ...               refuse to start.
    [Tags]    REQ_EXEC_007
    [Setup]    Clean Output    ${DATA_DIR}/repo-with-roots
    Create Leftover Container    bartleby-inst_test-with-roots_html
    ${result}=    Run Bartleby In    ${DATA_DIR}/repo-with-roots    html
    Should Contain    ${result.stdout}    leftover container
    File Should Exist    ${DATA_DIR}/repo-with-roots/target/bartleby/html/index.html
    [Teardown]    Remove Container If Present    bartleby-inst_test-with-roots_html

Interrupting A Build Removes The Container
    [Documentation]    Ctrl-C must cancel the build and still clean up, or the
    ...               next run collides with the abandoned container.
    [Tags]    REQ_EXEC_010
    [Setup]    Clean Output    ${DATA_DIR}/repo-no-roots
    ${handle}=    Start Process    ${BINARY}    html
    ...    cwd=${DATA_DIR}/repo-no-roots
    ...    stderr=STDOUT
    Wait Until Keyword Succeeds    20s    200ms
    ...    Container Should Exist    bartleby-inst_test-no-roots_html
    Send Signal To Process    SIGINT    ${handle}
    ${result}=    Wait For Process    ${handle}    timeout=60s    on_timeout=kill
    Log    ${result.stdout}
    Should Not Be Equal As Integers    ${result.rc}    0
    ...    msg=An interrupted build must not report success
    Wait Until Keyword Succeeds    30s    500ms
    ...    Container Should Not Exist    bartleby-inst_test-no-roots_html
    [Teardown]    Remove Container If Present    bartleby-inst_test-no-roots_html

*** Keywords ***
Run Bartleby In
    [Arguments]    ${repo}    ${args}
    ${result}=    Run Process    ${BINARY} ${args}
    ...    shell=True
    ...    cwd=${repo}
    ...    stderr=STDOUT
    Log    ${result.stdout}
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=bartleby ${args} exited with code ${result.rc}
    RETURN    ${result}

Clean Output
    [Documentation]    Empty target/bartleby without replacing the directory, to
    ...               avoid the macOS/Colima VirtioFS bind-mount staleness bug.
    [Arguments]    ${repo}
    Create Directory    ${repo}/target/bartleby
    Run Process    find ${repo}/target/bartleby -mindepth 1 -delete    shell=True

Create Failing Image
    [Documentation]    Tag a local copy of the transform image whose entrypoint
    ...               fails immediately. Committing an existing local image needs
    ...               no network and adds no meaningful disk.
    Remove Failing Image
    Docker    docker create --name bartleby-failimg ${REGISTRY}/hmd-tf-bartleby:stable
    ...    Could not create the container to commit from
    Docker    docker commit --change 'ENTRYPOINT ["/bin/false"]' bartleby-failimg ${REGISTRY}/hmd-tf-bartleby:${FAIL_TAG}
    ...    Could not commit the failing test image
    Docker    docker rm bartleby-failimg    Could not clean up the temporary container

Remove Failing Image
    Run Process    docker rm -f bartleby-failimg    shell=True
    Run Process    docker rmi -f ${REGISTRY}/hmd-tf-bartleby:${FAIL_TAG}    shell=True

Create Leftover Container
    [Arguments]    ${name}
    Remove Container If Present    ${name}
    ${result}=    Run Process
    ...    docker create --name ${name} ${REGISTRY}/hmd-tf-bartleby:stable true
    ...    shell=True
    Should Be Equal As Integers    ${result.rc}    0    msg=Could not create the leftover container

Remove Container If Present
    [Arguments]    ${name}
    Run Process    docker rm -f ${name}    shell=True

Container Should Exist
    [Arguments]    ${name}
    ${names}=    Container Names
    Should Contain    ${names}    ${name}    msg=Container ${name} does not exist yet

Container Should Not Exist
    [Arguments]    ${name}
    ${names}=    Container Names
    Should Not Contain    ${names}    ${name}    msg=Container ${name} is still there

Container Names
    [Documentation]    Every container name, running or not. Deliberately avoids
    ...               docker's --filter name=..., because a Robot cell containing
    ...               "name=" is parsed as a named argument.
    ${result}=    Run Process    docker ps -a --format {{.Names}}    shell=True
    RETURN    ${result.stdout}

Docker
    [Documentation]    Run one docker command and assert it succeeded.
    [Arguments]    ${command}    ${why}
    ${result}=    Run Process    ${command}    shell=True
    Log    ${result.stdout} ${result.stderr}
    Should Be Equal As Integers    ${result.rc}    0    msg=${why}: ${result.stderr}
