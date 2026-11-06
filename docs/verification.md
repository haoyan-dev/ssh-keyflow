# 検証記録

本書は SSH KeyFlow の仕様に対する完了確認に使う。成功した検証だけを記録し、外部環境が必要な確認は区別する。

## v0.2.0の検証（2026-11-06）

仕様は `.scratch/v0.2.0/spec.md`。以下のv0.1.0記録は過去の証拠として保持し、現行のPortable配布・CLI契約とは区別する。

- 共通処理：構造化要求とSnapshot、既存データ・排他・意図保存・照合・承認・別認証・所有設定保護の公開境界テスト成功。
- CLI：Cobraのヘルプ・版表示、非端末時の分岐、JSONと診断の分離、終了コード、入力・対象解決、直接操作を検証。Windows amd64実PTYと隔離USERPROFILEでネイティブEd25519生成が終了0、JSONで生成完了・接続未確認を確認。削除を拒否すると終了1で鍵ペアと状態を保持し、削除墓標なし。
- TUI：Bubble Tea/Bubbles/Lip Glossの公開Program境界で読取起動、一覧・詳細・発見・フォーム取消・狭い画面を検証。Windows実PTYで明示/引数なし起動、実OpenSSH鍵生成、パスフレーズ入力中Ctrl-C、同じID/パスで再開、取込失敗後の状態ハッシュ保持、80×24→32×12→50×18のリサイズ、終了0を確認。
- Portable：Python生成物テスト5件成功。6アーカイブ+SHA256SUMSの7件、実行ファイル+同梱文書、Unix0755/文書0644、チェックサム、失敗時の未公開・既存出力保持を確認。Windows amd64実バイナリの版注入・展開起動・ヘルプ・JSON一覧が成功。実行ファイルの移動・差し替え・削除後も一時ホームの鍵・状態・SSH設定・プロファイル・旧所有記録をバイト比較で保持。
- ビルド・全体検証：`go test -buildvcs=false ./...` 全件成功（app 65.426秒、ssh 25.301秒）、`go vet -buildvcs=false ./...` 成功。`python scripts/package_release.py --version v0.2.0 --output .tmp/release-ticket04-final` によるCGO無効6対象の実クロスビルド成功。実アーカイブのメンバー・同梱文書・権限・SHA256全6件を検算し、Windows amd64展開後の版表示は `SSH KeyFlow v0.2.0`。この配布物はローカルのみで未公開。
- ライセンス：6対象の実行用依存を照合し、固定モジュール版・Go・Unicodeの本文を同梱。詳細は [third-party-notices.md](third-party-notices.md)。
- レビュー修正：指定鍵の接続確認拒否・成功マーカー欠落をエラーとして返し、登録済み記録と接続失敗履歴を保持する公開操作テストに成功。作成・接続先追加・鍵作成途中の再開では、保存済み対象を承認前に表示し、拒否時に状態を保持することを確認。TUIはOS未入力を拒否し、明示指定後にレビューへ進む。通常のCLI一覧とJSONの管理対象外表示、元ファイル保持、SSH非実行を確認した。各修正はテスト失敗を先に確認してから実装した。修正後のGo全体テスト（app 71.624秒、ssh 33.872秒）、Portableテスト5件、vetが成功した。最終6対象アーカイブは統合後のコードから別途再生成する。

実ユーザーのSSH領域、永続PATH、プロファイル、接続先は変更していない。Windows amd64以外のネイティブ端末・実バイナリ実行、実SSHサーバーへの認証・登録・解除、Windows管理者共有登録先の昇格動作は未確認。クロスビルド成功はこれらの実機確認を代替しない。CI定義は3 OSのテストを行うが、この作業でホストCIを実行した証拠ではない。

