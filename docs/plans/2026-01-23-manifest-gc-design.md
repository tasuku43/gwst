---
title: "Design: gion manifest gc"
status: draft
date: 2026-01-23
---

# gion manifest gc (Design Note)

## Background / Goal

Workspaces can be in states that are either safe to delete automatically or should never be deleted automatically. `gion manifest gc` mechanically selects only those with extremely high confidence of being safe, removes them from `gion.yaml` (desired state), and relies on `gion apply` to reconcile (perform the actual deletion).

Manual removal (`gion manifest rm`) remains the explicit, human-judgment path.

## Responsibilities

- `gion manifest gc` is a **manifest mutation** command (it shrinks the inventory).
- Physical deletion (worktree remove / workspace dir remove) is performed by **`gion apply`**.
- Candidate selection is **conservative** (exclude when in doubt) to minimize accidental deletions.

## Assumptions / Data sources

- Decisions are based on local state only (**no implicit fetch/prune**).
- Dirty/ahead/behind detection follows `git status --porcelain=v2 -b` (via existing workspace status/state logic).
- Merge target selection is per-repo:
  1) Use `gion.yaml repos[].base_ref` when set (e.g. `origin/release/1.2`)
  2) Otherwise, use `origin/<default>` resolved from `refs/remotes/origin/HEAD`

## Rules (minimal set)

### Base exclusions (workspace-level)
Exclude any workspace if any repo is:
- dirty (uncommitted changes)
- unpushed (ahead > 0)
- diverged (ahead > 0 and behind > 0)
- unknown (cannot determine status: missing upstream, detached HEAD, git errors, etc.)

### Rule: strict merged (the only candidate rule)
For each repo, mark it as matched only if:
- `HEAD` is an ancestor of `origin/<target>` (i.e. `HEAD` is reachable from `origin/<target>`), and
- `HEAD != origin/<target>` (prevents deleting "created-only" workspaces)

A workspace becomes a GC candidate only when **all repos** match strict merged.

## Mode-specific notes

- `mode=review`:
  - The natural merge target is the PR base branch.
  - Therefore `gion manifest add --review` should store `repos[].base_ref = origin/<pr_base_branch>`.
  - `manifest gc` itself does not need special casing by mode if target selection is consistent (base_ref first) and strict merged is used.
- `mode=issue` assumes merging into the default branch (overrideable via `--base` / `base_ref` when needed).

## UX / Output

- Always show candidate list and reasons (per-repo strict merged + target context) before mutating the manifest.
- With `--no-apply`, only update `gion.yaml`.
- Otherwise, run `gion apply` once for the entire root (same pattern as other manifest mutations).
- If apply is declined/canceled (`n` / `Ctrl-C`), roll back `gion.yaml` (same as `manifest add`).

## Failure handling

- Git errors during evaluation are treated as `unknown` => skip (report as warnings).
- If there are 0 candidates, do nothing (do not modify the manifest).
