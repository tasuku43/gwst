# gwst - Git Workspaces for Human + Agentic Development

A git workspace tool and worktree manager for human + agentic development.

gwst moves local development from "clone-directory centric" to "workspace centric"
so humans and multiple AI agents can work in parallel without stepping on each other.

## Demo

https://github.com/user-attachments/assets/889e7f64-6222-4ad2-bc42-620dd1dd4139

## Why gwst

- In the era of AI agents, multiple actors edit in parallel and context collisions become common.
- gwst promotes directories into explicit workspaces and manages them safely with Git worktrees.
- It focuses on creating, listing, and safely cleaning up work environments.

## Who it's for

- People or teams managing work via GitHub Issues → batch-add inventory with `gwst manifest add --issue`.
- People or teams with frequent reviews → spin up review workspaces in bulk via `gwst manifest add --review`.
- People or teams changing multiple repos per task → presets create a task-level workspace (pseudo-monorepo).
- People or teams overwhelmed by many worktrees and risky cleanup → safe reconciliation via `gwst plan` + `gwst apply`.

## What makes gwst different

### 1) `gwst manifest add` is the center

One command, four creation modes:

```bash
gwst manifest add --repo git@github.com:org/repo.git
gwst manifest add --preset app PROJ-123
gwst manifest add --review https://github.com/owner/repo/pull/123   # GitHub only
gwst manifest add --issue https://github.com/owner/repo/issues/123  # GitHub only
```

If you omit options, gwst switches to an interactive flow:

```
$ gwst manifest add
Inputs
  • mode: s (type to filter)
    └─ repo - 1 repo only
    └─ issue - From an issue (multi-select, GitHub only)
    └─ review - From a review request (multi-select, GitHub only)
    └─ preset - From preset
```

Review/issue modes are also interactive (repo + multi-select):

```
$ gwst manifest add --review
Inputs
  • repo: org/gwst
  • pull request: s (type to filter)
Info
  • selected
    └─ #123 Fix status output
    └─ #120 Add repo prompt
```

```
$ gwst manifest add --issue
Inputs
  • repo: org/gwst
  • issue: s (type to filter)
Info
  • selected
    └─ #45 Improve preset flow
    └─ #39 Add doctor checks
```

### 2) Preset = pseudo-monorepo workspace

Define multiple repos as one task unit, then create them together:

```yaml
presets:
  app:
    repos:
      - git@github.com:org/backend.git
      - git@github.com:org/frontend.git
      - git@github.com:org/manifests.git
      - git@github.com:org/docs.git
```

```bash
gwst manifest add --preset app PROJ-123
```

### 3) Guardrails on cleanup

`gwst plan` + `gwst apply` refuse or ask for confirmation when removals are risky (dirty, unpushed, unknown, etc.):

```bash
gwst manifest rm PROJ-123
```

Omitting the workspace id prompts selection:

```
$ gwst manifest rm
Inputs
  • workspace: s (type to filter)
    └─ PROJ-123 [clean] - sample project
      └─ gwst (branch: PROJ-123-backend)
    └─ PROJ-124 [dirty changes] - wip
      └─ gwst (branch: PROJ-124-backend)
```

## Requirements

- Git
- gh CLI (optional; required for `gwst manifest add --review` and `gwst manifest add --issue` — GitHub only)

## Install

Recommended:

```bash
brew tap tasuku43/gwst
brew install gwst
```

Version pinning (optional):

```bash
mise use -g github:tasuku43/gwst
```
If you want to pin a specific version, use `mise use -g github:tasuku43/gwst@<version>`.

Manual (GitHub Releases):
- Download the archive for your OS/arch
- Extract and place `gwst` on your PATH
- Building from source requires Go 1.24+

For details and other options, see `docs/guides/INSTALL.md`.

## Quickstart (5 minutes)

### 1) Initialize the root

```bash
gwst init
```

This creates `GWST_ROOT` with the standard layout and a starter `gwst.yaml`.

Root resolution order:
1) `--root <path>`
2) `GWST_ROOT` environment variable
3) `~/gwst` (default)

Default layout example:

```
~/gwst/
├── bare/           # bare repo store (shared Git objects)
├── workspaces/     # task worktrees (one folder per workspace id)
└── gwst.yaml
```

### 2) Fetch repos (bare store)

```bash
gwst repo get git@github.com:org/backend.git
```

This stores the repository in the bare store (no working tree is created yet).
This step is required before creating a workspace.

### 3) Create a workspace

```bash
gwst manifest add --repo git@github.com:org/backend.git PROJ-123
```

You'll be prompted for a workspace id (e.g. `PROJ-123`, typically a Jira or ticket id).

Or run `gwst manifest add` with no args to pick a mode and fill inputs interactively.

### 4) Work and clean up

List workspaces:

```bash
gwst manifest ls
```

Open a workspace (prompts if omitted):

```bash
gwst open PROJ-123
```

This launches an interactive subshell at the workspace root (parent cwd unchanged) and
prefixes the prompt with `[gwst:<WORKSPACE_ID>]`.

Remove a workspace with guardrails (prompts if omitted):

```bash
gwst manifest rm PROJ-123
```

## Help and docs

- `docs/README.md` for documentation index
- `docs/spec/README.md` for specs index and status
- `docs/spec/commands/` for per-command specs
- `docs/spec/core/GWST.md` for gwst.yaml format
- `docs/spec/core/PRESETS.md` for preset format
- `docs/spec/core/DIRECTORY_LAYOUT.md` for the file layout
- `docs/spec/ui/UI.md` for output conventions
- `docs/concepts/CONCEPT.md` for the background and motivation

## Maintainer

- @tasuku43
