# bartleby

Render a repository's reStructuredText documentation — HTML, PDF, RevealJS
slides, PlantUML images — with Sphinx and LaTeX running inside Docker, so
nothing but Docker has to be installed on your machine.

## Install

macOS:

```bash
brew install neuronsphere/tap/bartleby
```

The `neuronsphere/tap` prefix is required — bartleby is not in Homebrew core,
so a bare `brew install bartleby` will not find it. That one command registers
the tap and installs the binary.

On Linux, take a tarball from the [latest
release](https://github.com/neuronsphere/hmd-cli-bartleby/releases/latest);
Homebrew supports casks on macOS only. To build from source, `make build`.

Docker (or [Colima](https://github.com/abiosoft/colima)) must be running when
you build.

## Use

Run it from the root of the repository you want to document:

```bash
bartleby              # every builder configured for every root document
bartleby html         # HTML only
bartleby pdf          # PDF only
bartleby explain      # ask Claude why the last build failed
bartleby reqs --check # check the requirements traceability matrix
```

Output lands in `target/bartleby/`, logs in `target/bartleby/logs/`.

Which builders run, and for which root documents, comes from `bartleby` in
`meta-data/manifest.json`. [`docs/install_and_run.rst`](docs/install_and_run.rst)
covers the manifest, every flag, and `$HMD_HOME` configuration.

## Agent skills and agents

The skills that describe how to do documentation and requirements work with
bartleby — and the agents that do it — are built into the binary:

```bash
bartleby skills                   # list them
bartleby skills install           # install to ~/.claude/skills
bartleby skills install --project # or into .claude/skills in this repository

bartleby agents                   # the same, for agents
bartleby agents install           # install to ~/.claude/agents
```

| Skill | Covers |
|-------|--------|
| `add-nerd` | Writing a NERD — a proposal for a change |
| `add-requirement` | Adding a requirement to the baseline and covering it with a test |
| `check-traceability` | Fixing a failing traceability check, honestly |
| `combine-docs` | Stitching several repositories' documentation into one build |
| `document-api` | Documenting a microservice API in RST |
| `plan-docs` | Deciding what kind of document something is, and where it belongs (Diátaxis) |

| Agent | Covers |
|-------|--------|
| `rst-doc-expert` | Authoring, organising, and building RST documentation |

Installing is safe to repeat, and anything you have edited locally is reported
and left alone rather than overwritten.

## Documentation

| Document | Contents |
|----------|----------|
| [`docs/install_and_run.rst`](docs/install_and_run.rst) | Installing, configuring, and every command and flag |
| [`docs/requirements/`](docs/requirements/index.rst) | What the CLI is required to do, traced to the tests that prove it |
| [`docs/releasing.rst`](docs/releasing.rst) | Cutting a release and updating the Homebrew cask |
| [`docs/proposals/`](docs/proposals/index.rst) | Design proposals |

The transform image the CLI runs is
[hmd-tf-bartleby](https://github.com/NeuronSphere/hmd-tf-bartleby).

## Development

```bash
make build     # build to src/go/bartleby/build/bartleby
make test      # Go unit tests
make reqs      # regenerate the requirements traceability matrix
make check     # fmt, vet, tests, and traceability — what CI should run
```

The Go CLI supersedes the original Python plugin, which remains in `src/python/`
for anyone still on `hmd bartleby`.
