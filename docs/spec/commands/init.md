---
title: "gwiac init"
status: implemented
---

## Synopsis
`gwiac init`

## Intent
Create the minimum directory and file layout under the resolved GWIAC root so other commands can operate safely.

## Behavior
- Validates that a root directory was resolved; otherwise fails.
- Creates (if absent) the directories `bare/` and `workspaces/` under the root with `0755` permissions.
- Creates `gwiac.yaml` if it does not exist, seeding it with:
  - `presets.example` that lists two GitHub repositories.
  - an empty `workspaces` map.
- Skips creation for items that already exist and reports them as skipped.

## Success Criteria
- Required directories exist and are directories.
- `gwiac.yaml` exists and is a file (either newly created or previously present).

## Failure Modes
- Root directory not provided or cannot be accessed.
- Filesystem errors while creating directories or writing `gwiac.yaml`.
