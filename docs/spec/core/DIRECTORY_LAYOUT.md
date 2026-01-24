---
title: "Directory layout under GWIAC_ROOT"
status: implemented
---

# Directory layout under GWIAC_ROOT

`gwiac` manages a single root directory (`GWIAC_ROOT`) that contains the bare repo store and workspaces.

## Root resolution

`GWIAC_ROOT` is resolved in this order:

1. `--root <path>`
2. `GWIAC_ROOT` environment variable
3. default `~/gwiac`

## Layout

```
GWIAC_ROOT/
  bare/         # bare repo store (shared Git objects)
  workspaces/   # workspaces (task-scoped worktrees)
  gwiac.yaml
  logs/         # created when --debug is used
```

## Workspaces

Each workspace is a directory under `workspaces/` and contains one or more repo worktrees:

```
GWIAC_ROOT/workspaces/<WORKSPACE_ID>/
  <alias1>/
  <alias2>/
  .gwiac/metadata.json   # workspace metadata (optional: mode, description, source_url, preset_name, base_branch)
```

`gwiac open` runs a subshell with `GWIAC_WORKSPACE=<WORKSPACE_ID>` set for the child process.
