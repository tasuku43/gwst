# gws コンセプト（確定版） v0.1

状態: Concept Approved（実装開始可能）  
実装言語: Go  
CLI 名: gws（`gws <subcommand>` で呼び出す）

## 1. なぜ gws が必要か（課題）

AI エージェント時代では、同一マシン上の同一コードベースに対して「複数主体（人間 + 複数エージェント）」が並列で変更を加えます。従来の clone ディレクトリを直接編集する運用では、次が顕在化します。

- 作業コンテキストの衝突（別タスクの変更や生成物が混ざる）
- “作業場の増殖” に人間の整理能力が追いつかない
- 削除（片付け）が怖いので残骸が溜まり、結果さらに危険になる
- agent が誤って破壊的操作をしやすい（rm -rf 等）

gws は、作業場を workspace（タスク単位ディレクトリ）へ昇格させ、worktree を規約化・安全化・一覧性を持って運用できるようにする。

## 2. gws のゴール / 非ゴール

### ゴール
- workspace（タスク）単位で作業場を生成し、複数リポジトリを同居させられる
- “マスター clone” で作業しない（bare repo store に履歴/オブジェクトを集約）
- worktree の作成・削除・回収（GC）を安全側で統制できる
- 人間もエージェントも使える（非対話/JSON などの拡張余地を残す）

### 非ゴール（MVPではやらない）
- GitHub/PR/Issue 状態連携（将来拡張）
- 既存リポジトリ管理ツールの完全置換（ghq等）
- worktree を「同一 repo で同一ブランチを複数同時に checkout」する高度運用
- Windows ネイティブのフルサポート（将来）

## 3. コア設計（B案: bare repo store + workspace worktree）

### 3.1 主要概念
- Repo store: 作業ディレクトリを持たない bare リポジトリを保管する領域（取得結果の保管庫）
- Workspace: タスク（チケット等）単位の作業ディレクトリ
- Worktree: Git worktree により生成される “実作業用” のチェックアウトディレクトリ

### 3.2 ルートディレクトリ
- 環境変数 `GWS_ROOT` でルートを指定できる
- 未指定の場合のデフォルトは `~/gws`

`$GWS_ROOT` 配下の固定構造（v0.1）:
- `$GWS_ROOT/repos/` : repo store（bare repo）
- `$GWS_ROOT/ws/`    : workspace 群（タスク単位）

## 4. 設定（Config）

### 4.1 設定ファイル
ユーザー設定:
- `~/.config/gws/config.yaml`

workspace ローカル:
- `$GWS_ROOT/ws/<WORKSPACE_ID>/.gws/manifest.yaml`

### 4.2 設定の優先順位
1. CLI フラグ（例: `--root`）
2. 環境変数（`GWS_ROOT`）
3. ユーザー設定ファイル（`~/.config/gws/config.yaml`）
4. デフォルト（`~/gws`）

## 5. Workspace ID とブランチ名（確定ルール）

### 5.1 Workspace ID は “Git ブランチ名として妥当” を必須にする
- v0.1 では workspace_id は refname（ブランチ名）として妥当であることを必須とする
- 無効な場合はエラーにし、修正候補を提示する（例: スペース→`_` 等）

### 5.2 ブランチ名は workspace_id と同一
- v0.1 の既定動作として、各 repo の worktree は `branch = workspace_id` を checkout する
- ブランチが存在しない場合は `base_ref`（既定 `origin/main`）から作成する

この制約により、従来の「ブランチ中心の体験」を “workspace中心” に置換し、追跡・回収（GC）を単純化する。

## 6. Repo 参照（repo spec）と repo store の配置

### 6.1 repo spec の入力形式（MVP）
- 推奨: フルの remote URL（SSH/HTTPS）
  - 例: `git@github.com:org/backend.git`
  - 例: `https://github.com/org/backend.git`

- 省略形（任意・MVPでサポート可）:
  - `github.com/org/repo`（`.git` は任意）

### 6.2 repo store のパス規約
repo store の物理パスは下記を基本とする（正規化後）:
- `$GWS_ROOT/repos/<host>/<owner>/<repo>.git`

正規化:
- 末尾 `.git` を除去して repo 名を決定
- `git@host:owner/repo` と `https://host/owner/repo` の双方を同じ repo key に正規化する

## 7. CLI（サブコマンド）— MVP

### 7.1 repo 操作
- `gws repo get <repo>`: repo store を作成または更新（clone --bare / fetch --prune）
- `gws repo ls`: repo store の一覧

### 7.2 workspace 操作
- `gws new <WORKSPACE_ID>`: workspace 作成（manifest 雛形生成）
- `gws add <WORKSPACE_ID> <repo> --alias <name>`: repo を workspace に追加（worktree 作成）
- `gws ls`: workspace 一覧
- `gws status <WORKSPACE_ID>`: workspace 内の各 repo の状態（dirty等）集計
- `gws rm <WORKSPACE_ID>`: workspace 削除（安全に worktree remove → ディレクトリ削除）

### 7.3 回収 / 診断
- `gws gc [--dry-run] [--older <duration>]`: stale workspace の候補提示と回収（安全ガード付き）
- `gws doctor [--fix]`: ロック残骸、欠損 worktree、参照不整合の検出（可能なら修復）

## 8. 安全性（Safe by default）

- dirty（未コミット変更）を検出した場合、削除・回収を拒否するのが既定
- `--dry-run` を重視し、破壊的操作は二段階を推奨
- “force” と “nuke” は分離（`--force` は対話省略/再実行、`--nuke` は破壊）

## 9. ロック（同時実行対策）

最低限、以下の排他を行う:
- repo 単位ロック: clone/fetch/worktree add/remove の競合回避
- workspace 単位ロック: ws add/rm/gc の競合回避

ロックには owner 情報（pid/host）とタイムアウトを持たせ、doctor で回収できる。

## 10. 実装方針（Go）

- Git 操作は原則 `git` コマンドを `os/exec` で呼び出す
- `git worktree list --porcelain` 等の機械向け出力をパースし、表示揺れに依存しない
- 依存ライブラリは最小化（MVPは標準ライブラリ中心）

## 11. MVP スコープ（確定）
- repo: get / ls
- workspace: new / add / ls / status / rm
- gc: dry-run と実行（安全ガード付き）
- doctor: 最低限の不整合検出（ロック残骸、欠損worktree、参照不整合の案内）

## 12. 将来拡張（バックログ）
- manifest 編集駆動（apply）
- テンプレート（複数 repo を一発で ws new）
- JSON 出力と安定スキーマ（agent 統合を強化）
- GitHub 連携（PR/Issue ステータスで gc 候補精度を上げる）
