.. bartleby release process

Releasing Bartleby
==================

Bartleby uses `GoReleaser <https://goreleaser.com>`_ to cross-compile the CLI
and publish it to GitHub Releases. A Homebrew cask in the
``neuronsphere/homebrew-tap`` repository is updated in the same pass, so
``brew install neuronsphere/tap/bartleby`` gets the new version.

Prerequisites
-------------

- Push access to ``neuronsphere/hmd-cli-bartleby``
- A GitHub token with write access to ``neuronsphere/homebrew-tap``:

  - for CI, stored as the repository secret ``HOMEBREW_TAP_TOKEN``
  - for a local release, exported as ``HOMEBREW_TAP_GITHUB_TOKEN``

  The built-in ``GITHUB_TOKEN`` will not do — it has no access outside the
  repository it runs in.

- `GoReleaser <https://goreleaser.com/install/>`__ installed locally, for local
  releases only

.. note::

   ``HOMEBREW_TAP_TOKEN`` is **not** currently set on this repository. Until it
   is, a tag pushed to CI publishes the binaries and skips the cask, with a
   warning in the job summary — the release is real but ``brew`` will not see
   the new version. Either add the secret or cut the release locally.

How a Release Works
-------------------

Releases are driven by Git tags matching ``v*``. When one is pushed,
``.github/workflows/release.yml`` runs GoReleaser, which:

1. Cross-compiles ``bartleby`` for four targets:

   - ``darwin/amd64`` (macOS Intel)
   - ``darwin/arm64`` (macOS Apple Silicon)
   - ``linux/amd64``
   - ``linux/arm64``

2. Packages each binary as ``bartleby_<version>_<os>_<arch>.tar.gz``.

3. Creates a GitHub Release with the tarballs and a checksums file attached.

4. Commits an updated ``Casks/bartleby.rb`` to ``neuronsphere/homebrew-tap``
   with the new version, URLs, and sha256 checksums.

The tag prefix matters: the workflow triggers on ``v*``, so the ``1.0.x`` tags
from the Python build train never fired it and never will.

Cutting a Release
-----------------

1. Ensure everything is merged to ``main`` and the tests pass:

   .. code-block:: bash

      make test
      make test-robot

2. Bump ``meta-data/VERSION`` to match the tag you are about to cut, and commit
   it.

3. Tag and push:

   .. code-block:: bash

      git tag v2.1.0
      git push origin v2.1.0

4. Watch the workflow in GitHub Actions, then verify:

   - the `release <https://github.com/neuronsphere/hmd-cli-bartleby/releases>`_
     has four tarballs and a checksums file
   - ``Casks/bartleby.rb`` in ``homebrew-tap`` names the new version
   - the install works:

     .. code-block:: bash

        brew update && brew upgrade --cask bartleby
        bartleby --version

Local Release (without CI)
--------------------------

To cut a release from a workstation — which is how the first one was done,
before the tap secret existed:

.. code-block:: bash

   export GITHUB_TOKEN="<token with write access to hmd-cli-bartleby>"
   export HOMEBREW_TAP_GITHUB_TOKEN="<token with write access to homebrew-tap>"
   git tag v2.1.0
   goreleaser release --clean

GoReleaser creates the release, which creates the tag on GitHub; pushing the
tag yourself is optional but keeps the two in step. The workflow notices a tag
that is already released and exits rather than publishing it a second time.

Rehearse first — this prints exactly what would be published and sends nothing:

.. code-block:: bash

   goreleaser release --snapshot --clean --skip=publish
   cat dist/homebrew/Casks/bartleby.rb

GoReleaser Configuration
-------------------------

``.goreleaser.yaml`` at the repository root drives all of it. The sections
worth knowing:

``force_token: github``
   Pins the release provider. GoReleaser otherwise picks it from whichever
   token happens to be in the environment, and a ``GITLAB_TOKEN`` exported for
   unrelated work is enough to make it treat this as a GitLab project and
   generate a cask full of ``gitlab.com`` URLs that do not exist. It did
   exactly that before the pin.

``builds``
   The Go source directory (``src/go/bartleby``), the target platforms, and the
   ``-ldflags`` that embed the version via ``-X main.version={{ .Version }}``.

``archives``
   The tarball naming template, which the cask's URLs are generated from.

``homebrew_casks``
   Which tap to write to, plus the cask's metadata and its ``postflight`` hook.

``changelog``
   Release notes from commit messages, excluding ``docs:`` and ``test:``.

A Cask, Not a Formula
---------------------

GoReleaser deprecated ``brews``: the formulae it generated for pre-compiled
binaries were a misuse of a mechanism meant to build from source. Casks are the
supported path, and the trade is that **Homebrew only supports casks on
macOS** — so Linux installs come from the release tarballs, which
``docs/install_and_run.rst`` documents.

The binaries are not signed or notarized, so macOS quarantines them. The cask
carries a ``postflight`` hook that clears the quarantine attribute; without it
``brew install`` succeeds and the binary is killed on first run. Signing and
notarizing would make the hook unnecessary.

Version Injection
-----------------

The version is set at build time through Go linker flags::

   -ldflags "-X main.version=2.0.0"

GoReleaser injects the Git tag. The ``Makefile`` injects ``meta-data/VERSION``
instead, which is also the HMD Python build's version input. Keep the two in
step: bump ``meta-data/VERSION`` in the same change as the tag, so a ``make
build`` binary and a released one report the same thing.

A binary built without the flag — a plain ``go build`` — reports ``dev``. The
CLI contract test in ``test/cli.robot`` asserts that a ``make build`` binary
does *not* report ``dev``, which catches the ldflag silently ceasing to apply.
