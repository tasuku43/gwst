# 出力仕様（MVP）

MVPでは “人間向け表示” を中心にしつつ、将来の agent 統合を見越して `--json` を用意する。

## `--json`（MVPで実装推奨）
- `gws ls --json`
- `gws status <ID> --json`
- `gws gc --dry-run --json`
- `gws doctor --json`
- `gws src ls --json`

JSON は schema_version を含める:
```json
{
  "schema_version": 1,
  "command": "status",
  "workspace_id": "PROJ-1234",
  "repos": [
    { "alias": "backend", "branch": "PROJ-1234", "dirty": false }
  ]
}
