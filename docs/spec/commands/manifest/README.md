---
title: "gwiac manifest"
status: implemented
aliases:
  - man
  - m
migrated_from: "docs/spec/commands/manifest.md"
---

## Synopsis
`gwiac manifest <subcommand> [args] [--no-apply] [--no-prompt]`

Aliases:
- `gwiac man` (alias of `gwiac manifest`)
- `gwiac m` (alias of `gwiac manifest`)

## Intent
Make `gwiac.yaml` the primary interface for day-to-day operations by providing interactive commands that edit the manifest (desired state) and, by default, reconcile the filesystem via `gwiac apply`.

This command family is the new home for "YAML editing" flows, including workspaces and presets.

## Non-negotiables
- Interactive UX is preserved for creation/removal flows.
- Mutations change `gwiac.yaml` first; filesystem changes happen only through `gwiac apply`.
- Destructive changes are applied only by `gwiac apply`, and `--no-prompt` must not allow destructive actions.
- Idempotent: repeated apply converges.

## Subcommands
Workspace inventory (default target):
- `gwiac manifest ls`
- `gwiac manifest add`
- `gwiac manifest rm`
- `gwiac manifest gc`
- `gwiac manifest validate`

Preset inventory:
- `gwiac manifest preset ls`
- `gwiac manifest preset add`
- `gwiac manifest preset rm`
- `gwiac manifest preset validate`

Preset aliases:
- `gwiac manifest preset` can be shortened as `gwiac manifest pre` and `gwiac manifest p`.

## Flag behavior
- `--no-apply` (mutating subcommands only): update `gwiac.yaml` and exit without running `gwiac apply`.
- `--no-prompt`: forwarded to `gwiac apply` when apply is run; rules follow `gwiac apply` spec (errors if destructive changes exist).

Notes:
- Preset subcommands modify inventory only; they do not run `gwiac apply` by default.

## Relationship with other commands
- `gwiac apply`: executes plan + confirmation + reconcile.
- `gwiac plan`: shows full diff; used for deeper review than the per-item drift indicators.
- `gwiac import`: rebuilds `gwiac.yaml` from the filesystem when drift must be captured back into inventory.
