---
title: "gws version"
status: implemented
---

## Synopsis
`gws version`

Also available as a global flag:
`gws --version`

## Intent
Print the gws version in a single line for support/debugging.

## Behavior
- `gws --version` prints the version and exits 0, without resolving `GWS_ROOT` or running any subcommand.
- `gws version` prints the same output and exits 0.
- Output format is a single line:
  - `gws <version> [<commit>] [<date>] (<go version> <os>/<arch>)`
- Default build values:
  - When not set via `-ldflags`, `<version>` defaults to `dev`.
  - `<commit>` and `<date>` are omitted when empty.

## Success Criteria
- Prints one line to stdout and exits 0.

