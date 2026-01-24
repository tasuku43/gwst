---
title: "gion manifest"
status: implemented
aliases:
  - man
  - m
migrated_from: "docs/spec/commands/manifest.md"
---

## Synopsis
`gion manifest <subcommand> [args] [--no-apply] [--no-prompt]`

Aliases:
- `gion man` (alias of `gion manifest`)
- `gion m` (alias of `gion manifest`)

## Intent
Make `gion.yaml` the primary interface for day-to-day operations by providing interactive commands that edit the manifest (desired state) and, by default, reconcile the filesystem via `gion apply`.

This command family is the new home for "YAML editing" flows, including workspaces and presets.

## Non-negotiables
- Interactive UX is preserved for creation/removal flows.
- Mutations change `gion.yaml` first; filesystem changes happen only through `gion apply`.
- Destructive changes are applied only by `gion apply`, and `--no-prompt` must not allow destructive actions.
- Idempotent: repeated apply converges.

## Subcommands
Workspace inventory (default target):
- `gion manifest ls`
- `gion manifest add`
- `gion manifest rm`
- `gion manifest gc`
- `gion manifest validate`

Preset inventory:
- `gion manifest preset ls`
- `gion manifest preset add`
- `gion manifest preset rm`
- `gion manifest preset validate`

Preset aliases:
- `gion manifest preset` can be shortened as `gion manifest pre` and `gion manifest p`.

## Flag behavior
- `--no-apply` (mutating subcommands only): update `gion.yaml` and exit without running `gion apply`.
- `--no-prompt`: forwarded to `gion apply` when apply is run; rules follow `gion apply` spec (errors if destructive changes exist).

Notes:
- Preset subcommands modify inventory only; they do not run `gion apply` by default.

## Relationship with other commands
- `gion apply`: executes plan + confirmation + reconcile.
- `gion plan`: shows full diff; used for deeper review than the per-item drift indicators.
- `gion import`: rebuilds `gion.yaml` from the filesystem when drift must be captured back into inventory.
