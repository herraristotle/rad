# Cross-platform release workflow — design

Date: 2026-11-24
Status: approved

## Intent

The repo ships 14 tags (`v0.1.0` … `v0.5.0`) but has no GitHub Release, and
`README.md:44` already promises "Prebuilt binaries are published on this
repository's releases page". Nothing in CI builds or publishes them — the only
workflow (`.github/workflows/go.yml`) builds and tests on master/PR.

Goal: pushing a version tag produces a GitHub Release with ready-to-run
binaries for every platform the project supports, with no manual steps.

## Goals

- One hosted GitHub Actions runner builds all target platforms and publishes
  them to a GitHub Release for the pushed tag.
- Assets match what the `Makefile` (`build_all`) already produces.
- The version string baked into the binary is the release tag, so `rad
  --version` reports it.
- Tests gate the release: a red test suite publishes nothing.
- `v0.5.0` (and any past tag) can be backfilled by hand.

## Non-goals

- No change to `go.yml`; master/PR CI stays as-is.
- No Homebrew tap, Docker image publishing, or package-manager distribution.
- No draft releases, no pre-release channel, no signing/notarization.
- No change to `Makefile`, `dockerfile`, or runtime code.

## Constraints

- `go.mod` declares `go 1.16`; CI uses Go 1.21, which cross-compiles every
  target cleanly (verified locally: all targets build, ~19 MB each).
- `CGO_ENABLED=0` is required for cross-compilation — same as the Makefile.
- `main.Version` in `cmd/rad/main.go:81` is the only version symbol; it is set
  via `-ldflags "-X main.Version=<tag>"`.
- macOS binaries are not signed; users may need to clear quarantine. Accepted.

## Approaches considered

**A. Plain matrix workflow with `gh release create` — chosen.**
A matrix over OS/arch runs `go build` with the version ldflags, packages each
binary, writes `checksums.txt`, and creates one Release with all assets.
Matches the Makefile targets exactly, full control of asset naming, no extra
config files to maintain.

**B. GoReleaser + `goreleaser-action`.**
Same artifacts, less workflow code, changelog and checksums for free. Rejected
because its advantages (brew taps, multi-binary packaging, signing pipeline)
are not in play for a single-binary repo, and it adds a second config file and
toolchain to keep in sync with the Makefile.

## Decisions

- **Trigger:** push of tags matching `v*`, plus `workflow_dispatch` with a `tag`
  input for backfilling older releases.
- **Runner:** `ubuntu-latest`, one job, `permissions: contents: write`.
- **Targets:** `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
  `windows/amd64` — the Makefile set.
- **Archive layout:** `rad_<tag>_<os>_<arch>.tar.gz`, and `.zip` on Windows,
  each containing the binary named `rad` / `rad.exe` at the archive root, so
  the README's "the binary is named `rad`" stays true after extraction.
- **Checksums:** single `checksums.txt` (SHA-256) attached to the release.
- **Release notes:** auto-generated from commits since the previous tag.
- **Ordering:** tests run first; publishing only starts if they pass.
- **Draft/prerelease:** none; a tag push publishes immediately.

## Left to the builder

- Exact action versions (`actions/checkout`, `actions/setup-go`) and whether to
  use `gh release create` versus `softprops/action-gh-release` for the upload.
- Whether the test step runs once in a separate job or inside the build job.
- Whether to add `windows/arm64` (it builds, but is outside the Makefile set).
