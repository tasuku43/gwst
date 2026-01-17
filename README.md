# gws - Git Workspaces for Human + Agentic Development

gws moves local development from "clone-directory centric" to "workspace centric"
so humans and multiple AI agents can work in parallel without stepping on each other.

## Why gws

- Keep one canonical Git object store (bare) and spin up task workspaces as worktrees
- Make "start a task" and "review a PR" repeatable with a single CLI

## Requirements

- Git
- Go 1.24+ (build/run from source)
- gh CLI (required for `gws create --review` and `gws create --issue` — GitHub only)

## Quickstart (5 minutes)

### 1) Initialize the root

```bash
gws init
```

This creates `GWS_ROOT` with the standard layout and a starter `templates.yaml`.

### 2) Define templates

Edit `templates.yaml` and list the repos you want in a workspace:

```yaml
templates:
  example:
    repos:
      - git@github.com:octocat/Hello-World.git
      - git@github.com:octocat/Spoon-Knife.git
```

Validate the file:

```bash
gws template validate
```

### 3) Fetch repos (bare store)

```bash
gws repo get git@github.com:octocat/Hello-World.git
gws repo get git@github.com:octocat/Spoon-Knife.git
```

### 4) Create a workspace

```bash
gws create --template example MY-123
```

Or create from a single repo:

```bash
gws create --repo git@github.com:octocat/Hello-World.git
```

Or run `gws create` with no args to pick a mode and fill inputs interactively.

### 5) Work and clean up

```bash
gws ls
gws open MY-123
gws status MY-123
gws rm MY-123
```

gws opens an interactive subshell at the workspace root.

## Review a PR (GitHub only)

```bash
gws create --review https://github.com/owner/repo/pull/123
```

- Creates `OWNER-REPO-REVIEW-PR-123`
- Fetches the PR head branch (forks not supported)
- Requires `gh` authentication

## Create from an Issue (GitHub only)

```bash
gws create --issue https://github.com/owner/repo/issues/123
```

- Creates `OWNER-REPO-ISSUE-123`
- Defaults branch to `issue/123`
- Requires `gh` authentication

## Provider support (summary)
- `gws create --repo` and `gws create --template` are provider-agnostic (any Git host URL).
- `gws create --review` and `gws create --issue` are GitHub-only today.

## How gws lays out files

gws keeps two top-level directories under `GWS_ROOT`:

```
GWS_ROOT/
  bare/        # bare repo store (shared Git objects)
  workspaces/  # task worktrees (one folder per workspace id)
  templates.yaml
```

Notes:

- Workspace id must be a valid Git branch name, and it becomes the worktree branch name.
- gws never changes your shell directory automatically.

## Root resolution

gws resolves `GWS_ROOT` in this order:

1. `--root <path>`
2. `GWS_ROOT` environment variable
3. `~/gws`

## Command overview

Core workflow:

- `gws init` - create root structure and `templates.yaml`
- `gws repo get <repo>` - create/update bare repo store
- `gws repo ls` - list repos already fetched
- `gws template ls` - list templates from `templates.yaml`
- `gws template validate` - validate `templates.yaml` entries
- `gws create --template <name> [<id>]` - create a workspace from a template
- `gws create --repo [<repo>]` - create a workspace from a repo (prompts for id)
- `gws add [<id>] [<repo>]` - add another repo worktree to a workspace
- `gws ls [--details]` - list workspaces and repos (optionally with git status details)
- `gws open [<id>]` - open a workspace in an interactive subshell
- `gws status [<id>]` - show branch, dirty/untracked, and ahead/behind
- `gws rm [<id>]` - remove a workspace (refuses if dirty)
- `gws path --workspace` - print a selected workspace path

Review workflow:

- `gws create --review <PR URL>` - create a workspace for a GitHub PR

Global flags:

- `--root <path>` - override `GWS_ROOT`
- `--no-prompt` - disable interactive prompts

## Repo spec format

Only SSH or HTTPS URLs are supported:

```
# SSH
git@github.com:owner/repo.git

# HTTPS
https://github.com/owner/repo.git
```

## Common tasks

### Add a repo to an existing workspace

```bash
gws add MY-123 git@github.com:org/another-repo.git
```

### Remove a workspace safely

```bash
gws status MY-123
gws rm MY-123
```

`gws rm` refuses if the workspace is dirty.

## Help and docs

- `docs/README.md` for documentation index
- `docs/spec/README.md` for specs index and status
- `docs/spec/core/TEMPLATES.md` for template format
- `docs/spec/core/DIRECTORY_LAYOUT.md` for the file layout
- `docs/spec/ui/UI.md` for output conventions

## Maintainer

- @tasuku43
