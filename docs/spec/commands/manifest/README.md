---
title: "gwst manifest"
status: implemented
aliases:
  - man
  - m
migrated_from: "docs/spec/commands/manifest.md"
---

## Synopsis
`gwst manifest <subcommand> [args] [--no-apply] [--no-prompt]`

Aliases:
- `gwst man` (alias of `gwst manifest`)
- `gwst m` (alias of `gwst manifest`)

## Intent
Make `gwst.yaml` the primary interface for day-to-day operations by providing interactive commands that edit the manifest (desired state) and, by default, reconcile the filesystem via `gwst apply`.

This command family is the new home for "YAML editing" flows, including workspaces and presets.

## Non-negotiables
- Interactive UX is preserved for creation/removal flows.
- Mutations change `gwst.yaml` first; filesystem changes happen only through `gwst apply`.
- Destructive changes are applied only by `gwst apply`, and `--no-prompt` must not allow destructive actions.
- Idempotent: repeated apply converges.

## Subcommands
Workspace inventory (default target):
- `gwst manifest ls`
- `gwst manifest add`
- `gwst manifest rm`
- `gwst manifest validate`

Preset inventory:
- `gwst manifest preset ls`
- `gwst manifest preset add`
- `gwst manifest preset rm`
- `gwst manifest preset validate`

Preset aliases:
- `gwst manifest preset` can be shortened as `gwst manifest pre` and `gwst manifest p`.

## Flag behavior
- `--no-apply` (mutating subcommands only): update `gwst.yaml` and exit without running `gwst apply`.
- `--no-prompt`: forwarded to `gwst apply` when apply is run; rules follow `gwst apply` spec (errors if destructive changes exist).

Notes:
- Preset subcommands modify inventory only; they do not run `gwst apply` by default.

## Relationship with other commands
- `gwst apply`: executes plan + confirmation + reconcile.
- `gwst plan`: shows full diff; used for deeper review than the per-item drift indicators.
- `gwst import`: rebuilds `gwst.yaml` from the filesystem when drift must be captured back into inventory.
