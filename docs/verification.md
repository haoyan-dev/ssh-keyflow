# 検証記録

本書は SSH KeyFlow の仕様に対する完了確認に使う。成功した検証だけを記録し、外部環境が必要な確認は区別する。

## 検証環境

- 開発環境：Windows amd64、Go 1.26.2、PowerShell 7、Git for Windows。
- 実際のユーザーのSSH設定・鍵や、未指定の接続先はテストで変更しない。
- CLIのテストでは一時ホームと外部プロセス境界を使う。インストーラーはローカル配布物と一時配置先を使う。

## 完了確認

| 対象 | 必要な証拠 | 結果 |
| --- | --- | --- |
| 起動と依存確認 | 不足コマンド、導入案内、再チェック、書込確認のCLIテスト | ticket01のStartup、MissingDependency、UnwritableStorageテスト成功 |
| JSONの整合性 | 破損、未対応version、固定ID、参照のCLIテスト | ticket01のInvalidState、StateRelationships、StateValidatesIDs、PublicOnlyImportテスト成功 |
| 永続化 | バックアップ、排他、書込失敗で正常状態保持 | ticket01のSuccessfulUpdate、LockedOrUnwritableUpdate、MutationLockCoversExternalValidationテスト成功 |
| 管理対象外の発見 | 鍵と再帰Include、循環、読取不可、Match exec非実行 | ticket01のListDiscoversRecursiveIncludesWithoutExecutingOrChangingThemテスト成功 |
| 管理対象外の保護 | 元ファイルと管理JSONへの自動取り込みがない | ticket01の一覧・取り込み境界テストで元設定保持と明示承認を確認 |
| 取り込み | 鍵ペア照合、公開鍵のみ、元パス保持、承認 | ticket01のImportValidatesPair、MismatchedPair、PublicOnlyImport、ImportRequiresExplicitConfirmationテスト成功。Windows実鍵でも確認 |
| 鍵生成 | Ed25519、自動名、衝突回避、端末入力、再開 | ticket02の生成・中断復旧・衝突保護のCLIテスト成功。Windows標準OpenSSHと実PTYによる本体からの生成・照合・一覧も成功 |
| 接続先追加 | アドレス・別名・ユーザー・ポート・OS、鍵の再利用 | ticket02のTailscale名・IPv6・Unicode・ポート・OS選択・既存鍵再利用・再開のCLIテスト成功 |
| 手動案内 | 3 OSのSSH有効化・登録・解除のコピペ内容 | ticket02/04のOS別案内と同じ生成スクリプトをCLI・POSIX・Windows標準NTFS fixtureで検証。解除用POSIXスクリプトはPerlを使用する |
| 自動登録 | 標準登録先確認、冪等、他の行保持、Windows共有範囲 | ticket03のCLI境界・POSIX/Windows標準NTFSスクリプトfixture成功。Windows管理者共有対象の構文・エラー処理を検証。実SSHサーバーは未確認 |
| 接続の検証 | 指定鍵のみ、ホスト鍵確認、変更時停止 | ticket03の引数分離・変更済みホスト鍵で停止するCLIテストと、実OpenSSHの空白入りパス解析が成功。実サーバーでの認証とホスト鍵ハンドシェイクは未確認 |
| 進捗と復旧 | 登録と接続結果の分離、通信断後の照合、鍵保持 | ticket02/03の中断復旧・結果不明の読取照合・追記履歴・独立した接続結果のCLIテスト成功。解除・削除の復旧はticket04で確認する |
| SSH別名 | 衝突、Includeの差分・承認・バックアップ、拒否時案内 | ticket03の既存設定保持、所有権・ダイジェスト、不一致停止、Include承認・バックアップ・中断復旧テスト成功 |
| 登録解除 | 別認証の新規接続、最終承認、対象鍵だけ除去、冪等 | ticket04の別指紋・別認証制約、承認、結果不明の読取照合、到達不能時の保持、OS別実スクリプトfixtureが成功。実SSH認証は未確認 |
| ローカル削除 | 登録残存・不一致・到達不能で停止、削除前確認 | ticket04の残存登録・不一致停止、削除意図と中断復旧、再出現パス拒否、固定ID・履歴・設定保持のCLIテスト成功 |
| 秘密値保護 | JSON・出力・コマンド引数にパスワードを残さない | 認証値はOpenSSHの元端末入力へ委譲。CLIテストで端末入力保持と認証方式の引数制約を検証。実PTYで鍵生成・鍵ペア取込を確認 |
| インストーラー | 最新・指定版、OS/CPU、更新・削除、PATH所有権 | ticket05のローカルHTTP配布物・一時ホームによる24テスト成功。Windows実バイナリのインストール・更新・削除・SSHデータ保持も成功 |
| 配布 | 6対象のクロスビルド、CIとリリース定義 | ticket05で6対象のビルド・9配布物の作成と8件のチェックサムを確認。CI・リリース定義あり。本体完成後に再ビルドする |
| 実コマンド | 一時ホームでWindowsのOpenSSH鍵取り込み・生成確認 | Windows標準OpenSSHで検証用鍵生成、公開鍵のみの取り込み、PTYでの鍵ペア照合・取り込みと一覧表示、本体からの鍵生成が成功 |
| レビュー | main初期コミットを基点にStandardsとSpecの独立レビュー | 未確認 |

