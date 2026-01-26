# gion — task workspaces (built on Git worktrees) as code, with guardrails.

Git workspaces as code, with guardrails.  
Define a YAML inventory, then plan/apply to reconcile safely.

Worktree sprawl brings pain:
- Risky cleanup: deleting dirty or unpushed work by mistake.
- Cognitive load: when worktrees multiply, it’s hard to remember where you did what.
- Bulk creation friction: worktrees are powerful but tedious to set up and place.

gion makes it safe and repeatable: declare task workspaces in YAML, review diffs (including deletion risk), then create and clean them up in bulk.  
You don’t have to edit YAML directly—`gion manifest ...` lets you add/remove workspaces interactively and updates the inventory behind the scenes.

## Overview

gion manages task-based Git workspaces on top of worktrees.  
You declare desired workspaces in `gion.yaml`, preview drift with `gion plan`, and reconcile with `gion apply`.

## Who it’s for

- Developers working on tasks that span multiple repositories.
- GitHub-centric, PR/issue-driven workflows (spin up local worktrees for reviews/issues and work in parallel).
- People using AI agents to create and remove many worktrees, while reviewing diffs and deletion risk before making changes.

## Features

- **Reproducible inventory:** `gion.yaml` is the source of truth
- **Bulk create, safely:** spin up many worktrees at once with guardrails.
- **Bulk cleanup, safely:** remove worktrees in bulk with guardrails (diff + confirmation, risk surfaced for dirty/unpushed/diverged/unknown).
- **Fast navigation:** `giongo` jumps to any workspace or repo
- **Multi-repo tasks:** group repos under a single workspace via presets
- **GitHub-aware entry points:** create from PRs or issues with `gh`

## Guardrails (safety by default)

- **Plan-first:** always shows a diff before applying changes.
- **Deletion risk visibility:** removal plans include a risk summary (e.g., dirty / unpushed / diverged / unknown).
  - **dirty:** working tree has changes.
  - **unpushed:** local branch is ahead of upstream.
  - **diverged:** local and upstream have both advanced.
  - **unknown:** status cannot be determined (e.g., git error or missing upstream).
- **Confirm destructive changes:** removals require an explicit confirmation; `--no-prompt` refuses destructive changes.
- **Conservative bulk cleanup:** merged-worktree cleanup excludes anything uncertain and acts only when it is highly likely safe.
- **Clear safety boundary:** gion reconciles only under `GION_ROOT/` (workspaces + repo stores).

## Quickstart (5 minutes)

### 1) Install

```bash
brew tap tasuku43/gion
brew install gion
```

### 2) Initialize the root (once per machine)

```bash
gion init
```

By default, gion uses `~/gion` as the root. This creates:

```text
~/gion/
  gion.yaml
  bare/        # bare repo store (shared Git objects)
  workspaces/  # task workspaces (worktrees)
```

### 3) Prepare a repo store

```bash
gion repo get git@github.com:org/backend.git
```

This creates a bare repo store at:

```text
~/gion/bare/github.com/org/backend.git
```

### 4) Create a workspace (review first)

Add a workspace to the inventory without applying yet:

```bash
gion manifest add --no-apply --repo git@github.com:org/backend.git PROJ-123
```

Review the plan (example output, trimmed):

```bash
gion plan
```

```text
Plan
  • + add workspace PROJ-123
    └─ backend  PROJ-123
       repo: github.com/org/backend.git
```

Apply the changes (this will prompt if anything is destructive):

```bash
gion apply
```

This creates a workspace worktree at:

```text
~/gion/workspaces/PROJ-123/backend
```

### 5) Jump into a workspace (interactive)

```bash
cd "$(giongo --print)"
```

### 6) Remove safely (guardrails on by default)

```bash
gion manifest rm PROJ-123
```

Tip: If you prefer the shortest path, omit `--no-apply` and `gion manifest add` will run `gion apply` by default.

## Terminology

```
GION_ROOT/  (safety boundary: gion only touches under this directory)
├─ gion.yaml                      # desired state (inventory)
│
├─ bare/                          # shared Git object store (bare clones)
│  └─ github.com/org/
│     ├─ backend.git              # bare repo store (shared)
│     ├─ frontend.git
│     └─ infra.git
│
└─ workspaces/                    # task-scoped directories (each contains worktrees)
   ├─ PROJ-123/                   # workspace_id (task)
   │  ├─ backend/                 # worktree checkout (repo: backend)
   │  │  ├─ .git                  # gitdir file -> points into .../backend.git/worktrees/...
   │  │  └─ ...                   # working directory (your changes live here)
   │  ├─ frontend/
   │  └─ infra/
   │
   └─ PROJ-456/
      └─ backend/
```

- **Workspace:** a task-scoped directory under `GION_ROOT/workspaces/<WORKSPACE_ID>/` that can contain multiple repos.
- **Worktree:** a Git worktree checkout for a repo, placed under a workspace (e.g. `.../workspaces/<id>/<alias>/`).
- **Repo store:** a shared bare clone cache under `GION_ROOT/bare/` (used to create and update worktrees efficiently).
- **Manifest:** the inventory file `gion.yaml` and the `gion manifest ...` subcommands that update it.

## Demo (45s)

Coming soon (new demo video in progress).

This demo shows:
- Create many workspaces at once
- Jump fast with `giongo`
- Clean up in bulk with guardrails (plan warns before deletions)

## Installation

### Homebrew

```bash
brew tap tasuku43/gion
brew install gion
```

### Other options

