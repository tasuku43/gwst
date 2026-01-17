---
title: "gws template ls"
status: implemented
---

## Synopsis
`gws template ls`

## Intent
Show available workspace templates defined in `templates.yaml` so users can choose one when creating workspaces.

## Behavior
- Loads `<root>/templates.yaml`; fails if the file is missing or unreadable.
- Parses template entries (supports current and legacy `repos` formats).
- Prints template names in sorted order, and for each template lists its repositories in display form.

## Success Criteria
- Templates are listed with their repo entries; if none exist, the command reports that no templates were found.

## Failure Modes
- Root not resolved.
- `templates.yaml` missing, unreadable, or invalid YAML.
