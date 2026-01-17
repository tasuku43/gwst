---
title: "Release pipeline architecture"
status: implemented
---

# Release pipeline architecture

This document explains the architecture and sequence of the v0.1.0 release pipeline.

## Components

- Git tag: `vX.Y.Z` (e.g. `v0.1.0`)
- GitHub Actions workflow: `.github/workflows/release.yml`
- GoReleaser config: `.goreleaser.yaml`
- GitHub Release artifacts:
  - `gwst_vX.Y.Z_macos_arm64.tar.gz`
  - `gwst_vX.Y.Z_macos_x64.tar.gz`
  - `gwst_vX.Y.Z_linux_arm64.tar.gz`
  - `gwst_vX.Y.Z_linux_x64.tar.gz`
  - `checksums.txt` (SHA256)
- Homebrew Formula: `Formula/gwst.rb`
- Formula updater script: `.github/scripts/update-homebrew-formula.sh`

## Architecture (high-level)

```mermaid
flowchart LR
  Dev[Maintainer] -->|push tag vX.Y.Z| GH[GitHub]
  GH -->|trigger| GA[GitHub Actions: release.yml]
  GA --> GR[GoReleaser]
  GR --> RLS[GitHub Release + artifacts]
  RLS -->|download| Users[Users]

  RLS -->|checksums.txt + tag| Updater[update-homebrew-formula.sh]
  Updater --> PR[PR updating Formula/gwst.rb]
  GH -->|auto-merge PR after CI| Main[main branch]
  Main --> Brew[brew tap + brew install]
  RLS --> Mise[mise github backend]
```

## Sequence (tag to install)

```mermaid
sequenceDiagram
  participant Dev as Maintainer
  participant GH as GitHub
  participant GA as GitHub Actions
  participant GR as GoReleaser
  participant RLS as GitHub Release
  participant PR as PR (Formula update)
  participant Brew as Homebrew user
  participant Mise as mise user

  Dev->>GH: push tag vX.Y.Z
  GH->>GA: trigger release workflow
  GA->>GR: goreleaser release --clean
  GR->>RLS: create/update release + upload artifacts
  GA->>GA: run formula update script (from dist/checksums.txt)
  GA->>PR: open PR to update Formula/gwst.rb
  PR-->>GH: CI passes
  GH->>GH: auto-merge PR (stable tags only)
  Brew->>GH: brew tap tasuku43/gwst (pulls Formula/gwst.rb)
  Brew->>RLS: download tar.gz + verify sha256
  Brew->>Brew: install gwst
  Mise->>RLS: download tar.gz + install (github backend)
```

## Notes

- `gwst version` correctness is guaranteed for **GitHub Releases binaries** by `-ldflags` injected by GoReleaser.
- Homebrew formula is updated via a PR after each stable release tag; GitHub auto-merge is enabled by the release workflow.
- Homebrew formula updates are performed for **stable tags only** (tags without a `-...` prerelease suffix).
