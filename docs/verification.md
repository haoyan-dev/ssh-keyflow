# 検証記録

本書は SSH KeyFlow の仕様に対する完了確認に使う。成功した検証だけを記録し、外部環境が必要な確認は区別する。

## v0.2.0の検証（2026-11-06）

仕様は `.scratch/v0.2.0/spec.md`。以下のv0.1.0記録は過去の証拠として保持し、現行のPortable配布・CLI契約とは区別する。

- 最終完了記録：[v0.2.0-review.md](v0.2.0-review.md)。レビュー修正後の `158bd2a` から6対象を再ビルドし、`.tmp/release-v0.2.0-final-158bd2a` に6アーカイブとSHA256SUMSを生成した。許可メンバー・同梱文書・Unix権限・全チェックサムを検算し、最終Windows ZIPの展開起動と読取JSON、保存先未作成を確認した。
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

## PR #1 follow-up verification (2026-11-07)

Source commit `b045f7c` (parent `97ec7f9`) fixes the three subsequent PR review findings. Verification ran on Windows amd64 with Go 1.26.2, Python 3.12, isolated temporary homes and external-process fixtures; no real user SSH files or servers were used.

- Manual/default structured unregister: a public `app.Run` regression failed for missing/refused/approved callbacks before the fix. Afterward it checks exact target visibility before consent, unchanged state bytes on refusal/missing callback, and approved unknown unregister evidence plus manual guidance.
- Terminal stdin with redirected stdout: `go test` gives its child nonterminal stdin, so the terminal-dependent regression skips there. Built `internal/cli` with `go test -c -buildvcs=false -o .tmp/cli-regression.test.exe ./internal/cli`, then ran the executable directly under the native Windows PTY with `-test.run=TestTerminalInputWithRedirectedOutputShowsHelp -test.v`. Before the fix it failed with exit 2 and empty stdout; afterward it passed with help/status 0. Explicit `tui` still rejects redirected stdout with status 2.
- Public Bubble Tea event tests fail before the layout fix at 32x12 (13 rendered lines), then pass for empty/populated lists, full help with page scrolling, and resize in both directions. Existing focused-form and review/selection tests also pass. The helper explicitly delivers WindowSizeMsg before interaction, avoiding startup event scheduling as a source of false evidence.
- `go test -buildvcs=false ./...`, `go vet -buildvcs=false ./...` and `python scripts/test_portable.py -v` (5 tests) pass for the corrected source. The initial full run took app 71.866s and ssh 30.451s; the final layout adjustment was also covered by the targeted TUI suite.

Six-target final release generation remains an integration check after these fixes are merged. The native PTY regression deliberately skips in ordinary nonterminal CI; its direct Windows execution above supplies the missing descriptor condition. Live remote authentication and native Linux/macOS terminal checks remain outside this follow-up.

### Integration completion (2026-11-07)

- Integrated cleanly by fast-forward at `9177555`; executable source/tests are `b045f7c`, with subsequent changes confined to documentation.
- Independent Standards and Spec re-reviews found no remaining findings. Focused app tests independently covered missing/refused/approved manual consent and unknown-result reconciliation. Four TUI regressions and additional public-entry 32x12 probes covered list/help, form validation, review scrolling and back/toggle controls. The compiled CLI regression passed directly in a Windows PTY.
- `python scripts/package_release.py --version v0.2.0 --output .tmp/release-v0.2.0-final-b045f7c` rebuilt all six actual CGO-disabled target binaries from the integrated code. Inspection passed for seven assets, all six SHA256 values, exact archive members, notices bytes and Unix executable/document permissions.
- The extracted final Windows amd64 binary reported `SSH KeyFlow v0.2.0`; `keys list --json` succeeded with valid JSON, no diagnostic output and no `.ssh` directory created in the isolated home.

Earlier hosted CI at `97ec7f9` passed the Ubuntu, macOS and Windows jobs plus cross-builds. That result predates these code fixes; final-head hosted CI is tracked separately on PR #1. Hosted native tests and cross-builds do not establish interactive Linux/macOS TUI or live SSH-server behavior. No user SSH data or remote server was accessed, and no release was published.

## Source-hosted installers (2026-11-07)

Tested uncommitted installer changes over `3e71b0e90bf4e364a821fc741d288cbc8bc8515e`
on Windows amd64, Python 3.12.13, Windows PowerShell 5.1.26100.9711,
PowerShell 7.6.9 and Git Bash. The scripts consume existing two-member Portable
archives; application code and release packaging are unchanged.

The tested script SHA256 values are:

| File | SHA256 |
| --- | --- |
| `install.sh` | `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f` |
| `install.ps1` | `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f` |
| `scripts/test_installers.py` | `ee4c8c38ac9e5f99cb09dccea4bb17dd96bb85b90678f702831625ef5770bd57` |
| `scripts/test_installer_profiles.py` | `0b08067ce8da9956b186f28a220d64ff4d7656c42fd7afc293059d74a20a0b9c` |

- The initial installer suite passed all 50 tests in 203.911s. After extending
  Windows uninstall rollback, the 26 Windows cases passed again in 84.720s.
  After the shell byte-offset fix, all 24 shell cases passed again in 132.616s.
  Fixtures cover stable/version/architecture selection, both pipeline entry
  points, checksum and archive rejection, ownership, legacy receipts, update,
  removal and failed writes.
