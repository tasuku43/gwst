---
title: "gion init"
status: implemented
---

## Synopsis
`gion init`

## Intent
Create the minimum directory and file layout under the resolved GION root so other commands can operate safely.

## Behavior
- Validates that a root directory was resolved; otherwise fails.
- Creates (if absent) the directories `bare/` and `workspaces/` under the root with `0755` permissions.
- Creates `gion.yaml` if it does not exist, seeding it with:
  - `presets.example` that lists two GitHub repositories.
  - an empty `workspaces` map.
- Skips creation for items that already exist and reports them as skipped.

## Success Criteria
- Required directories exist and are directories.
- `gion.yaml` exists and is a file (either newly created or previously present).

## Failure Modes
- Root directory not provided or cannot be accessed.
- Filesystem errors while creating directories or writing `gion.yaml`.
