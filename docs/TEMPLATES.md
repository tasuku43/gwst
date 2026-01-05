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

ルール:
- `gws new` でテンプレート名を指定（未指定なら対話）
- repo は `gws repo get` 済みであることが前提（未取得ならエラー）
- `gws template add` は対話形式でテンプレートを追加する
- `gws template show <name>` でテンプレート内容を確認できる
- `gws template rm <name>` でテンプレートを削除できる