- The focused profile suite initially passed 29 of 30 cases, including all
  16 Windows cases. Its remaining Git Bash case exposed text-mode CRLF
  normalization in the byte-offset calculation. Enabling awk binary input
  fixed the non-UTF-8/CRLF/no-final-newline regression; its focused rerun passed.
  These tests also cover complete rollback after profile replacement, directory
  and read-only profiles, locked files, byte/mode/Windows ACL preservation, and
  IEX caller variables including a read-only version variable.
  The final shell profile suite passed all 14 cases in 59.290s; together with
  the 16 unchanged Windows cases, all 30 profile cases are verified.
- `python3.12.exe scripts/test_portable.py -v` passed all five tests in
  22.917s, including a native executable build/run.
- Final scripts installed the actual published v0.2.0 Windows amd64 archive
  using Windows PowerShell 5.1, reported `SSH KeyFlow v0.2.0`, updated through
  latest stable resolution with PowerShell 7 and explicit PATH consent, and
  uninstalled through Windows PowerShell 5.1. Only a temporary installation
  directory and profile were used. Uninstall restored the exact original profile
  bytes and left the installation directory empty; smoke fixtures were removed.
- Separate read-only Standards and Specification reviews resolved the profile
  atomicity, rollback, caller-scope and noninteractive prompt findings. Shell
  syntax, PowerShell 5.1/7 parsing and `git diff --check` passed. Both reviewers
  also cleared the final shell byte-offset change without additional findings.

At the local implementation checkpoint, the three-OS CI matrix included both
installer suites but had not run this change on hosted CI. Native Linux/macOS
and ARM64 runtime execution had not been checked locally; shell fixtures
substitute platform identity under Git Bash. Tests use isolated homes and
loopback release fixtures and do not modify real user SSH data, profiles,
persistent PATH or remote registrations. Publication was subsequently authorized
and verified as recorded below.

## v0.2.1 published release (2026-11-07)

Release [SSH KeyFlow v0.2.1](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.2.1)
was published on 2026-11-07 at 07:32:13 UTC (16:32:13 JST). Annotated tag
`v0.2.1` (tag object `bd71f741a1d3418cabe43d2a50a03355832637b1`) points to
implementation commit `3e5674430cf7d0a2e031aa1c214a70af5c666982`.
GitHub's latest release API returned `v0.2.1`, with draft and prerelease both
false. The release title and notes include the one-line installation commands,
usage, update behavior and verification limits.

- [Release workflow 42351144039](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42351144039)
  succeeded on the tagged commit: Ubuntu, macOS and Windows Go tests, Portable
  tests, installer/profile tests, six-target crossbuilds, Linux amd64 version
  injection, and the publish job all passed.
- [Main workflow 42351143967](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42351143967)
  also passed on the same implementation commit.
- Each OS ran all five Portable tests. Windows executed all 50 installer and
  30 profile cases with no skips. Ubuntu and macOS each executed 24 shell
  installer and 14 shell profile cases; their 26 Windows installer and
  16 Windows profile cases were correctly skipped.
- Downloaded all seven published assets with `gh release download v0.2.1`.
  Verified the exact six expected archive names, six unique manifest entries,
  every archive SHA256, exactly two regular members, executable/notice names,
  tagged notice bytes and Unix permissions (0755 executable, 0644 notices).
  Notice comparison used the tag's Git blob, avoiding local Windows checkout
  newline conversion. The downloaded manifest SHA256 was
  `95d9dda43ceaacb80097f7a5037e5aec82418e50659e3d6ae0c3ef8c3b149faa`.
- The Windows amd64 ZIP SHA256 was
  `720c2bcc3992374ab69aa8564cda7ddfc61030234f069ae3aaf4ab7878c7d4eb`.
  Its extracted executable reported `SSH KeyFlow v0.2.1`.
- Downloaded `install.sh` and `install.ps1` from the public raw `main` URLs.
  Both matched their tagged Git blobs and the script hashes recorded above.
- The downloaded PowerShell installer installed pinned v0.2.1 through Windows
  PowerShell 5.1, updated through latest-stable resolution with PowerShell 7 and
  explicit PATH consent, and uninstalled through PowerShell 5.1. This used only
  an isolated installation directory and profile under the workspace `.tmp/`.
  Both installed binaries reported `SSH KeyFlow v0.2.1`. Noninteractive install
  preserved the profile, explicit consent added the owned stanza, and removal
  restored the exact original profile bytes and left the install directory
  empty. Verification fixtures were removed after the checks.

Hosted Linux/macOS shell tests provide native installer-fixture evidence.
Execution of the published ARM64 binaries, interactive Linux/macOS TUI behavior,
live SSH-server operations and elevated Windows shared administrator targets
remain unverified. No real user SSH data, profiles or remote registrations were
modified. A documentation-only follow-up records these results on main without
moving the released tag.

## v0.2.2 release verification (2026-11-07)

Annotated tag `v0.2.2` (tag object
`03f7df3d339a180b432fae54367afaf76e23e9ab`) identifies commit
`1a6d7d8829322cf3fd985c46a2100495d4d606d0`, the merge of PR #2.
Changes since v0.2.1 are limited to development policies and verification
records; application and installer source are unchanged.

- Before tagging, [main CI 42353432984](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42353432984)
  passed Go, Portable, installer and profile tests on Ubuntu, macOS and
  Windows, plus six-target crossbuilds and Linux amd64 version injection.
- Local verification used Windows 11 amd64, Go 1.26.2 and Python 3.12.14.
  `go vet -buildvcs=false ./...` passed. Built all six release targets with
  `v0.2.2` injected and verified the seven-asset set, all archive checksums,
  exact two-member archive contents, notice bytes and Unix modes. The
  extracted Windows amd64 binary reported `SSH KeyFlow v0.2.2`.
