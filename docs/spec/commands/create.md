---
title: "gwst create"
status: implemented
---

## Synopsis
`gwst create [--preset <name> | --review [<PR URL>] | --issue [<ISSUE_URL>] | --repo [<repo>]] [<WORKSPACE_ID>] [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]`

## Intent
Unify all workspace creation flows under a single command and keep "create" semantics consistent across modes.

## Modes and selection
- Exactly one of `--preset`, `--review`, `--issue`, or `--repo` can be specified. If multiple are provided, error.
- If none are provided and prompts are allowed, enter an interactive mode picker.
  - The picker presents `preset`, `repo`, `review`, `issue` and supports arrow selection with filterable search.
  - `review` and `issue` are GitHub-only modes; `repo` and `preset` are provider-agnostic.
- If none are provided and `--no-prompt` is set, error.
- When prompts are used, mode flags (`--preset`, `--review`, `--issue`, `--repo`) still run the unified create prompt flow so the Inputs section is rendered as a single in-place interaction.
- When a repo is determined (via selection, URL parse, or preset resolution), start a background fetch immediately to reduce lead time.
  - Prefetch only needs to ensure the default branch is up-to-date for upcoming worktree creation.
  - Wait for any in-flight prefetch before creating the worktree(s).
  - If fetch is unnecessary due to the existing freshness/grace checks, skip the prefetch.

### Inputs output (prefetch hint)
When prefetch starts, show it under the repo entry so users can tell background fetch began.

Example:
```
Inputs
  • repo: git@github.com:tasuku43/gwst.git
    └─ prefetch: git fetch origin (background)
```

## Mode: preset
Same behavior as the former `gwst new`.

### Behavior
- Resolves `--preset` and `WORKSPACE_ID`; if either is missing and `--no-prompt` is not set, interactively prompts for both. With `--no-prompt`, missing values cause an error.
- Validates `WORKSPACE_ID` using `git check-ref-format --branch` and fails on invalid IDs.
- Loads the specified preset from `gwst.yaml`; errors if the preset is missing.
- When prompting is allowed, asks for an optional description (`description`). Empty input is allowed; non-empty values are saved to `gwst.yaml` and `.gwst/metadata.json`.
- Preflights preset repositories to see which stores are absent.
  - If repos are missing and prompting is allowed, offers to run `gwst repo get` for them before proceeding.
  - With `--no-prompt`, missing repos cause an error.
