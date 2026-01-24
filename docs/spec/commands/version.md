---
title: "gion version"
status: implemented
---

## Synopsis
`gion version`

Also available as a global flag:
`gion --version`

## Intent
Print the gion version in a single line for support/debugging.

## Behavior
- `gion --version` prints the version and exits 0, without resolving `GION_ROOT` or running any subcommand.
- `gion version` prints the same output and exits 0.
- Output format is a single line:
  - `gion <version> [<commit>] [<date>] (<go version> <os>/<arch>)`
- Default build values:
  - When not set via `-ldflags`, `<version>` defaults to `dev`.
  - `<commit>` and `<date>` are omitted when empty.

## Success Criteria
- Prints one line to stdout and exits 0.
