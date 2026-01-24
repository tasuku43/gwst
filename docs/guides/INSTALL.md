---
title: "Install gwiac"
status: implemented
---

# Install gwiac

Supported platforms (v0.1.0):
- macOS: `amd64`, `arm64`
- Linux: `amd64`, `arm64`

For compatibility policy, see `docs/spec/core/COMPATIBILITY.md`.

## Install via GitHub Releases (manual)

1. Download a release archive for your OS/arch from GitHub Releases.
2. Extract and place `gwiac` on your `PATH`.
3. Verify:
   - `gwiac version`

## Install via Homebrew

Homebrew uses GitHub Releases as the source of truth (stable tags only).

Install:
- `brew tap tasuku43/gwiac`
- `brew install gwiac`

Notes:
- Homebrew is intended for installing the latest stable release.
- Pre-release tags (e.g. `v0.1.0-rc.1`) are not published to Homebrew.
- For version pinning, prefer `mise` (see below).

## Install via mise

`mise` can install tools from GitHub Releases.

Example (pin a version):
- `mise use -g github:tasuku43/gwiac@v0.1.0`

Example (track latest):
- `mise use -g github:tasuku43/gwiac@latest`

Verify:
- `gwiac version`
