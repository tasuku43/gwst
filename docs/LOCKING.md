# ロック戦略（MVP）

目的:
- 人間 + 複数エージェントの同時実行で破壊的競合が起きないようにする

## ロック単位
1) repo ロック
- 対象: repo store 単位
- 目的: clone/fetch/worktree add/remove の競合回避

2) workspace ロック
- 対象: workspace 単位
- 目的: ws add/rm/gc の競合回避

## ロックファイル
- repo: `$STORE/.gws/lock`
- ws: `$WS/.gws/lock`

## ロック情報
ロックファイルには以下を記録（テキストで十分）:
- pid
- hostname
- started_at
- command（任意）

## タイムアウト / 回収
- 一定時間を超えたロックは doctor が “疑わしい” として報告
- `doctor --fix` で回収できる（ただし安全のため確認が望ましい）