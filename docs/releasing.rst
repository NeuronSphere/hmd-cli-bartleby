.. bartleby release process

Releasing Bartleby
==================

Bartleby uses `GoReleaser <https://goreleaser.com>`_ to cross-compile the CLI
binary and publish it to GitHub Releases. A Homebrew formula in the
``neuronsphere/homebrew-tap`` repository is updated automatically so that users
can install the latest version with ``brew``.

Prerequisites
-------------

- Push access to ``neuronsphere/hmd-cli-bartleby``
- A GitHub Personal Access Token (PAT) with ``repo`` scope on
  ``neuronsphere/homebrew-tap``, stored as the repository secret
  ``HOMEBREW_TAP_TOKEN`` in ``hmd-cli-bartleby``
- `GoReleaser <https://goreleaser.com/install/>`_ installed locally (only
  needed for local releases — CI handles this automatically)

How a Release Works
-------------------

The release pipeline is driven by Git tags. When a tag matching ``v*`` is
pushed, the GitHub Actions workflow at ``.github/workflows/release.yml`` runs
GoReleaser, which:

1. Cross-compiles the ``bartleby`` binary for four targets:

   - ``darwin/amd64`` (macOS Intel)
   - ``darwin/arm64`` (macOS Apple Silicon)
   - ``linux/amd64``
   - ``linux/arm64``

2. Packages each binary into a tarball named
   ``bartleby_<version>_<os>_<arch>.tar.gz``.

3. Creates a GitHub Release on ``hmd-cli-bartleby`` with the tarballs
   attached.

4. Pushes an updated ``Formula/bartleby.rb`` (with the new version and
   sha256 checksums) to ``neuronsphere/homebrew-tap``.

Cutting a Release
-----------------

1. Ensure all changes are merged to ``main`` and tests pass:

   .. code-block:: bash

      make test
      make test-robot

2. Update the version in ``meta-data/VERSION``:

   .. code-block:: bash

      echo "1.2.0" > meta-data/VERSION

3. Commit, tag, and push:

   .. code-block:: bash

      git add meta-data/VERSION
      git commit -m "release: v1.2.0"
      git tag v1.2.0
      git push origin main --tags

4. Monitor the release workflow in GitHub Actions. Once complete, verify:

   - The `GitHub Release <https://github.com/neuronsphere/hmd-cli-bartleby/releases>`_
     contains four tarballs.
   - The ``Formula/bartleby.rb`` in ``homebrew-tap`` has been updated with
     the new version and checksums.

5. Verify the Homebrew install:

   .. code-block:: bash

      brew update
      brew upgrade bartleby
      bartleby --help

Local Release (without CI)
--------------------------

If you need to cut a release from your workstation rather than through CI:

.. code-block:: bash

   export GITHUB_TOKEN="ghp_..."
   export HOMEBREW_TAP_GITHUB_TOKEN="ghp_..."
   goreleaser release --clean

GoReleaser Configuration
-------------------------

The ``.goreleaser.yaml`` at the repository root controls the entire release
pipeline. Key sections:

``builds``
   Specifies the Go source directory (``src/go/bartleby``), target
   platforms, and ``-ldflags`` used to embed the version at compile time via
   ``-X main.version={{ .Version }}``.

``archives``
   Controls the tarball naming template. The names must match the URL
   pattern in the Homebrew formula.

``brews``
   Tells GoReleaser which tap repository, directory, and formula metadata to
   use when auto-updating the Homebrew formula after a successful release.

``changelog``
   Auto-generates release notes from commit messages, excluding commits
   prefixed with ``docs:`` or ``test:``.

Version Injection
-----------------

The binary's version is set at build time through Go linker flags::

   -ldflags "-X main.version=1.2.0"

Both the project ``Makefile`` (reading ``meta-data/VERSION``) and GoReleaser
(using the Git tag) inject this value. The ``bartleby --help`` output and
any future ``bartleby version`` subcommand will reflect it.
