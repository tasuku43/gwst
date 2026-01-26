---
title: "Command reference"
status: implemented
---

# Command reference

This page is a user-oriented overview of gion commands.
For detailed behavior contracts (including edge cases and output rules), see `docs/spec/commands/`.

## Quick cheat sheet

Typical flow:

```bash
gion init
gion repo get git@github.com:org/repo.git
gion manifest add --repo git@github.com:org/repo.git PROJ-123
gion plan
gion apply
```

Common cleanup:

```bash
gion manifest gc
gion manifest rm PROJ-123
```

## gion (main CLI)

- `gion init` - initialize the root layout (`bare/`, `workspaces/`, `gion.yaml`).
- `gion repo get <repo>` - create/update a bare repo store for a remote repo.
- `gion repo ls` - list known bare repo stores under `GION_ROOT/bare/`.
- `gion manifest ...` - day-to-day inventory front-end (interactive by default).
- `gion plan` - show the diff between `gion.yaml` and the filesystem (no changes).
- `gion apply` - reconcile the filesystem to match `gion.yaml` (prompts before destructive changes).
- `gion import` - rebuild `gion.yaml` from the filesystem (when the filesystem is the source of truth).
- `gion doctor [--fix | --self]` - check workspace/repo health.
- `gion version` - print version.
- `gion help [command]` - show help (examples: `gion help manifest`, `gion help repo`).

### Global flags

- `--root <path>` - override `GION_ROOT`.
- `--no-prompt` - disable interactive prompts (destructive changes are still blocked).
- `--debug` - write debug logs to `<GION_ROOT>/logs/`.

## gion manifest (inventory front-end)

Workspace inventory:

- `gion manifest ls` - list workspaces and show drift tags.
- `gion manifest add ...` - add workspace entries, then runs `gion apply` by default.
- `gion manifest rm <id>...` - remove workspace entries, then runs `gion apply` by default.
- `gion manifest gc` - conservatively remove workspaces that are highly likely safe to delete, then runs `gion apply` by default.
- `gion manifest validate` - validate `gion.yaml` inventory.

Preset inventory:

- `gion manifest preset ls|add|rm|validate` - manage presets in `gion.yaml`.

Common flags:

- `--no-apply` - update `gion.yaml` only (do not run `gion apply`).
- `--no-prompt` - disable interactive prompts (forwarded to `gion apply` when apply is run).

## giongo (jump tool)

`giongo` is a companion binary for fast navigation. It does not change any state.

- `giongo --print` - select a destination and print its path.
- `giongo init` - print a shell function for `cd "$(giongo --print ...)"` integration.

## Further reading

- Use cases: `docs/guides/USECASES.md`
- Install: `docs/guides/INSTALL.md`
- Specs (contracts): `docs/spec/README.md`