- Downloaded `install.sh` and `install.ps1` from their public raw `main`
  URLs and compared their bytes with Git blobs at
  `1a6d7d8829322cf3fd985c46a2100495d4d606d0`. SHA256 values were
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`
  and `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`,
  respectively. This check downloaded source only; it did not execute an
  installer or modify user SSH data, profiles or PATH.

Release [SSH KeyFlow v0.2.2](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.2.2)
was published on 2026-11-07 at 08:25:28 UTC (17:25:28 JST). GitHub's latest
release API returned `v0.2.2`, with draft and prerelease both false.

- [Release workflow 42354854403](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42354854403)
  succeeded on the exact tagged commit. Ubuntu, macOS and Windows passed Go,
  Portable, installer and profile tests; six-target crossbuilds, Linux amd64
  version injection and publication also passed.
- The remote annotated tag object and peeled commit matched the identities
  above. Downloaded all seven published assets with `gh release download
  v0.2.2` and confirmed the exact six archive names and `SHA256SUMS`.
- Verified six unique manifest entries, every archive SHA256, exactly two
  regular members per archive, executable/notice names, tagged notice bytes
  and Unix modes (0755 executable, 0644 notices, tar uid/gid zero). Notice
  comparison used the tag's Git blob to avoid checkout newline conversion.
  The downloaded manifest SHA256 was
  `3a9c2c522842d8e70ec786280f69e1c3fc017b67184b266f90ebc73efbf2ad0c`.
- The published Windows amd64 ZIP SHA256 was
  `3b176d44778676cb27cc6bde004a37becc852fc547763f671a09c83ff8701251`.
  Its extracted executable reported `SSH KeyFlow v0.2.2` on Windows 11 amd64.
  The temporary executable was removed after this check.
- Release notes describe the documentation changes, installation/update
  instructions, successful publication checks and verification limits.

Native execution of the published Linux/macOS and ARM64 binaries, live
SSH-server operations, interactive Linux/macOS TUI behavior and elevated
Windows shared administrator targets remain unverified. The v0.2.2 checks
did not exercise a native install/update/removal cycle; hosted installer
tests used isolated fixtures. No real user SSH data, profiles, persistent
PATH or remote registrations were modified. This documentation-only follow-up
records publication without moving the released tag or changing binary assets.

## TUI UX統合実装のローカル検証（2026-11-08 JST）

検証対象はクリーンな `284a00bcf7078042f072804a5ea36da725a4fd11`（`test/06-tui-journeys`）。開始時のintegrationは `58148f57f0ca1ad336ed210be0fabba3dd35f39d`、最新の文書更新 `a7319738c6adf050fd632e111f90dd8069a2a5a8` まで取り込み済み。本体は `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` と同一で、06の `internal/` 変更は公開TUI境界の横断テスト1件だけ。後続の証跡文書更新は実装・テストのソースを変えない。

環境はWindows amd64、Go1.26.2、Windows PowerShell5.1.26100.9711、既存CPython3.12.13。Pythonの実体は `C:\Users\haoyan-ts\AppData\Roaming\uv\python\cpython-3.12.13-windows-x86_64-none\python.exe`。Herdrのローカル専用端末で独立コマンドを実行し、全件exit0を確認した。

| 検証 | コマンド・成功結果 |
| --- | --- |
| Go全体 | `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'`：cmd2.239秒、app46.640秒、cli2.564秒、ssh27.857秒、tui11.024秒。stateはテストなし。既存native fixtureのためプロセス限定ポリシーを指定し、永続設定は変更していない。 |
| 静的確認 | `go vet -buildvcs=false ./...` 成功。 |
| Portable | `python scripts/test_portable.py -v`：5件、5.376秒。 |
| Installer | `python scripts/test_installers.py -v`：50件、226.021秒。 |
| Profile | `python scripts/test_installer_profiles.py -v`：30件、109.056秒。 |
| 本体ビルド | `go build -buildvcs=false -o .tmp/ux06-verification/ssh-keyflow.exe ./cmd/ssh-keyflow` と `--version`：`SSH KeyFlow dev`。 |
| 実配布物 | `python scripts/package_release.py --version v0.2.0 --output .tmp/ux06-verification/package`：CGO無効の6対象クロスビルド、6アーカイブとSHA256SUMSの7配布物。ローカル検証用でタグ・公開なし。 |
| 配布物監査 | 全6件のSHA256、各アーカイブの実行ファイル＋元と同一のライセンス文書のみ、Unix0755/0644、uid/gid0、実行形式とGOOS/GOARCH/CGO/trimpath情報を確認。展開したWindows amd64の `--version` は `SSH KeyFlow v0.2.0`。help・読取JSONも成功し、隔離ホームの `.ssh` は作られなかった。残る5対象の実バイナリ実行は未確認。 |

追加の `TestJourneyUnregisterReconciliationUnlocksDeletionWithoutRevivingOnboarding` は、削除阻止から登録解除の不在照合、再起動後の古い追加作業抑制、別セッションでの保護付きローカル削除、墓標・履歴の保持までを既存fixtureで確認した。既存の手動案内、未知結果、接続確認失敗、別名とIncludeの別承認、stale再読込、Tasks投影のテストも全体実行に含む。実SSHの成功をfake Runnerから推定しない。

