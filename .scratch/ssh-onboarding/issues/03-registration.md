# 03: 自動登録・指定鍵の接続確認・SSH別名設定

Status: ready-for-human
Resolution: completed
Blocked by: 02

## What to build

自動登録・指定鍵の接続確認・SSH別名設定を対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [x] 既存鍵またはパスワードで新規ログインし、自分の公開鍵を重複なく登録する
- [x] OSと標準登録先の確認、Windows共有管理者ファイルの影響確認を行う
- [x] ホスト鍵検証を保持し、指定鍵だけで接続結果を確認する
- [x] 結果不明を再確認し、登録状態と接続結果を分けて記録する
- [x] 別名衝突と管理対象外設定を保護し、Includeの差分・バックアップ・承認または手動案内が機能する
- [x] CLI境界で登録と設定保護をTDDで検証する

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。

2026-11-04: Completed implementation at the agreed app.Run boundary using the TDD skill. Choices 1/a and 3 perform native interactive bootstrap, guarded shared-script registration, read-only reconciliation before retry, independent target-key verification and optional owned config/approved Include edits. Original terminal input is preserved; known_hosts and known_hosts2 use the isolated Home, with quoted spaced paths and changed-key failures preserved. Durable intent and append-only registration events are saved under the whole-operation lock. Configuration digests, exact backups and interrupted Include recovery protect unowned/changed files. Manual onboarding, provenance and creation resume remain covered.

Evidence: go test ./... -count=1 and go vet ./... pass. Focused native ssh -G argument parsing passes without a network connection. Automatic POSIX scripts run in isolated Git Bash fixtures; Windows standard-account automatic registration/reconciliation runs with real NTFS ACLs in isolated homes. PowerShell administrator syntax/error fixtures pass. docs/application-contract.md defines the schema, operation lifecycle, config ownership and ticket04 handoff.

Validation gaps: no live SSH server/password/key/host-key handshake, native Linux/macOS server execution or elevated Windows shared-administrator ACL execution. Script fixtures and native local parsing do not establish those results. Tests never modify real user .ssh files or remote servers. README.md and docs/verification.md remain parent-owned; integration is merged into this ticket branch only.