## v0.1.0の検証環境

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
| 進捗と復旧 | 登録と接続結果の分離、通信断後の照合、鍵保持 | ticket02/03/04の中断復旧・結果不明の読取照合・追記履歴・独立した接続結果・削除チェックポイントのCLIテスト成功 |
| SSH別名 | 衝突、Includeの差分・承認・バックアップ、拒否時案内 | ticket03の既存設定保持、所有権・ダイジェスト、不一致停止、Include承認・バックアップ・中断復旧テスト成功 |
| 登録解除 | 別認証の新規接続、最終承認、対象鍵だけ除去、冪等 | ticket04の別指紋・別認証制約、承認、結果不明の読取照合、到達不能時の保持、OS別実スクリプトfixtureが成功。実SSH認証は未確認 |
| ローカル削除 | 登録残存・不一致・到達不能で停止、削除前確認 | ticket04の残存登録・不一致停止、削除意図と中断復旧、再出現パス拒否、固定ID・履歴・設定保持のCLIテスト成功 |
| 秘密値保護 | JSON・出力・コマンド引数にパスワードを残さない | 認証値はOpenSSHの元端末入力へ委譲。CLIテストで端末入力保持と認証方式の引数制約を検証。実PTYで鍵生成・鍵ペア取込を確認 |
| インストーラー | 最新・指定版、OS/CPU、更新・削除、PATH所有権 | ticket05のローカルHTTP配布物・一時ホームによる24テスト成功。Windows実バイナリのインストール・更新・削除・SSHデータ保持も成功 |
| 配布 | 6対象のクロスビルド、CIとリリース定義 | 最終本体7703cebで6対象ビルド、9配布物、全8件のSHA256、単一バイナリ構成とUnix0755を確認。Windows amd64実行で版表示成功。CI・リリース定義あり |
| 実コマンド | 一時ホームでWindowsのOpenSSH鍵取り込み・生成確認 | Windows標準OpenSSHで検証用鍵生成、公開鍵のみの取り込み、PTYでの鍵ペア照合・取り込みと一覧表示、本体からの鍵生成が成功 |
| レビュー | main初期コミットを基点にStandardsとSpecの独立レビュー | 初回Standards3件・Spec2件を7703cebで修正。独立した再レビューで全件解消、新規不備0件。詳細はfinal-review.md |

## 要件の照合

`spec.md` の34ストーリーだけでなく、仕様の一部である `requirements.md` と `plan.md` を照合対象にした。

| ストーリー | requirements の対象 | 実装と検証の対応 |
| --- | --- | --- |
| 1–3 | 環境チェック、保存先、対話メニュー | ticket01の起動・依存不足・JSON拒否・書込確認と、全6操作のCLI境界テスト |
| 4–7 | 管理対象外、再帰Include、明示取り込み、公開鍵のみ | ticket01の読取専用探索・循環・Match非実行・ペア照合・公開鍵のみの取込 |
| 8–12 | Ed25519、自動命名、接続先、鍵再利用、Tailscale | ticket02の意図保存・衝突拒否・同じパスの再開・接続先入力テスト |
| 13–16 | 自動登録、重複回避、OS選択、手動案内 | ticket02/03の共通登録生成・OS別fixture。7703cebで新規追加から自動登録・検証・設定への継続も確認 |
| 17–21 | ホスト鍵、指定鍵検証、独立進捗、別名とInclude | ticket03の認証引数・読取照合・登録履歴・所有設定・差分承認・バックアップ・復旧 |
| 22–28 | 別認証、解除、ローカル削除、結果不明の復旧 | ticket04の新規接続制約・別指紋・承認・残存登録による削除拒否・墓標と復旧。7703cebで永続バックアップ・原子的置換と部分書込・中断・対象変更のfixtureも確認 |
| 29–30 | 認証値保護、正常JSON保存、排他 | 元端末を渡す外部プロセス境界と全操作を覆う排他、書込拒否・正常バックアップ・不正状態拒否 |
| 31–34 | 6対象、インストール、更新、アンインストール | ticket05の24件の配布fixture・6対象実ビルド・所有PATHだけの削除・SSHデータ保持 |

Tailscaleは通常のSSH経路として扱い、端末探索・Tailscale SSHのポリシー変更・全ディスク探索は含めない。取り込んだ既存鍵は方式と元パスを保持し、管理名と別名とは別の固定IDで履歴を参照する。記録だけを外す操作、自動更新、v0.1.0ではエージェント用の非対話JSON APIは追加していない。v0.2.0では直接CLIと読取一覧JSONを追加した。

## 統合時の検証と外部環境の制約

2026-11-06：最終本体 `7703ceb` を `implement/ssh-keyflow` にfast-forward統合。ルートで `go test ./... -count=1` が成功（app:46.497秒、ssh:24.096秒）、`go vet ./...` も成功。サンドボックスのfixtureアクセス拒否後、承認された通常権限で再実行した。配布スクリプトは変更しておらず、既存24件の成功結果を維持する。