coordinatorの最終ネイティブ証跡は同じ本体0be6を使い、手動Registerの未記録／以前検証済みの組と再起動、32x12の未知登録Task、60x18の実中断済み鍵作成Task、Ctrl+C後の最終結果と同一プロセスの承認拒否、20個の長い日本語別名、80x24／32x12の検索・縦スクロール、実31x11→60x18の入力保持・変更操作阻止・復帰が成功した。最終ナビゲーション／リサイズの記録はa731973で、古い02／04の同項目を置き換える。実ssh-keygenのパスフレーズ入力待ちをCtrl+Cで中断し鍵ファイルなしで復帰した証拠は、別の旧ソース6fdc973として保持し、最終ソースでの再実行とは扱わない。

受入条件ごとのテスト名、ネイティブ証跡の正確なSHA、ローカル配布物の全ハッシュ、独立レビューと貼り付け修正の未完了ゲートは [06 coverage](../.scratch/tui-ux/coverage-06.md)、引き継ぎは [handoff-06](../.scratch/tui-ux/handoff-06.md) を参照。現時点で全ACの完了承認はしていない。Linux/macOSネイティブ、実SSHサーバーへの認証・ホスト鍵ハンドシェイク・登録・解除、Windows管理者共有先の昇格実行、ホストCIはこの06作業では未検証。実ユーザーの鍵・SSH設定・プロファイル、実接続先は変更していない。

## TUI UX 最終レビュー後の検証（2026-11-08 JST）

製品・テストの確定SHAは `26269049b747db02988ad00db2b62dcff22119f0`、fresh mergerによる最終統合は `c155e2adf6c0b0963fb68085c9d036286f188539`。後続変更は完了証跡文書のみ。Windows amd64 / Go1.26.2 / PowerShell5.1 / CPython3.12.13 / Herdr0.1.18。プロセス限定ExecutionPolicy Bypassを用い永続設定は変更していない。

- `go test -buildvcs=false ./... -count=1`: PASS（cmd2.028秒、app41.354秒、cli2.343秒、ssh23.251秒、tui10.282秒）。focused TUI9.037秒、`go vet -buildvcs=false ./...`、native dev build / versionもPASS。統合mergeでfocused TUIを再実行しPASS（8.910秒、01:13:11.764–01:13:22.216 JST）。
- `python -X utf8 scripts/package_release.py --version v0.2.0 --output .tmp/ux-review-ac11-verification/package`: 6対象クロスビルドと7配布物PASS。ローカルsmoke版番号でありtag・公開なし。全ハッシュ、各archiveの実行ファイル＋同一notices、Unix mode/uid/gid、CPU/Go metadata、展開Windows amd64のversion/help/read-only JSONを検査。残り5対象は実行していない。全digestは [handoff](../.scratch/tui-ux/handoff-review-fixes.md) に保存。
- Portable5 / Installer50 / Profile30の先行284a00b PASSは変更のないscript契約に引き続き適用。TUI修正に対して全Goと実packageは確定SHAで再検証した。
- 独立した規約・仕様レビューを別々に再確認し、S1/P1–P5をすべて解決。最後のP5は32x12で長いパス末尾を1200msのカーソル点滅後も保持し、実カーソル移動・編集時の欄復帰を確認。native binary SHA256は `8038c3a6537f128d4e69f083d39941f66f3d0562a3db2592bdc7153699342c1f`。
- ba0f69dで実貼り付け、Help/Review/dirty draft、実31x11→60x18復帰、保存済み各種TaskのReview、結果リンクのF5/scroll保持、実Windows ssh-keygen中断と再起動を確認。パスフレーズは入力していない。P5の本体差分はviewport条件だけなので、不変部分のnative証拠はその正確なSHAを保持する。

現在のAC01–12受入状況は [coverage](../.scratch/tui-ux/coverage-06.md)、独立レビューは [final review](../.scratch/tui-ux/final-review.md)、操作順・state hashは [native verification](../.scratch/tui-ux/native-verification.md)。全6件ready-for-human、未チェック受入条件なし。

Linux/macOS/Windows arm64のnative、実SSH認証・ホスト鍵・サーバー登録/解除、Windows管理者共有先の昇格実行、hosted CIは未検証。fake Runnerやcross-buildをその証明とは扱わない。実ユーザーの鍵・設定・プロフィール・接続先、mainは変更せず、push/tag/release/転送なし。

## PR CI fixture correction (2026-11-08 JST)

Public PR: https://github.com/haoyan-dev/ssh-keyflow/pull/4. Initial public head `55f92ae6da8116e974444369ddb3cd21afb1bad0` reproduced macOS TUI failures in push run `42391651145` and PR run `42391960644`: creation/configuration guards rejected the `/var` symlink ancestor in `t.TempDir()` paths before the intended test scenario ran. Ubuntu and crossbuild passed in the push run.

The correction resolves only the handoff/result fixture homes and interrupted-creation fixture to their physical paths with `filepath.EvalSymlinks`, matching the existing app fixture pattern. Production guards, behavior, test assertions and fake external boundaries are unchanged. Windows local `go test -count=1 ./internal/tui` PASS (9.447s), `go vet ./internal/tui` PASS, Herdr t73 exit0. The correction commit and full hosted CI results are tracked in PR #4; previous product/native evidence remains applicable, while the published test source now additionally contains this fixture correction. No release/tag publication.

## v0.3.0 release verification (2026-11-08 JST)

The release candidate is `85be65b0dbd29bdf2add37806b898069581c2815`,
the merge of PR #4 into main. Origin/main and tags were refreshed before
preparation; v0.3.0 was unused locally and remotely. The candidate worktree
was clean during all local checks. Its tree is identical to the final PR
head `be53dee0e48750d493be92b57b2b600367f92c0f`, which passed separate
Standards and Specification reviews with no remaining material findings.

