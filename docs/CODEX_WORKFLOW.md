# Codex CLI で gws を実装するための運用

このリポジトリは、Codex CLI を “実装エージェント” として活用する前提でドキュメントを整備する。

## 1) Codex が読む指示ファイル
- ルートの `AGENTS.md` に、実装規約・安全規約・テスト規約を記載する
- 必要なら `AGENTS.override.md` をローカルで使う（個人の一時ルール）

## 2) 推奨ワークフロー（人間 + Codex）
1. `tasks/MVP.md` のタスクを 1 つ選ぶ
2. Codex に「そのタスクだけを実装し、DoD を満たす」よう依頼する
3. Codex は必ず `go test ./...` を通し、差分を小さく保つ
4. 人間がレビューし、次タスクへ

## 3) codex exec を使った “スクリプト的実行” の推奨
- 非対話で走らせたい場合は `codex exec` を使う
- 出力を機械で扱うなら `--json` を付ける
- 構造化結果が必要なら `--output-schema` を使う

## 4) 安全性
- `rm -rf` や `sudo` 等は原則禁止（AGENTS.md に明記）
- CI など隔離環境以外では `danger-full-access` は使わない
