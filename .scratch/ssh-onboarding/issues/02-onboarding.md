# 02: 鍵生成・接続先追加・手動登録案内・中断後の再開

Status: ready-for-human
Resolution: completed
Blocked by: 01

## What to build

鍵生成・接続先追加・手動登録案内・中断後の再開を対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [x] 安全な自動名のEd25519鍵を端末対話で生成し、既存鍵を上書きしない
- [x] 接続先ごとの新規鍵または管理対象鍵の再利用を対話で選べる
- [x] OS別のSSH有効化と登録用コピペ手順を生成し、未確認OSは選択する
- [x] 秘密値を記録せず、意図と進捗を永続化し中断後に同じ鍵を再利用できる
- [x] CLI境界の生成・再開・手動経路をTDDで検証する

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。

2026-11-04: 実装完了。`app.Run` 境界の偽Runner・一時ホームで red/green を確認。`TestCreateEd25519PersistsIntentAndListsCreatedProvenance`、`TestInterruptedCreationRecoversSamePathWithoutOverwriting`、`TestNewDestinationResumesSameKeyAfterInterruption` が保存済み意図・操作全体ロック・同一ID/パスの復旧を検証する。ファイルなし・秘密鍵のみ・ペアあり・公開鍵のみを区別し、半生成ファイルの上書き・自動削除をしない。`TestCreationDoesNotClaimFilesAppearingDuringConfirmation` は確認中に現れた無関係ファイルを保護する。

既存鍵の再利用と imported 来歴、Unicode、IPv6/Tailscale・ポート、未確認OSからの選択、登録履歴 unknown と接続 unverified の分離を検証。`TestDestinationRechecksImportedKeyAfterConfirmation` と `TestCreationAndOnboardingHistoryRejectsInconsistentState` は不一致・不正履歴で停止する。`TestAbandonedOnboardingCannotClaimStandaloneCreation` は中止したオンボーディングと独立生成を分離し、紐付けには明示的ジョブIDを使用する。

`TestDestinationAliasCannotShadowRecursiveUnmanagedHost` は再帰Includeのリテラル別名衝突を拒否し、`TestWildcardHostDefaultsPreserveManualOnboarding` は通常の `Host *` と `backup-*` について警告を表示して手動経路を維持する。既存configは変更せず、Match execも実行しない。

`TestManualGuidanceIsCopyableForEveryDestinationOS` で Linux/macOS/Windows標準・管理者のSSH有効化と登録案内を確認。生成LinuxスクリプトはGit Bash一時ホームで実行し、コメント・他鍵・改行を保持、options付き同一鍵を重複追記しない。Windows PowerShell 5.1で両種スクリプトの構文を検証し、標準アカウントはNTFS一時ディレクトリでACL作成・UTF-8 BOM/CRLF保持・冪等実行が成功。管理者共有対象は対応する昇格アカウントでの実行が未検証。macOS実機および実SSHサーバーへの操作は未実施。全操作は一時ホーム/偽プロセスまたは明示したスクリプトfixtureのみ。

親タスクによるネイティブWindows検証（2026-11-04）: 実PTY・システムOpenSSH、隔離ルート `.tmp/native-created-smoke` で `5/c/Native created/yes`、空パスフレーズ2回、一覧・終了が成功（exit 0）。自動名 `id_ed25519_Native_created_20261104T091557Z`、fingerprint `SHA256:mZiikTwMW/AGWXywutspZcKJiv41OnGnRw7bMZjnZQE`。本体がペアを導出/照合し `origin=created connection=unverified` を表示。実ユーザーのSSHデータは未使用。完了済みSession2860の再ポーリングはしない。

検証: gofmt、`go test ./...`、`go vet ./...` が成功。Windows sandboxの一時fixtureアクセス拒否後に昇格して実行、Goキャッシュはルート `.tmp/go-cache` を使用。integration `implement/ssh-keyflow` の最新 `b6099cf`（指定の `331c2fe` を含む）を取り込み済み。配布ファイルや他チケットへの独自編集、配布スイート再実行はなし。

契約: version 1にoptional `creations`/`onboardings`、created来歴、Windows accountを追加。`internal/ssh.StandardRegistration` と既存登録ペアを03で再利用できる。Interactive SSHのStdinは端末を維持し、将来のスクリプトは安全に引用したコマンドpayloadで渡す。詳細は `docs/application-contract.md`。

2026-11-13: v0.1.0（`ee3a00f`）で公開済み。v0.5.0（`6a09684`、PR #8、ADR 0001）で、自動登録・解除・接続検査・設定編集・ローカル鍵削除は廃止された。鍵生成とローカル作成の再開は現行版に残る。