- [Main CI 42392798662](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42392798662)
  passed on this exact candidate: Ubuntu, macOS and Windows Go, Portable,
  installer and profile tests, plus six-target crossbuilds and Linux amd64
  version injection. Both final push and PR CI also passed.
- Before tagging, local Windows 11 amd64 (10.0.26300), Go 1.26.2,
  CPython 3.12.13 and Windows PowerShell 5.1 ran from an isolated release
  worktree. Uncached `go test -buildvcs=false ./... -count=1` passed
  (cmd 1.924s, app 42.475s, cli 2.271s, ssh 23.027s, tui 10.021s);
  `go vet -buildvcs=false ./...` passed. Process-only ExecutionPolicy
  Bypass supported native test fixtures without changing persistent policy.
- `python -X utf8 scripts/package_release.py --version v0.3.0` built all
  six targets from the clean candidate with the intended release version.
  Local artifact audit verified exactly seven assets, six unique manifest
  rows, all SHA256 values, exactly two regular members per archive, notice
  bytes, Unix 0755/0644 and uid/gid zero, executable format/CPU, and
  GOOS/GOARCH/CGO-disabled/trimpath build metadata. Windows amd64 reported
  `SSH KeyFlow v0.3.0`; help and all three read-only JSON lists passed
  under a temporary home without creating `.ssh`.
  The local manifest SHA256 was
  `87c86f75bfa93d7b04c3a16b5eeeed96fa25ef1fd0be094c1923c322f0c2fad8`;
  local Windows amd64 ZIP SHA256 was
  `7bcb34fa55c6803b79a197eb3f90020c03202cc6c6b8b8d3d9a4211da9b8b035`.
  Local archives use Windows checkout notice bytes; published workflow
  archives are audited separately against the candidate's Git blob.

Release notes cover the TUI changes since v0.2.2, installation/update,
continued version-1 state compatibility and verification limits. Native
interactive Windows TUI evidence remains in the preceding implementation
sections. Linux/macOS interactive TUI, Windows arm64 execution, live SSH
authentication/host-key/registration/revocation and elevated Windows shared
administrator targets remain unverified. Crossbuilds and fixture tests do
not establish those behaviors. Verification uses isolated data; real user
SSH data, persistent PATH, profiles and remote registrations are preserved.

Release [SSH KeyFlow v0.3.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.3.0)
was published at 2026-11-07T17:16:18Z. Annotated tag object
`3657d9b6e56d0e1ba07089f5c84254bdca6ce658` and its remote peeled commit
both match the intended candidate. GitHub's latest API returned v0.3.0,
with draft and prerelease both false.

- [Release workflow 42394936168](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42394936168)
  passed on the exact tagged commit: three-OS Go, Portable, installer and
  profile tests, six-target crossbuild, Linux amd64 version smoke, and
  the publish job all succeeded.
- Downloaded all seven published assets and repeated the archive audit
  against the candidate's Git-blob notices. Every checksum, exact member
  set, Unix mode/uid/gid, executable format/CPU and Go build setting passed.
  Published manifest SHA256:
  `5068b8e389a8ecfde317d9d64d98f4ea8d6091f8fdefb2037c1f36ee679050be`.
  Published Windows amd64 ZIP SHA256:
  `bbcabb4a9d9e0093542efe8862d87e723ce3971abf8834dbfdf923593d2f2295`.
- Extracted published Windows amd64 reported `SSH KeyFlow v0.3.0`;
  help and three read-only JSON lists passed under a temporary home,
  without creating `.ssh`. The other five published binaries were not
  executed locally.
- Downloaded the two source-hosted installers from raw main before
  publication and compared them byte-for-byte with main commit
  `85be65b0dbd29bdf2add37806b898069581c2815`.
  `install.sh` SHA256:
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`;
  `install.ps1` SHA256:
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`.
  This was source comparison only. No native install/update/removal cycle
  was repeated for this release; hosted installer tests used isolated
  fixtures. The evidence follow-up changes documentation only.
- Separate read-only Standards and Specification reviews found no
  material preparation issues. Release notes include the successful
  publication checks and the remaining native/live-server limits above.

This documentation follow-up records publication without moving the tag or
changing published binary assets.

## Entry detail redesign (2026-11-08 JST)