- Creates the workspace directory `<root>/workspaces/<WORKSPACE_ID>`; fails if it already exists.
- Applies the preset by adding a worktree for each repo:
  - Alias defaults to the repo name.
  - Branch defaults to `WORKSPACE_ID`.
  - Refreshes the bare store only when the default branch is stale or missing (checked via `git ls-remote`, unless a recent fetch exists within `GWST_FETCH_GRACE_SECONDS`, default 30s); skips fetch when already up-to-date.
  - Base ref is auto-detected (prefers `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin).
- If a repo is determined before worktree creation (preset selection path), start prefetching for each repo immediately and wait for completion before creating worktrees.
- When prompting is allowed, collects per-repo branch names interactively:
  - For each repo in the preset, prompt: `branch for <alias> [default: <WORKSPACE_ID>]:`
  - The input box is pre-filled with `<WORKSPACE_ID>` so users can press Enter to accept or append (e.g., `-hotfix`) without retyping.
  - Empty input still accepts the default (`WORKSPACE_ID`). Input is validated via `git check-ref-format --branch`.
  - Duplicate branch names across repos are allowed; a duplicate entry is warned and the user can confirm or re-enter.
- Renders a summary of created worktrees and suggests `gwst open`.

### Success Criteria
- Workspace directory exists with one worktree per preset repo, each on branch `WORKSPACE_ID`.

### Failure Modes
- Missing or invalid workspace ID.
- Preset not found or contains empty repo entries.
- User declines required `repo get` operations.
- Git errors while adding worktrees or fetching repos.
- Workspace already exists.

## Mode: repo
Create a workspace from a selected repo without using a preset.

### Behavior
- If `<repo>` is provided, use it directly and skip repo selection.
- Without `<repo>`, requires prompts/TTY to select a repo.
- Step 1 (selection path only): select a repo (searchable single-select) from existing repo stores.
- Step 2: same flow as preset mode after selection:
  - Decide the workspace ID.
  - Input an optional description.
  - Decide per-repo branch names (same validation and duplicate handling as preset mode).
- Preflights selected repos and offers to run `gwst repo get` if stores are missing (same as preset mode).
- Once the repo is determined, start prefetch immediately and wait for completion before creating the worktree.

### Success Criteria
- Workspace directory exists with one worktree for the selected repo, on the chosen branch.

### Failure Modes
- Repo selection empty or canceled.
- Repo stores missing and user declines/forbids `repo get`.
- Workspace already exists or invalid workspace ID.
- Git errors while adding worktrees or fetching repos.

## Mode: review
Same behavior as the former `gwst review`.

### Behavior
- If `PR URL` is provided:
  - Accepts GitHub PR URLs only (e.g., `https://github.com/owner/repo/pull/123`); rejects other hosts or malformed paths.
  - Uses `gh api` to fetch PR metadata (requires authenticated GitHub CLI): PR number, head ref, and repositories.
  - Saves the PR title as the workspace description in both `gwst.yaml` and `.gwst/metadata.json`.
  - Rejects forked PRs (head repo must match base repo).
  - Selects the repo URL based on `defaultRepoProtocol` (SSH preferred, HTTPS fallback).
  - Workspace ID is `<OWNER>-<REPO>-REVIEW-PR-<number>` (owner/repo uppercased); errors if it already exists.
  - Ensures the repo store exists, prompting to run `gwst repo get` if missing (unless `--no-prompt`, which fails instead).
  - Fetches the PR head ref into the bare store: `git fetch origin <head_ref>`.
  - Adds a worktree under `<root>/workspaces/<OWNER>-<REPO>-REVIEW-PR-<number>/<alias>` where:
    - Creates a local branch `<head_ref>` tracking `origin/<head_ref>`.
    - `gwst.yaml` stores the branch name as `<head_ref>`.
    - `.gwst/metadata.json` stores `mode=review` and the PR URL as `source_url`.
- Once the repo is determined, start prefetch immediately and wait for completion before creating the worktree.
- If `PR URL` is omitted and prompts are allowed (interactive picker):
  - `--no-prompt` with no URL => error.
  - Step 1: pick a repo from fetched bare stores whose origin remote is GitHub. Display `alias (owner/repo)`; filterable by substring.
  - Step 2: fetch open PRs for the repo via `gh api` (latest 50 open, sorted by updated desc).
- Step 3: multi-select PRs using the same add/remove loop as `gwst preset add` (filterable list; `<Enter>` adds; `<Ctrl+D>` or `done` to finish; minimum 1 selection).
  - For each selected PR:
    - Workspace ID = `<OWNER>-<REPO>-REVIEW-PR-<number>` (owner/repo uppercased).
    - Creates a local branch matching the PR head ref, tracking `origin/<head_ref>`.
    - `gwst.yaml` stores the branch name as the PR head ref.
    - Workspace description = PR title.
    - `.gwst/metadata.json` stores `mode=review` and the PR URL as `source_url`.
    - Fork PRs remain rejected.
  - Flags other than `--no-prompt` are not allowed in picker mode (error if provided).
  - Creation is sequential; an error on one PR stops further creation and reports successes/failures so far.
- Output uses Inputs/Steps/Result only (no header line). When multiple workspaces are created, Result lists each workspace/worktree added.

### Success Criteria
- For URL mode: new workspace `<OWNER>-<REPO>-REVIEW-PR-<number>` exists with a worktree checked out to the PR head branch.
- For picker mode: each selected PR produces a workspace with the same guarantees; partial success is reported if a later item fails.

### Failure Modes
- Invalid or unsupported PR URL; non-GitHub host.
- Fork PR detected.
- Missing or unauthenticated `gh` CLI.
- Repo store missing and user declines/forbids `repo get`.
- Git errors fetching the PR head or creating the worktree.
- Picker mode: no TTY available, repo selection empty, PR fetch fails (auth/network/API), or zero PRs selected.

## Mode: issue
Same behavior as the former `gwst issue`.

### Behavior
- If `ISSUE_URL` is provided:
  - Accepts GitHub issue URLs only (e.g., `https://github.com/owner/repo/issues/123`).
  - Parse the URL to obtain `owner`, `repo`, and `issue number`.
  - Workspace ID: defaults to `<OWNER>-<REPO>-ISSUE-<number>` (owner/repo uppercased); can be overridden with `--workspace-id`. Must pass `git check-ref-format --branch`. If the workspace already exists, error.
  - Branch: defaults to `issue/<number>`. Before proceeding, prompt the user with the default and allow editing unless `--no-prompt` or `--branch` is supplied.
  - Uses `gh api` to fetch the issue title and saves it as the workspace description in both `gwst.yaml` and `.gwst/metadata.json` (requires authenticated GitHub CLI).
  - `.gwst/metadata.json` stores `mode=issue` and the issue URL as `source_url`.
    - If the branch exists in the bare store, use it.
    - If the branch exists on `origin` but not locally, fetch it and create a tracking branch.
    - If not, create it from a base ref.
  - Base ref: defaults to the standard detection used by `gwst add` (prefer `HEAD`, then `origin/HEAD`, then `main`/`master`/`develop` locally or on origin). `--base` overrides detection; must resolve in the bare store or as `origin/<ref>`.
  - Repo resolution:
    - Only the repo that owns the issue is used (single-repo flow). No preset support in MVP.
    - If the repo store is missing, prompt to run `gwst repo get <repo>`; with `--no-prompt`, fail.
  - Worktree location: `<root>/workspaces/<WORKSPACE_ID>/<repo_name>`; branch = `issue/<number>`.
  - Existing worktree collision: if the target workspace already has a worktree on the target branch, error (no reuse in MVP).
- Once the repo is determined, start prefetch immediately and wait for completion before creating the worktree.
- If `ISSUE_URL` is omitted and prompts are allowed (interactive picker):
  - `--no-prompt` with no URL => error.
  - Step 1: pick a repo from fetched bare stores whose origin remote resolves to a supported host. Display `alias (host/owner/repo)`; filterable by substring.
  - Step 2: fetch open issues for the chosen repo via `gh api` (GitHub only). Default fetch: latest 50 open issues sorted by updated desc.
- Step 3: multi-select issues using the same add/remove loop as `gwst preset add` (filterable list; `<Enter>` adds; `<Ctrl+D>` or `done` to finish; minimum 1 selection).
  - For each selected issue:
    - Workspace ID = `<OWNER>-<REPO>-ISSUE-<number>` (owner/repo uppercased, no per-item override in this flow).
    - Branch defaults to `issue/<number>` and can be edited per issue in a list editor; duplicate branches must be re-entered.
    - Workspace description = issue title.
    - Base ref detection and repo missing handling are the same as the URL path.
  - Flags `--workspace-id`, `--branch`, and `--base` are only valid when a single issue is targeted (URL path). In picker mode with multiple selections, using these flags is an error.
  - Creation is sequential; an error on one issue stops further creation and reports successes/failures so far.
- Output uses Inputs/Steps/Result (no header line). When multiple workspaces are created, Result lists each workspace/worktree added.

### Success Criteria
- For URL mode: workspace `<root>/workspaces/<OWNER>-<REPO>-ISSUE-<number>` exists with a worktree for the issue repo checked out to branch `issue/<number>`.
- For picker mode: each selected issue produces a workspace with the same guarantees; partial success is reported if a later item fails.

### Failure Modes
- Unsupported or invalid issue URL; non-GitHub host.
- Workspace already exists.
- Repo store missing and user declines/forbids `repo get`.
- Git errors when creating/fetching branch or adding the worktree.
- Base ref cannot be detected.
- Provided `--base` or `--branch` is invalid or conflicts with existing worktree in the target workspace.
- Picker mode: no TTY available, repo selection empty, issue fetch fails (auth/network/API), or zero issues selected.

## Removed commands
- `gwst new`, `gwst review`, and `gwst issue` are removed. Users should use `gwst create` with the corresponding mode flag.
