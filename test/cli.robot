*** Settings ***
Documentation     Contract tests for the bartleby CLI surface: versioning, help,
...               and the errors it must report rather than swallow.
...
...               These tests need neither Docker nor the network, so they run in
...               seconds and are the right place to catch a flag that stops
...               being wired up. `make test-cli` runs just this suite.
...
...               Each test is tagged with the requirements it verifies; see
...               docs/requirements/.
Library           Process
Library           OperatingSystem

*** Variables ***
${BINARY}       ${CURDIR}/../src/go/bartleby/build/bartleby
${DATA_DIR}     ${CURDIR}/data

*** Test Cases ***

Version Flag Reports A Version
    [Documentation]    --version must work and must not print the literal "dev",
    ...               which would mean the -X main.version ldflag was dropped.
    [Tags]    REQ_CLI_006
    ${result}=    Run Bartleby    --version    ${DATA_DIR}/repo-with-roots    expect_rc=0
    Should Contain    ${result.stdout}    bartleby
    Should Not Contain    ${result.stdout}    dev

Version Subcommand Matches The Flag
    [Tags]    REQ_CLI_006
    ${flag}=    Run Bartleby    --version    ${DATA_DIR}/repo-with-roots    expect_rc=0
    ${sub}=     Run Bartleby    version      ${DATA_DIR}/repo-with-roots    expect_rc=0
    Should Be Equal    ${flag.stdout}    ${sub.stdout}

Help Lists Every Command
    [Tags]    REQ_CLI_009
    ${result}=    Run Bartleby    --help    ${DATA_DIR}/repo-with-roots    expect_rc=0
    FOR    ${command}    IN    html    pdf    slides    puml    update-image    configure    version
        Should Contain    ${result.stdout}    ${command}
    END

Help Lists Every Persistent Flag
    [Tags]    REQ_CLI_009
    ${result}=    Run Bartleby    --help    ${DATA_DIR}/repo-with-roots    expect_rc=0
    FOR    ${flag}    IN    --shell    --root-doc    --autodoc    --gather    --title
    ...    --no-timestamp-title    --confidential    --default-logo    --html-default-logo
    ...    --pdf-default-logo
        Should Contain    ${result.stdout}    ${flag}
    END

Unknown Builder Is An Error Not A Silent No-Op
    [Documentation]    A --shell value no root declares used to print
    ...               "No builds to run." and exit 0, which reads as success.
    [Tags]    REQ_SEL_002    REQ_CLI_007
    ${result}=    Run Bartleby    --shell confluence    ${DATA_DIR}/repo-no-roots    expect_rc=1
    Should Contain    ${result.stderr}    no builder named "confluence"
    Should Contain    ${result.stderr}    available: html, pdf

Shell Flag Contradicting A Subcommand Is Rejected
    [Documentation]    `bartleby html --shell pdf` used to ignore the flag.
    [Tags]    REQ_CLI_002_SPEC001
    ${result}=    Run Bartleby    html --shell pdf    ${DATA_DIR}/repo-with-roots    expect_rc=1
    Should Contain    ${result.stderr}    --shell

Unknown Root Document Is An Error
    [Tags]    REQ_SEL_004
    ${result}=    Run Bartleby    --root-doc nope    ${DATA_DIR}/repo-with-roots    expect_rc=1
    Should Contain    ${result.stderr}    nope
    Should Contain    ${result.stderr}    available: main

Gather Outside The Docs Repo Is Rejected
    [Documentation]    Gather rewrites docs/, so it must refuse to run anywhere
    ...               except a hmd-docs-bartleby checkout.
    [Tags]    REQ_GATH_001
    ${result}=    Run Bartleby    --gather hmd-lib-widget    ${DATA_DIR}/repo-with-roots    expect_rc=1
    Should Contain    ${result.stderr}    hmd-docs-bartleby

Malformed Manifest Is Reported
    [Documentation]    A manifest that cannot be parsed must fail loudly instead
    ...               of being treated as absent and silently built with defaults.
    [Tags]    REQ_MAN_002
    ${repo}=    Set Variable    ${TEMPDIR}/bartleby-bad-manifest
    Create Directory    ${repo}/meta-data
    Create File    ${repo}/meta-data/manifest.json    {"bartleby": {"roots":
    ${result}=    Run Bartleby    html    ${repo}    expect_rc=1
    Should Contain    ${result.stderr}    manifest.json
    [Teardown]    Remove Directory    ${repo}    recursive=True

Unknown Flag Is Rejected
    [Tags]    REQ_CLI_008
    ${result}=    Run Bartleby    --not-a-flag    ${DATA_DIR}/repo-with-roots    expect_rc=1
    Should Contain    ${result.stderr}    unknown flag

Unexpected Argument Is Rejected
    [Documentation]    bartleby takes no positional arguments; a stray word is
    ...               far more likely to be a mistake than something to ignore.
    [Tags]    REQ_CLI_008
    ${result}=    Run Bartleby    htlm    ${DATA_DIR}/repo-with-roots    expect_rc=1
    Should Contain    ${result.stderr}    htlm

A Runtime Error Does Not Print Usage
    [Documentation]    Usage text is for a malformed command line. Printing it
    ...               after a runtime failure buries the error that matters.
    [Tags]    REQ_CLI_007
    ${result}=    Run Bartleby    --shell confluence    ${DATA_DIR}/repo-no-roots    expect_rc=1
    Should Start With    ${result.stderr}    Error:
    Should Not Contain    ${result.stderr}    Usage:
    Should Be Empty    ${result.stdout}

*** Keywords ***
Run Bartleby
    [Documentation]    Run the binary in a directory and assert its exit code.
    ...               stderr is captured separately so error text can be asserted.
    [Arguments]    ${args}    ${cwd}    ${expect_rc}=0
    ${result}=    Run Process    ${BINARY} ${args}
    ...    shell=True
    ...    cwd=${cwd}
    Log    ${result.stdout}
    Log    ${result.stderr}
    Should Be Equal As Integers    ${result.rc}    ${expect_rc}
    ...    msg=bartleby ${args} exited with ${result.rc}, expected ${expect_rc}
    RETURN    ${result}
