# gws - Git Workspaces for Human + Agentic Development

gws moves local development from "clone-directory centric" to "workspace centric"
so humans and multiple AI agents can work in parallel without stepping on each other.

## Why gws

- Keep one canonical Git object store (bare) and spin up task workspaces as worktrees
- Make "start a task" and "review a PR" repeatable with a single CLI
- Separate human browsing clones from agent/task worktrees

## Requirements

- Git
- Go 1.24+ (build/run from source)
- gh CLI (optional; no longer required for `gws review`)

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

### 3) Fetch repos (bare + src)

```bash
gws repo get git@github.com:octocat/Hello-World.git
gws repo get git@github.com:octocat/Spoon-Knife.git
```

### 4) Create a workspace

```bash
gws new --template example MY-123
```

Or run `gws new` with no args to select a template and workspace id interactively.

### 5) Work and clean up

```bash
gws ls
gws status MY-123
gws rm MY-123
```

gws prints the workspace path so you can `cd` into it.

## Review a PR/MR (GitHub, GitLab, Bitbucket)

```bash
gws review https://github.com/owner/repo/pull/123
# or
gws review https://gitlab.com/owner/repo/-/merge_requests/123
# or
gws review https://bitbucket.org/owner/repo/pull-requests/123
```

- Creates `REVIEW-PR-123`
- Fetches the PR/MR ref directly (forks supported)
- No `gh` dependency

## How gws lays out files

gws keeps three top-level directories under `GWS_ROOT`:

```
GWS_ROOT/
  bare/        # bare repo store (shared Git objects)
  src/         # normal clones for human browsing
  workspaces/  # task worktrees (one folder per workspace id)
  templates.yaml
```

Notes:

- Workspace id must be a valid Git branch name, and it becomes the worktree branch name.
- `src/` is a regular clone and does not share local branches with `workspaces/`.
- gws never changes your shell directory automatically.

## Root resolution

gws resolves `GWS_ROOT` in this order:

1. `--root <path>`
2. `GWS_ROOT` environment variable
3. `~/gws`

## Command overview

Core workflow:

- `gws init` - create root structure and `templates.yaml`
- `gws repo get <repo>` - create/update bare repo and a `src/` clone
- `gws repo ls` - list repos already fetched
- `gws template ls` - list templates from `templates.yaml`
- `gws new [--template <name>] [<id>]` - create a workspace from a template
- `gws add [<id>] [<repo>]` - add another repo worktree to a workspace
- `gws ls` - list workspaces and repos
- `gws status [<id>]` - show branch, dirty/untracked, and ahead/behind
- `gws rm [<id>]` - remove a workspace (refuses if dirty)

Review workflow:

- `gws review <PR URL>` - create a workspace for a GitHub PR

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

- `docs/CLI.md` for command details
- `docs/TEMPLATES.md` for template format
- `docs/DIRECTORY_LAYOUT.md` for the file layout
- `docs/UI.md` for output conventions

## Maintainer

- @tasuku43
