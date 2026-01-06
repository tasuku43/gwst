# gws review（PR レビュー用ワークスペース）

状態: draft（MVP 以降）

## 目的
- PR レビュー専用の導線で workspace を作成する
- 人間/エージェントがレビュー開始時に迷わず作業環境を整えられる

## コマンド
- `gws review <PR URL>`

## 対象範囲（MVP 以降）
- GitHub の PR URL のみ対応
- fork PR は対象外
- 既存 workspace がある場合はエラー

## 前提
- `gh` が利用可能であること（認証済み）

## 仕様（動作）
1. PR URL から `owner/repo` と PR 番号を取得
2. `gh` で PR 情報を取得
   - headRefName（PR ブランチ名）
   - headRepository（fork でないことを確認）
3. repo store が未取得なら `repo get` と同等の導線で取得（対話）
4. workspace を作成
   - workspace ID は `REVIEW-PR-<number>` を使用（例: `REVIEW-PR-123`）
   - 既存 workspace がある場合はエラー
5. PR の head を取得して worktree を作成
   - `git fetch origin <headRefName>` で PR ブランチを取得
   - local branch は PR head から作成する
   - workspace ID と worktree ブランチ名は一致しない

## 出力
- `gws new` と同じ UX（インデント/進捗表示）
- 最後に workspace ツリーを表示

## エラー
- PR URL が GitHub 以外: エラー
- fork PR: エラー
- `gh` 未インストール/未認証: エラー
- repo store 未取得で `repo get` を拒否: エラー
- workspace が既に存在: エラー

## 今後の検討
- fork PR 対応
- `workspace_id` のカスタム（`--id`）
- `--no-fetch` などのオプション
