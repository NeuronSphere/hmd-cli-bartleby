*** Settings ***
Documentation     Integration tests for the Go bartleby CLI binary.
...               Runs the binary against the peer hmd-tf-bartleby project and
...               verifies that expected output files are produced.
Library           Process
Library           OperatingSystem

*** Variables ***
# Override BINARY on the command line: robot --variable BINARY:/path/to/bartleby ...
${BINARY}         ${CURDIR}/../src/go/bartleby/build/bartleby
${TARGET_REPO}    ${CURDIR}/../../hmd-tf-bartleby
${OUTPUT_DIR}     ${TARGET_REPO}/target/bartleby
# Title must be a single hyphen-separated token: spaces break the LaTeX Makefile
# target name and underscores break LaTeX text mode (treated as subscript).
${PDF_TITLE}      bartleby-docs

*** Test Cases ***
HTML Build Produces Index File
    [Documentation]    Running 'bartleby html' should produce html/index.html in target/bartleby
    [Setup]    Clean Output Dir
    Run Bartleby    html
    File Should Exist    ${OUTPUT_DIR}/html/index.html

PDF Build Produces A PDF File
    [Documentation]    Running 'bartleby pdf --title' should produce a timestamped PDF in
    ...               target/bartleby. The --title flag avoids underscores in the document
    ...               title (the timestamp format %Y-%m-%d_%H_%M_%S contains underscores
    ...               which break LaTeX outside math mode when used as a title).
    [Setup]    Clean Output Dir
    Run Bartleby    pdf --title ${PDF_TITLE}
    ${pdfs}=    List Files In Directory    ${OUTPUT_DIR}    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}    msg=No PDF found in ${OUTPUT_DIR}
    Log    PDF produced: ${pdfs}[0]

Default Build Produces Both HTML And PDF
    [Documentation]    Running 'bartleby --title' with no subcommand builds all default builders
    [Setup]    Clean Output Dir
    Run Bartleby    --title ${PDF_TITLE}
    File Should Exist    ${OUTPUT_DIR}/html/index.html
    ${pdfs}=    List Files In Directory    ${OUTPUT_DIR}    *.pdf    absolute=True
    Should Not Be Empty    ${pdfs}    msg=No PDF found after default build

*** Keywords ***
Run Bartleby
    [Documentation]    Execute the bartleby binary with optional subcommand and flags
    ...               from the target repo directory.
    [Arguments]    ${args}
    ${result}=    Run Process    ${BINARY} ${args}
    ...    shell=True
    ...    cwd=${TARGET_REPO}
    ...    stdout=STDOUT
    ...    stderr=STDOUT
    Log    ${result.stdout}
    Should Be Equal As Integers    ${result.rc}    0
    ...    msg=bartleby ${args} exited with code ${result.rc}

Clean Output Dir
    [Documentation]    Empty target/bartleby without removing the directory itself.
    ...               Keeping the same inode avoids a macOS/Colima VirtioFS bind-mount
    ...               staleness bug where rm+mkdir causes Docker to lose the mount point.
    Create Directory    ${OUTPUT_DIR}
    Run Process    find ${OUTPUT_DIR} -mindepth 1 -delete    shell=True
