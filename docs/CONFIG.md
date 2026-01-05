# gws 設定ファイル仕様（MVP）

場所:
- `~/.config/gws/config.yaml`

## 例

```yaml
version: 1

# 省略時: GWS_ROOT または ~/gws
root: ""

paths:
  repos_dir: "repos"
  ws_dir: "ws"

defaults:
  base_ref: "origin/main"
  ttl_days: 30

naming:
  workspace_id_must_be_valid_refname: true
  branch_equals_workspace_id: true

repo:
  # 省略形入力（github.com/org/repo）の解決用。MVPでは "github.com" 固定でも可
  default_host: "github.com"
  default_protocol: "https"  # "https" or "ssh"
```

## 仕様

- paths.* は root からの相対パス 
- defaults.base_ref は新規ブランチ作成時の基点 
- ttl_days は gc の既定（--older が優先） 
- workspace_id_must_be_valid_refname=true の場合、無効 ID はエラー
