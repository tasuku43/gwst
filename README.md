# gion — inventory-driven Git workspaces with guardrails

Create and delete whole workspaces (single- or multi-repo) safely using Git worktrees.
Designed for human + agentic development: each task (or each agent) gets an isolated workspace directory.

## Demo

https://github.com/user-attachments/assets/889e7f64-6222-4ad2-bc42-620dd1dd4139

## The core: “make many, remove many” without fear

gion’s strongest feature is not just creating worktrees — it’s making workspace lifecycle **repeatable and safe**:

- **Declare** the desired workspaces in `gion.yaml` (inventory / desired state)
- **Declare** the desired workspaces in `gion.yaml` (inventory / desired state)
- **Diff** with `gion plan` (read-only)
- **Reconcile** with `gion apply` (shows a plan; asks for confirmation for destructive changes)

This is what lets you spin up dozens of workspaces for parallel work — and clean them up in bulk with guardrails.

### Example: `gion plan` shows risk before removals

When a plan includes removals, `gion plan` inspects each repo in the workspace and summarizes risk (dirty/unpushed/diverged/unknown) so you can review before running `apply`.

How to generate this on your machine (optional):

```bash
gion manifest add --repo <REPO> <WORKSPACE_ID>
# (optional) make the workspace risky: dirty changes / unpushed commits / etc.
gion manifest rm <WORKSPACE_ID> --no-apply
gion plan
```

<!-- BEGIN: gion plan removal-risk example (paste real output) -->
```text
PASTE REAL OUTPUT OF: gion plan
```
<!-- END: gion plan removal-risk example (paste real output) -->

## Quickstart (5 minutes)

### 1) Install

```bash
brew tap tasuku43/gion
brew install gion
```

Other options:

- Version pinning with mise (optional): `mise use -g github:tasuku43/gion@<version>`
- Manual install via GitHub Releases (download archive → put `gion` on your PATH)
- Build from source requires Go 1.24+

For details and other options, see `docs/guides/INSTALL.md`.

### 2) Initialize a root

```bash
gion init
```

Root resolution order:
1) `--root <path>`
2) `GION_ROOT` environment variable
3) `~/gion` (default)

Default layout:

```
~/gion/
├── bare/           # shared bare repo store
├── workspaces/     # task workspaces (one directory per workspace id)
└── gion.yaml      # inventory (desired state)
```

### 3) Create a workspace (interactive front-end to the inventory)

The happy path is `gion manifest add`: it writes `gion.yaml` and (by default) runs `gion apply`.
Run it with no args to choose a mode interactively (`repo` / `preset` / `review` / `issue`).

```bash
gion manifest add --repo git@github.com:org/backend.git PROJ-123
```

Open a workspace:

```bash
gion open PROJ-123
```

### 4) Remove safely (bulk cleanup with a plan)

```bash
gion manifest rm PROJ-123
```

`gion manifest rm` updates `gion.yaml` and (by default) runs `gion apply`, which prints a plan and enforces confirmation for destructive removals. Omit the workspace id to select interactively (multi-select).

### Interactive shortcuts (omit args to prompt)

gion falls back to interactive prompts when you omit required args (unless `--no-prompt` is set):

```bash
gion manifest add   # mode picker: repo / preset / review / issue
gion manifest rm    # workspace multi-select
gion open           # workspace picker
```

## The “create in bulk” entry points

### 1) From PRs / issues (GitHub only)

These modes are built to create workspaces from your existing workflow.

Direct URL (single workspace):

```bash
gion manifest add --review https://github.com/owner/repo/pull/123
gion manifest add --issue  https://github.com/owner/repo/issues/123
```

Notes:
- Requires `gh` (authenticated) to fetch metadata.
- For bulk creation, run `gion manifest add` with no args and use the interactive picker (supports multi-select), then confirm a single `apply`.

### 2) From presets (multi-repo “task workspace”)

A preset is a lightweight “pseudo-monorepo” template: one task creates multiple repos together.

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

## Power move: edit `gion.yaml` by hand (AI-friendly)

`gion.yaml` is just YAML. You can edit it directly (humans or AI), then review/apply changes:

```bash
gion plan
gion apply
```

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
- If filesystem is the truth (someone changed worktrees manually), use `gion import` to rebuild `gion.yaml` from the current state and see a diff.

### Example: `gion import` prints a unified diff

How to generate this on your machine (optional):

```bash
gion manifest add --repo <REPO> <WORKSPACE_ID>
gion manifest rm <WORKSPACE_ID> --no-apply
gion import
```

<!-- BEGIN: gion import unified-diff example (paste real output) -->
```text
PASTE REAL OUTPUT OF: gion import
```
<!-- END: gion import unified-diff example (paste real output) -->

## Requirements

- Git
- `gh` CLI (optional; required for `gion manifest add --review` and `gion manifest add --issue` — GitHub only)

## Help and docs

- `docs/README.md` for documentation index
- `docs/spec/README.md` for specs index and status
- `docs/spec/commands/` for per-command specs
- `docs/spec/core/INVENTORY.md` for `gion.yaml` format
- `docs/spec/core/PRESETS.md` for preset format
- `docs/spec/core/DIRECTORY_LAYOUT.md` for the file layout
- `docs/spec/ui/UI.md` for output conventions
- `docs/concepts/CONCEPT.md` for background and motivation

## Maintainer

- @tasuku43
