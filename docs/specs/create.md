---
title: "gws create"
status: planned
---

## Synopsis
`gws create [--template <name> | --review [<PR URL>] | --issue [<ISSUE_URL>]] [<WORKSPACE_ID>] [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]`

## Intent
Unify all workspace creation flows under a single command and keep "create" semantics consistent across modes.

## Modes and selection
- Exactly one of `--template`, `--review`, or `--issue` can be specified. If multiple are provided, error.
- If none are provided and prompts are allowed, enter an interactive mode picker.
  - The picker presents `template`, `review`, `issue` and supports arrow selection with filterable search.
- If none are provided and `--no-prompt` is set, error.

## Mode: template
Same behavior as the former `gws new`.

### Behavior
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
  - Refreshes the bare store only when the default branch is stale or missing (checked via `git ls-remote`, unless a recent fetch exists within `GWS_FETCH_GRACE_SECONDS`, default 30s); skips fetch when already up-to-date.
  - Base ref is auto-detected (prefers `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin).
- When prompting is allowed, collects per-repo branch names interactively:
  - For each repo in the template, prompt: `branch for <alias> [default: <WORKSPACE_ID>]:`
  - The input box is pre-filled with `<WORKSPACE_ID>` so users can press Enter to accept or append (e.g., `-hotfix`) without retyping.
  - Empty input still accepts the default (`WORKSPACE_ID`). Input is validated via `git check-ref-format --branch`.
  - Duplicate branch names across repos are allowed; a duplicate entry is warned and the user can confirm or re-enter.
- Renders a summary of created worktrees and suggests `cd` into the workspace.

### Success Criteria
- Workspace directory exists with one worktree per template repo, each on branch `WORKSPACE_ID`.

### Failure Modes
- Missing or invalid workspace ID.
- Template not found or contains empty repo entries.
- User declines required `repo get` operations.
- Git errors while adding worktrees or fetching repos.
- Workspace already exists.

## Mode: review
Same behavior as the former `gws review`.

### Behavior
- If `PR URL` is provided:
  - Accepts GitHub PR URLs only (e.g., `https://github.com/owner/repo/pull/123`); rejects other hosts or malformed paths.
  - Uses `gh api` to fetch PR metadata (requires authenticated GitHub CLI): PR number, head ref, and repositories.
  - Rejects forked PRs (head repo must match base repo).
  - Selects the repo URL based on `defaultRepoProtocol` (SSH preferred, HTTPS fallback).
  - Workspace ID is `REVIEW-PR-<number>`; errors if it already exists.
  - Ensures the repo store exists, prompting to run `gws repo get` if missing (unless `--no-prompt`, which fails instead).
  - Fetches the PR head ref into the bare store: `git fetch origin <head_ref>`.
  - Adds a worktree under `<root>/workspaces/REVIEW-PR-<number>/<alias>` where:
    - Branch is `<head_ref>`.
    - Base ref is `refs/remotes/origin/<head_ref>`.
- If `PR URL` is omitted and prompts are allowed (interactive picker):
  - `--no-prompt` with no URL => error.
  - Step 1: pick a repo from fetched bare stores whose origin remote is GitHub. Display `alias (owner/repo)`; filterable by substring.
  - Step 2: fetch open PRs for the repo via `gh api` (latest 50 open, sorted by updated desc).
  - Step 3: multi-select PRs using the same add/remove loop as `gws template new` (filterable list; `<Enter>` adds; `<Ctrl+D>` or `done` to finish; minimum 1 selection).
  - For each selected PR:
    - Workspace ID = `REVIEW-PR-<number>`.
    - Branch = PR head ref; base ref = `refs/remotes/origin/<head_ref>`.
    - Fork PRs remain rejected.
  - Flags other than `--no-prompt` are not allowed in picker mode (error if provided).
  - Creation is sequential; an error on one PR stops further creation and reports successes/failures so far.
- Output uses Inputs/Steps/Result only (no header line). When multiple workspaces are created, Result lists each workspace/worktree added.

### Success Criteria
- For URL mode: new workspace `REVIEW-PR-<number>` exists with a worktree checked out to the PR head branch.
- For picker mode: each selected PR produces a workspace with the same guarantees; partial success is reported if a later item fails.

### Failure Modes
- Invalid or unsupported PR URL; non-GitHub host.
- Fork PR detected.
- Missing or unauthenticated `gh` CLI.
- Repo store missing and user declines/forbids `repo get`.
- Git errors fetching the PR head or creating the worktree.
- Picker mode: no TTY available, repo selection empty, PR fetch fails (auth/network/API), or zero PRs selected.

## Mode: issue
Same behavior as the former `gws issue`.

### Behavior
- If `ISSUE_URL` is provided:
  - Parse the URL to obtain `owner`, `repo`, and `issue number`.
  - Workspace ID: defaults to `ISSUE-<number>`; can be overridden with `--workspace-id`. Must pass `git check-ref-format --branch`. If the workspace already exists, error.
  - Branch: defaults to `issue/<number>`. Before proceeding, prompt the user with the default and allow editing unless `--no-prompt` or `--branch` is supplied.
    - If the branch exists in the bare store, use it.
    - If not, create it from a base ref.
  - Base ref: defaults to the standard detection used by `gws add` (prefer `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin). `--base` overrides detection; must resolve in the bare store or as `origin/<ref>`.
  - Repo resolution:
    - Only the repo that owns the issue is used (single-repo flow). No template support in MVP.
    - If the repo store is missing, prompt to run `gws repo get <repo>`; with `--no-prompt`, fail.
  - Worktree location: `<root>/workspaces/<WORKSPACE_ID>/<repo_name>`; branch = `issue/<number>`.
  - Existing worktree collision: if the target workspace already has a worktree on the target branch, error (no reuse in MVP).
- If `ISSUE_URL` is omitted and prompts are allowed (interactive picker):
  - `--no-prompt` with no URL => error.
  - Step 1: pick a repo from fetched bare stores whose origin remote resolves to a supported host. Display `alias (host/owner/repo)`; filterable by substring.
  - Step 2: fetch open issues for the chosen repo from the host API (GitHub via `gh api`; other hosts may be added later). Default fetch: latest 50 open issues sorted by updated desc.
  - Step 3: multi-select issues using the same add/remove loop as `gws template new` (filterable list; `<Enter>` adds; `<Ctrl+D>` or `done` to finish; minimum 1 selection).
  - For each selected issue:
    - Workspace ID = `ISSUE-<number>` (no per-item override in this flow).
    - Branch = `issue/<number>`.
    - Base ref detection and repo missing handling are the same as the URL path.
  - Flags `--workspace-id`, `--branch`, and `--base` are only valid when a single issue is targeted (URL path). In picker mode with multiple selections, using these flags is an error.
  - Creation is sequential; an error on one issue stops further creation and reports successes/failures so far.
- Output uses Inputs/Steps/Result (no header line). When multiple workspaces are created, Result lists each workspace/worktree added.

### Success Criteria
- For URL mode: workspace `<root>/workspaces/ISSUE-<number>` exists with a worktree for the issue repo checked out to branch `issue/<number>`.
- For picker mode: each selected issue produces a workspace with the same guarantees; partial success is reported if a later item fails.

### Failure Modes
- Unsupported or invalid issue URL.
- Workspace already exists.
- Repo store missing and user declines/forbids `repo get`.
- Git errors when creating/fetching branch or adding the worktree.
- Base ref cannot be detected.
- Provided `--base` or `--branch` is invalid or conflicts with existing worktree in the target workspace.
- Picker mode: no TTY available, repo selection empty, issue fetch fails (auth/network/API), or zero issues selected.

## Removed commands
- `gws new`, `gws review`, and `gws issue` are removed. Users should use `gws create` with the corresponding mode flag.
