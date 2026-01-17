---
title: "gws init"
status: implemented
---

## Synopsis
`gws init`

## Intent
Create the minimum directory and file layout under the resolved GWS root so other commands can operate safely.

## Behavior
- Validates that a root directory was resolved; otherwise fails.
- Creates (if absent) the directories `bare/` and `workspaces/` under the root with `0755` permissions.
- Creates `templates.yaml` if it does not exist, seeding it with an `example` template that lists two GitHub repositories.
- Skips creation for items that already exist and reports them as skipped.

## Success Criteria
- Required directories exist and are directories.
- `templates.yaml` exists and is a file (either newly created or previously present).

## Failure Modes
- Root directory not provided or cannot be accessed.
- Filesystem errors while creating directories or writing `templates.yaml`.
