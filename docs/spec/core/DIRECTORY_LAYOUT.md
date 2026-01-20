---
title: "Directory layout under GWST_ROOT"
status: implemented
---

# Directory layout under GWST_ROOT

`gwst` manages a single root directory (`GWST_ROOT`) that contains the bare repo store and workspaces.

## Root resolution

`GWST_ROOT` is resolved in this order:

1. `--root <path>`
2. `GWST_ROOT` environment variable
3. default `~/gwst`

## Layout

```
GWST_ROOT/
  bare/         # bare repo store (shared Git objects)
  workspaces/   # workspaces (task-scoped worktrees)
  gwst.yaml
  logs/         # created when --debug is used
```

## Workspaces

Each workspace is a directory under `workspaces/` and contains one or more repo worktrees:

```
GWST_ROOT/workspaces/<WORKSPACE_ID>/
  <alias1>/
  <alias2>/
  .gwst/metadata.json   # workspace metadata (mode, description, source_url, preset)
```

`gwst open` runs a subshell with `GWST_WORKSPACE=<WORKSPACE_ID>` set for the child process.