## 外部環境を必要とする確認

ticket01の統合コミットは `3fba832`。統合後の `go test ./...` と `go vet ./...` は成功。後続チケットによる変更は最終検証で再確認する。

ticket02の統合コミットは `b29e530`。統合後の `go test ./...` と `go vet ./...` は成功。追加14件のCLIテスト、生成POSIX登録スクリプトのGit Bash実行、Windows標準アカウントのNTFS登録・冪等性fixtureが成功。SSH有効化と登録の手動案内は3 OS分を検証済み。解除案内はticket04で確認する。Windows共有管理者ファイルの昇格実行、macOS実機、実SSHサーバーへの操作は未確認。

ticket03の統合コミットは `14ea6d4`。統合後の `go test ./... -count=1`（app: 40.437秒）と `go vet ./...` が成功。自動登録のPOSIXスクリプトfixture、Windows標準NTFS登録・読取照合、Windows管理者用PowerShell構文とエラー処理、実OpenSSHの `ssh -G` による引数解析が成功。CLI模擬応答による認証制約・ホスト鍵変更時停止は、実サーバーへの接続成功とは区別する。

ticket04の統合コミットは `cda1cb8`。統合後の `go test ./... -count=1`（app: 34.474秒、ssh: 10.246秒）と `go vet ./...` が成功。Linux/macOS向け解除スクリプトはGit Bashの隔離fixtureで実行し、Windows標準アカウントはPowerShell/NTFSで読取・解除・再実行・元bytes/ACL SDDL保持を確認。別認証・共有管理者承認・不明結果・ローカル削除の復旧をCLI境界で検証。

2026-11-04: 親タスクによる配布スイート再実行は24件成功（69.134秒）。Git Bash fixtureがサンドボックスでアクセス拒否となったため、承認済みの通常権限環境で再実行した。

ticket05の統合コミットは `331c2fe`。`python scripts/test_installers.py -v` は24件成功（68.079秒）。`python scripts/package_release.py --version v0.0.0-ticket05 --output .tmp/distribution-release-v0.0.0-ticket05` は6対象成功。実Windows amd64配布物の `--version` は `SSH KeyFlow v0.0.0-ticket05`。公開先とGitHub Actionsでの実行は未設定・未確認。

- 実際のGitHub Releasesからの取得：公開先とリリースの設定後に確認する。
- Linux/macOS/Windowsの実SSHサーバーへの登録・接続・解除：ユーザーが検証対象を指定した後に確認する。
- 各OS・CPUでのネイティブ実行：クロスビルドの成功とは区別する。
