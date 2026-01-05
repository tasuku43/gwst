# ディレクトリ設計（確定）

## ルート
- `GWS_ROOT` があればそれを使う
- 無ければ `~/gws`
- `templates.yaml` は `GWS_ROOT` 直下に配置

## 配下構造（固定）
- `$GWS_ROOT/bare/`   : repo store（bare repo）
- `$GWS_ROOT/src/`    : human working tree
- `$GWS_ROOT/ws/`     : workspace (AI)
- `$GWS_ROOT/ws/<ID>/`:
    - `<alias>/`        : worktree 作業ディレクトリ
    - `.gws/`           : gws 管理メタ

## workspace メタ
- `$GWS_ROOT/ws/<ID>/.gws/manifest.yaml`（意図）
- `$GWS_ROOT/ws/<ID>/.gws/state.json`（観測結果。MVPでは任意）
- `$GWS_ROOT/ws/<ID>/.gws/lock`（workspace ロック）

## repo store のパス
- `$GWS_ROOT/bare/<host>/<owner>/<repo>.git`
- `$GWS_ROOT/bare/<host>/<owner>/<repo>.git/.gws/lock`（repo ロック）