- Version pinning with mise (optional): `mise use -g github:tasuku43/gion@<version>`
- Manual install via GitHub Releases (download archive → put `gion` and `giongo` on your PATH)
- Build from source requires Go 1.24+

### Requirements

- Git
- `gh` CLI (optional; required for `gion manifest add --review` and `gion manifest add --issue` — GitHub only)

## Usage

For a command overview, see `docs/guides/COMMANDS.md` (or run `gion help`).

### Core workflow (Declare → Diff → Reconcile)

- **Declare** desired workspaces in `gion.yaml` (inventory / desired state)
- **Diff** with `gion plan` (read-only)
- **Reconcile** with `gion apply` (shows a plan; confirms destructive changes)

Example plan (add + remove, trimmed):

```text
Plan
  • + add workspace PROJ-123
    └─ backend  PROJ-123
       repo: github.com/org/backend.git
  • - remove workspace PROJ-099
    └─ backend  PROJ-099
       risk: dirty (unstaged=2)
       sync: upstream=origin/main ahead=1 behind=0

Apply destructive changes? (default: No)
```

### Create workspaces

Interactive front-end to the inventory:

```bash
gion manifest add
```

Run with flags to skip prompts:

```bash
gion manifest add --repo git@github.com:org/backend.git PROJ-123
```

### Move fast with giongo

`giongo` is a small companion binary that jumps into a workspace or repo using a picker.  
It does not change any state.

Example (zsh function):

```bash
giongo() {
  if [[ "$1" == "init" || "$1" == "--help" || "$1" == "-h" || "$1" == "--version" || "$1" == "--print" ]]; then
    command giongo "$@"
    return $?
  fi
  local dest
  dest="$(command giongo --print "$@")" || return $?
  [[ -n "$dest" ]] && cd "$dest"
}
```

Shortcut (auto-generate the function for your shell):

```bash
eval "$(giongo init)"
```

Notes:
- `giongo init` outputs a bash/zsh function definition.
- For a permanent setup, paste the output into `~/.zshrc` or `~/.bashrc`.

### Cleanup

Manual removal (explicit human judgment):

```bash
gion manifest rm PROJ-123
```

Automatic cleanup (conservative):

```bash
gion manifest gc
```

`gion manifest gc` removes workspace entries from `gion.yaml` only when they are highly likely safe to delete, then (by default) runs `gion apply` to reconcile.

GC safety rules (summary):
- Excludes any workspace with dirty / unpushed / diverged / unknown state.
- Considers a workspace safe only when all repos are strictly merged into their target base.
- Uses Git data from the local repo store (no PR metadata).

### Import

If the filesystem is the source of truth, rebuild the inventory:

```bash
gion import
```

## Inventory (`gion.yaml`)

### Root (`GION_ROOT`)

`GION_ROOT` is resolved in this order:

1. `--root <path>`
2. `GION_ROOT` environment variable
3. default `~/gion`

### Location and layout

- Inventory file: `<GION_ROOT>/gion.yaml`
- Bare repo stores (shared Git objects): `<GION_ROOT>/bare/`
- Workspaces (task directories containing worktrees): `<GION_ROOT>/workspaces/`

Invariants (short):
- `version: 1` is the current inventory schema; future changes will be versioned.
- gion only reads/writes under `GION_ROOT/` (safety boundary).
- Workspace IDs must be valid Git branch names (used as worktree branches).

`gion.yaml` is plain YAML. You can edit it directly (humans or AI), then review/apply changes:

```bash
gion plan
gion apply
```

For the full schema, see `docs/spec/core/INVENTORY.md`.

Minimal example:

```yaml
version: 1
workspaces:
  PROJ-123:
    description: "fix login flow"
    mode: repo
    repos:
      - alias: backend
        repo_key: github.com/org/backend.git
        branch: PROJ-123
```

Notes:
- `gion.yaml` is gion-managed: commands rewrite the whole file, so comments/ordering may not be preserved.
- Avoid relying on ordering/comments; keep human notes in `description` or a separate file.
- If you hand-edit `gion.yaml`, run `gion plan` to review the diff before `gion apply`.
- For day-to-day edits, prefer `gion manifest ...`; treat direct YAML edits as advanced or automation-only, and always review with `gion plan`.
- If your local manual changes are the source of truth, use `gion import` to follow them back into `gion.yaml`.

## Examples

### From PRs / issues (GitHub only)

This path is optimized for bulk creation from PRs/issues with one apply.

Interactive bulk selection (multi-select in the picker):

```bash
gion manifest add
```

Notes:
- Requires `gh` (authenticated) to fetch metadata.
- The picker supports bulk selection of PRs/issues, then a single apply.

Direct URL (single workspace):

```bash
gion manifest add --review https://github.com/owner/repo/pull/123
gion manifest add --issue  https://github.com/owner/repo/issues/123
```

### From presets (multi-repo “task workspace”)

```yaml
presets:
  app:
    repos:
      - git@github.com:org/backend.git
      - git@github.com:org/frontend.git
      - git@github.com:org/infra.git
```

```bash
gion manifest add --preset app PROJ-123
```

## Troubleshooting / FAQ

- `gion manifest add --review/--issue` requires `gh` authentication.
- `gion apply` prompts before destructive changes; use `gion plan` to preview.
- `gion.yaml` is rewritten by gion; use `gion import` if the filesystem is your source of truth.

## Contributing

See `CONTRIBUTING.md`.

## Security

See `SECURITY.md`.

## License

See `LICENSE`.

## Maintainer

- @tasuku43
