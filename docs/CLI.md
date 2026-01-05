# gws CLI 仕様（MVP）

このドキュメントは「実装時に迷わない」ための最小限の仕様を定義する。

## グローバル

### コマンド形式
- `gws <command> [flags] [args]`

### 共通フラグ（MVP）
- `--root <path>`: GWS_ROOT を上書き
- `--no-prompt`: 対話を行わない（MVPでは破壊的操作を拒否してもよい）
- `--json`: 機械可読出力（MVPでは ls/status のみでも可）
- `--verbose`: 実行ログを増やす
- `--quiet`: 最低限の出力

### ルート解決
`--root` > `GWS_ROOT` > `~/.config/gws/config.yaml` の root > `~/gws`

## コマンド一覧（MVP）

- `gws new <ID>`
- `gws add <ID> <repo> --alias <name>`
- `gws ls`
- `gws status <ID>`
- `gws rm <ID>`
- `gws gc [--dry-run] [--older <duration>]`
- `gws doctor [--fix]`
- `gws repo get <repo>`
- `gws repo ls`

## repo

### gws repo get <repo>
目的:
- repo store（bare）を作成・更新する

挙動:
- repo store が無い: `git clone --bare <remote> <store>`
- ある: `git -C <store> fetch --prune`

成功条件:
- `<store>` が存在し、`fetch` が成功している

### gws repo ls
目的:
- repo store 一覧を出す

MVP出力:
- repo_key, store_path, remote_url, last_fetch_at（あれば）

## workspace

### gws new <WORKSPACE_ID>
目的:
- `$GWS_ROOT/ws/<id>/` と `.gws/manifest.yaml` を作成

制約:
- WORKSPACE_ID は Git ブランチ名として妥当な文字列であること（refname check）

### gws add <WORKSPACE_ID> <repo> --alias <name>
目的:
- workspace 配下に worktree を作成する

既定ルール:
- branch = WORKSPACE_ID
- base_ref = 設定の defaults.base_ref（既定 origin/main）

挙動:
1. repo get を内部的に実行（store を最新化）
2. `<ws>/<id>/<alias>` を作業ディレクトリとして決定
3. ブランチが存在しない場合は base_ref から作成して worktree add
4. manifest に追記
5. last_used_at を更新

失敗条件（MVP）
- alias が衝突している
- WORKSPACE_ID ブランチが既に別の worktree で checkout されている（Gitが拒否するはず）

### gws ls
目的:
- workspace 一覧（MVPはディレクトリ走査 + manifest読取）

### gws status <WORKSPACE_ID>
目的:
- workspace 内の各 worktree の dirty 状態等を集計

MVPで返す最小項目:
- repo alias
- branch（= workspace_id）
- HEAD short sha
- dirty（true/false）
- untracked_count（概算でよい）

### gws rm <WORKSPACE_ID>
目的:
- workspace を安全に削除

挙動:
1. workspace lock 取得
2. 各 repo の worktree remove（git経由）
3. workspace ディレクトリ削除

安全ガード:
- dirty なら拒否（既定）
- `--nuke` でのみ強制削除（MVPでは `--nuke` 未実装でも可）

## gc

### gws gc [--dry-run] [--older <duration>]
目的:
- stale workspace を列挙し、回収する

stale 判定（MVP）:
- manifest.last_used_at が `--older` を超える
- または policy.ttl_days を超える

## doctor

### gws doctor [--fix]
目的:
- よくある壊れ方を検出する

MVP対象:
- ロック残骸（一定時間以上古い）
- manifest はあるが worktree が存在しない
- repo store はあるが remote が取れない