Final product/test source:
`73fb603a2760207845bac607a8d2dd7e8b86423e` on local branch
`feat/entry-detail-ux`, based on refreshed main
`76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.
The completion follow-up changes evidence and ticket status only.

- Windows 11 amd64, Go 1.26.2, clean source tree:
  `go test -buildvcs=false ./... -count=1` PASS
  (cmd 1.722s, app 41.832s, cli 2.202s, ssh 22.301s, tui 10.972s);
  `go vet -buildvcs=false ./...` and native development build PASS.
  Native fixtures used process-only PowerShell ExecutionPolicy Bypass.
- `--version` reports `SSH KeyFlow dev`. Final executable SHA256:
  `3c9d69822dbb3cbcfc0796cafc76b0ddd7aa87f0e3402cc8fef7fc343d92bb15`.
- Native console inspection covered 80x24, 60x18 and 32x12 on preceding
  `f6b6427c984146fd897342fe03e783c633372b8d`. The final source only refines
  the below-60-column Task footer, its test and review record; its corrected
  32x12 Task/technical/Help navigation was repeated natively. Public-event
  tests cover all four overview and technical entry kinds at every size.
- Synthetic public-only fixture state remained byte-identical through native
  navigation. Separate local standards and specification review passes found
  no unresolved issues. The review, specification and completed ticket are in
  [entry-detail-ux](../.scratch/entry-detail-ux/spec.md); exact commands,
  native source versions, hashes and review results are in the
  [verification record](../.scratch/entry-detail-ux/verification.md).

Linux/macOS/Windows arm64 native behavior, real SSH/server effects, elevated
Windows shared administrator destinations and hosted CI remain untested for
this change. Packaging/installer suites were not repeated; those sources and
the state/CLI/backend contracts are unchanged. All work is committed locally;
no push, merge, tag or release.

## Local detail colors and panels (2026-11-08 JST)

Product/test source `6fb51e850ef39fa76afeb4024df37e2dfab87246` on
`feat/entry-detail-ux`, verified from a clean source tree. The subsequent
completion commit records evidence and local ticket status only.

- Windows 11 amd64 / Go 1.26.2: full uncached
  `go test -buildvcs=false ./... -count=1` PASS (cmd 3.892s, app 47.215s,
  cli 2.799s, ssh 25.705s, tui 13.790s), `go vet -buildvcs=false ./...`
  PASS and native development build PASS. Native fixtures used process-only
  PowerShell ExecutionPolicy Bypass.
- Final binary reports `SSH KeyFlow dev`; SHA256:
  `55661fba7db5cc71b624f5a12ac715ee7a26a986de26d725f6a3ed03e18f19d4`.
- Native color inspection on that executable: destination panels at 80x24,
  key panels and wrapped deletion blocker at 60x18, registration/Task plain
  headings at 32x12, scrolling and Task technical/Help/back restoration.
  Earlier exploratory inherited NO_COLOR inspection retained readable labels.
  Color-enabled settings were process-local.
- Automated checks cover semantic colors, ANSI-stripped labels, all four
  overview/technical kinds at supported sizes, 59-column threshold behavior,
  width reflow through Help/technical origins, Unicode, scrolling, unchanged
  saved state and no external commands. Separate local standards/specification
  reviews have no unresolved findings.

Review → specification → ticket → implementation artifacts, exact native
conditions, fixture hash and platform limits are in the
[panel verification record](../.scratch/entry-detail-ux/panel-verification.md)
and [ticket 02](../.scratch/entry-detail-ux/issues/02-detail-color-panels.md).
Native resize itself, other OS/CPU targets, live SSH and hosted CI were not
tested; packaging/installer suites were not repeated. All work remains local.

## Whole-TUI readability redesign (2026-11-08 JST)

Product/test source `280ccebb61d539590c55c019595d4e2029c5d4be` on local
`feat/entry-detail-ux`, based on refreshed main
`76d392826f757e8aa4cfdb72d2d7b5216660f0fc`. The completion follow-up changes
documentation and ticket status only.

- Windows 11 amd64 / Go1.26.2: full uncached
  `go test -buildvcs=false ./... -count=1` PASS (cmd1.837s, app43.903s,
  cli2.256s, ssh24.045s, tui16.363s), `go vet -buildvcs=false ./...` PASS,
  native build PASS. Native test fixtures used process-only PowerShell
  ExecutionPolicy Bypass.
- Binary reports `SSH KeyFlow dev`; SHA256:
  `e4f8c6541d0f6b711783db99db3413d3085c43ec2e72888a7139468befd9b337`.
- Final source generated54 page captures across80×24,60×18,32×12. Public-event
  regressions cover shared paging/selected IDs, same-name discrimination,
  long-value form context, resized overlay focus, picker origin, supported result
  actions, session instructions and history scope. Existing safety/handoff
  journeys remain passing.
- Native final-binary inspection covered32×12 with explicit `NO_COLOR` and80×24
  with color enabled. The isolated synthetic state remained byte-identical.
  Astra's separate standards/safety and specification/UI reviews have no
  remaining blocking finding.

The [local preview](../.scratch/tui-readability/preview.md),
[design](../.scratch/tui-readability/design.md),
[review resolutions](../.scratch/tui-readability/review-resolutions.md), and
[exact verification record](../.scratch/tui-readability/verification.md) document
the review → spec → ticket → implementation workflow and remaining platform
limits. Native60×18 and runtime resize, other OS/CPU targets, live SSH and hosted
CI were not repeated; automated WindowSize journeys cover resizing and all three
sizes. Packaging/installer sources were unchanged and their suites were not
repeated. All work remains local.

## Selected A grouped-panel implementation (2026-11-08 JST)

Product/test source `fbb2c377031d9526c1a35670973a7f3c2be8cc6c` on local
`feat/tui-grouped-panels`, based on refreshed main
`76d392826f757e8aa4cfdb72d2d7b5216660f0fc`. The completion follow-up changes
documentation and ticket status only.

- Windows amd64 / Go 1.26.2: full uncached
  `go test -buildvcs=false ./... -count=1`, `go vet -buildvcs=false ./...`
  and native development build PASS. Existing Windows fixture scripts used
  process-only `PSExecutionPolicyPreference=Bypass`.
- Final `.build/ssh-keyflow.exe` reports `SSH KeyFlow dev`; SHA256:
  `a06ebffa3c5a2498d9edefcda238ef8e6fe3660c65a7fff4e97e3872df82bf1c`.
- Generated 240 production frames across 40 page/scenario families, three
  sizes (80×24, 60×18, 32×12), and both document ends, with matching text/ANSI.
  Public-event regressions cover geometry, paging, reference discrimination,
  exact selected IDs, forms, overlays, reflow and existing safety behavior.
- Final native 80×24 `NO_COLOR` launch → unregister picker → Esc → quit
  passed with isolated synthetic state byte-identical. Earlier implementation
  inspection covered destination overview and technical panels.
- Separate Astra standards/correctness and specification/visual reviews PASS
  for the final source with no unresolved blockers. The standards reviewer
  independently ran the TUI suite and TUI vet; full-repository and native
  results were implementer-reported to both reviewers.

The [selected specification](../.scratch/tui-grouped-panels/spec.md),
[local test instructions](../.scratch/tui-grouped-panels/README.md) and
[detailed verification](../.scratch/tui-grouped-panels/verification.md) record
the review → spec → tickets → implementation workflow and review resolutions.
Native narrow sizes and runtime resize were not repeated; automated journeys
cover those layouts. Other native targets, live SSH, hosted CI and packaging/
installer suites were not repeated. No push, merge, tag or publication.

## v0.4.0 release verification (2026-11-08 JST)

The release candidate is `a191d39afeae20b3dbf75663e10b538839b53672`.
PR #6 merged it into main with merge commit
`5a4710db25d552af5932afa4dd0101edbcb18bd5` at 2026-11-08T10:58:58Z.
Ancestry and identical trees were verified after refreshing origin/main;
both identify tree `54971e8f178de50a5e9bf0cc6ca36601462fab46`.
The annotated tag targets the reviewed, tested source commit already merged
into main, rather than the additional merge commit.

Preparation and exact-candidate checks:

- The source working tree was clean during local verification. Windows amd64,
  Go 1.26.2 and CPython 3.12.13 ran the full uncached
  `go test -buildvcs=false ./... -count=1` (cmd 1.900s, app 46.617s,
  cli 2.255s, ssh 30.232s, tui 24.443s) and
  `go vet -buildvcs=false ./...`: PASS. Native fixture scripts inherited only
  process-local `PSExecutionPolicyPreference=Bypass`.
- Hosted Verify runs `42473210314` (push) and `42473214254` (PR) both passed
  on the exact source. Each ran Go, Portable, installer and profile suites on
  Linux/macOS/Windows, plus six-target crossbuild and Linux amd64 CI-version
  smoke. The installer suites were not redundantly rerun locally.
- `python -X utf8 scripts/package_release.py --version v0.4.0 --output
  .tmp/release-v0.4.0/local-assets` built six real targets from that source.
  The audit verified seven assets, six unique LF manifest rows and all hashes,
  two regular members per archive, notices matching checkout bytes, Unix
  modes 0755/0644 and uid/gid 0, platform formats and GOOS/GOARCH/CGO-disabled/
  trimpath metadata, and embedded version markers. Local manifest SHA256:
  `c61de919769645db3c3d41709fb7d2c1ff0b0b4b8c057f0067e7e34a99b51457`.
- Extracted local Windows amd64 reported `SSH KeyFlow v0.4.0`; help and all
  three read-only JSON lists passed under a temporary home without creating
  `.ssh`. Other target binaries were not executed locally.
- Separate Astra release standards and specification reviews approved the
  candidate. The stale layout-contract sentence was corrected before final
  CI; release notes cover the changes since v0.3.0, version-1 compatibility,
  install/update instructions and remaining native/live-SSH limits.

Publication and downloaded-asset audit:

- [SSH KeyFlow v0.4.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.4.0)
  was published at 2026-11-08T11:08:32Z. Annotated tag object
  `de78904c5576970bc805240c0c831c2074dd8b21` and its remote peeled commit
  match the intended candidate. GitHub's latest API returned v0.4.0 with
  draft and prerelease both false.
- [Release workflow 42473838603](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42473838603)
  passed on that exact tagged source: three-OS Go/Portable/installer/profile
  tests, six-target crossbuild, Linux CI-version smoke and publication.
- Downloaded all seven published assets. Verified six unique manifest entries,
  every archive SHA256, exactly two regular archive members, notices matching
  the candidate's Git blob, Unix modes/ownership, platform/build metadata and
  embedded version markers. Published manifest SHA256:
  `29d62cead9fa219f5837907c56d36d37a6afd63552b87e5f58092bd7a5fdc513`.
- Extracted published Windows amd64 reported `SSH KeyFlow v0.4.0`; help and
  three read-only JSON lists passed in a temporary home without creating
  `.ssh`. The other five published binaries were not executed locally.
- Applied the reviewed release notes explicitly after workflow publication.
  Release target metadata is the exact candidate SHA; notes match the reviewed
  file, asset identity/size/update metadata are unchanged, and remote tag
  object/target plus stable/latest classification were rechecked.
- Source-hosted `main/install.sh` and `main/install.ps1` matched their Git
  blobs at source-hosted installer main commit
  `5a4710db25d552af5932afa4dd0101edbcb18bd5`. SHA256 respectively:
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`
  and `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`.
  No real install/update/removal cycle was repeated; hosted tests use isolated
  fixtures.
