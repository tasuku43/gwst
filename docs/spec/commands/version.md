---
title: "gwiac version"
status: implemented
---

## Synopsis
`gwiac version`

Also available as a global flag:
`gwiac --version`

## Intent
Print the gwiac version in a single line for support/debugging.

## Behavior
- `gwiac --version` prints the version and exits 0, without resolving `GWIAC_ROOT` or running any subcommand.
- `gwiac version` prints the same output and exits 0.
- Output format is a single line:
  - `gwiac <version> [<commit>] [<date>] (<go version> <os>/<arch>)`
- Default build values:
  - When not set via `-ldflags`, `<version>` defaults to `dev`.
  - `<commit>` and `<date>` are omitted when empty.

## Success Criteria
- Prints one line to stdout and exits 0.
