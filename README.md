# gwst - Git Workspaces for Human + Agentic Development

A git workspace tool and worktree manager for human + agentic development.

gwst moves local development from "clone-directory centric" to "workspace centric"
so humans and multiple AI agents can work in parallel without stepping on each other.

## Why gwst

- In the era of AI agents, multiple actors edit in parallel and context collisions become common.
- gwst promotes directories into explicit workspaces and manages them safely with Git worktrees.
- It focuses on creating, listing, and safely cleaning up work environments.

## Who it's for

- People or teams managing work via GitHub Issues → batch-create with `gwst create --issue`.
- People or teams with frequent reviews → spin up review workspaces in bulk via `gwst create --review`.
- People or teams changing multiple repos per task → templates create a task-level workspace (pseudo-monorepo).
- People or teams overwhelmed by many worktrees and risky cleanup → guardrails in `gwst rm`.

## What makes gwst different

### 1) `create` is the center

One command, four creation modes:

```bash
gwst create --repo git@github.com:org/repo.git
gwst create --template app PROJ-123
gwst create --review https://github.com/owner/repo/pull/123   # GitHub only
gwst create --issue https://github.com/owner/repo/issues/123  # GitHub only
```

If you omit options, gwst switches to an interactive flow:

```
$ gwst create
Inputs
  • mode: s (type to filter)
    └─ repo - 1 repo only
    └─ issue - From an issue (multi-select, GitHub only)
    └─ review - From a review request (multi-select, GitHub only)
    └─ template - From template
```

Review/issue modes are also interactive (repo + multi-select):

```
$ gwst create --review
Inputs
  • repo: org/gwst
  • pull request: s (type to filter)
Info
  • selected
    └─ #123 Fix status output
    └─ #120 Add repo prompt
```

```
$ gwst create --issue
Inputs
  • repo: org/gwst
  • issue: s (type to filter)
Info
  • selected
    └─ #45 Improve template flow
    └─ #39 Add doctor checks
```

### 2) Template = pseudo-monorepo workspace

Define multiple repos as one task unit, then create them together:

```yaml
templates:
  app:
    repos:
      - git@github.com:org/backend.git
      - git@github.com:org/frontend.git
      - git@github.com:org/manifests.git
      - git@github.com:org/docs.git
```

```bash
gwst create --template app PROJ-123
```

### 3) Guardrails on cleanup

`gwst rm` refuses or asks for confirmation when workspaces are dirty, unpushed, or unknown:

```bash
gwst rm PROJ-123
```

Omitting the workspace id prompts selection:

```
$ gwst rm
Inputs
  • workspace: s (type to filter)
    └─ PROJ-123 [clean] - sample project
      └─ gwst (branch: PROJ-123-backend)
    └─ PROJ-124 [dirty changes] - wip
      └─ gwst (branch: PROJ-124-backend)
```

## Requirements

- Git
- gh CLI (optional; required for `gwst create --review` and `gwst create --issue` — GitHub only)

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

This creates `GWST_ROOT` with the standard layout and a starter `templates.yaml`.

Root resolution order:
1) `--root <path>`
2) `GWST_ROOT` environment variable
3) `~/gwst` (default)

Default layout example:

```
~/gwst/
├── bare/           # bare repo store (shared Git objects)
├── workspaces/     # task worktrees (one folder per workspace id)
└── templates.yaml
```

### 2) Fetch repos (bare store)

```bash
gwst repo get git@github.com:org/backend.git
```

This stores the repository in the bare store (no working tree is created yet).
This step is required before creating a workspace.

### 3) Create a workspace

```bash
gwst create --repo git@github.com:org/backend.git
```

You'll be prompted for a workspace id (e.g. `PROJ-123`, typically a Jira or ticket id).

Or run `gwst create` with no args to pick a mode and fill inputs interactively.

### 4) Work and clean up

List workspaces:

```bash
gwst ls
```

Open a workspace (prompts if omitted):

```bash
gwst open PROJ-123
```

This launches an interactive subshell at the workspace root (parent cwd unchanged) and
prefixes the prompt with `[gwst:<WORKSPACE_ID>]`.

Remove a workspace with guardrails (prompts if omitted):

```bash
gwst rm PROJ-123
```

## Help and docs

- `docs/README.md` for documentation index
- `docs/spec/README.md` for specs index and status
- `docs/spec/commands/` for per-command specs (create/add/rm/etc.)
- `docs/spec/core/TEMPLATES.md` for template format
- `docs/spec/core/DIRECTORY_LAYOUT.md` for the file layout
- `docs/spec/ui/UI.md` for output conventions
- `docs/concepts/CONCEPT.md` for the background and motivation

## Maintainer

- @tasuku43
