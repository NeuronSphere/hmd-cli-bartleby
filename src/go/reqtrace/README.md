# reqtrace

Generates and checks a requirements traceability matrix from sphinx-needs directives and
test annotations.

Requirements are declared in `docs/requirements/*.rst` as `.. req::` / `.. spec::`
directives. Coverage is declared where the test lives — a Go test in a `// Requirements:`
doc comment, a Robot test in its `[Tags]`. `reqtrace` reads both sides and writes
`docs/requirements/traceability.rst`, failing when a requirement has no test, a test
references a requirement that does not exist, or the generated matrix is stale.

## Use

```sh
go install github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace/cmd/reqtrace@latest

reqtrace                 # regenerate the matrix
reqtrace -check          # fail on a gap or stale output; for CI
reqtrace -repo /path     # explicit repository root (default: walk up for meta-data/manifest.json)
reqtrace -quiet          # print nothing on success
```

### Listing needs as JSON

`-format json` prints every need instead of writing the matrix. Each entry has its
status, a normalised stage (`draft`, `proposed`, `in_progress`, `implemented`, `deferred`,
`closed`, `unknown`), its links, a digest of its text, and the tests that cover it. No
Sphinx build is needed.

```sh
reqtrace -format json                      # this repository; or, run in a directory of
                                           # repositories, every one of them
reqtrace -format json -workspace DIR       # every repository up to two levels below DIR
reqtrace -format json -open -summary       # open needs, counted per repository
reqtrace -format json -repo NAME -stage draft -type spec
reqtrace -format json -changes             # add what changed since the previous call
reqtrace -diff OLD.json NEW.json           # what changed between two listings
```

A workspace listing is served from an index under `$REQTRACE_CACHE_DIR` (default: the user
cache directory). Each call refreshes the index from directory and file modified times and
the repositories' git `HEAD`, and re-reads only what moved. There is no file watcher.
`-rebuild` ignores the index.

Or as a library:

```go
import "github.com/neuronsphere/hmd-cli-bartleby/src/go/reqtrace"
```

## Licence

**Apache License 2.0** — see `LICENSE`.

This module is licensed separately from the rest of the repository, which is under the
Business Source License 1.1. `reqtrace` is a self-contained tool with no dependencies
outside the Go standard library, carved out into its own module so it can be consumed by
projects that cannot take a BSL dependency.

## Layout

This is a nested Go module. Its version tags therefore carry the directory prefix:

```sh
git tag src/go/reqtrace/v0.1.0
```

Plain `v0.1.0` tags apply to the repository root module, not to this one.
