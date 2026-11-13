# 04: 別認証確認・公開鍵登録解除・ローカル鍵削除

Status: ready-for-human
Resolution: completed

## What to build

別認証確認・公開鍵登録解除・ローカル鍵削除を対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [x] 対象鍵と多重化接続を除いた別認証による新規ログインを確認する
- [x] 最終承認の後に指定した公開鍵だけを解除し、他の鍵とコメントを保持する
- [x] 別認証不能や到達不能では自動解除を止め、手動解除手順を示し鍵を残す
- [x] 登録が残る間や要確認状態ではローカル鍵削除を止める
- [x] 解除済み再実行、結果不明とローカル削除をCLI境界でTDD検証する

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。

2026-11-04: ticket03 統合後の `implement/ssh-keyflow` (`a2c21d0`) を取り込み完了。共有 `StandardRevocation` の同一スクリプトを自動解除と手動案内に使用。引用符付き options 内の偽一致を除外し type+blob の完全一致だけを削除する。Linux/macOS と Windows 標準/共有管理者ターゲットに対応し、未対応設定・危険なパス/権限では停止する。

別鍵の導出 fingerprint と内容 digest を検証し、対象鍵・agent・証明書・既定鍵・多重化・認証 fallback を除いた新規ログインを要求する。host/user/port/target/fingerprint の最終承認、共有管理者の個別承認、保存済み解除 intent/history、結果不明時の読取専用照合を実装。到達不能・別認証不能では同じスクリプトを案内しローカル鍵を保持する。

メニュー6は独立した承認と削除 intent/checkpoint を要求し、登録残存/結果不明/変更ファイル/未完 Include 編集を拒否する。削除再開・再出現パス拒否・公開鍵のみの削除を検証。key/registration ID、作成/取込 provenance、履歴と tombstone を保持し、所有 digest 付き専用 config の該当 binding のみ再生成する。main Include、backup、ConfigEdits は保持する。契約は `docs/application-contract.md` に追記。

検証: `go test ./... -count=1` PASS (`cmd/ssh-keyflow` 0.768s、`internal/app` 32.934s、`internal/ssh` 10.656s)、`go vet ./...` PASS。既存の「4/6未実装」テストは実装済みメニューを妨げるため未知入力7/invalidの無変更テストへ更新。app.Run 境界の RED→GREEN、実際の guarded POSIX payload (Linux/macOS Git Bash fixture)、Windows PowerShell/NTFS の inspect/remove/retry と bytes/SDDL 保持、拒否ケースを含む。ユーザーの SSH ファイルと実サーバーにはアクセスしていない。

検証範囲: native Linux/macOS・実SSH接続/host-key handshake・昇格済み Windows 共有管理者での成功実行は未検証。Windows 標準アカウントの実スクリプト実行と共有管理者の権限/承認拒否は fixture 検証済み。README.md と docs/verification.md、配布コードは変更していない。外部環境での検証は親の独立レビューへ引き継ぐ。

2026-11-13: v0.1.0（`ee3a00f`）で公開済み。v0.5.0（`6a09684`、PR #8、ADR 0001）で、自動登録・解除・接続検査・設定編集・ローカル鍵削除は廃止された。この課題の機能は現行版にない。
