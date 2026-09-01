# Changelog

## 2026-09-01

### Added

- `bartleby explain` asks Claude what went wrong in the last build, in one
  request. It sends the Sphinx warnings in full, the tail of the build log, the
  LaTeX error slice for a PDF build, the repository name, version and manifest,
  and the source lines every warning refers to — mapping the container's
  temporary paths (`/tmp/tmpXXXX/source/index.rst`) back to the repository's own
  files. Overlapping excerpts are merged; the payload is capped and says what was
  dropped.
- `--explain`, or `BARTLEBY_EXPLAIN`, has a failed build explain itself. The
  explanation is advisory: the build's own error and exit code survive whatever
  happens, including having no credentials at all.
- `--dry-run` prints the exact payload and sends nothing, because the evidence
  includes excerpts of the user's documentation.
- Credentials come from the Anthropic SDK's own resolution — `ANTHROPIC_API_KEY`,
  `ANTHROPIC_AUTH_TOKEN`, an `ant auth login` profile, or workload identity —
  rather than one variable. Default model `claude-opus-5`, overridable with
  `--model` / `BARTLEBY_EXPLAIN_MODEL`. The answer is written to
  `target/bartleby/logs/<builder>-explain.md`.
- The prompt is replaceable: `--prompt-file`, `BARTLEBY_EXPLAIN_PROMPT_FILE`,
  `BARTLEBY_EXPLAIN_PROMPT`, or a repository's `.bartleby/explain-prompt.md`.
- `--image` / `BARTLEBY_IMAGE` runs an explicitly named transform image, which is
  what makes a locally built one usable: `bartleby --image hmd-tf-bartleby:local`.
- 16 requirements for the two features (areas `EXPL` and `EXEC`), with tests. The
  request path is covered through a stub requester, so the suite needs no API key.

### Changed

- Depends on `github.com/anthropics/anthropic-sdk-go`.


## 2026-08-31

Requirements-driven documentation. What the CLI must do is now written down as
sphinx-needs items, every requirement is linked to the tests that verify it, and
the link is enforced.

### Added

- `docs/requirements/`: 85 requirement items (60 `req`, 25 `spec`) covering the
  command surface, builder and root-document selection, manifest reading, config
  resolution, container execution, sources, gather, the HMD environment, autodoc,
  and the traceability process itself. IDs are area-coded —
  `HMD_CLI_BARTLEBY_REQ_<AREA>_<NNN>`.
- Coverage is declared in the test source: a `// Requirements:` doc comment on a
  Go test, `[Tags]` on a Robot test. All 126 Go tests and 32 Robot tests are
  annotated.
- `tools/reqtrace` generates `docs/requirements/traceability.rst` — a coverage
  table by requirement, sphinx-needs `needtable` views, and a `test` item per
  test linked to what it verifies. `make reqs` regenerates it.
- `make reqs-check`, now part of `make check`, fails on a requirement with no
  test, a test with no requirement, a reference to a requirement that does not
  exist, a duplicate ID, a broken `:links:` target, or a stale generated page.
  It needs neither Docker nor Sphinx.
- `trace-exempt` for requirements no automated test can reasonably cover. Two
  qualify, both about pulling a multi-gigabyte image; each says how it is
  verified instead.
- Tests for behaviour the requirements demanded and nothing covered: container
  exit-code handling, log streaming, leftover-container recovery and
  cancellation (through a narrowed `dockerAPI` interface, so no daemon is
  needed), Docker endpoint detection, the `configure` flow, and a guard that no
  non-test source reintroduces a compose invocation.
- Robot tests for the interrupt path, a container that exits non-zero, a
  leftover container, PlantUML rendering, output-directory creation, autodoc
  without a Python package, unknown flags and arguments, and that a runtime
  error prints no usage block. Fixtures `repo-with-puml` and `repo-with-sources`.
- This repository now has a `bartleby` manifest section, which is also how it
  configures sphinx-needs (`needs_id_regex`, `needs_warnings`) for its own docs.

### Fixed

- **A failing build was reported as a success.** `ContainerWait` was registered
  with `WaitConditionNotRunning` before the container started; a created but
  unstarted container is already "not running", so the wait returned immediately
  with status 0 and every exit code was ignored. It now waits for
  `WaitConditionNextExit`. Found by writing the requirement for it.
- Two long-standing documentation warnings: a duplicate link target in
  `releasing.rst`, and an elided `...` that made a JSON example unparseable.

### Known issue (in hmd-tf-bartleby, not the CLI)

- The transform image does not fail on a failed Sphinx build: `hmd_tf_bartleby.py`
  captures the exit code and only logs it, so a document that does not compile
  produces a container that exits 0. The CLI reports what the container tells it,
  so a broken document still looks like a successful build end to end.


## 2026-08-28

The Go CLI reaches parity with the Python implementation and becomes the primary
interface. It talks to the Docker API directly — no `docker-compose` binary and no
`docker-compose-<shell>.yaml` written into `target/`.

### Fixed

- `-s/--shell` was parsed and never read: every invocation built all builders, and
  `bartleby html --shell pdf` silently ignored the flag. The flag now selects
  builders, accepts a comma-separated list, and a contradiction with a subcommand
  is an error.
- `-a/--autodoc` never reached the container. `AUTODOC` was emitted as Go's `true`
  while the transform image compares against Python's `"True"`.
- `--version` did not exist, so GoReleaser's `-X main.version` ldflag was
  discarded. Both `--version` and `bartleby version` now report it.
