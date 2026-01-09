---
title: "gws new"
status: implemented
---

## Synopsis
`gws new [--template <name>] [<WORKSPACE_ID>]`

## Intent
Create a new workspace directory under `<root>/workspaces` and populate it with worktrees defined by a template.

## Behavior
- Resolves `--template` and `WORKSPACE_ID`; if either is missing and `--no-prompt` is not set, interactively prompts for both. With `--no-prompt`, missing values cause an error.
- Validates `WORKSPACE_ID` using `git check-ref-format --branch` and fails on invalid IDs.
- Loads the specified template from `templates.yaml`; errors if the template is missing.
- Preflights template repositories to see which stores are absent.
  - If repos are missing and prompting is allowed, offers to run `gws repo get` for them before proceeding.
  - With `--no-prompt`, missing repos cause an error.
- Creates the workspace directory `<root>/workspaces/<WORKSPACE_ID>`; fails if it already exists.
- Applies the template by adding a worktree for each repo:
  - Alias defaults to the repo name.
  - Branch defaults to `WORKSPACE_ID`.
  - Fetches the bare store before adding the worktree.
  - Base ref is auto-detected (prefers `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin).
- When prompting is allowed, collects per-repo branch names interactively:
  - For each repo in the template, prompt: `branch for <alias> [default: <WORKSPACE_ID>]:`
  - The input box is pre-filled with `<WORKSPACE_ID>` so users can press Enter to accept or append (e.g., `-hotfix`) without retyping.
  - Empty input still accepts the default (`WORKSPACE_ID`). Input is validated via `git check-ref-format --branch`.
  - Duplicate branch names across repos are allowed; a duplicate entry is warned and the user can confirm or re-enter.
- Renders a summary of created worktrees and suggests `cd` into the workspace.

## Success Criteria
- Workspace directory exists with one worktree per template repo, each on branch `WORKSPACE_ID`.

## Failure Modes
- Missing or invalid workspace ID.
- Template not found or contains empty repo entries.
- User declines required `repo get` operations.
- Git errors while adding worktrees or fetching repos.
- Workspace already exists.
