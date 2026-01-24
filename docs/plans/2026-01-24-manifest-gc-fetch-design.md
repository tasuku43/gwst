---
title: "Design: manifest gc default fetch"
status: draft
date: 2026-01-24
---

# manifest gc default fetch (Design Note)

## Goal

Ensure `gion manifest gc` evaluates merge state using up-to-date base refs from bare repo stores, so merged workspaces are detected even when `gion apply` has not fetched.

## Decision

- Default behavior: fetch only base refs needed for merge checks from bare repo stores.
- Opt-out flag: `--no-fetch`.
- Only repos referenced by workspace entries are fetched (ignore presets).
- Fetch is performed against bare stores, not worktrees.

## Behavior details

- Collect distinct `repos[].repo_key` values from all workspaces.
- For each repo, fetch only `origin/<base>` for merge checks (or default branch if `base_ref` is unset).
- If a repo fetch fails, treat that repo as unknown for the workspace and skip it from GC candidates (warn with workspace + repo context).

## Rationale

- `manifest gc` should be conservative but correct; stale refs silently hide merge state.
- Fetching bare stores keeps worktrees untouched and matches existing repo-store usage.
- `--no-fetch` preserves offline workflows.
