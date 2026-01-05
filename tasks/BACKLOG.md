# gws Backlog (post-MVP)

- Declarative workflow:
    - manifest 編集 → `gws ws apply`
- Templates:
    - `gws ws new <id> --repo ...` をテンプレ化
- JSON outputs:
    - schema_version を固定し、エラーも JSON で安定返却
- GitHub integration:
    - PR merge 状態で gc 候補精度を上げる
- Advanced safety:
    - “nuke” の明確な設計と監査ログ
- Windows support
