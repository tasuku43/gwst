---
title: "gwst gc"
status: planned
pending:
  - rules-implementation
  - confirmation-ux
  - review-upstream-exclusion
  - created-only-exclusion
  - reason-format
---

## Synopsis
`gwst gc`

## Intent
Bulk-delete workspaces that are highly likely safe to remove, using conservative rules. This is intentionally separated from manual removal flows, which remain the explicit/human-judgment path.

## Scope
- **GC**: automatic, bulk, conservative. Exclude when in doubt.
- **Manual removal**: explicit, interactive, for human judgment.

## Definitions
- **Clean**: no uncommitted changes in any repo.
- **Unpushed**: local branch is ahead of upstream (from `git status --porcelain=v2 -b`).
- **Unknown**: status cannot be determined (e.g., git error, no HEAD).
- **Base exclusions**: any workspace containing Dirty / Unpushed / Unknown repos is excluded.

## Safe-to-remove Rules (extensible)
- Rules are predicates that return `(matched bool, reason string)`.
- A workspace is a candidate only if:
  - all repos pass base exclusions, and
  - each repo matches at least one rule.
- Rules are evaluated from a list (array) to allow easy future extension.
- Initial rules (OR):
  1) **Merged into origin default branch**: repo `HEAD` is reachable from `origin/<default>` (default resolved from `origin/HEAD`).
     - Reason: `merged`
- **Remote selection**: fixed to `origin` only.
- **Review exclusion**: if repo `HEAD` equals its upstream `origin/<head_ref>` and is **not** merged into origin default, exclude.
- **Created-only exclusion**: avoid deleting workspaces created but not started (e.g., template/repo/issue)
  - Even if `HEAD` equals origin default, these should not be GC candidates.
  - Requires a clear heuristic or explicit mode metadata to avoid false positives.

## Data Collection / Performance
- Gather repo info once per repo, then reuse for all rules and exclusions.
- Source of truth for status: `git status --porcelain=v2 -b` (no implicit fetch/prune).
- Rule evaluation must not re-run expensive git commands per rule; use shared snapshot data.

## Behavior
- Scans all workspaces under root.
- For each workspace:
  - Collect status and structural info per repo.
  - Apply base exclusions.
  - Evaluate rules and collect reasons.
- Prints candidates with reasons (always shown before deletion).
- Deletes all candidates in one run (single confirmation; no per-item selection).
- Confirmation requires typing `y` to proceed; any other input cancels.
- Removes worktrees and workspace directory (same removal semantics as `gwst apply` workspace removals).

## Output
- Summary: scanned / candidates / deleted / skipped.
- Candidate list: workspace id + reasons (rule names/strings, repo context).
  - Reason strings should be short (e.g., `[merged]`).

## Success Criteria
- Dirty/Unpushed/Unknown are always excluded.
- Candidates are explainable with rule reasons.
- Rule evaluation is extensible without refactoring the core scan loop.

## Failure Modes
- Git status or rule errors => treat as unknown, skip, and report warning.
- Removal errors for a candidate => report and continue.
