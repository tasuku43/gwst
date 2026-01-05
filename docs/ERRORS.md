# エラーコード（MVP）

exit code（提案）:
- 0: success
- 10: invalid args / not found
- 20: blocked by safety guard（dirty 等）
- 30: external dependency failure（git 実行失敗、ネットワーク等）
- 40: lock acquisition failure

MVPでは、stderr に人間向け説明を出す。
`--json` 時は、可能なら JSON で error を返す（将来拡張で安定化）。
