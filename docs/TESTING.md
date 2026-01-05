# テスト計画（MVP）

## 方針
- 単体テスト: 正規化、パス解決、manifest I/O、config 優先順位
- 統合テスト: 一時ディレクトリに git リポジトリを作成し、bare store → worktree add/remove を検証

## 統合テストの戦略
- `git init` で “ローカルのダミーリモート” を作り、それを remote URL として扱う
- fetch/prune などはローカルでも成立するようにする

## DoD（MVP）
- `go test ./...` が通る
- 主要コマンド（repo get、ws new/add/status/rm）が統合テストで一通り動く
