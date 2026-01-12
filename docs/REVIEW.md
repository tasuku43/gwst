# gws create --review（PR レビュー用ワークスペース）

状態: draft（MVP 以降）

## 目的
- PR レビュー専用の導線で workspace を作成する
- 人間/エージェントがレビュー開始時に迷わず作業環境を整えられる

## コマンド
- `gws create --review <PR URL>`

## 対象範囲（MVP 以降）
- GitHub の PR URL に対応
- fork PR は非対応
- 既存 workspace がある場合はエラー

## 仕様（動作）
1. PR URL から `host/owner/repo` と番号を取得（GitHub: pull/<n>）
2. repo store が未取得なら `repo get` と同等の導線で取得（対話）
3. workspace を作成
   - workspace ID は `REVIEW-PR-<number>` を使用（例: `REVIEW-PR-123`）
4. PR の head ref を fetch
   - `git fetch origin <head_ref>`
5. fetch した ref を base に worktree を作成
   - worktree ブランチ名は PR の head ref と同じ

## 出力
- `gws create` と同じ UX（インデント/進捗表示）
- 最後に workspace ツリーを表示

## エラー
- 対応していないホスト: エラー
- repo store 未取得で `repo get` を拒否: エラー
- workspace が既に存在: エラー

## 今後の検討
- `workspace_id` のカスタム（`--id`）
- `--no-fetch` などのオプション
