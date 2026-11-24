# Cross-Platform Release Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a GitHub Actions workflow that builds `rad` for five OS/arch targets on tag push and publishes the archives to a GitHub Release.

**Architecture:** One workflow file with three jobs: `test` (gate), `build` (matrix over five targets, each producing one archive artifact), and `publish` (downloads all archives, writes `checksums.txt`, creates or updates the Release). Version comes from the pushed tag via `-ldflags "-X main.Version=<tag>"`.

**Tech Stack:** GitHub Actions (hosted `ubuntu-latest`), Go 1.21, `gh` CLI, `tar`/`zip`, `sha256sum`.

**Spec:** `docs/superpowers/specs/2026-11-24-release-workflow-design.md`

## Global Constraints

- CI Go version: `1.21` (`go.mod` declares `go 1.16`; 1.21 is the verified cross-compile toolchain).
- `CGO_ENABLED=0` for every build.
- Targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`. Do not add `windows/arm64` — it is outside the Makefile set.
- Archive names: `rad_<tag>_<goos>_<goarch>.tar.gz`, and `.zip` for windows. `<tag>` includes the leading `v`, e.g. `rad_v0.5.0_linux_amd64.tar.gz`.
- Archive contents: exactly one file at the archive root, named `rad` (`rad.exe` on windows). No enclosing directory.
- Version symbol: `main.Version` (declared at `cmd/rad/main.go:81`), set with `-ldflags "-X main.Version=<tag>"`.
- Workflow permissions: `contents: write`.
- Do not modify `.github/workflows/go.yml`, `Makefile`, `dockerfile`, or any Go source.

## Review Focus

- **`workflow_dispatch` checks out the wrong ref.** On a manual run the default checkout is the branch, which would bake a branch SHA (or an empty version) into the binary and tag the wrong commit. Checkout must use the requested tag explicitly.
- **Archives contain a nested directory.** `tar czf archive.tar.gz dist/` produces `dist/rad` inside; users following the README expect `rad` at the root after extraction. Build into a scratch dir and archive the file by name.
- **Windows binary not named `rad.exe`.** Without the `.exe` suffix the file is not runnable on Windows.
- **Checksums file lists itself or omits an archive.** `sha256sum * > checksums.txt` captures the empty file it is about to write; the glob must be `rad_*` and the file created after hashing.
- **Re-running a release for an existing tag fails.** `gh release create` errors if the Release exists; the publish step must create only when missing and use `--clobber` on upload.
- **Tests failing still publishes.** `publish` must depend on `test`, not run in parallel with it.

---

### Task 1: Add the release workflow

**Files:**
- Create: `.github/workflows/release.yml`
- Test: none (verification is YAML validation plus a local run of the packaging commands)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: workflow named `Release`, triggered by `v*` tags and `workflow_dispatch` with a required `tag` input; assets named per Global Constraints.

- [ ] **Step 1: Write `.github/workflows/release.yml`**

Top-level shape:

```yaml
name: Release

on:
  push:
    tags: ["v*"]
  workflow_dispatch:
    inputs:
      tag:
        description: "Tag to release (e.g. v0.5.0)"
        required: true
        type: string

permissions:
  contents: write
```

`test` job: `runs-on: ubuntu-latest`; `actions/checkout@v4` with `ref: ${{ inputs.tag || github.ref_name }}`; `actions/setup-go@v5` with `go-version: "1.21"`; then `go test ./...`.

`build` job: `needs: test`, `runs-on: ubuntu-latest`, `strategy.fail-fast: false`, matrix `include` with these five entries (goos, goarch, ext):

```yaml
          - {goos: linux,   goarch: amd64, ext: ""}
          - {goos: linux,   goarch: arm64, ext: ""}
          - {goos: darwin,  goarch: amd64, ext: ""}
          - {goos: darwin,  goarch: arm64, ext: ""}
          - {goos: windows, goarch: amd64, ext: ".exe"}
```

Job-level `env` in both `build` and `publish`:

```yaml
    env:
      TAG: ${{ inputs.tag || github.ref_name }}
