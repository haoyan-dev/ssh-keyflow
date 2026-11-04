# 01: 起動・環境確認・管理対象外の一覧・既存鍵の取り込み

Status: ready-for-agent
Blocked by: None (can start immediately)

## What to build

起動・環境確認・管理対象外の一覧・既存鍵の取り込みを対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [ ] 起動時に依存コマンド、書き込み、JSONを確認し、不足の導入案内と再確認ができる
- [ ] 対話メニューから一覧と明示的な既存鍵取り込みを実行できる
- [ ] 再帰Include探索はMatch execを実行せず、管理対象外の元ファイルを変更しない
- [ ] 固定ID、version、バックアップ、排他、不一致の要確認、公開鍵のみと鍵ペア確認が機能する
- [ ] CLI境界で正常経路と破損・競合・対象外保護をTDDで検証する

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。
