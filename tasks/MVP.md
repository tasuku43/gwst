# gws MVP tasks (v0.1)

このタスクリストは、Codex CLI で 1 タスクずつ実装する前提で書かれている。
各タスクは “小さく完結” し、`go test ./...` を通して終える。

## 進捗サマリ

ステータス種別: To Do / Doing / Done

| Task ID | Goal (short) | Status |
| --- | --- | --- |
| MVP-001 | Initialize repository skeleton | Done |
| MVP-010 | Implement root resolution | Done |
| MVP-011 | Implement config loader | Done |
| MVP-012 | Git command runner | Done |
| MVP-020 | Repo spec normalization | Done |
| MVP-021 | gws repo get | Done |
| MVP-022 | gws repo ls | Done |
| MVP-030 | gws ws new | Done |
| MVP-031 | manifest read/write library | Done |
| MVP-032 | gws ws add (worktree add) | Done |
| MVP-033 | gws ws ls | Done |
| MVP-034 | gws ws status | To Do |
| MVP-035 | gws ws rm | To Do |
| MVP-040 | gws gc --dry-run | To Do |
| MVP-041 | gws gc (execute) | To Do |
| MVP-042 | gws doctor | To Do |
| MVP-050 | Unit tests for normalization & config | To Do |
| MVP-051 | Integration tests for repo get + ws add/rm | To Do |
| MVP-060 | Basic Makefile or justfile | To Do |
| MVP-070 | src get/ls (human working tree) | To Do |
| MVP-080 | template (workspace templates) | To Do |

## MVP Definition of Done (全体)
- コマンド:
    - `gws repo get|ls`
    - `gws ws new|add|ls|status|rm`
    - `gws gc --dry-run` と実行
    - `gws doctor`（最低限）
- ルート解決: `--root` > `GWS_ROOT` > config > default(~/gws)
- ディレクトリ構造: `<root>/repos`, `<root>/ws`
- workspace_id は refname 妥当性を必須、ブランチ名 = workspace_id
- 主要な統合テストが通る（temp dir + local dummy remote）
- `go test ./...` が常に green

---

## EP0: Repo bootstrap & docs

### MVP-001: Initialize repository skeleton
Goal:
- Go module / basic folder structure / docs placeholders を作る

Acceptance:
- `docs/` と `tasks/` が存在し、最低限の README がある
- `go test ./...` が通る（空でもよい）

Notes:
- CLI framework は MVP では標準ライブラリでよい（独自 dispatcher）

---

## EP1: Core libs (config/root/log/exec)

### MVP-010: Implement root resolution
Goal:
- `--root`, `GWS_ROOT`, config, default の優先順位で root を決定

Acceptance:
- `gws --root /tmp/x ...` が root を上書きする
- env `GWS_ROOT` が反映される
- config.yaml の root が読める
- 未指定時は `~/gws`

Files:
- internal/config
- internal/paths

### MVP-011: Implement config loader
Goal:
- `~/.config/gws/config.yaml` をロードし、デフォルト値を埋める

Acceptance:
- YAML が読める
- 想定キー（docs/CONFIG.md）を解釈できる
- 未設定でもクラッシュしない

### MVP-012: Git command runner
Goal:
- `git` を `os/exec` で実行し、stdout/stderr/exit を扱える共通関数を作る

Acceptance:
- `git --version` が取れる
- 失敗時に stderr が取り出せる

---

## EP2: Repo store

### MVP-020: Repo spec normalization
Goal:
- remote URL / github.com/org/repo を repo_key (host/owner/repo) に正規化

Acceptance:
- SSH形式とHTTPS形式を同一 repo_key にできる
- `.git` の有無に耐える
- 不正形式はわかりやすくエラー

### MVP-021: gws repo get
Goal:
- repo store を作成（clone --bare）/ 更新（fetch --prune）

Acceptance:
- 新規: `<root>/repos/<host>/<owner>/<repo>.git` が作られる
- 既存: fetch が走り、失敗時はエラーがわかる

### MVP-022: gws repo ls
Goal:
- repo store の一覧表示（MVP）

Acceptance:
- `<root>/repos` 配下を走査して一覧できる
- 破損ディレクトリは警告扱い

---

## EP3: Workspace

### MVP-030: gws ws new
Goal:
- workspace dir + `.gws/manifest.yaml` 作成

Acceptance:
- `<root>/ws/<id>` が作られる
- workspace_id refname チェックがある
- manifest が生成される

### MVP-031: manifest read/write library
Goal:
- manifest の読み書きと更新（repos追記、last_used_at更新）

Acceptance:
- ws add で repos に追記できる
- 冪等（同じ repo/alias を二重登録しない）

### MVP-032: gws ws add (worktree add)
Goal:
- repo store を最新化し、`<root>/ws/<id>/<alias>` に worktree を作成する

Rules:
- branch = workspace_id
- base_ref = defaults.base_ref（origin/main 既定）

Acceptance:
- ブランチがなければ base_ref から作成される
- 既にブランチがあればそのブランチで checkout
- manifest に store_path, worktree_path が入る

### MVP-033: gws ws ls
Goal:
- workspace 一覧を出す

Acceptance:
- `<root>/ws/*` を列挙し、manifest があれば読み、無ければ警告

### MVP-034: gws ws status
Goal:
- workspace 内の repo の状態（dirty等）を集計

Acceptance:
- `git status --porcelain` で dirty を判定
- alias ごとに結果を出せる

### MVP-035: gws ws rm
Goal:
- workspace を安全に削除

Acceptance:
- dirty があれば拒否（既定）
- worktree remove を git 経由で実施
- workspace ディレクトリが消える

---

## EP4: GC & Doctor

### MVP-040: gws gc --dry-run
Goal:
- stale workspace の候補を列挙

Acceptance:
- `--older 30d` などで候補が出る
- pinned は除外

### MVP-041: gws gc (execute)
Goal:
- 候補 workspace を安全に回収

Acceptance:
- dirty は拒否
- ws rm 相当の削除を行う

### MVP-042: gws doctor
Goal:
- 最低限の不整合検出

Acceptance:
- 古いロックファイルを検出
- manifest はあるが worktree が無い等を検出してヒントを出す
- `--fix` はMVPでは “ロック回収のみ” でもよい

---

## EP5: Tests

### MVP-050: Unit tests for normalization & config
Acceptance:
- 正規化と root 解決にテストがある

### MVP-051: Integration tests for repo get + ws add/rm
Acceptance:
- temp dir で local dummy remote を作り、一通り通る

---

## EP6: Release hygiene (optional for MVP)

### MVP-060: Basic Makefile or justfile
Acceptance:
- `make test`, `make fmt` 等が動く

### MVP-061: CI (GitHub Actions) basic
Acceptance:
- go test が走る

---

## EP7: Human working tree (src)

### MVP-070: src get / ls
Goal:
- human 向けの作業ツリーを `$GWS_ROOT/src` に用意する

Acceptance:
- `gws src get <repo>` で `$GWS_ROOT/src/<host>/<owner>/<repo>` が作成される
- `gws src ls` で `src` 配下の一覧が出せる

---

## EP8: Templates

### MVP-080: template + new
Goal:
- `$GWS_ROOT/templates.yaml` で workspace テンプレートを管理する
- `gws new` でテンプレート適用を行えるようにする

Acceptance:
- `gws template ls` でテンプレート名を一覧できる
- `gws new --template <name> <id>` で template の repos が `ws` に追加される
- template 未指定時は対話で template と workspace id を入力できる
- repo get 未実行の repo があればエラーで中断する