同じコミットから `python scripts/package_release.py --version v0.0.0-final --output <root>/.tmp/release-final-7703ceb` が6対象成功。9配布物・manifest全8件のSHA256・各アーカイブの単一実行ファイル・Unix0755を検査。Windows amd64版の実行結果は `SSH KeyFlow v0.0.0-final`。ローカル検証用の版であり公開リリースではない。独立再レビューの結果と追加検証の余地は [final-review.md](final-review.md) に記録した。

ticket01の統合コミットは `3fba832`。統合後の `go test ./...` と `go vet ./...` は成功。後続チケットによる変更は最終検証で再確認する。

ticket02の統合コミットは `b29e530`。統合後の `go test ./...` と `go vet ./...` は成功。追加14件のCLIテスト、生成POSIX登録スクリプトのGit Bash実行、Windows標準アカウントのNTFS登録・冪等性fixtureが成功。SSH有効化と登録の手動案内は3 OS分を検証済み。解除案内はticket04で検証済み。Windows共有管理者ファイルの昇格実行、macOS実機、実SSHサーバーへの操作は未確認。

ticket03の統合コミットは `14ea6d4`。統合後の `go test ./... -count=1`（app: 40.437秒）と `go vet ./...` が成功。自動登録のPOSIXスクリプトfixture、Windows標準NTFS登録・読取照合、Windows管理者用PowerShell構文とエラー処理、実OpenSSHの `ssh -G` による引数解析が成功。CLI模擬応答による認証制約・ホスト鍵変更時停止は、実サーバーへの接続成功とは区別する。

ticket04の統合コミットは `cda1cb8`。統合後の `go test ./... -count=1`（app: 34.474秒、ssh: 10.246秒）と `go vet ./...` が成功。Linux/macOS向け解除スクリプトはGit Bashの隔離fixtureで実行し、Windows標準アカウントはPowerShell/NTFSで読取・解除・再実行・元bytes/ACL SDDL保持を確認。別認証・共有管理者承認・不明結果・ローカル削除の復旧をCLI境界で検証。

2026-11-04: 親タスクによる配布スイート再実行は24件成功（69.134秒）。Git Bash fixtureがサンドボックスでアクセス拒否となったため、承認済みの通常権限環境で再実行した。統合済み本体f17cca0の6対象再ビルドは `python scripts/package_release.py --version v0.0.0-validation --output .tmp/release-validation-complete` で成功。9配布物とmanifestの全8件のSHA256を検算し、Windows amd64版の実行結果は `SSH KeyFlow v0.0.0-validation`。配布物はローカル検証用で未公開。

ticket05の統合コミットは `331c2fe`。`python scripts/test_installers.py -v` は24件成功（68.079秒）。`python scripts/package_release.py --version v0.0.0-ticket05 --output .tmp/distribution-release-v0.0.0-ticket05` は6対象成功。実Windows amd64配布物の `--version` は `SSH KeyFlow v0.0.0-ticket05`。公開先とGitHub Actionsでの実行は未設定・未確認。

## 初回公開リリースの検証（2026-11-06）

正式版 [v0.1.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.1.0) を公開した。タグは `ee3a00f4f86171ab3860fbc5f09b0fdf8462040f` を指し、最新正式リリースとして取得できることを確認した。

- [mainのCI](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42153679151) と [リリースワークフロー](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42154035210) が成功。Linux・macOS・WindowsのGoテスト、インストーラー25項目（OS固有のskipを含む）、6対象のクロスビルドを実行した。
- 公開された全9配布物をGitHub Releasesから取得し、manifest全8件のSHA256、各アーカイブの単一実行ファイル構成、Unix実行権限0755、`install.sh`のLF改行を検証した。
- 公開された`install.ps1`をWindows PowerShell 5.1で実行し、一時配置先への最新正式版の導入、指定版による更新、削除が成功。実バイナリの表示は`SSH KeyFlow v0.1.0`。一時プロファイルのPATH追記と、削除後の元bytes完全一致も確認した。実際のユーザーのSSH設定・鍵・PowerShellプロファイルは変更していない。
- CIで検出したGNU ddの末尾コピーによる作業ファイル切り詰めは`conv=notrunc`で修正した。末尾改行のない元プロファイルとUnicode・引用符を含むパスの回帰テストが成功。macOSの実パス・コマンド配置とWindowsのアカウント・ACLに合わせて隔離fixtureを補正し、管理対象外パスやACLの保護は保持した。

## 未検証の範囲

- Linux/macOS/Windowsの実SSHサーバーへの登録・接続・解除：ユーザーが検証対象を指定した後に確認する。
- 各OS・CPUでのネイティブ実行：クロスビルドの成功とは区別する。
