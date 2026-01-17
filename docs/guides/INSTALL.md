---
title: "Install gws"
status: implemented
---

# Install gws

Supported platforms (v0.1.0):
- macOS: `amd64`, `arm64`
- Linux: `amd64`, `arm64`

For compatibility policy, see `docs/spec/core/COMPATIBILITY.md`.

## Install via GitHub Releases (manual)

1. Download a release archive for your OS/arch from GitHub Releases.
2. Extract and place `gws` on your `PATH`.
3. Verify:
   - `gws version`

## Install via Homebrew

Homebrew support is part of Epic 1. This project uses GitHub Releases as the source of truth.

Install:
- `brew tap tasuku43/gws`
- `brew install gws`

Notes:
- Homebrew is intended for installing the latest stable release.
- For version pinning, prefer `mise` (see below).

## Install via mise

`mise` can install tools from GitHub Releases.

Example (pin a version):
- `mise use -g github:tasuku43/gws@v0.1.0`

Example (track latest):
- `mise use -g github:tasuku43/gws@latest`

Verify:
- `gws version`
