# gwiac — inventory-driven Git workspaces with guardrails

Create and delete whole workspaces (single- or multi-repo) safely using Git worktrees.
Designed for human + agentic development: each task (or each agent) gets an isolated workspace directory.

## Demo

https://github.com/user-attachments/assets/889e7f64-6222-4ad2-bc42-620dd1dd4139

## The core: “make many, remove many” without fear

gwiac’s strongest feature is not just creating worktrees — it’s making workspace lifecycle **repeatable and safe**:

- **Declare** the desired workspaces in `gwiac.yaml` (inventory / desired state)
- **Declare** the desired workspaces in `gwiac.yaml` (inventory / desired state)
- **Diff** with `gwiac plan` (read-only)
- **Reconcile** with `gwiac apply` (shows a plan; asks for confirmation for destructive changes)

This is what lets you spin up dozens of workspaces for parallel work — and clean them up in bulk with guardrails.

### Example: `gwiac plan` shows risk before removals

When a plan includes removals, `gwiac plan` inspects each repo in the workspace and summarizes risk (dirty/unpushed/diverged/unknown) so you can review before running `apply`.

How to generate this on your machine (optional):

```bash
gwiac manifest add --repo <REPO> <WORKSPACE_ID>
# (optional) make the workspace risky: dirty changes / unpushed commits / etc.
gwiac manifest rm <WORKSPACE_ID> --no-apply
gwiac plan
```

<!-- BEGIN: gwiac plan removal-risk example (paste real output) -->
```text
PASTE REAL OUTPUT OF: gwiac plan
```
<!-- END: gwiac plan removal-risk example (paste real output) -->

## Quickstart (5 minutes)

### 1) Install

```bash
brew tap tasuku43/gwiac
brew install gwiac
```

Other options:

- Version pinning with mise (optional): `mise use -g github:tasuku43/gwiac@<version>`
- Manual install via GitHub Releases (download archive → put `gwiac` on your PATH)
- Build from source requires Go 1.24+

For details and other options, see `docs/guides/INSTALL.md`.

### 2) Initialize a root

```bash
gwiac init
```

Root resolution order:
1) `--root <path>`
2) `GWIAC_ROOT` environment variable
3) `~/gwiac` (default)

Default layout:

```
~/gwiac/
├── bare/           # shared bare repo store
├── workspaces/     # task workspaces (one directory per workspace id)
└── gwiac.yaml      # inventory (desired state)
```

### 3) Create a workspace (interactive front-end to the inventory)

The happy path is `gwiac manifest add`: it writes `gwiac.yaml` and (by default) runs `gwiac apply`.
Run it with no args to choose a mode interactively (`repo` / `preset` / `review` / `issue`).

```bash
gwiac manifest add --repo git@github.com:org/backend.git PROJ-123
```

Open a workspace:

```bash
gwiac open PROJ-123
```

### 4) Remove safely (bulk cleanup with a plan)

```bash
gwiac manifest rm PROJ-123
```

`gwiac manifest rm` updates `gwiac.yaml` and (by default) runs `gwiac apply`, which prints a plan and enforces confirmation for destructive removals. Omit the workspace id to select interactively (multi-select).

### Interactive shortcuts (omit args to prompt)

gwiac falls back to interactive prompts when you omit required args (unless `--no-prompt` is set):

```bash
gwiac manifest add   # mode picker: repo / preset / review / issue
gwiac manifest rm    # workspace multi-select
gwiac open           # workspace picker
```

## The “create in bulk” entry points

### 1) From PRs / issues (GitHub only)

These modes are built to create workspaces from your existing workflow.

Direct URL (single workspace):

```bash
gwiac manifest add --review https://github.com/owner/repo/pull/123
gwiac manifest add --issue  https://github.com/owner/repo/issues/123
```

Notes:
- Requires `gh` (authenticated) to fetch metadata.
- For bulk creation, run `gwiac manifest add` with no args and use the interactive picker (supports multi-select), then confirm a single `apply`.

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
gwiac manifest add --preset app PROJ-123
```

## Power move: edit `gwiac.yaml` by hand (AI-friendly)

`gwiac.yaml` is just YAML. You can edit it directly (humans or AI), then review/apply changes:

```bash
gwiac plan
gwiac apply
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
- `gwiac.yaml` is gwiac-managed: commands rewrite the whole file, so comments/ordering may not be preserved.
- If filesystem is the truth (someone changed worktrees manually), use `gwiac import` to rebuild `gwiac.yaml` from the current state and see a diff.

### Example: `gwiac import` prints a unified diff

How to generate this on your machine (optional):

```bash
gwiac manifest add --repo <REPO> <WORKSPACE_ID>
gwiac manifest rm <WORKSPACE_ID> --no-apply
gwiac import
```

<!-- BEGIN: gwiac import unified-diff example (paste real output) -->
```text
PASTE REAL OUTPUT OF: gwiac import
```
<!-- END: gwiac import unified-diff example (paste real output) -->

## Requirements

- Git
- `gh` CLI (optional; required for `gwiac manifest add --review` and `gwiac manifest add --issue` — GitHub only)

## Help and docs

- `docs/README.md` for documentation index
- `docs/spec/README.md` for specs index and status
- `docs/spec/commands/` for per-command specs
- `docs/spec/core/INVENTORY.md` for `gwiac.yaml` format
- `docs/spec/core/PRESETS.md` for preset format
- `docs/spec/core/DIRECTORY_LAYOUT.md` for the file layout
- `docs/spec/ui/UI.md` for output conventions
- `docs/concepts/CONCEPT.md` for background and motivation

## Maintainer

- @tasuku43