- Separate final standards and specification/evidence reviews approved the
  publication record with no blocking findings. The standards review also
  confirmed the final notes, tag, workflow and latest status against GitHub.

The publication evidence follow-up changes only this record. It does not move
the tag or change binary assets. Live SSH/server mutations, other native
OS/CPU interactive sessions and elevated Windows shared-administrator behavior
were not repeated. Cleanup is limited to task-owned branches whose tips are
ancestors of integrated main, after checking worktree cleanliness.

## 必須機能への縮小：dev（2026-11-08）

仕様は [.scratch/essential-key-utility/spec.md](../.scratch/essential-key-utility/spec.md)、
チケットは [01](../issues/01-essential-key-backend.md)・
[02](../issues/02-essential-key-cli-tui.md)・
[03](../issues/03-essential-key-contract-verification.md)。
この節だけが簡素化後の実装に対する証拠であり、上記の公開版・自動登録機能の記録は履歴として保持する。

- ソース：`22275867764d74f5dc60c4fc8d48a70d141e856b`。
  ベースは `404a3ebe616799b582a9bc8438bf47360065847c`。
  作業ブランチは `refactor/01-essential-key-utility`。
  ソースコミット時の作業ツリーはクリーン。後続の完了記録はチケット・仕様の状態と本節だけを変更し、
  実行コード・依存・配布スクリプトを変更しない。
