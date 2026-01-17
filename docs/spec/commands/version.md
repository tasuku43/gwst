---
title: "gwst version"
status: implemented
---

## Synopsis
`gwst version`

Also available as a global flag:
`gwst --version`

## Intent
Print the gwst version in a single line for support/debugging.

## Behavior
- `gwst --version` prints the version and exits 0, without resolving `GWST_ROOT` or running any subcommand.
- `gwst version` prints the same output and exits 0.
- Output format is a single line:
  - `gwst <version> [<commit>] [<date>] (<go version> <os>/<arch>)`
- Default build values:
  - When not set via `-ldflags`, `<version>` defaults to `dev`.
  - `<commit>` and `<date>` are omitted when empty.

## Success Criteria
- Prints one line to stdout and exits 0.

