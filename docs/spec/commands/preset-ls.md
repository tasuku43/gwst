---
title: "gwst preset ls"
status: implemented
---

## Synopsis
`gwst preset ls`

## Intent
Show available workspace presets defined in `gwst.yaml` so users can choose one when creating workspaces.

## Behavior
- Loads `<root>/gwst.yaml`; fails if the file is missing or unreadable.
- Parses preset entries (supports current and legacy `repos` formats).
- Prints preset names in sorted order, and for each preset lists its repositories in display form.

## Success Criteria
- Presets are listed with their repo entries; if none exist, the command reports that no presets were found.

## Failure Modes
- Root not resolved.
- `gwst.yaml` missing, unreadable, or invalid YAML.
