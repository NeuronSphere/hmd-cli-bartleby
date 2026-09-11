# Changelog

## Unreleased

### Added

- `bartleby docx` and `bartleby pptx`, for the transform's new pandoc builders.
  `REQ_CLI_002` is amended rather than duplicated, and a test asserts every
  subcommand pins the builder it is named for — wiring one to the wrong shell is
  otherwise invisible without running a real build.
- Five more skills, and the licence position on each recorded in the skill
  itself rather than left for someone to rediscover:
  - `diagram-c4` — C4 architecture diagrams in PlantUML. Verified that
    `!include <C4/C4_Container>` renders with networking disabled, since
    PlantUML bundles the C4 macros; the `!include https://raw.githubusercontent…`
    form found in most examples makes every render depend on GitHub.
  - `write-from-template` — a rubric over The Good Docs Project templates
    (**MIT No Attribution**, so vendorable with no obligation), keyed to the
    Diátaxis modes.
  - `review-prose` — the Google developer documentation style guide, **CC BY
    4.0**, so its rules are quoted directly.
  - `review-plain-language` — a separate review from style: whether the reader
    can find, understand, and act on it. Carries the :rfc:`2119` keyword
    discussion.
  - `review-vale` — Vale configuration, style packages, vocabularies, and CI.
    Notes that `.rst` support needs `docutils` and reports nothing without it,
    which looks exactly like a clean run.
- `add-requirement` now covers RFC 2119: that `shall` and `must` are the same
  thing, that `should` does not belong in a baseline because no test can fail
  on it, and RFC 2119's own instruction to use the keywords sparingly.

- `plan-docs` skill: decide which of the four Diátaxis modes a piece of content
  belongs to — tutorial, how-to, reference, explanation — and audit an existing
  doc set against them. Written as a restatement in our own words, citing
  <https://diataxis.fr>, and quoting nothing: Diátaxis is CC BY-SA 4.0, so
  verbatim inclusion would make the file adapted material and oblige it to carry
  CC BY-SA 4.0 too. The skill says so, in case someone later wants to quote it.


## 2026-09-04 — v2.1.0

### Added

- `bartleby reqs` and `bartleby reqs --check` — the traceability tool is now
  reachable from the CLI, so anyone who has bartleby needs no second install.
  It and the standalone `reqtrace` binary are two front doors onto one
  `reqtrace.Run`, rather than each orchestrating the steps, so they cannot drift
  into disagreeing about what a check means. Requirement `TRACE_010`.
- `reqtrace` ships as its own Homebrew cask: `brew install neuronsphere/tap/reqtrace`.
  A separate cask rather than a second binary in bartleby's, because two casks
  cannot both link the same binary name — bundling it would have made installing
  it alone impossible, and being adoptable without bartleby (or its BSL licence)
  is the reason it was carved out.
- `reqtrace -version`, injected at build time, defaulting to `dev`. Tested by
  building with the ldflag and running the binary, since the failure it guards
  against is the ldflag silently ceasing to apply.
- Requirements `TRACE_008` and `TRACE_009` for the carve-out and the version
  flag. The tests assert the module path, the absence of any `require`
  directive, and the Apache licence, rather than trusting review to notice a
  dependency creeping in.

### Changed

- `reqtrace` moved out of the CLI module into `src/go/reqtrace`, its own
  Apache-2.0 module with no dependency outside the standard library. The CLI
  module no longer contains it at all.
- Every Makefile target that checks sources — `test`, `test-race`, `cover`,
  `vet`, `fmt`, `fmt-check`, `tidy` — now walks both modules. They only covered
  `src/go/bartleby`, so after the carve-out `make check` stopped testing
  reqtrace while the traceability matrix still counted its tests as coverage.
- One release tag now publishes eight archives and two casks. GoReleaser's
  per-module tag support (`monorepo.tag_prefix`) is Pro-only, so reqtrace's cask
  version tracks the bartleby release; its Go module keeps its own tag line for
  `go install`. `docs/releasing.rst` spells out the consequence.


## 2026-09-04 — v2.0.0

First released version of the Go CLI: **v2.0.0**. The major bump records that
the Python entrypoint is no longer the product — the Go binary is.

### Added

- `bartleby agents` — the bundled agents install the same way, to
  `~/.claude/agents`. An agent installs as a single `<name>.md` rather than a
  directory, because that is what an agent runtime reads. Like the skills, they
  previously shipped only in the Python package.
- `bartleby skills` — the bundled agent skills are embedded in the binary and
  install with `bartleby skills install`, defaulting to `~/.claude/skills`, or
  `--project` for `.claude/skills` in the repository. `list` and `show` read them
  without installing. Previously the skills reached users only through the Python
  package, which a Homebrew install does not have. Re-running installs nothing
  twice, and a locally edited skill is reported and left alone unless `--force`.
- Two skills for requirements work, usable in any repository: `add-requirement`
  (write a requirement into the baseline and cover it with a test in the same
  change) and `check-traceability` (diagnose each way the check fails, and the
  judgment about what a test may claim to cover). Both drive `reqtrace`, which is
  installable on its own: `go install
  github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/cmd/reqtrace@latest`.
- A Homebrew cask, so installing is one command:
  `brew install neuronsphere/tap/bartleby`. The tap prefix is required; bartleby
  is not in Homebrew core. Linux installs come from the release tarballs, since
  Homebrew supports casks on macOS only.
- A `README.md`, which the repository did not have.

### Changed

- The skills and agents moved to `src/go/bartleby/{skills,agents}/` so the Go
  binary can embed them. The install rules they share — idempotence, never
  clobbering a local edit, reporting each outcome — live in one place
  (`internal/bundle`), so the two cannot drift apart.
  `setup.py` copies them from there, so the Python package still ships the same
  files from one source. Their own instructions now say `bartleby` rather than
  the legacy `hmd bartleby`.
- `meta-data/VERSION` is `2.0.0`, matching the release.
- `.goreleaser.yaml` publishes a `homebrew_casks` entry instead of the
  deprecated `brews` one. GoReleaser deprecated the formulae it generated for
  pre-compiled binaries; casks are the supported path. The cask clears
  `com.apple.quarantine` in a `postflight` hook, without which Gatekeeper kills
  an unsigned binary on first run.
- `force_token: github` pins the release provider. GoReleaser picks it from
  whichever token is in the environment, and a `GITLAB_TOKEN` exported for
  unrelated work was enough to make it generate a cask pointing at `gitlab.com`
  URLs that do not exist.

### Fixed

- The release workflow no longer republishes a tag that has already been
  released, which is what a release cut from a workstation produces. When
  `HOMEBREW_TAP_TOKEN` is absent it publishes the binaries and warns, instead of
  failing the release.
- `archives.format` renamed to `formats`, and the config declares `version: 2`.
  As it stood, `goreleaser check` failed.


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
