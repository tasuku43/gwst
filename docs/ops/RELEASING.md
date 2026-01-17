---
title: "Releasing gwst"
status: implemented
---

# Releasing gwst (v0.1.0)

This document describes how we ship gwst as a “touchable OSS” with a reliable install path.

For the pipeline architecture and sequence, see `docs/ops/RELEASE_PIPELINE.md`.

## Source of truth

- **GitHub Releases** is the source of truth for distributed binaries.
- `go install` is **not** a supported install method for end users.
- Pre-release tags (e.g. `v0.1.0-rc.1`) do not update Homebrew automatically.

## Release goals (Epic 1)

- Tag push creates a GitHub Release automatically.
- Release includes **macOS + Linux** binaries for **amd64 + arm64**.
- Release includes `checksums.txt` (SHA256).
- Release notes include links to:
  - Install guide: `docs/guides/INSTALL.md`
  - Compatibility policy: `docs/spec/core/COMPATIBILITY.md`
- `gwst version` (and `gwst --version`) shows `vX.Y.Z` in the GitHub Releases binaries.

## Versioning and build metadata

`gwst version` output is populated via `-ldflags` at build time:

- `internal/cli.version` = tag (e.g. `v0.1.0`)
- `internal/cli.commit` = short commit hash
- `internal/cli.date` = build date

`GoReleaser` is responsible for injecting these.

## Release steps (operator checklist)

1. Ensure CI is green on `main`.
2. Create a tag locally:
   - `git tag v0.1.0`
3. Push the tag:
   - `git push origin v0.1.0`
4. Confirm GitHub Actions `release` workflow finished successfully.
5. Confirm the GitHub Release has:
   - macOS + Linux archives for amd64/arm64
   - `checksums.txt`
   - release note links to INSTALL and COMPATIBILITY
6. For stable tags (no `-rc` / no prerelease suffix), confirm the Homebrew update PR was created and auto-merged.
7. Smoke test by downloading a release artifact and running:
   - `./gwst version`
   - `./gwst --version`

## Distribution

- **Homebrew**: We publish an install path in `docs/guides/INSTALL.md`.
- **mise**: We publish an install path in `docs/guides/INSTALL.md` (GitHub Releases backend).
