---
title: "Directory layout under GWS_ROOT"
status: implemented
---

# Directory layout under GWS_ROOT

`gws` manages a single root directory (`GWS_ROOT`) that contains the bare repo store and workspaces.

## Root resolution

`GWS_ROOT` is resolved in this order:

1. `--root <path>`
2. `GWS_ROOT` environment variable
3. default `~/gws`

## Layout

```
GWS_ROOT/
  bare/         # bare repo store (shared Git objects)
  workspaces/   # workspaces (task-scoped worktrees)
  templates.yaml
  logs/         # created when --debug is used
```

## Workspaces

Each workspace is a directory under `workspaces/` and contains one or more repo worktrees:

```
GWS_ROOT/workspaces/<WORKSPACE_ID>/
  <alias1>/
  <alias2>/
  .gws/metadata.json   # optional workspace metadata (e.g. description)
```

`gws open` runs a subshell with `GWS_WORKSPACE=<WORKSPACE_ID>` set for the child process.