- 環境：Windows amd64、Go 1.26.2、Python 3.12、PowerShell 7.6.9、
  Windows OpenSSH 9.5p2。Goキャッシュは作業用 `.tmp/go-cache`。
  インストーラーは一時ホーム・配布物のfixtureを使い、Windows PowerShell 5.1と
  Git Bash経由のshを含む。実ユーザーの鍵・SSH設定・プロファイル・PATH、
  接続先サーバーは変更していない。
- ソース確定後の `go test -buildvcs=false -count=1 ./...` と
  `go vet -buildvcs=false ./...` は成功。
  読取操作の無変更、鍵ペア・指紋・ダイジェスト、不正状態の拒否、v1履歴保持、
  削除途中の対象鍵だけの隔離、作成中断から同じパスへの再開、上書き拒否、
  ロック・バックアップ失敗、確認前の出力失敗で保存しないことを検証した。
  CLIの終了コード・JSONと診断の分離・非端末での読取、TUIの検索・戻る・スクロール・
  最小32×12・入力カーソル・ネイティブ端末引き渡しも対象に含む。
- `scripts/test_portable.py -v` は確定ソースで5件成功。
  `scripts/test_installers.py -v` は50件、
  `scripts/test_installer_profiles.py -v` は30件成功。
  後者2スイートはソース確定前の統合作業中に実行したが、その対象となる
  `install.sh`・`install.ps1`・`scripts/` はベースから確定ソースまで差分ゼロである。
  インストーラーfixtureはアプリのリモート操作を実行しない。
- `scripts/package_release.py --version v0.0.0-check --output
  .tmp/essential-key-utility-assets-2227586` を確定ソースから実行し、
  Linux / macOS / Windows × amd64 / arm64の6対象を実クロスビルドした。
  `v0.0.0-check` はローカル検証用の版注入であり、タグ・リリースを作成していない。
  配布物は6アーカイブと `SHA256SUMS` の7件。
  manifestのSHA256は
  `cd1ff422d9e2a16dd55d2bcb2a8902639760db354274ee8058d29fb7ddd66c80`。
- 実アーカイブ全6件について、manifestのLF・一意な6行・全SHA256、
  実行ファイルと `THIRD_PARTY_NOTICES.txt` の2メンバー、
  同梱文書のチェックアウトとのバイト一致、Unixの0755/0644・uid/gid 0、
  GOOS / GOARCH / CGO無効 / trimpath情報、検証版の埋め込みを照合した。
  6対象の実行用モジュールの和集合22件すべてについて同梱ライセンス項目を確認した。
- 展開したWindows amd64配布版で `SSH KeyFlow v0.0.0-check`、
  `keys list --json` と `hosts list --json` を確認。
  空の一時ホームで読取だけでは `.ssh` が作成されないことを確認した。
  同じ配布版をWindows実PTYと別の一時ホームで実行し、
  確認入力からネイティブssh-keygenへ引き渡して空パスフレーズのEd25519作成が終了0、
  未確認状態で保存されることを確認した。
  TUIのKeys表示・公開鍵表示・OSC52要求の出力・正常終了も確認した。
  端末側のクリップボード内容の読戻しは行っていない。
- 生成スニペットをネイティブ `ssh -G` で解釈し、
  ローカルの `ProxyCommand=ssh -V` を使って実際のIdentityFile展開を検証した。
  ネットワーク接続なしで、空白・アポストロフィ・`%` を含むパスを元の鍵へ解決できた。
  `${...}`を含むパスは誤ったコマンドを生成せず、公開鍵の配置案内だけを残す。
  接続コマンドは、利用者がスニペットを貼り付けた後の `ssh 別名` に統一した。
- StandardsとAstraの仕様レビューを別々に実施。
  出力失敗の無視、保持する状態検証のテスト不足、OpenSSHのパス展開、
  ヘルプ経由の再読込後の古いコピー復元、フォームの `?` 入力を修正し、回帰テストを追加した。
  `.bak`を恒久v1バックアップと誤解させる文言を修正し、必要な場合は最初の変更前に
  v1を利用者が別途退避する契約へ統一した。OSC52非対応時は折り返し画面の手動選択ではなく、
  CLI出力で公開鍵の1行を保つ案内とした。最終レビューの残指摘はない。

本作業ではpush・mainへのマージ・タグ作成・公開・ホストCIを実行していない。
Linux / macOS / Windows arm64のネイティブ実行、実OSシグナルによる中断、
他の端末でのOSC52受信・クリップボード読戻し、実SSHサーバーへの設定・接続は未検証。
クロスビルドとローカルfixtureの成功を、それらの実機確認とは区別する。
作業用の空パスフレーズ鍵と実験スクリプトは削除し、
ローカル配布物はレビュー用に保持する。
並行作業の3作業ツリーは各作業ブランチへコミット済みでクリーンにし、
main統合前の履歴保全のため保持する。