- A root document declared without `root_doc` was sent as an empty string instead
  of defaulting to `index`.
- A builder the manifest does not declare printed `No builds to run.` and exited
  0. It is now an error that lists the available builders.
- An unreadable or malformed `meta-data/manifest.json` was indistinguishable from
  no manifest at all and silently produced a default build. Both are now reported.
- The container name was built from the raw manifest name, so a repo directory
  containing spaces or punctuation failed container creation with an opaque API
  error. Names are sanitized to Docker's character rules.
- Title sanitization only handled spaces and underscores, leaving `& % # $ ^ \ ~`
  and friends to break LaTeX.
- `HMD_BARTLEBY_CONFIDENTIAL` was compared against `"true"` exactly, so `True` or
  `1` did nothing.
- `PIP_USERNAME`/`PIP_PASSWORD` were detected and then dropped, so credentialed
  autodoc builds lost their index URL. A `pip.conf` is now generated in a mode-600
  temporary file, mounted read-only, and deleted after the build.
- Ctrl-C left the container running, which then collided with the next run.
  Interrupts now cancel the build and still remove the container.
- `DOCKER_TLS_VERIFY`, `DOCKER_CERT_PATH`, and `DOCKER_API_VERSION` were ignored
  whenever `DOCKER_HOST` or a docker context supplied the endpoint.

### Added

- `bartleby.sources`: external documentation trees are staged into
  `docs/_sources/`, added to the root document's toctree via the
  `.. bartleby-sources::` marker or before an "Indexes and tables" section, and
  the docs tree is restored afterwards whether the build succeeds or fails.
- `-g/--gather` for assembling sibling repositories' docs, which now validates
  every named repo before it empties `docs/`.
- `bartleby configure`, writing defaults to `$HMD_HOME/.config/hmd.env`.
- `$HMD_HOME/.config/hmd.env` is loaded on startup. Values already in the
  environment win; a missing file is silent, an unreadable one warns.
- Per-builder config from `bartleby.config.builders.<shell>` and
  `HMD_BARTLEBY_<SHELL>_CONFIG`, plus object-form builders
  (`{"shell": ..., "config": ...}`), which previously failed to parse.
- `confidential` can be set in `bartleby.config`.
- Docker endpoint detection falls back to the Colima, Docker Desktop, and Rancher
  socket locations, so the binary works without `DOCKER_HOST` exported.
- Go unit tests across manifest parsing, build planning, config precedence,
  sources, gather, sanitization, env loading, and container configuration; plus a
  Docker-free `test/cli.robot` contract suite. `make check`, `make cover`,
  `make test-race`, and `make fmt` were added.

### Changed

- Builds run in a deterministic order (root name, then builder name).
- Image pull progress is printed as readable lines instead of the raw JSON stream.
- `update-image` treats an image that is not present locally as nothing to do.
- The Robot suites no longer depend on a sibling `hmd-tf-bartleby` checkout, and
  no longer create stray files named `STDOUT` in the fixture directories.


## 2026-02-26

- feat: add AI agent and skills for RST documentation, API docs, NERDs, and doc combining
- feat: add external documentation sources support via bartleby.sources manifest config

## 2026-02-25

- feat: add slides subcommand for RevealJS slideshow rendering
- feat: support selecting multiple root documents via comma-separated -rd argument
- fix: html and pdf subcommands now respect the -rd flag
- fix: use default builders (html, pdf) when no bartleby manifest config present

## 2025-04-21

- fix: update hmd-cli-app version to 1.2

## 2024-10-18

- fix: bumps dep versions

## 2023-11-01

- feat: reads conf params from env vars
- feat: adds root document feature

## 2023-09-13

- fix: adds custom doc title args

## 2023-03-15

- fix: loads hmd env in update image
- feat: adds update-image command

## 2023-03-09

- feat: adds logo parameters
- feat: adds default config logo values

## 2023-03-08

- fix: adds load_hmd_env to command

## 2023-03-03

- fix: fixes missing var in puml cmd
- feat: adds configure command
- feat: adds HMD_BARTLEBY_CONFIDENTIALITY_STATEMENT

## 2023-03-01

- fix: bumps hmd-cli-app

## 2023-02-13

- fix: bumps hmd-cli-app version

## 2022-11-28

- fix: updates hmd-cli-app version
- feat: updates input path to be prj root

## 2022-09-29

- feat: bumps default img version

## 2022-07-19

- feat: updates indexes and bumps tf-bartleby version

## 2022-07-15

- feat: bumps tf-bartleby version

## 2022-03-16

- fix: corrects docker compose file

## 2022-03-11

- feat: bumps image version

## 2022-03-09

- feat: adds puml command to generate images

## 2022-03-07

- fix: creates target dir if not exists

## 2022-03-04

- feat: adds gather mode and support for NEP022

## 2022-02-11

- feat: adds support for image version override with env vars

## 2022-01-20

- fix: fixes gitignore egg-info

## 2022-01-19

- fix: remove all dependency on hmd repo home

## 2022-01-18

- feat: update repo path when no hmd repo home exists

## 2021-12-09

- feat: add autodoc flag
- fix: use the correct version to find the bartleby tf image and add docs
- test: add placeholder test
- feat: add support for multiple commands from shell input

## 2021-12-07

- feat: initial code checkin

## 2021-12-06

- feat: add python tech
- feat: generate initial repo structure
