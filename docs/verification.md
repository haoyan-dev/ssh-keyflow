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
| 鍵生成 | Ed25519、自動名、衝突回避、端末入力、再開 | 未確認 |
| 接続先追加 | アドレス・別名・ユーザー・ポート・OS、鍵の再利用 | 未確認 |
| 手動案内 | 3 OSのSSH有効化・登録・解除のコピペ内容 | 未確認 |
| 自動登録 | 標準登録先確認、冪等、他の行保持、Windows共有範囲 | 未確認 |
| 接続の検証 | 指定鍵のみ、ホスト鍵確認、変更時停止 | 未確認 |
| 進捗と復旧 | 登録と接続結果の分離、通信断後の照合、鍵保持 | 未確認 |
| SSH別名 | 衝突、Includeの差分・承認・バックアップ、拒否時案内 | 未確認 |
| 登録解除 | 別認証の新規接続、最終承認、対象鍵だけ除去、冪等 | 未確認 |
| ローカル削除 | 登録残存・不一致・到達不能で停止、削除前確認 | 未確認 |
| 秘密値保護 | JSON・出力・コマンド引数にパスワードを残さない | 未確認 |
| インストーラー | 最新・指定版、OS/CPU、更新・削除、PATH所有権 | 未確認 |
| 配布 | 6対象のクロスビルド、CIとリリース定義 | 未確認 |
| 実コマンド | 一時ホームでWindowsのOpenSSH鍵取り込み・生成確認 | Windows標準OpenSSHで検証用鍵生成、公開鍵のみの取り込み、PTYでの鍵ペア照合・取り込みと一覧表示が成功。本体からの鍵生成は未確認 |
| レビュー | main初期コミットを基点にStandardsとSpecの独立レビュー | 未確認 |

## 外部環境を必要とする確認

ticket01の統合コミットは `3fba832`。統合後の `go test ./...` と `go vet ./...` は成功。後続チケットによる変更は最終検証で再確認する。

- 実際のGitHub Releasesからの取得：公開先とリリースの設定後に確認する。
- Linux/macOS/Windowsの実SSHサーバーへの登録・接続・解除：ユーザーが検証対象を指定した後に確認する。
- 各OS・CPUでのネイティブ実行：クロスビルドの成功とは区別する。
