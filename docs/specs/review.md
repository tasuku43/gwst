---
title: "gws review"
status: implemented
pending:
  - interactive-pr-picker
  - multi-pr-workspaces
---

## Synopsis
`gws review [<PR URL>] [--no-prompt]`

## Intent
Create a review-focused workspace for a GitHub pull request, checking out the PR head branch in a dedicated worktree.

## Behavior
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

## Success Criteria
- For URL mode: new workspace `REVIEW-PR-<number>` exists with a worktree checked out to the PR head branch.
- For picker mode: each selected PR produces a workspace with the same guarantees; partial success is reported if a later item fails.

## Failure Modes
- Invalid or unsupported PR URL; non-GitHub host.
- Fork PR detected.
- Missing or unauthenticated `gh` CLI.
- Repo store missing and user declines/forbids `repo get`.
- Git errors fetching the PR head or creating the worktree.
- Picker mode: no TTY available, repo selection empty, PR fetch fails (auth/network/API), or zero PRs selected.
