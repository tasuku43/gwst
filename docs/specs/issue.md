---
title: "gws issue"
status: implemented
---

## Synopsis
`gws issue <ISSUE_URL> [--workspace-id <id>] [--branch <name>] [--base <ref>] [--no-prompt]`

## Intent
Create a workspace tailored to a single issue, wiring a branch named after the issue and checking it out as a worktree so work can start immediately without manual setup.

## Supported hosts (MVP)
- GitHub, GitLab, Bitbucket issue URLs (e.g., `https://github.com/owner/repo/issues/123`). Other hosts reject with an error.

## Behavior
- Parse the issue URL to obtain `owner`, `repo`, and `issue number`.
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
- Output: mirror existing style (`gws review`/`gws new`): header, Info section shows issue title/number/repo, Steps, Result with workspace + worktree summary, suggestion to `cd`.

## Success Criteria
- Workspace `<root>/workspaces/ISSUE-<number>` exists with a worktree for the issue repo checked out to branch `issue/<number>`.

## Failure Modes
- Unsupported or invalid issue URL.
- Workspace already exists.
- Repo store missing and user declines/forbids `repo get`.
- Git errors when creating/fetching branch or adding the worktree.
- Base ref cannot be detected.
- Provided `--base` or `--branch` is invalid or conflicts with existing worktree in the target workspace.
