# workspace templates 仕様（MVP）

場所:
- `$GWS_ROOT/templates.yaml`

目的:
- workspace に追加する repo 群をテンプレートとして管理する

例:
```yaml
templates:
  webapp:
    repos:
      - github.com/org/frontend
      - github.com/org/backend
```

互換性:
- 過去の形式（`repos: - repo: ...`）も読み込み可能

ルール:
- `gws create --template <name>` でテンプレート名を指定（未指定なら対話）
- repo は `gws repo get` 済みであることが前提（未取得ならエラー）
- テンプレートの編集は `templates.yaml` を直接編集する
- 編集後は `gws template validate` で整合性を確認する

repo get の補助:
- 未取得 repo がある場合は `gws create --template` が対話で `repo get` を実行するか確認する

repo 形式:
- `git@github.com:owner/repo.git`
- `https://github.com/owner/repo.git`
