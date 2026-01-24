---
title: "gwiac plan"
status: implemented
---

## Synopsis
`gwiac plan [--root <path>] [--no-prompt]`

## Intent
Compute and display the diff between `gwiac.yaml` and the filesystem without applying changes, so users can review intended actions.

## Behavior
- Loads `<root>/gwiac.yaml`; errors if missing or invalid.
- Scans `<root>/workspaces` to build the current state.
- Computes a plan with `add`, `remove`, and `update` actions:
  - `add`: workspace or repo entry exists in manifest but not on filesystem.
  - `remove`: exists on filesystem but not in manifest.
  - `update`: exists in both but differs by repo alias, repo key, or branch.
- Renders a human-readable plan summary and exits without changes.
  - `remove` actions include a risk summary by inspecting each repo in the workspace:
    - Prints `risk:` only when non-clean (e.g., `dirty`, `unpushed`, `diverged`, `unknown`).
    - `sync:` (ahead/behind) if applicable.
    - `changes: clean` if no working tree changes.
    - For dirty repos, `changes:` counts and `files:` with the modified/untracked/conflicted file list.
- `--no-prompt` is accepted but has no effect (kept for CLI consistency).

## Success Criteria
- Plan is printed to stdout; exit status is 0 even if the plan is empty.

## Failure Modes
- Manifest file missing or invalid.
- Filesystem or git errors while scanning workspaces.
