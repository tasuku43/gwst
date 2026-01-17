---
title: "gws repo ls"
status: implemented
---

## Synopsis
`gws repo ls`

## Intent
List bare repo stores currently managed under the GWS root.

## Behavior
- Scans `<root>/bare` for directories ending with `.git` (nested by host/owner/repo).
- Emits each entry as `<repo_key>\t<store_path>`, where `repo_key` is the path relative to `bare/` using forward slashes.
- Collects and reports non-fatal warnings encountered while walking the directory tree.

## Success Criteria
- Existing repo stores are listed; if none exist, the command succeeds with an empty result.

## Failure Modes
- Root path inaccessible or not a directory.
- Filesystem errors while scanning the `bare/` tree.
