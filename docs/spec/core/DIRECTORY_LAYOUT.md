---
title: "Directory layout under GION_ROOT"
status: implemented
---

# Directory layout under GION_ROOT

`gion` manages a single root directory (`GION_ROOT`) that contains the bare repo store and workspaces.

## Root resolution

`GION_ROOT` is resolved in this order:

1. `--root <path>`
2. `GION_ROOT` environment variable
3. default `~/gion`

## Layout

```
GION_ROOT/
  bare/         # bare repo store (shared Git objects)
  workspaces/   # workspaces (task-scoped worktrees)
  gion.yaml
  logs/         # created when --debug is used
```

## Workspaces

Each workspace is a directory under `workspaces/` and contains one or more repo worktrees:

```
GION_ROOT/workspaces/<WORKSPACE_ID>/
  <alias1>/
  <alias2>/
  .gion/metadata.json   # workspace metadata (optional: mode, description, source_url, preset_name, base_branch)
```

`gion open` runs a subshell with `GION_WORKSPACE=<WORKSPACE_ID>` set for the child process.
