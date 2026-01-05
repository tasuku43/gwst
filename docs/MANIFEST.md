# workspace manifest 仕様（MVP）

場所:
- `$GWS_ROOT/ws/<ID>/.gws/manifest.yaml`

目的:
- “意図” を記録する（現状観測は state.json を想定）

## 最小スキーマ（MVP）

```yaml
schema_version: 1
workspace_id: "PROJ-1234"

created_at: "2026-01-04T12:00:00Z"
last_used_at: "2026-01-04T12:00:00Z"

policy:
  pinned: false
  ttl_days: 30

repos:
  - alias: "backend"
    repo_spec: "git@github.com:org/backend.git"
    repo_key: "github.com/org/backend"
    store_path: "/home/user/gws/repos/github.com/org/backend.git"
    worktree_path: "/home/user/gws/ws/PROJ-1234/backend"
    branch: "PROJ-1234"
    base_ref: "origin/main"
    created_branch: true
```

## 更新ルール（MVP）

- ws new: manifest 新規作成 
- ws add: repos 配列に追記し、last_used_at 更新 
- ws status: last_used_at 更新（運用判断。MVPでは更新しないでも可）