```

`build` steps, in order:
1. Checkout as above.
2. Setup Go 1.21.
3. Build: `mkdir -p dist` then `CGO_ENABLED=0 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} go build -ldflags "-X main.Version=${TAG}" -o "dist/rad${{ matrix.ext }}" ./cmd/rad`.
4. Package, `working-directory: dist`:
   - windows: `zip "rad_${TAG}_${{ matrix.goos }}_${{ matrix.goarch }}.zip" "rad${{ matrix.ext }}"`
   - otherwise: `tar czf "rad_${TAG}_${{ matrix.goos }}_${{ matrix.goarch }}.tar.gz" "rad${{ matrix.ext }}"`
   Select with a single `run` block using `if [ "${{ matrix.goos }}" = "windows" ]`.
5. Upload: `actions/upload-artifact@v4`, `name: rad-${{ matrix.goos }}-${{ matrix.goarch }}`, `path: dist/rad_*`, `if-no-files-found: error`.

`publish` job: `needs: build`, `runs-on: ubuntu-latest`, steps:
1. `actions/download-artifact@v4` with `path: dist`, `merge-multiple: true`.
2. Write checksums: `cd dist && sha256sum rad_* > checksums.txt`.
3. Create/upload, with `GH_TOKEN: ${{ github.token }}`:
   `gh release view "$TAG" >/dev/null 2>&1 || gh release create "$TAG" --title "$TAG" --generate-notes`
   then `gh release upload "$TAG" dist/* --clobber`.

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/release.yml')); print('ok')"`
Expected: `ok`

- [ ] **Step 3: Lint the workflow**

Run: `go install github.com/rhysd/actionlint/cmd/actionlint@latest && "$(go env GOPATH)/bin/actionlint" .github/workflows/release.yml`
Expected: no output, exit 0. If the install fails (no network), note it and continue — Step 4 is the substantive check.

Then confirm the job wiring, which is what enforces the test gate:

Run: `grep -nE '^  (test|build|publish):|^    needs:' .github/workflows/release.yml`
Expected: `build:` is followed by `needs: test`, and `publish:` is followed by `needs: build`.

- [ ] **Step 4: Reproduce the packaging commands locally**

Create a scratch dir outside the repo (`/tmp/rad-release-check`), then for each of the five matrix entries run the exact build and package commands from Step 1 with `TAG=v0.5.0-test`. Expected results:

- Five archives exist: `rad_v0.5.0-test_{linux_amd64,linux_arm64,darwin_amd64,darwin_arm64,windows_amd64}.tar.gz` plus the `.zip` for windows.
- `tar tzf rad_v0.5.0-test_linux_amd64.tar.gz` prints exactly `rad` (no directory prefix); `unzip -l` on the windows archive shows `rad.exe`.
- Extract the linux/amd64 archive and run `./rad --version`: output ends with `rad version v0.5.0-test`.
- `sha256sum rad_* > checksums.txt` produces exactly five lines (the five archives; `checksums.txt` itself is not matched), and `sha256sum -c checksums.txt` reports all five OK.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "Add cross-platform release workflow publishing to GitHub Releases"
```

---

### Task 2: Document release downloads in the README

**Files:**
- Modify: `README.md:42-46` (Installation section)

**Interfaces:**
- Consumes: asset naming from Task 1.
- Produces: nothing later tasks depend on.

- [ ] **Step 1: Replace the prebuilt-binaries line**

Current line 44 reads: "Prebuilt binaries are published on this repository's [releases page](https://github.com/herraristotle/rad/releases); the binary is named `rad`. You can also build from source as above."

Replace it with a short block that states: releases are created automatically when a `v*` tag is pushed; assets are `rad_<tag>_<os>_<arch>.tar.gz` (`.zip` on Windows); extract and the binary is `rad` (`rad.exe`); `checksums.txt` is attached for SHA-256 verification, with the command `sha256sum -c checksums.txt`.

- [ ] **Step 2: Verify the claims match the workflow**

Re-read `.github/workflows/release.yml` asset names and confirm each name pattern quoted in the README appears verbatim in the workflow. Fix any mismatch.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "Document release asset names and checksum verification"
```

---

### Task 3: Verify the release end to end

**Files:**
- None modified.

**Interfaces:**
- Consumes: Tasks 1 and 2.
- Produces: nothing.

This task requires the branch to be on GitHub and a tag to exist on the remote, so it runs after merge and needs the human partner's go-ahead to trigger.

- [ ] **Step 1: Confirm no release exists for the target tag**

Run: `gh release view v0.5.0`
Expected: "release not found" (or the existing release, if one has since been created — then skip to Step 3).

- [ ] **Step 2: Trigger the workflow manually against the existing tag**

Run: `gh workflow run release.yml -f tag=v0.5.0`
Then: `gh run list --workflow=release.yml --limit 1` and `gh run watch <id> --exit-status`.
Expected: run completes successfully; all three jobs green.

- [ ] **Step 3: Inspect the published release**

Run: `gh release view v0.5.0 --json assets -q '.assets[].name' | sort`
Expected, sorted:
`checksums.txt`, `rad_v0.5.0_darwin_amd64.tar.gz`, `rad_v0.5.0_darwin_arm64.tar.gz`, `rad_v0.5.0_linux_amd64.tar.gz`, `rad_v0.5.0_linux_arm64.tar.gz`, `rad_v0.5.0_windows_amd64.zip`.

- [ ] **Step 4: Spot-check one asset**

Run: `gh release download v0.5.0 -p 'rad_v0.5.0_linux_amd64.tar.gz' -O /tmp/rad-rel.tar.gz && tar tzf /tmp/rad-rel.tar.gz`
Expected: exactly `rad`. Extract and run `./rad --version`; expect the output to end with `rad version v0.5.0`.

- [ ] **Step 5: Re-run against the same tag to prove idempotency**

Run: `gh workflow run release.yml -f tag=v0.5.0`, watch it, then `gh release view v0.5.0 --json assets -q '.assets | length'`.
Expected: the run succeeds (the create step is skipped because the release exists, uploads use `--clobber`) and the asset count is still 6 — no duplicate or missing assets.
