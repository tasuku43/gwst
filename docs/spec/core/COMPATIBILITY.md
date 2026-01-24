---
title: "Compatibility policy"
status: implemented
---

# Compatibility policy

## Versioning

gwiac uses SemVer-style versioning (`vX.Y.Z`).

During `v0.x`:
- Breaking changes may happen, but will be documented clearly in release notes.
- We aim to keep the core workflow stable, but do not guarantee backward compatibility.

Starting at `v1.0.0`:
- Breaking changes will only happen in major version bumps.

## Supported install methods

Supported:
- GitHub Releases (binaries)
- Homebrew (stable releases via this repo's Formula)
- mise (GitHub Releases backend)

Not supported:
- `go install ...@vX.Y.Z` as an end-user install method

## Supported platforms (v0.1.0)

- macOS: `amd64`, `arm64`
- Linux: `amd64`, `arm64`

Windows is not part of the v0.1.0 distribution plan.
