# 安全設計（破壊的操作のガード）

## 原則
- Safe by default
- dirty（未コミット変更）や未追跡ファイルがある場合、削除・回収を拒否する

## dirty 判定（MVP）
- `git status --porcelain` が非空なら dirty=true
- 未追跡ファイルを含めるかは設定で将来切替可能（MVPは含める）

## 破壊的フラグ
- `--force`: 対話省略・再実行寄り（MVPでは省略可）
- `--nuke`: dirty でも削除（MVPでは実装してもよいが、最小なら未実装でもよい）

## 推奨運用
- `gws gc --dry-run` → 候補確認 → `gws gc --older ...` 実行
- pinned workspace は回収しない
