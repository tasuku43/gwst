# gws CLI 仕様（MVP）

このドキュメントは「実装時に迷わない」ための最小限の仕様を定義する。

## グローバル

### コマンド形式
- `gws <command> [flags] [args]`

### 共通フラグ（MVP）
- `--root <path>`: GWS_ROOT を上書き
- `--no-prompt`: 対話を行わない（MVPでは破壊的操作を拒否してもよい）
- `--verbose` / `-v`: 実行ログを増やす（`GWS_VERBOSE=1` でも可）

### 出力
- MVP では人間向け表示のみを提供する（JSON 出力は将来拡張）

### ルート解決
`--root` > `GWS_ROOT` > `~/gws`

## コマンド一覧（MVP）

- `gws new [--template <name>] [<ID>]`
- `gws add <ID> <repo>`
- `gws ls`
- `gws status <ID>`
- `gws rm <ID>`
- `gws doctor [--fix]`
- `gws init`
- `gws repo get <repo>`
- `gws repo ls`
- `gws template ls`
- `gws review <PR URL>`

## repo

### gws repo get <repo>
目的:
- repo store（bare）を作成・更新する

挙動:
- repo store が無い: `git clone --bare <remote> <store>`
- ある: `fetch` は行わない（`gws new` 時に最新化する）
- `src/<host>/<owner>/<repo>` に作業ツリーを作成（既存なら何もしない）

入力形式:
- `git@github.com:owner/repo.git`
- `https://github.com/owner/repo.git`

成功条件:
- `<store>` が存在し、`fetch` が成功している

### gws repo ls
目的:
- repo store 一覧を出す

MVP出力:
- repo_key, store_path, remote_url, last_fetch_at（あれば）

## template

### gws template ls
目的:
- `$GWS_ROOT/templates.yaml` に定義されたテンプレート名を一覧する

テンプレートは `templates.yaml` を直接編集する

## init

### gws init
目的:
- `$GWS_ROOT` 配下に必要なディレクトリ/設定ファイルを作成する

挙動:
- `bare/`, `src/`, `workspaces/` を作成（既存ならスキップ）
- `templates.yaml` を作成（既存ならスキップ）
  - `example` テンプレート（複数 repo）を同梱

## workspace

### gws new [--template <name>] [<WORKSPACE_ID>]
目的:
- `$GWS_ROOT/workspaces/<id>/` を作成

制約:
- WORKSPACE_ID は Git ブランチ名として妥当な文字列であること（refname check）
- template 未指定時は対話で template と WORKSPACE_ID を入力する

### gws add <WORKSPACE_ID> <repo>
目的:
- workspace 配下に worktree を作成する

既定ルール:
- branch = WORKSPACE_ID
- base_ref = origin/HEAD（空なら自動検出。必要なら main/master/develop を順に探索）

挙動:
1. repo get 済みであることを前提に store を最新化（未取得ならエラー）
2. `<workspaces>/<id>/<repo_name>` を作業ディレクトリとして決定
3. ブランチが存在しない場合は base_ref から作成して worktree add
4. workspace 配下に worktree を作成

テンプレート適用時の repo get:
- 未取得 repo がある場合、対話で `repo get` を実行して続行するか確認する

失敗条件（MVP）
- alias が衝突している
- WORKSPACE_ID ブランチが既に別の worktree で checkout されている（Gitが拒否するはず）

### gws ls
目的:
workspace 一覧（MVPはディレクトリ走査）

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
1. 各 repo の worktree remove（git経由）
2. workspace ディレクトリ削除

安全ガード:
- dirty なら拒否（既定）

## doctor

### gws doctor [--fix]
目的:
- よくある壊れ方を検出する

MVP対象:
- repo store はあるが remote が取れない

## review

### gws review <PR URL>
目的:
- PR レビュー専用の導線で workspace を作成する

挙動:
1. PR/MR URL から host/owner/repo/番号を取得（GitHub, GitLab, Bitbucket Cloud）
2. repo store が未取得なら `repo get` と同等の導線で取得（対話）
3. PR/MR の ref を直接 fetch し、専用ブランチで worktree を作成する

制約:
- GitHub / GitLab / Bitbucket Cloud 対応（fork PR/MR も可）
- 既存 workspace がある場合はエラー
- workspace ID は `REVIEW-PR-<number>`
