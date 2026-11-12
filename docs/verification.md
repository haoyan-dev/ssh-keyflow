# 検証記録

本書は SSH KeyFlow の仕様に対する完了確認に使う。成功した検証だけを記録し、外部環境が必要な確認は区別する。

## 2026-11-11: v0.9.1 release and publication audit

- The user authorized publication of v0.9.1. Candidate:
  `3508d7ac77e4ecdba343ac6222f6af6ff3ffafd7`, the refreshed `origin/main`.
  The candidate working tree was clean. Local and remote tag reservations
  were checked before creating the annotated tag.
- Selected PATCH after v0.9.0 for the release-publication fix in PR #26.
  Other changes are versioning policy and verification documentation.
  No CLI, JSON, state, installation, or runtime contract changes require
  migration from v0.9.0; existing downgrade limitations remain applicable.
- Main Verify run 42971345135 passed all three OS test jobs and crossbuild
  on the exact candidate. The release workflow repeated its required
  verification and build matrix from the tag rather than duplicating that
  matrix locally.
- Local environment: Windows amd64, Go 1.26.2, Python 3.12.10,
  Windows PowerShell 5.1, Git Bash and PowerShell 7 available.
  `go test ./...`, `go vet ./...`, Portable tests (5), and release tests (17)
  passed against the candidate. Installer tests (50) and profile tests (30)
  also passed. All checks began on the clean candidate; only this documentation
  record changed while installer checks ran, with test inputs unchanged.
- Publication behavior reviewed separately from contract compatibility:
  the shared queue serializes tags; numeric stable comparison advances
  latest only to a newer version; lookup failures stop publication and
  prereleases cannot advance latest. v0.9.1 is newer than current v0.9.0.
  Existing independent standards/specification reviews for PR #26 reported
  zero findings. This release adds no application source changes.
- Annotated tag object `9ebf3f51a7c10f90ce5c308866ee31aca26fb0f3`
  matches locally and remotely and targets the candidate. Only
  `refs/tags/v0.9.1` was pushed. Release workflow:
  [42975321482](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42975321482).
  All three OS test jobs, crossbuild and publication passed on the candidate.
- [v0.9.1](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.9.1)
  is published, stable, not draft or prerelease, and returned by
  `releases/latest`. The prepared notes cover changes, compatibility,
  installation and verification limits. Their published content matches
  the prepared text; asset IDs and SHA256 digests were unchanged by the edit.
- Downloaded exactly seven assets: six OS/CPU archives and `SHA256SUMS`.
  Manifest entries are unique, LF-only and match each archive's SHA256.
  Every asset's size and digest also match GitHub metadata. Each archive
  contains exactly the regular executable and `THIRD_PARTY_NOTICES.txt`.
  Notices match the tagged Git blob; Unix entries have modes 0755/0644
  and uid/gid zero. All six binaries embed v0.9.1, and Go metadata confirms
  their expected GOOS/GOARCH, CGO disabled and trimpath (built with Go 1.26.0).
- The downloaded Windows amd64 executable returns `SSH KeyFlow v0.9.1`.
  Read-only key/destination JSON lists return empty arrays, guide help and
  TUI help succeed, and the isolated home is not created. Temporary extracted
  binaries were removed. An initial audit assertion incorrectly expected
  linker flags in build metadata; it was corrected to inspect the embedded
  version directly, and the complete audit then passed.
- Source-hosted installer inputs were fetched through GitHub's contents API at
  main commit `3508d7ac77e4ecdba343ac6222f6af6ff3ffafd7` and matched the Git
  blobs byte for byte. SHA256: `install.sh`
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`;
  `install.ps1`
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`.
  Source-hosted installer evidence is tracked separately from binary releases.
- The retrieved PowerShell installer passed production latest installation,
  same-version update and uninstall with PowerShell 7 in a disposable
  directory. The installed version is v0.9.1; executable/notices receipt
  hashes match, and notices match the tagged source. Noninteractive execution
  declined PATH persistence; the receipt records no profile ownership.
  Seeded SSH config and profile bytes were preserved after every operation.
  The temporary installation was removed.
- Standards review of this release record: only `docs/verification.md`
  changes; candidate, environments, reusable evidence and limitations are
  explicit, with immutable published assets preserved. `git diff --check`
  passed. Separate specification review: version/target selection, annotated
  tag, required CI, notes, all seven assets, native smoke, latest status and
  installer source attribution are accounted for, with no open findings.
- Limits: published-binary and installer execution covers Windows amd64 only.
  Other targets received build and archive inspection, not native execution.
  Elevated Windows ssh-agent changes, native macOS/Linux agent behavior,
  OSC52 delivery and live SSH connections were not exercised. This single
  publication verifies hosted queue syntax and normal latest advancement,
  not simultaneous release races or live older-series/prerelease publication;
  those decision cases remain covered by mocked regression tests.

## Release latest guard (2026-11-11)

- Integration: PR #26 merged at
  `7641bc0b44dedf3269963271f453077763399bd8`. Reviewed head
  `b4cefe0d91c37f106685661f739e43a7dc7b2151` passed hosted push run 42969232585
  and PR run 42969239502, including Linux/macOS/Windows tests and the six-target
  cross-build. Both independent review axes reported zero findings.
- The local limitations below remain historical evidence. Hosted test/build
  execution is now verified; actual publication and queue scheduling are still
  untested because this task created no release.
- Scope: `.scratch/release-latest/spec.md`; implementation on
  `fix/01-release-latest`, based on
  `662d7b7b4999a3569b71fd5abbae2d121fb5d3e8` plus the reviewed working diff.
  This is local implementation evidence, not a committed release candidate or
  a hosted CI result.
- Environment: Windows amd64, PowerShell, Python 3.12.10, Go 1.26.2, Git Bash.
  Broad checks ran with filesystem access needed for Go's cache and temporary
  installer fixtures; release tests mock all publication calls.
- `python scripts/test_release.py -v`: 17 tests passed, including numeric
  1.9.1/1.10.0 ordering, equal versions, first stable release, drafts/previews,
  paginated absence checks, malformed responses, HTTP/transport failures, and
  every serial ordering of two stable candidates plus a preview.
- `go test ./...` and `go vet ./...`: passed.
- `python scripts/test_portable.py -v`: 5 passed, including Windows native
  version injection and preservation of user data.
- `python scripts/test_installers.py -v`: 50 passed.
- `python scripts/test_installer_profiles.py -v`: 30 passed.
- Workflow YAML parsing and wiring inspection passed: the publication job
  depends on verification, uses a constant cross-tag concurrency group with
  cancellation disabled and queue max, and invokes the guarded publisher.
  The new release tests are included in the three-OS CI job.
- A read-only live lookup through the helper successfully parsed the
  repository's current latest response. No live release mutation was made.
- Standards review: Python uses only the standard library; publication
  preserves explicit repository selection, verified tag, generated notes, and
  built-asset arguments. Existing releases are never edited/deleted on retry.
- Specification review: all local acceptance cases passed. A latest 404 alone
  cannot trigger first-release publication; a successful listing without a
  published stable release is also required.
- Limitations: hosted queue scheduling, live publication, and non-Windows native
  execution remain untested. Queue order is modeled locally; actual concurrency
  exclusion depends on GitHub Actions. Manual publishers and older workflow
  definitions do not share this guard. The unchanged six-target build matrix
  remains assigned to release CI and was not duplicated locally for this fix.
- Final diff and whitespace checks passed. No release tag, push, or publication
  occurred during this implementation.

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

## v0.5.0候補（2026-11-08）

- 対象ソース：`6a09684180e1731aae9d75a834759283f9228ee1`。
  v0.4.0から非互換CLI・保存形式の変更を含むため、0.xのMINORを更新する。
  README・アプリケーション契約・ADRに適用版とv1事前退避を明記した。
  リリースノート案は作業用 `.tmp/v0.5.0-release-notes.md` に保持した（2026-11-11時点で作業領域から削除済み）。
- PRは [#8](https://github.com/haoyan-dev/ssh-keyflow/pull/8)。
  この候補後の追加差分は本記録とチケットの状態だけとし、タグは上記ソースを指す。
- macOSで初回CIが失敗した `TestNativeHandoffPreservesOriginalStdinAndInterruptedCreation` は、
  テスト用 `t.TempDir()` のリンク祖先が鍵作成の安全検査に拒否されていた。
  バックエンドの既存fixtureと同様に `filepath.EvalSymlinks` でテスト用ホームを正規化し、
  stdin同一性・呼出回数・中断進捗・再承認の検証を維持した。
  本番のリンク拒否条件・端末引き渡し処理は変更していない。
  修正後のmacOS hosted CIで全検証が成功した。
- クリーンな候補からWindows amd64 / Go 1.26.2 / Python 3.12 / PowerShell 7.6.9で、
  `go test -buildvcs=false -count=1 ./...`、`go vet -buildvcs=false ./...`、
  Portable 5件、インストーラー50件、プロファイル30件を再実行し、すべて成功した。
- 同じ候補のhosted CI：
  [push](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42482360552) と
  [PR](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42482364411)。
  Linux / macOS / WindowsのGo・Portable・インストーラー・プロファイル検証、
  6対象の実クロスビルドとLinux amd64版表示が成功した。
- `scripts/package_release.py --version v0.5.0 --output .tmp/v0.5.0-candidate-6a09684`
  を同じクリーンな候補から実行し、6対象・7配布物を生成した。
  manifest SHA256：`ebf814fc1567912271ff55f8a279685fbbbf2d61c658363a01ea5946362e3d02`。
  全ハッシュ、アーカイブの正確な2メンバー、同梱文書のバイト一致、Unix権限・uid/gid、
  GOOS / GOARCH / CGO無効 / trimpath / v0.5.0埋め込み、実行用モジュール22件の
  ライセンス対応を照合した。Windows amd64を展開して
  `SSH KeyFlow v0.5.0`、空のkeys/hosts JSON、一時ホームへの読取時無変更を確認した。
- Astra Specは版・移行文書を承認し、Standardsは同文書とmacOS fixture修正を承認した。
  実行コードは `22275867764d74f5dc60c4fc8d48a70d141e856b` から不変。

注釈付き `v0.5.0` タグは上記候補がmainへ統合済みであることを確認して作成する。
タグpushは自動公開を開始するため、具体的な版・SHAへの公開承認を別途確認する。
この記録時点でv0.5.0タグのpush・GitHub Release公開は未実施。
公開済み配布物のダウンロード検証は未実施であり、上記はローカル候補の証拠である。
他OS/CPUのネイティブ対話動作、実OSシグナルによる中断、クリップボード読戻し、
実SSHサーバーへの接続・設定は未検証。テスト用ジャンクションは削除した。

## v0.5.0公開後の検証（2026-11-08 JST）

- ユーザーが `v0.5.0` と対象コミット
  `6a09684180e1731aae9d75a834759283f9228ee1` の公開を承認した。
  [PR #8](https://github.com/haoyan-dev/ssh-keyflow/pull/8) はmainの
  `2ed184fcee8db3d2d1ddd8c69abd5eb47661271c` へマージ済みで、
  対象ソースがその祖先であることを確認した。
- 注釈付きタグ `v0.5.0` のオブジェクトは
  `a37fecb569064c80a5d7c7b010c3de9adcb9e806`。
  リモート未使用を再確認してこのタグだけをpushした。
  [Release workflow 42484308830](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42484308830)
  は上記対象コミットで実行し、Linux / macOS / Windows検証、
  6対象クロスビルド、publishがすべて成功した。
  リモートのタグオブジェクトとdereference先も上記2つのSHAに一致する。
- [正式リリース v0.5.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.5.0)
  は2026-11-08 13:24:20 UTC（同日22:24:20 JST）に公開された。
  draft / prereleaseはともにfalse、latest APIは `v0.5.0` を返すことを確認した。
  配布物は契約どおり6アーカイブと `SHA256SUMS` の7件である。
  公開済みタグ・バイナリアセットは変更していない。
- 7配布物を公開先から `.tmp/v0.5.0-published/` へ取得した。
  全ファイルのSHA256がGitHub APIのasset digestに一致し、
  manifestのLF・一意な6行・全アーカイブのSHA256も一致した。
  公開manifestのSHA256は
  `c6ad0e9273d37277dbbce0b625141f590b0cabc789ac006b315171d024cca0d3`。
- 全アーカイブについて、実行ファイルと `THIRD_PARTY_NOTICES.txt` だけを含むこと、
  Unixの0755/0644・uid/gid 0、GOOS / GOARCH / CGO無効 / trimpath /
  v0.5.0埋め込み、実行用モジュール22件の同梱ライセンス対応を確認した。
  noticeは対象コミットのGit blobとバイト一致する。
  初回の比較はWindowsチェックアウトのCRLF変換により不一致となったため、
  検証スクリプトの比較基準をタグのGit原本へ修正して全件を再実行した。
  配布物の修正は不要だった。
- 公開Windows amd64版を一時ホームで実行し、
  `SSH KeyFlow v0.5.0`、空のkeys/hosts JSON、
  読み取りで `.ssh` が作成されないことを確認した。
- main配信の `install.sh` / `install.ps1` をraw mainから取得し、
  インストーラー配信元コミット
  `2ed184fcee8db3d2d1ddd8c69abd5eb47661271c` のファイルとバイト一致を確認した。
  SHA256はそれぞれ
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f` /
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`。
  取得したPowerShell版をWindows amd64 / PowerShell 7.6.9で実行し、
  latestからの新規導入・同版更新・削除がすべて成功した。
  本体のv0.5.0表示・所有記録の本体ハッシュ・同梱noticeを照合し、
  一時ホームに置いた既存SSHデータとプロファイルのバイト保持を確認した。
  PATH保存は承認せず、ユーザーの実環境は変更していない。
- 取得済み配布物・API metadata・監査JSON・インストーラーsmoke結果は
  `.tmp/v0.5.0-published*` にレビュー用として保持した（2026-11-11時点で作業領域から削除済み）。
  インストーラーsmokeと配布版smokeの一時ホームは終了時に削除した。

Linux / macOS / Windows arm64のネイティブ対話動作、実OSシグナルによる中断、
クリップボード読戻し、実SSHサーバーへの接続・設定は未検証。
POSIXインストーラーは配信ソース一致と既存fixture / hosted CIで確認し、
公開済み配布物を使った実機インストールはWindows amd64のみで実施した。

## v0.5.1公開準備（2026-11-08）

- ユーザーがタグ方針変更のpush・main統合・新規リリースを依頼した。
  最新正式版v0.5.0からの差分は方針と検証記録・チケットのみであり、
  互換性のある修正としてPATCH版v0.5.1を選択する。
- 方針レビュー対象は `cbb239b8dfbe03c257b7584e7244cb44b7c97674`。
  Astraの仕様レビューと別途の規約適合確認で残指摘なし。
  Windows / PowerShell上で `git diff --check`、文書参照と
  README・アプリケーション契約・公開ワークフローとの整合を確認した。
- アプリのソース、依存、テスト、梱包・インストーラー、CI設定はv0.5.0から不変。
  v0.5.0のGo vetなど入力不変の検証証拠を再利用する。
  本PRと統合後コミットのCI結果を確認してから注釈付きタグを作成し、
  公開ワークフローでv0.5.1を注入した6対象をビルドする。
- タグ対象は本PR統合後の最新origin/mainとする。
  確定した候補SHA・CI・リリース・配布物検証結果は公開後の文書PRで追記する。
  本節の準備時点ではv0.5.1のタグpush・公開・公開物確認は未実施。
  実機検証の限界はv0.5.0記録と同じであり、v0.5.1のWindows amd64版は
  公開後に版表示と一時ホームでの読み取りsmokeを確認する。

## v0.5.1確定候補（2026-11-08）

- [PR #10](https://github.com/haoyan-dev/ssh-keyflow/pull/10) はmainの
  `de4c88c2ef9e2813e5da750a113ebd34c446c430` へマージ済み。
  このコミットを注釈付きv0.5.1タグの対象とする。
  本依頼はpush・main統合・新規公開を含み、追加の版・SHA承認は不要とする。
- レビュー済みPR HEAD `4c9edbac2f05085be2c957e0bf1baea306c80b06` と
  候補のツリーはともに `ac11738676e1730ebbb73efe18024fa781818ad3` で一致する。
  Astraの仕様レビューは方針・公開準備・日本語ノートを承認し、
  別途の規約確認も残指摘なし。Windows / PowerShellで差分チェックと
  クリーンな候補ツリーを確認した。
- 上記PR HEADの
  [push CI 42486871834](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42486871834) と
  [PR CI 42486939422](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42486939422) は
  合計8チェックが成功した。Linux / macOS / WindowsのGo・Portable・
  インストーラー・プロファイル検証と6対象クロスビルドを含む。
  ツリー一致によりこの証拠を候補へ適用し、統合後CIも完了を確認してからタグをpushする。
- v0.5.0との差分は `AGENTS.md`、アプリケーション契約、検証記録、チケットのみ。
  Go vetの対象ソース・依存は不変のためv0.5.0の成功証拠を再利用する。
  版注入は変わるビルド入力として扱い、公開ワークフローでv0.5.1版を新規生成する。
  公開後の記録追加は別の文書PRとし、このタグは移動しない。

## v0.5.1公開の検証（2026-11-08 JST）

- 対象コミットは `de4c88c2ef9e2813e5da750a113ebd34c446c430`。
  [統合後CI 42487428590](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42487428590)
  はこの正確なSHAで4ジョブすべて成功した。
  リモートmainとタグの未使用、クリーンなmainを再確認し、
  注釈付きタグ `v0.5.1` だけをpushした。
  タグオブジェクトは `3882a514c2849f91040b506fe7ab7ff150a22ecf`、
  dereference先は上記ソースSHAで、ローカルとリモートが一致する。
- [Release workflow 42487948659](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42487948659)
  はv0.5.1タグの同じソースSHAで全5ジョブが成功した。
  3 OSの検証・6対象クロスビルド後にpublishが同じタグからv0.5.1を注入して
  6アーカイブと `SHA256SUMS` を生成・公開した。
- [正式リリース v0.5.1](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.5.1)
  は2026-11-08 14:10:53 UTC（同日23:10:53 JST）に公開された。
  draft / prereleaseはfalse、latest APIはv0.5.1を返す。
  公開配布物は契約どおり7件で、全件を `.tmp/v0.5.1-published/` へ取得した。
  各ファイルのサイズ・SHA256はGitHub APIのasset size / digestと一致する。
- `SHA256SUMS` はLF・重複なしの6行で、全アーカイブのハッシュが一致する。
  manifestのSHA256は
  `c14b5d0afc4bef7dad1ecc554f50228629b624ec6ca38bae600ebe9905a724b6`。
  全アーカイブの正確な2メンバー、noticeのタグGit blobとのバイト一致、
  Unixの0755/0644・uid/gid 0、GOOS / GOARCH / CGO無効 / trimpath、
  v0.5.1文字列の埋め込み、実行用モジュール22件のライセンス対応を確認した。
  初回監査に追加した「go version -mに版注入引数が表示される」という条件は
  実際のビルド情報に合わなかったため、監査側のその条件だけを除いて全件を再実行した。
  公開バイナリ・タグ・配布物の修正は行っていない。
- Windows amd64 / Go 1.26.2 / Python 3.12.13 / PowerShell 7.6.9で監査した。
  公開Windows amd64版は `SSH KeyFlow v0.5.1` と表示し、
  一時ホームのkeys / hosts JSONが空、読取後に `.ssh` が作られないことを確認した。
  公開バイナリのビルド情報はGo 1.26.0であり、監査用Goと区別する。
- main配信の両インストーラーをraw mainから取得し、配信元コミット
  `de4c88c2ef9e2813e5da750a113ebd34c446c430` のGit原本とのバイト一致を確認した。
  `install.sh` / `install.ps1` のSHA256はそれぞれ
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f` /
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`。
  取得したPowerShell版のlatest新規導入・同版更新・削除を一時環境で実行し、
  本体v0.5.1表示、所有記録ハッシュ、同梱notice、既存SSHデータとプロファイルの
  バイト保持を確認した。PATH保存は承認せず、実ユーザー環境は変更していない。
- 公開物・API metadata・監査JSON・インストーラーsmoke結果は
  `.tmp/v0.5.1-*` と監査スクリプトに保持した（2026-11-11時点で作業領域から削除済み）。検証用の一時ホームは終了時に削除した。
  この記録は公開後の文書PRで統合し、公開済みタグは移動しない。
- Astraは最終の仕様・証拠・日本語ノートを承認し、別途の規約確認と
  `git diff --check` も成功した。承認済みノートを公開本文へ反映してUTF-8で照合し、
  全7アセットのID / digestが本文変更前と不変であることを確認した。

Linux / macOS / Windows arm64のネイティブ対話動作、実OSシグナルによる中断、
クリップボード読戻し、実SSHサーバーへの接続・設定は未検証。
POSIXインストーラーは配信ソース一致とfixture / hosted CIで確認し、
公開済み配布物の実機導入はWindows amd64のみで確認した。


## v0.5.2確定候補（2026-11-08）

- ユーザーの明示的な「tag this and cleanup the stale branches」により、
  最新origin/main `3f224795889447168df432b90fa3642b58bc4e1f` を
  注釈付きv0.5.2タグの対象に選択した。版・完全SHAをpush前に報告した。
  最新正式版v0.5.1の後の未使用PATCH版であり、新しい非互換変更はない。
- v0.5.1からの差分は `docs/verification.md` と
  `issues/04-release-tag-policy.md` の公開後記録だけである。
  アプリのソース・依存・テスト・ビルド・配布・CI入力は不変のため、
  v0.5.0で実施したGo vetの成功証拠を再利用する。
  [候補main CI 42489564460](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42489564460)
  は正確な候補SHAで全4ジョブ成功し、3 OSのGo・Portable・
  インストーラー・プロファイル検証と6対象クロスビルドを確認した。
- PR #11で統合済みの検証記録はAstraの仕様レビューと別途の規約確認を通過済み。
  Windows amd64 / PowerShell 7.6.9で最新refs・タグ未使用・候補の
  クリーンな作業ツリーを確認する。v0.5.2の版注入は変わる入力として扱い、
  Release workflowから新規ビルドして公開後の配布物を検証する。
- 本記録は別の文書ブランチで保存し、公開候補の内容を変更しない。
  公開後の証拠を同ブランチへ追記してPRで統合する。

## 古い作業ブランチの整理（v0.5.2依頼）

- `refactor/01-essential-key-utility`
  (`0e3efc151e3f8b01432886b73fa312909822d20f`) と
  `docs/03-v050-publication-verification`
  (`53639ffe7ceaf3c272710d29410e5e443f8dbe0e`) は
  mainへの祖先関係を確認し、ローカル・リモートの両方から削除した。
- 次の3本は並列作業のスナップショットであり、コミット自体はmainの祖先でない。
  全変更パスを比較し、mainへの取り込みまたは後続修正への置換を
  Astraの仕様レビューと独立した規約確認で確認してから、ローカルブランチと
  `.worktrees/essential-cli` / `essential-docs` / `essential-tui` を削除した。
  未追跡・ignoredファイルを含めてすべてクリーンで、未統合作業はなかった。
  - `refactor/02-essential-cli`
    (`79d47627d80e5213e29d4f4e838aa1e334463a1f`)：
    CLI・出力・状態処理は一致し、残差2ファイルはmainの
    `ssh 別名` ガイドと対応テストへの改善だけだった。
  - `refactor/02-essential-tui`
    (`fcc8243280f9043e8e31d2674fa38f7d55fe9bb6`)：
    残差はmainのmacOS用一時パス解決とテスト診断改善だけだった。
  - `docs/03-essential-contract`
    (`89ff53edc47bc25b7108c322e5482e4e65284c24`)：
    残差4ファイルはmainの確定版・移行案内・ガイド・コピー代替・タグ方針だった。
- 公開済みタグとmainの履歴は変更していない。

## v0.5.2公開の検証（2026-11-08 JST）

- 注釈付きタグv0.5.2の対象は
  `3f224795889447168df432b90fa3642b58bc4e1f`、
  タグオブジェクトは `2ab7245ee9e11bb8b08adf37627f8291bbfe38b9`。
  最新mainとクリーンな作業ツリー、ローカル・リモートで版未使用を
  再確認して、このタグだけをpushした。
  [Release workflow 42490574611](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42490574611)
  は同じ対象SHAで3 OSの検証、6対象クロスビルド、publishの全5ジョブに成功した。
  同じタグからv0.5.2を注入した配布物を生成している。
- [正式リリース v0.5.2](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.5.2)
  は2026-11-08 14:42:23 UTC（同日23:42:23 JST）に公開された。
  draft / prereleaseはfalse、latest APIはv0.5.2を返す。
  配布物は6アーカイブとSHA256SUMSの7件だけで、
  `.tmp/v0.5.2-published/` へ取得した全件のサイズ・SHA256が
  GitHub APIのsize / digestと一致した。
- manifestはLF・重複なしの6行で全アーカイブのハッシュが一致し、
  manifestのSHA256は
  `1b116a94d5a85a6a297c3ea098873023394a53fcadf467ce09776253c7357dda`。
  全アーカイブの正確な2メンバー、noticeの対象Git blobとのバイト一致、
  Unixの0755/0644・uid/gid 0、GOOS / GOARCH / CGO無効 / trimpath、
  v0.5.2文字列の埋め込み、実行用モジュール22件の同梱ライセンス対応を確認した。
- 監査環境はWindows amd64 / Go 1.26.2 / Python 3.12.13 / PowerShell 7.6.9。
  公開Windows amd64版は `SSH KeyFlow v0.5.2` と表示した。
  一時ホームで空のkeys / hosts JSONと、読み取りで `.ssh` を作成しないことを確認した。
- raw mainから取得した `install.sh` / `install.ps1` は、
  配信元main `3f224795889447168df432b90fa3642b58bc4e1f` のGit原本と
  バイト一致した。SHA256はそれぞれ
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f` /
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`。
  公開PowerShellインストーラーでlatestの新規導入・同版更新・削除に成功し、
  v0.5.2表示、所有記録のハッシュ、notice、既存SSHデータとプロファイルの
  バイト保持を確認した。PATH保存は承認せず、一時ホームは終了時に削除した。
- API metadata・配布物・監査JSON・インストーラーsmoke結果を
  `.tmp/v0.5.2-*` に保持した（2026-11-11時点で作業領域から削除済み）。この証拠と整理記録は文書PRで統合し、
  公開済みタグとバイナリは変更しない。

Linux / macOS / Windows arm64のネイティブ対話操作、実OSシグナルによる中断、
クリップボード読戻し、実SSHサーバーへの接続・設定は未検証。
POSIXインストーラーは配信ソース一致とfixture / hosted CIで確認し、
公開済み配布物の実機導入はWindows amd64のみで確認した。

## 端末別コピペスクリプト（2026-11-09 JST）

- 基点は取得済みorigin/main
  `e5e826c32cb79b5267cd5ee4f327ad516ffb659c`。
  作業ブランチは `feat/05-platform-copy-paste-snippets`、
  検証対象の実装コミットは
  `e67d05d18c8ddafacaaf6a63271b11a1db0cdc3f`。
  同コミットのクリーンなツリーで `go test ./...` と
  `go vet ./...` が成功した。直前に同一のGo入力で実行した全テストの
  成功結果をGoのキャッシュで再利用した。追加のネイティブ詳細確認も成功した。
  この記録とチケット完了欄の追記は文書だけであり、Goの検証入力を変更しない。
- 環境はWindows amd64、Go 1.26.2、
  Windows PowerShell 5.1.26100.9711、PowerShell 7.6.9、
  `C:\Program Files\Git\bin\sh.exe` のPOSIXシェル。
  スクリプトはテスト用ディレクトリを参照するように置き換え、
  実ユーザーのSSH鍵、config、ProgramData、永続環境設定は変更していない。
  各PowerShellは自身の標準モジュールを探索する環境で起動した。
- Linux/macOS向けの公開鍵配置とクライアント設定のスクリプトをGit Bashで実行し、
  既存内容、末尾改行のないファイル、同じ鍵の再配置、
  別アカウントでの拒否、非通常ファイルの拒否を確認した。
  クライアント設定は両PowerShellでも実行した。
  既存の別Hostを保持し、通常・グループ・引用符・`Host=alias` の同じ別名を拒否した。
  空白・アポストロフィ・`%`・`$()`・バッククォート・Unicodeを含む
  IdentityFileの内容が展開されずに保存された。
- Windows標準アカウントの配置スクリプトを両PowerShellで実行し、
  既存鍵の保持、重複追加の防止、ユーザー不一致の拒否、
  DACLの継承停止と本人・SYSTEM・Administratorsだけの許可を確認した。
  UTF-16のconfig / authorized_keysを追記前に拒否し、元のバイト列を保持した。
  管理者向けスクリプトは両PowerShellで構文と昇格ガードを確認し、
  非昇格時にディレクトリを作成しないことを確認した。
- 既存のネイティブOpenSSHテストで、スクリプトに含むHostスニペットの
  パースと `%` の展開をネットワーク接続なしに確認した。
  TUIテストで第1ステップからの表示、ステップごとの実行用文字列だけのコピー、
  前後移動、Helpからの復帰、F5後に古いスクリプトを復活させないことを確認した。
  公開鍵のみ、展開不能な秘密鍵パス、同一性検証、CLIのリダイレクト、
  読み取りによる状態非変更の既存テストも成功した。
- 規約レビュー: 専用ブランチ、共通バックエンド、追加依存なし、
  非実行・状態非変更の境界、ADR 0002による旧契約の明示的更新、
  README・用語・CLI/TUI契約と差分の整合、`git diff --check` を確認した。
  未解決の規約違反はない。
- 仕様レビュー: チケット05と仕様書の受け入れ条件を別途照合し、
  接続先OSと実行元OSの区別、Windowsアカウント種別、
  コピー単位、既存データ保持、重複・権限・リテラル値、
  公開鍵のみの扱いとコピー失効を確認した。未解決の仕様差分はない。

Linux/macOS実機での権限・シェル動作、昇格したWindows管理者向けスクリプト、
実SSHサーバーへの配置・接続、OSC52クリップボードの読戻しは未検証。
Windowsのシンボリックリンク作成権限がないため、リンク拒否の実行部分は
ローカルでは省略した。非通常ファイルの拒否は実行済みで、
リンク拒否のテストは権限のある環境で実行できる。
梱包・インストーラー・リリースの入力契約は変更していないため、
今回の変更では配布マトリックスやPythonの梱包・インストーラーテストを再実行しない。
push・main統合・タグ付け・公開はこの依頼の範囲に含めない。

## v0.6.0リリース準備（2026-11-09 JST）

- 後続の利用者依頼「merge to main and tag the version」により、
  PR統合・タグpush・自動公開を承認された。
  origin/mainとタグを再取得し、最新正式版がv0.5.2であることを確認した。
  端末別コピペスクリプトは後方互換の機能追加なので、次のMINOR版v0.6.0を選ぶ。
  状態スキーマや配布契約の変更、追加の移行作業はない。
- 検証済み実装 `e67d05d18c8ddafacaaf6a63271b11a1db0cdc3f` と
  準備開始時HEAD `874d7255537bfa2c0d853415083cc1a56b484fb7` の差は
  `docs/verification.md` とチケット05だけで、Go・依存・ビルド入力は同一。
  前節のGoテスト・vet・ネイティブスクリプト検証を再利用する。
  PRとリリースのhosted CIで3 OSテスト、Pythonの梱包・インストーラーテスト、
  6対象クロスビルドとLinux amd64の版注入を改めて確認する。
- 最終候補はPR統合後の最新origin/mainとし、対象SHAとCI結果を
  タグpush前に記録する。正式公開後は7配布物、manifest、内部バージョン、
  タグ・ソース・workflowの対応とWindows amd64の実行を検証する。
  公開後記録は別の文書PRで統合し、公開タグとバイナリを変更しない。

## v0.6.0タグ候補の確定（2026-11-09 JST）

- [PR #13](https://github.com/haoyan-dev/ssh-keyflow/pull/13) をmerge commitで統合し、
  最新origin/mainの `1227e27e610f750804417a8e475cdf4952c49407` を候補に確定した。
  検証済みPR HEAD `b937e03d037675db974dbd05d7fc4e3c39fde813` と
  `git diff --exit-code` でツリー全体の一致を確認した。
  [PR CI 42496355022](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42496355022) と
  [push CI 42496307994](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42496307994) は
  両方とも3 OSテスト・6対象クロスビルドの全4ジョブに成功した。
  branch protection・ruleset・必須レビューの設定はなく、全8チェックの成功後に統合した。
- Windows amd64 / Python 3.12.13でもREADMEの
  `test_portable.py`（5件）、`test_installers.py`（50件）、
  `test_installer_profiles.py`（30件）が成功した。
  実装コミットのGoテスト・vetは入力が同一なので再利用した。
  実クロスビルドはhosted CIで成功しており、同じマトリックスをローカルで重複実行しない。
- v0.6.0はローカル・origin双方で未使用であり、最新正式版v0.5.2からの
  後方互換の機能追加として選択した。候補のクリーンなmainに注釈付きタグを作成し、
  意図したタグだけをpushする。候補SHAはpush前に利用者へ報告した。
  この候補記録は別の文書ブランチで先に保存し、公開後記録とともに文書PRで統合する。

## v0.6.0公開の検証（2026-11-09 JST）

- 注釈付きタグv0.6.0の対象は
  `1227e27e610f750804417a8e475cdf4952c49407`、
  タグオブジェクトは `b47d526043cd451e0df9d83bc6281445004497f4`。
  クリーンな最新mainと版未使用を再確認し、このタグだけをpushした。
  originのタグオブジェクトとpeeled commitもローカルと一致した。
  [main CI 42496877016](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42496877016) と
  [Release workflow 42496973891](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42496973891)
  は同じ候補SHAで成功した。Releaseの3 OSテスト・6対象クロスビルド・publishの
  全5ジョブが成功し、タグからv0.6.0を注入した配布物を公開した。
- [正式リリース v0.6.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.6.0) は
  2026-11-08 15:59:45 UTC（2026-11-09 00:59:45 JST）に公開された。
  draft / prereleaseはfalse、latest APIはv0.6.0を返す。
  前版からの変更、コピー操作の変更、更新方法、追加移行不要、検証限界を
  リリースノートに記載した。ノート更新の前後で全asset IDとdigestが不変だった。
- `.tmp/v0.6.0-published/` に全配布物を取得し、
  Linux・macOS・Windowsのamd64/arm64用6アーカイブとSHA256SUMSだけであること、
  全7件のサイズとSHA256がGitHub APIのsize / digestに一致することを確認した。
  manifestはLF・重複なしの6行で、全アーカイブのハッシュが一致した。
  manifest自身のSHA256は
  `5e5ebd103f5b40b803091d870f2b56ff196bf4479cacd07ebf10e521d51dd599`。
- 全アーカイブが実行ファイルとTHIRD_PARTY_NOTICES.txtの正確な2メンバーを含み、
  noticeが候補のGit blobとバイト一致した。
  Unixの0755/0644・uid/gid 0、GOOS / GOARCH / CGO無効 / trimpath、
  全6バイナリのv0.6.0埋め込み、実行用モジュール22件のライセンス対応を確認した。
  公開Windows amd64版は `SSH KeyFlow v0.6.0` と表示し、
  一時ホームで空のkeys / hosts JSONを返し、読み取りで `.ssh` を作成しなかった。
- 監査環境はWindows amd64 / Go 1.26.2 / Python 3.12.13 / PowerShell 7.6.9。
  main `1227e27e610f750804417a8e475cdf4952c49407` を配信元として、
  raw mainのinstall.sh / install.ps1がそれぞれGit原本とバイト一致した。
  SHA256は
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f` /
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`。
  公開PowerShellインストーラーでlatestの新規導入・同版更新・削除に成功し、
  v0.6.0表示、所有記録ハッシュ、notice、既存SSHデータ・プロファイルの
  バイト保持を確認した。PATH保存は承認せず、一時ホームは終了時に削除した。
- 初回のローカル確認に加え、hosted CIでLinux/macOSのPOSIXスクリプト実行と
  0700/0600の権限確認を含むGoテストが成功した。
  公開済みバイナリのネイティブsmokeはWindows amd64のみであり、
  他の5対象の公開バイナリ実行、対話TUI、OSC52読戻し、
  実SSHサーバーへの配置・接続、昇格した管理者配置のローカル実行は未検証。
- featureブランチはmainへの統合とクリーン状態を確認してローカル・origin双方から削除した。
  配布物、API metadata、監査JSON、インストーラーsmokeとworkflowログは
  `.tmp/v0.6.0-*` に保持した（2026-11-11時点で作業領域から削除済み）。
  この公開後記録は文書PRで統合し、タグ・配布物を変更せず追加リリースも行わない。
- 公開後文書の規約レビュー: 文書とチケットのみの差分、正確なSHA・workflow・URL、
  監査出力との一致、`git diff --check` を確認し、未解決の違反なし。
  仕様レビュー: リリース契約の候補記録・公開検証・installerの別main追跡・
  制約の明記・不変性・後続文書PRを別途照合し、未解決の差分なし。

## 2026-11-09: ガイドのローカル・接続先プラットフォーム選択

- 対象はissue 06、仕様は `.scratch/guide-platform-selection/spec.md`。
  更新済みorigin/main `c226ffb5302fed539f36aa7400d8732ddd55e935` を基点に
  `feat/06-guide-platform-selection` で実装した。
  検証対象ソースは `b292eb1cdfb017d7c86081ea9ee60f7a708da284`。
  実行時の作業ツリーとこのコミットのGoソース・依存関係・ビルド入力は同一で、
  コミット後のクリーン状態も確認した。
- Windows amd64、Go 1.26.2、PowerShell 7.6.9の環境で
  `go test ./...` と `go vet ./...` が成功した。
  app 17.879秒、cli 3.459秒、tui 1.581秒で成功し、
  cmdは入力変更なしのキャッシュを使用した。
  新規テストは3×3のOS組み合わせ、既定OS・アカウントの維持、
  異なるローカルOSでの秘密鍵パス必須、指定OSのパス形式、
  無効入力時のスクリプト抑止、公開鍵のみの扱い、状態バイトの保持、
  共通バックエンド・CLI・TUI間の選択の受け渡しを確認した。
- TUIテストでプラットフォームフォームの既定値、検証エラー、
  戻る・再編集・取消、全ステップのプラットフォーム表示、
  スクリプトだけのコピー、help/backを含むF5無効化を確認した。
  既存のGit Bash・Windows PowerShell・PowerShell 7向けスクリプトテストも
  同じsuiteで成功し、一時ファイル以外のSSH設定や鍵を変更しない。
- `go build -o .build/ssh-keyflow.exe ./cmd/ssh-keyflow` が成功した。
  作成したWindows amd64バイナリは `SSH KeyFlow dev` を表示し、
  `hosts guide --help` でlocal-os / remote-os / windows-account /
  identity-fileを確認した。開発バイナリは無視対象の `.build/` に保持する。
- 規約レビューは別の読み取り専用レビュアーがAGENTS.md・契約・用語集・ADRと
  変更差分を照合し、入力境界検証・読み取り専用性・エスケープ・コピーと
  再読込の扱いに未解決の指摘なしと報告した。
  仕様レビューは実装担当が別途、issueの全受入条件と仕様の各項目を
  backend/CLI/TUIテストと照合し、未解決の差分なし。
  `git diff --check` も成功した。
- 配布・installer・依存関係は変更せず、Pythonの配布テストとリリース用の
  全対象ビルドは再実行していない。hosted Linux/macOS CI、実WSL、
  対話TUIとOSC52読戻し、実サーバーへの配置・接続は未検証。
  Windows上のPOSIX権限・symlink権限・管理者昇格の制約は前の検証と同じ。
  端末内の指定秘密鍵パスは利用者の申告であり、一致の検証・転送はしない。
  この追補は文書だけで、上記ソースの検証入力を変更しない。
  本変更はローカルコミットのみで、push・merge・新しいタグは行っていない。

## 2026-11-09: Managed key/destination removal and dissociation

- Source commit: `fa9442bbd74a30abc4797abb4abd13cdb9f597fa` on
  `feat/07-remove-managed-records`, based on freshly fetched `origin/main`
  `c226ffb5302fed539f36aa7400d8732ddd55e935`.
  Worktree: `.worktrees/remove-managed-records/`. The existing guide-platform
  worktree/branch at `a5acdde40f662991b791ac65446f40271bb60d55` stays unchanged.
- Environment: Windows amd64, Go 1.26.2, Python 3.12.13, PowerShell 7.6.9,
  installed Git Bash and Windows PowerShell. Tests use temporary homes and
  fixture processes; no real SSH data, persistent PATH, or remote access changes.
- `go test ./...` and `go vet ./...` passed against the exact source/test tree
  committed as the source SHA above. Subsequent `git diff HEAD -- internal cmd
  go.mod go.sum scripts` was empty. Formatting of all changed Go files and
  `git diff --check` passed.
- Regression coverage includes key/destination removal and association-only
  removal, unrelated associations, public/private/config preservation, exact
  prior-state backup, confirmation decline/missing/error, preview-output failure,
  existing locks, backup failures, missing files, ambiguous/missing identities,
  nonexistent associations, pending legacy deletion quarantine, archived-state
  validation, history/creation retention, and key/alias reuse with fresh IDs.
  TUI tests exercise fixed-ID targeting, associated-only pickers, cancellation,
  stale-state guards, native confirmation handoff, result reload, and 32×12 fit.
  CLI tests check required flags and nonterminal rejection before storage changes.
- `python scripts/test_portable.py -v` passed all 5 tests.
  `python scripts/test_installers.py -v` passed all 50 tests, and
  `python scripts/test_installer_profiles.py -v` passed all 30 tests.
  `python scripts/package_release.py --version v0.0.0-check --output
  .tmp/remove-verification-dist` built all six OS/CPU targets successfully.
  The generated set contained exactly seven assets, six matching SHA256 checksums,
  and exactly the executable and notices in each archive. The Windows amd64 binary
  printed `SSH KeyFlow v0.0.0-check`; help for all three new commands succeeded.
  This was local verification only, with no tag or publication.
- Standards review, performed separately: existing common-backend/terminal/lock/
  atomic-save/report conventions retained; no dependency changes, physical key
  deletion, remote execution, or unrelated feature edits. Archived IDs remain
  validated and new associations require active records. No unresolved findings.
- Specification review, performed separately against
  `.scratch/remove-managed-records/spec.md`: all three CLI commands and both TUI
  actions implemented; cascade versus selective removal, confirmations, data
  preservation, reuse, error handling, help, and domain/API documentation match
  the acceptance criteria. No unresolved findings.
- Limits: native binary smoke was Windows amd64 only; other targets were
  crossbuilt. No manual full-screen TUI/physical console Ctrl+C session,
  live SSH-server exercise, hosted CI, push, merge, or release was performed.
  Older binaries reject states containing the optional archive arrays; a separate
  prechange state copy is required when downgrading to such binaries.

## 2026-11-10: v0.7.0 preparation

- The user authorized a PR to `main` and publication of the next version.
  Refreshed `origin/main` remains
  `c226ffb5302fed539f36aa7400d8732ddd55e935`; the feature worktree is clean at
  `3ecb3b2` before release-document preparation. The existing guide-platform
  branch/worktree remains separate and unchanged.
- The latest published stable version is v0.6.0. v0.7.0 is unused locally and
  remotely and is selected as the next MINOR for the new removal/dissociation
  feature and its pre-v1 state compatibility impact. Release notes are in
  `.scratch/remove-managed-records/release-v0.7.0.md`, including installation,
  file preservation, downgrade migration, and verification limits.
- Reuse the passing local verification for source commit
  `fa9442bbd74a30abc4797abb4abd13cdb9f597fa` recorded above: `git diff --exit-code
  fa9442b HEAD -- internal cmd scripts go.mod go.sum` passes, and the Windows
  amd64 / Go 1.26.2 / PowerShell 7.6.9 conditions are unchanged. Further changes
  are documentation only. Hosted PR and release CI remain required.
- Final standards review: terminal confirmation, locking, atomic save/backup,
  domain vocabulary, CLI/TUI integration, archive identity validation, and
  documentation comply with the existing contracts. No unresolved findings.
- Final specification review, separate from standards: selective dissociation,
  cascaded removal, file/history preservation, cancellation/error handling,
  reuse with fresh IDs, and CLI/TUI acceptance criteria are covered. No unresolved
  findings. No unrelated guide-platform changes are included.
- The release target will be the latest `origin/main` after this PR is merged.
  Record its exact candidate SHA and CI evidence before pushing the annotated
  tag, then integrate candidate/publication records through a documentation PR.

## 2026-11-10: v0.7.0 candidate

- [PR #15](https://github.com/haoyan-dev/ssh-keyflow/pull/15) merged at
  2026-11-09 15:26:54 UTC (2026-11-10 00:26:54 JST). Candidate and latest
  `origin/main`: `2a70645c5e1c7f03ed96ec97a08718961010b3c0`.
  `git diff --exit-code d989c08964ae02a6fd0a497a1bb66a5f74b4f162 origin/main`
  confirms the entire candidate tree is identical to the final tested PR head.
  Checked out this candidate cleanly before starting the documentation branch.
- All eight checks passed on final PR head
  `d989c08964ae02a6fd0a497a1bb66a5f74b4f162`: [push run
  42643438459](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42643438459)
  and [PR run
  42643534983](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42643534983).
  Each passed Linux, macOS, and Windows Go/Portable/installer/profile suites plus
  six-target packaging and Linux amd64 injected-version smoke. Reuse these
  results for the identical candidate tree, along with the unchanged-source
  local vet and verification recorded above. No required review or protection
  rule is configured; mergeability was clean and no check was failing/pending.
- v0.7.0 is the next stable MINOR after v0.6.0 for this pre-v1 feature and archive
  compatibility change. The tag remains unused locally/remotely. The user
  authorized publication; report version and full target SHA before pushing
  only the annotated v0.7.0 tag. Let the release workflow rerun its verification
  and build matrix before publishing.
- Candidate records are committed separately on
  `docs/v070-publication-verification` before tag push and will be integrated
  with publication results in a documentation PR. They do not change the
  selected candidate commit or move a published tag.

## 2026-11-10: v0.7.0 publication verification

- Annotated tag v0.7.0 identifies
  `2a70645c5e1c7f03ed96ec97a08718961010b3c0`; tag object
  `ce235f6d6983305e2c437db9c0c0ff2053cbc591`. The remote tag object and peeled
  commit match the local values. Only this tag was pushed, after rechecking that
  the version was unused, latest stable was v0.6.0, and remote main still matched
  the candidate.
- [Main integration CI
  42644767845](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42644767845)
  passed all four jobs at the candidate SHA. Release verification also passed
  its three OS suites and six-target crossbuild at that same tagged commit
  before the publish job started.
- [Release workflow
  42644953520](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42644953520)
  succeeded with all five jobs at the tag's source SHA, including publish.
  [SSH KeyFlow v0.7.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.7.0)
  was published on 2026-11-09 at 15:35:49 UTC (2026-11-10 00:35:49 JST).
  Draft/prerelease are false; the latest-release API returns v0.7.0. Release
  notes cover changes since v0.6.0, downgrade compatibility, installation/update,
  and verification limits. Updating only the notes/title preserved all seven
  asset IDs and digests.
- Downloaded every published asset to `.tmp/v0.7.0-published/`: exactly six
  OS/CPU archives and SHA256SUMS. Each size and SHA256 matches GitHub API
  metadata. The manifest has exactly six unique LF-only entries, all matching
  their archive hashes. Its SHA256 is
  `d570f76829d1055635691727fcab76b3bcac353c525de79197eb598b2b70ffb9`.
- Each archive contains exactly the regular executable and
  THIRD_PARTY_NOTICES.txt; notices match the tagged Git blob byte for byte.
  Unix modes are 0755/0644 with uid/gid zero. All six binaries report the
  expected GOOS/GOARCH, CGO disabled, and trimpath in Go build metadata and
  contain v0.7.0. All 22 runtime dependencies match the licensed module/version
  inventory at the tagged source.
- Published Windows amd64 smoke: exact version `SSH KeyFlow v0.7.0`, empty
  keys/creations and hosts/associations JSON in a temporary home, and successful
  help for keys remove, hosts remove, and hosts dissociate. Read-only operations
  did not create `.ssh`.
- Source-hosted installers were fetched separately from raw `main` at commit
  `2a70645c5e1c7f03ed96ec97a08718961010b3c0` and match their Git blobs byte for
  byte. SHA256: install.sh
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`;
  install.ps1
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`.
  Source delivery is tracked against this main commit independently of the tag.
- The downloaded PowerShell installer passed latest install, same-version
  update, and uninstall in a disposable home, using production GitHub URLs.
  Installed version was v0.7.0; binary receipt hashes and bundled notices
  matched. Existing SSH data/configuration and profile bytes were preserved.
  No persistent PATH change was approved; the temporary home was removed.
- Publication audit environment: Windows amd64, Go 1.26.2, Python 3.12.13,
  PowerShell 7.6.9. Native smoke covers Windows amd64 only. Other published
  binaries, manual fullscreen TUI, OSC52 readback, and live SSH servers are not
  exercised locally. Hosted tests and crossbuilds do not establish native runtime
  behavior of all six published binaries.
- Audit JSON, release metadata before/after notes, asset fingerprints, source
  installers, published assets, and workflow logs were retained under
  `.tmp/v0.7.0-*` in the original workspace; they were no longer present
  as of 2026-11-11. Temporary auditor and installer homes
  were removed. Integrate these records through a documentation PR without
  moving the published tag or changing binary assets.
- Documentation standards review: only verification and local-ticket records
  change, links/SHAs/timestamps agree with audit outputs, and whitespace checks
  pass. No unresolved findings. Separate specification review: candidate
  identity, passing CI, asset/version/source agreement, stable/latest status,
  installer source tracking, migration notes, limits, and publication
  immutability satisfy the release contract. No unresolved findings.

## 2026-11-10: Guide platform selection integration

- The user accepted integration through a PR to `main`. Refreshed base:
  `87f58ba68afe1c42827f3205ebe25e71be8ffc0f`. Merged it into
  `feat/06-guide-platform-selection` without rewriting published history.
  Integration source commit: `9db8806143c6f2d3880853573d52fa381cc0187f`.
  Resolved four conflicts in CLI flag registration, TUI action titles, the
  application contract, and this verification log, preserving both guide
  options and v0.7.0 removal/dissociation behavior.
- Windows amd64, Go 1.26.2, Python 3.12.13, PowerShell 7.6.9:
  `go test ./... -count=1` and `go vet ./...` passed. The tested source/test
  tree matches the integration commit, whose clean working tree was confirmed.
  Existing tests cover the 3x3 platform matrix, invalid options, public-only
  guidance, unchanged saved records, TUI form/copy/refresh/cancellation, and
  removal/dissociation confirmation, target selection and data preservation.
- Built the integrated Windows amd64 executable and confirmed
  `SSH KeyFlow dev`. Help exposes all four guide options and retains
  `keys remove`, `hosts remove`, and `hosts dissociate`. Go formatting of all
  11 changed Go files passed after normalizing checkout CRLF to LF;
  `git diff --check` passed and no merge markers remain.
- `python scripts/test_portable.py -v` passed all 5 tests, and
  `python scripts/test_installers.py -v` passed all 50 tests.
- `python scripts/test_installer_profiles.py -v` passed all 30 tests.
  These three Python suites ran against the same integration commit's inputs;
  later changes are verification and ticket documentation only.
- Standards review, performed separately: shared backend and validation,
  existing form/navigation patterns, read-only guide generation, domain
  vocabulary and documentation remain consistent. No dependencies or saved
  state schema changes. No unresolved findings.
- Specification review, performed separately against
  `.scratch/guide-platform-selection/spec.md`: explicit local/destination OS,
  account selection, foreign-terminal path validation, unchanged defaults,
  public-only behavior, identity checks and script-only copying satisfy the
  acceptance criteria. Removal and dissociation remain reachable in CLI/TUI.
  No unresolved findings.
- Native binary smoke is Windows amd64 only. Fullscreen TUI/clipboard and live
  SSH-server sessions were not manually exercised. Hosted verification and
  six-target packaging will run on the PR; no new tag or release is requested.

## 2026-11-10: TUI color, spacing, and panes

- Source commit: `1aef8445c9ce0c952a26f965839d0b8f6a25513f` on
  `feat/08-tui-panes`. Base was refreshed `origin/main`
  `58d15de1bed2a2a23654b5b02accbcdbc7e2deed`; starting working tree was clean.
  Specification: `.scratch/tui-panes/spec.md`; local ticket: issue 08.
- Environment: Windows amd64, Go 1.26.2, PowerShell. `go test ./... -count=1`
  passed during implementation. Subsequent changes were confined to TUI
  menu text, message truncation, and removal of a temporary preview test.
  `go test ./internal/tui -count=1` passed against the final source/test inputs
  committed above; other package inputs were unchanged.
- `go vet ./...`, Go formatting, and `git diff --check` passed against the
  final inputs. Built `.build/08-tui-panes.exe`; `--version` returned
  `SSH KeyFlow dev`, and `tui --help` passed. Lip Gloss was already present
  and became a direct dependency; module versions and go.sum are unchanged.
- Inspected fixture-based terminal renders for keys, search, import validation,
  and a compact form at 80x24 and 32x12. Regression tests additionally cover
  40x16, 60x20, and 120x40; complete pane boundaries; minimum-size search with
  stale feedback; paging with 30 records; long ASCII/CJK input; caret bounds;
  fixed form actions; NO_COLOR; and rejection of untrusted terminal controls.
  Existing tests continue to cover raw clipboard scripts/public keys, guide
  navigation, draft/back behavior, reload invalidation, and native handoff.
- Standards review: reused existing viewport/textinput and installed styling
  components, kept presentation separate from operation/state logic, retained
  domain terminology and sanitization, and followed local issue/branch policy.
  No unresolved findings.
- Separate specification review: color and non-color focus cues, entry spacing,
  labeled content/input/search/messages/menu panes, persistent form controls,
  and the 32x12 minimum satisfy the three requested improvements and their
  recorded acceptance criteria. No unresolved findings.
- Limits: visual inspection used deterministic rendered text rather than a
  live fullscreen session. Clipboard delivery and appearance across terminal
  themes were not manually tested. Native smoke checks cover Windows amd64.
  Packaging/installer inputs were unchanged, so their Python suites and the
  release build matrix were not repeated. No push, merge, or release.

## 2026-11-10: v0.8.0 release preparation

- The user's subsequent "tag and push" request authorizes integration and
  publication. Refreshed remote branches and tags; latest stable is v0.7.0.
  Select v0.8.0 for backward-compatible TUI and guide platform features.
  The version is unused locally and no tag has been created at this stage.
- Clean tested source: `a39fce80e0fe30d8621848716ec81cec2c6345ac` on
  `feat/08-tui-panes`, based on current `origin/main`
  `58d15de1bed2a2a23654b5b02accbcdbc7e2deed`. On Windows amd64 with
  Go 1.26.2, fresh `go test ./... -count=1` and `go vet ./...` passed.
  Subsequent release preparation changes only documentation; source, tests,
  dependencies, and build inputs match that tested commit.
- Python packaging and installer suites are running locally. Hosted Verify
  must pass on the final PR state before merge. The hosted crossbuild and
  Release workflow perform six-target packaging, avoiding a duplicate local
  build matrix. Record the final integrated candidate SHA and published-asset
  checks in the post-publication documentation PR.
- Release notes cover both changes since v0.7.0, installation, compatibility,
  the inherited pre-v0.7.0 downgrade limit, and native/manual testing limits.
  Standards review: presentation remains separate from application operations,
  domain terms and terminal sanitization are retained, and the release follows
  the repository's PR/tag workflow. No unresolved findings.
- Separate specification review against `.scratch/tui-panes/spec.md`: all
  three UI requests and size, keyboard, copy, and state-preservation criteria
  are covered by the implementation evidence above. No unresolved findings.

## 2026-11-10: v0.8.0 integrated candidate

- The user requested "tag and push". Integrated the UI through
  [PR #18](https://github.com/haoyan-dev/ssh-keyflow/pull/18); release candidate
  is `b0b353e875a4d3d4396012602cb5dd67e3e7a646` on `main`.
  Its full tree matches PR head `dbb5310511ade78abcca01352d591385747fb6ed`.
  The implementation, tests, dependencies, installers, and build inputs also
  match the clean locally tested `a39fce80e0fe30d8621848716ec81cec2c6345ac`;
  subsequent changes were documentation only.
- On Windows amd64, Go 1.26.2, Python 3.12.13, and PowerShell 7.6.9, fresh
  `go test ./... -count=1`, `go vet ./...`, and Python Portable (5),
  installer (50), and installer-profile (30) tests passed. Whitespace checks
  and the final clean working tree check passed before tagging.
- PR Verify [42835513500](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42835513500)
  and branch Verify
  [42835400257](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42835400257)
  passed on the final PR head: Go and Python suites on Linux, macOS, and Windows,
  plus actual six-target packaging and Linux amd64 injected-version smoke.
  Repository branch protection and rulesets required no additional reviews;
  separate standards and specification reviews above had no unresolved findings.
- Refreshed branch/tag references, confirmed v0.8.0 was unused locally and
  remotely, and selected a MINOR increment from stable v0.7.0 for compatible
  TUI and explicit guide-platform features. Reported the full candidate SHA
  and version before pushing. Annotated tag object
  `87109928772889638b90744c46ae668ad51fbc28` peels locally and remotely to
  the candidate above; pushed only `refs/tags/v0.8.0`.
- Release notes include changes since v0.7.0, installation/update guidance,
  no required v0.7.0 migration, the inherited pre-v0.7.0 downgrade limit, and
  native/manual verification limits. The release build matrix runs in the
  hosted workflow; no duplicate local six-target build was needed.

## 2026-11-10: v0.8.0 publication audit

- Published [SSH KeyFlow v0.8.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.8.0)
  at `2026-11-10T14:48:09Z`. Release workflow
  [42836604663](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42836604663)
  succeeded: all three OS verification jobs, six-target crossbuild, and publish.
  Its source SHA is the annotated tag's peeled candidate
  `b0b353e875a4d3d4396012602cb5dd67e3e7a646`. Stable, nondraft,
  nonprerelease, and latest-release metadata all identify v0.8.0.
- Downloaded the actual six archives and SHA256SUMS from GitHub. The asset set
  has exactly seven files; all GitHub asset sizes/digests and six manifest
  checksums match. Manifest SHA256:
  `7d2241fb98b002a9eca4fd0c744a5f04de90dfbea54fb2fc25e12dc2b3c04357`.
  Manifest names are unique and use the required SHA256/two-space/LF format.
- Each archive contains exactly its platform executable and the candidate's
  `THIRD_PARTY_NOTICES.txt`, as regular files. Unix modes are 0755/0644 with
  zero uid/gid. All six binaries have the expected GOOS/GOARCH, CGO disabled,
  trimpath, and embedded v0.8.0. All 22 runtime dependency module/version pairs
  are present in the recorded notice sources.
- The downloaded Windows amd64 executable returns `SSH KeyFlow v0.8.0`.
  Read-only key/destination JSON lists and TUI help pass without creating an
  SSH directory in the isolated home. Guide help includes all four platform
  options; key/destination removal and dissociation help remain available.
- Replaced generated release notes with the prepared changes, compatibility,
  installation, and limits text. Checked exact note content and confirmed
  asset IDs/digests did not change. The published tag and binary assets remain
  immutable.
- Downloaded `main`-hosted installers from the production source URLs and
  compared their bytes with main commit
  `b0b353e875a4d3d4396012602cb5dd67e3e7a646`. SHA256: install.sh
  `b5b9e46bcee1f36deb9df80e65c671afd192a9ac09817277fbaf33be896d0e2f`;
  install.ps1
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`.
  Source delivery is tracked independently of the release tag.
- The downloaded PowerShell installer passed latest installation, same-version
  update, and uninstall using production GitHub URLs and a disposable home.
  Installed version, binary receipt hash, and bundled notices match. Existing
  SSH data, configuration, and profile bytes were preserved, with no persistent
  PATH change. Temporary binary-audit and installer homes were removed.
- Environment: Windows amd64, Go 1.26.2, Python 3.12.13, PowerShell 7.6.9.
  Native published-binary and live installer smoke covers Windows amd64 only.
  Other native targets, fullscreen appearance across terminal themes, OSC52
  delivery, and live SSH servers remain untested locally. Hosted tests and
  crossbuilds do not establish native behavior of all six published binaries.
- Downloaded assets, metadata before/after notes, fingerprints, audit JSON,
  source installers, and workflow logs were kept under `.tmp/v0.8.0-*`;
  they were no longer present as of 2026-11-11.
  These post-release records are integrated through a documentation PR,
  requiring no new version or tag movement.
- Documentation standards review: only verification and local issue records
  change; URLs, SHAs, timestamps, and checks agree with retained audit outputs.
  Whitespace checks pass. No unresolved findings.
- Separate release-specification review: candidate identity, passing CI,
  asset/version/source agreement, stable/latest status, installer source
  tracking, compatibility notes, limits, and publication immutability satisfy
  AGENTS.md and the distribution contract. No unresolved findings.

## 2026-11-11: ssh-agent autostart step

- Source commit: `d5a8456e108f8f477ca6fd673403a7cde397e709` on
  `feat/09-ssh-agent-autostart` (feature `43a885a`, review fixes `d5a8456`).
  Base was refreshed `origin/main` `a08bf976994e3ae91955ee65773a0070388f9251`.
  Specification: `.scratch/ssh-agent-autostart/spec.md`; local ticket 09;
  GitHub issue #20; ADR 0005.
- Environment: Windows amd64, Go 1.26.2, Windows PowerShell 5.1.26100,
  PowerShell 7.6.10, Git for Windows sh with OpenSSH 10.2p1. `go test ./...
  -count=1` and `go vet ./...` passed on the commit above. Built
  `.build/09-agent.exe`; `--version` returned `SSH KeyFlow dev`.
- Detection reads only the private-key format header. Unit tests cover OpenSSH
  `none`/cipher, encrypted and plain PEM/PKCS#8, unrecognized, truncated and
  missing files. Native ssh-keygen keys with and without a passphrase match.
- Guide tests cover unprotected (3 steps, no agent config), protected and
  undeterminable keys (4 steps, `AddKeysToAgent yes`, notice on config and
  agent steps when undeterminable), macOS-only `UseKeychain`, public-only keys,
  every local platform via IdentityFile override, CLI `hosts guide` output and
  TUI step navigation/copy. Native `ssh -G` parses the Host block for all three
  local platforms.
- Native script checks: the POSIX check reported a disposable ssh-agent on a
  temporary socket as reachable and an absent socket as missing; the agent was
  stopped with `ssh-agent -k`. The Windows script ran in both PowerShell
  editions, non-elevated, against the real service (Manual/Running): it refused
  the change as non-Administrator and the service state was unchanged.
- Earlier runs of the POSIX test (before `d5a8456`) left three disposable
  ssh-agent processes on `kfa*` temp sockets; they were stopped and the
  directories removed. The Windows ssh-agent service was not touched.
- Standards review (separate pass): ticket summary/comment out of date (fixed);
  duplicated test fixtures and notice text built outside the step (fixed);
  `~` quoted literally in the macOS ssh-add hint (fixed); POSIX test tool
  lookup and agent cleanup (fixed). Remaining judgement calls accepted:
  `openSSHPrivate`/`privateWithCipher` live in different test packages, and
  `agentSetup` follows the existing platform-string switch of `clientSetup`.
- Specification review (separate pass): this record was missing (added);
  Windows wording contradicted key persistence (fixed); no warning without ssh
  on PATH (fixed); notice absent from the config step (fixed); CLI coverage
  (added); stale detection text in the ticket and issue #20 (fixed); refresh
  wording recorded in the spec Decisions.
- Limits: no elevated Windows run, so `Set-Service`/`Start-Service` were not
  executed. macOS launchd/Keychain and Linux desktop/systemd agents were not
  run natively; their scripts are check-only and were exercised in Git sh. No
  real SSH connection loaded a key through AddKeysToAgent. Packaging and
  installer inputs are unchanged.

## 2026-11-11: ssh-agent autostart integration

- PR #22 merged into `main` as merge commit
  `f9956c92c26027bfeae642361e9e56b3eab4adcf`, matching the repository's
  merge-commit method. Head `880680609d755c1a4a9e6b6737235dd1b982b425` passed
  PR and branch CI (runs 42943892214 and 42943946815): tests on ubuntu, macOS
  and Windows plus crossbuild. The merged tree is identical to that head.
- `main` CI run 42951426479 on the merge commit passed the same jobs.
- Issue #20 closed by the merge. Ticket PR #21 was closed unmerged because
  #22 contains its commit `7c2add5`. Both task branches were confirmed to be
  ancestors of `origin/main`, then deleted locally and on `origin`.
- No version tag or release was requested; the earlier verification record
  remains the source of local test evidence and limits.

## 2026-11-11: v0.9.0 release and publication audit

- The user requested "tag a release". Latest stable was v0.8.0; v0.9.0 was
  unused locally and remotely. Selected a MINOR increment for the
  backward-compatible ssh-agent guide step (ticket 09, #20). Candidate is
  `origin/main` `662d7b7b4999a3569b71fd5abbae2d121fb5d3e8` (merge of #23).
  Since v0.8.0 no dependency, packaging, or installer input changed.
- On that clean commit (Windows amd64, Go 1.26.2, Python 3.12.10), fresh
  `go test ./... -count=1`, `go vet ./...`, and Python Portable (5),
  installer (50), and installer-profile (30) tests passed. Main CI run
  42951426479 passed on the feature merge; the candidate adds documentation only.
- Annotated tag object `c3c4d2bc643fddc13c23762f715db32b9f39e6a1` peels
  locally and remotely to the candidate; only `refs/tags/v0.9.0` was pushed.
  Release run
  [42955302241](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42955302241)
  passed verify (three OS tests, crossbuild) and publish on the candidate.
- [v0.9.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.9.0) is
  stable, not draft or prerelease, and `releases/latest` returns v0.9.0.
  Generated notes were replaced with the prepared changes, compatibility,
  installation, and limits text; exact note content was checked and asset IDs
  and digests were unchanged by the edit.
- Downloaded all seven assets. `SHA256SUMS` verifies the six archives. Each
  archive contains only `ssh-keyflow` (`.exe` on Windows) and
  `THIRD_PARTY_NOTICES.txt`; tar entries are 0755/0644 with uid/gid 0. All six
  binaries report the expected GOOS/GOARCH, CGO disabled, trimpath, embed
  `v0.9.0`, and contain the `AddKeysToAgent yes` guide text. Bundled notices are
  identical across archives and match the tagged `THIRD_PARTY_NOTICES.txt`.
- Downloaded Windows amd64 binary: `--version` returns `SSH KeyFlow v0.9.0`;
  read-only key/destination JSON lists, guide help with all four platform
  options, and TUI help pass without creating files in an isolated home.
- Production `install.ps1` and `install.sh` from `main` match `662d7b7`
  (install.ps1 SHA256
  `88e76aca2e2d04dd41cb0a77fd8a896587240469386e5baac1b2c4f7a038a27f`).
  The downloaded PowerShell installer passed latest installation, same-version
  update, and uninstall in PowerShell 7.6.10 with a disposable install directory
  and profile. Installed executable and notices match the release asset and
  the receipt hash. A seeded SSH config and the profile were byte-identical
  afterwards, the PATH prompt was declined non-interactively, and the user
  PATH was unchanged. Temporary homes and extracted binaries were removed.
  The downloaded assets in `.tmp/v0.9.0-assets` were deleted after the audit
  at the user's request. The immutable assets and their SHA256 digests remain
  available from the release page and `SHA256SUMS`.
- Limits: native published-binary and installer smoke covers Windows amd64
  only. The elevated Windows ssh-agent service change, native macOS/Linux
  agents, OSC52 delivery, and live SSH servers were not exercised. Crossbuilds
  do not establish native behavior on all six targets. The published tag and
  assets remain immutable.


## 2026-11-11: project documentation system

- Tested implementation source: `76dc5b2826468c0c744be18f49bc2e852115979c`,
  based on refreshed `origin/main` `1d2f44eb0449b389629e01fe71b970f715badd47`.
  Scope: reorganized OSS documentation, tutorial, contribution/maintenance
  guidance, offline Markdown checks, and a documentation CI job. Runtime,
  dependencies, installer, packaging, and release-publication inputs are unchanged.
- Environment: Windows amd64, Go 1.26.2, Python 3.12.10; normal desktop credentials
  were required for the existing hc pipe and GitHub access. Tests ran in task-owned
  hc terminal `docs-validation` (t1t), which exited 0. No core restart occurred.
- Passing checks: `go test ./... -count=1`, `go vet ./...`,
  `python scripts/test_portable.py -v` (5),
  `python scripts/test_release.py -v` (17),
  `python scripts/test_installers.py -v` (50), and
  `python scripts/test_installer_profiles.py -v` (30).
  These ran before/during the implementation commit with identical applicable
  source inputs. After the final checker changes,
  `python scripts/test_docs.py -v` (10) and `python scripts/check_docs.py`
  passed on the committed implementation tree (24 maintained pages).
- Built the native executable without version injection: `--version` returned
  `SSH KeyFlow dev`. Checked `--help` for 31 documented CLI command lines, then
  read empty key/destination JSON lists in an isolated home and confirmed no home
  files were written. This validates command/flag availability, not execution of
  every mutation shown in the tutorial.
- GitHub's read-only GFM rendering API rendered all 11 main user/contributor pages;
  parsed HTML confirmed single top-level headings, links, code blocks, and tables
  where present. This is structural rendering evidence, not screenshot-based QA.
- README is 73 lines and retains all seven original second-level headings as
  navigation points. Compared the old long-form prose to the relocated guides and
  release page; operational guidance is retained with adjusted relative links.
- Standards and specification self-reviews were performed separately in
  [.scratch/documentation-system/review.md](../.scratch/documentation-system/review.md).
  Resolved findings: misleading sequential CLI examples and fenced HTML anchors
  incorrectly qualifying as link targets. No remaining implementation finding;
  external PR review remains pending.
- This record and ticket updates add evidence only. Reuse implementation evidence
  when their check inputs remain identical, rerun the documentation check, and let
  PR CI verify the final head, including the existing six-target crossbuild.
- Limits: external URLs are outside the offline checker; this is a scoped Markdown
  validator, not a complete Markdown parser. No live SSH server, elevated service
  setup, OSC52 delivery, native macOS/Linux tutorial, or end-to-end interactive
  tutorial was exercised. No hosted documentation site or application release was
  requested by the clarified scope. Publication is a review PR; integration awaits
  review, and no version tag is needed for this documentation change.


## 2026-11-11: v0.9.2 release preparation

- Proposed version: **v0.9.2**; previous highest published stable: **v0.9.1**
  (`3508d7ac77e4ecdba343ac6222f6af6ff3ffafd7`). Refreshed remote branches/tags
  and release listings before selection; v0.9.2 was unreserved locally/remotely.
- Version rationale: compatible documentation fixes and documentation-validation
  improvements since v0.9.1 require PATCH in 0.x development. Runtime, dependencies,
  state, installers, packaging, publication script/workflow, and bundled notices
  are unchanged. This is neither a feature increment nor a 1.0.0 readiness decision.
- Full tested source: `aaa8defaa11fdada24b9c0e37175b8c3681cc679`, clean integrated
  main before preparation. The release notes are tracked in
  [.scratch/release-v0.9.2/release-notes.md](../.scratch/release-v0.9.2/release-notes.md).
  They cover all changes since v0.9.1, unchanged compatibility/migration rules,
  installation/update instructions, and outstanding translation work.
- Environment: Windows amd64, Go 1.26.2, Python 3.12.10. Fresh checks in task-owned
  hc terminal `release-v092-checks` exited 0: `go test ./... -count=1`,
  `go vet ./...`, `python scripts/check_docs.py` (24 maintained pages),
  `python scripts/test_docs.py -v` (10), `python scripts/test_portable.py -v` (5),
  and `python scripts/test_release.py -v` (17).
- Passing main CI [42985556698](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42985556698)
  tested this exact source: docs, three-OS Go/Portable/release/installer/profile
  tests, and six-target crossbuild. Reuse its installer (50) and profile (30)
  evidence because the installer/test/runtime/dependency inputs are unchanged;
  the local environment matches the earlier Windows validation. Do not duplicate
  the six-build matrix locally; the tag workflow performs its own matrix.
- Standards self-review: correct work branch, local tracker/dependencies, complete
  tracked notes, authoritative contract/policy paths, and no tag before preparation
  integration. No unresolved standards finding.
- Specification self-review: changes since the previous release are accounted for;
  language requirements are explicitly pending translations, not shipped language
  support. Notes contain compatibility, migration, installation, and limits.
  No unresolved preparation finding; publication/audit gates remain pending.
- After this preparation merges, refresh main and compare its candidate against
  the tested source. Preparation-only records/notes may reuse unchanged check
  inputs; rerun documentation links and require passing candidate CI. Record the
  exact candidate and comparison in the annotated tag, published notes, and audit.
- Limits: no live SSH server, elevated Windows service setup, OSC52 delivery,
  native macOS/Linux interactive tutorial, or all-target native execution.
  Crossbuilds are not native smoke tests. No new state migration is introduced;
  existing backup and downgrade limits remain. Published-asset and installer
  checks will follow publication and be integrated in a separate audit PR.


## 2026-11-11: v0.9.2 publication audit

- [Preparation PR #31](https://github.com/haoyan-dev/ssh-keyflow/pull/31) merged before tagging.
  Final candidate/tagged source: `50ee1ba49b1378c8b68d42c9ec080a795ed19a76`.
  Its tree equals preparation head `67cfe6af001639d33534372ea9229088fa1dd8ef`,
  whose PR/branch CI runs 42986958424 and 42986950327 passed all ten checks.
  Only preparation notes/records differ from tested source
  `aaa8defaa11fdada24b9c0e37175b8c3681cc679`; unchanged-input evidence was reused
  and documentation links rerun on the clean final candidate.
- Annotated tag object `2533ab46d6f79ce346b185948b72472d9cebb10d` is identical locally and remotely and
  peels to the candidate. Only `refs/tags/v0.9.2` was pushed. The annotation
  records the version rationale, tested source, final source, and comparison.
- [Release workflow 42987960776](https://github.com/haoyan-dev/ssh-keyflow/actions/runs/42987960776)
  passed three-OS verification, documentation checks, six-target crossbuild,
  and publication on the exact candidate. Its publish step built the actual
  release binaries with the v0.9.2 version injection.
- [v0.9.2](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.9.2) is published, stable,
  not draft/prerelease, and is returned by `releases/latest`. The PATCH rationale
  is compatible documentation/checking improvements since v0.9.1; Chinese,
  English, and Japanese translations remain pending documentation tickets 04–05.
- Replaced generated notes with the prepared changes, compatibility/migration,
  installation, limits, full final source, and evidence comparison. Exact note
  content was checked, and all seven asset IDs, digests, and sizes were unchanged.
- Downloaded all seven assets. SHA256SUMS has exactly six unique archive entries
  with SHA256/two-space/LF formatting; all archive hashes and all seven GitHub
  asset digests match. Exact names, public asset identities/digests, binary hashes,
  and installer source evidence are preserved in
  [published-assets.json](../.scratch/release-v0.9.2/published-assets.json).
- Each archive contains only its executable and THIRD_PARTY_NOTICES.txt.
  Tar members are regular files, with modes 0755/0644 and uid/gid 0; zip members
  are regular files. Every notice matches the tagged source notice byte-for-byte.
  All six binaries have expected GOOS/GOARCH, CGO_ENABLED=0, trimpath, no VCS
  revision stamping, Go 1.26.0 build metadata, and one embedded v0.9.2 tag string.
- Audit correction: the first audit script incorrectly required linker flags in
  `go version -m`; those flags are absent from these binaries' metadata. The
  corrected check verifies the embedded tag and uses workflow evidence for the
  injection flags. No tag or asset was changed. Runtime version output was tested
  only on the available native Windows amd64 target, not inferred for other targets.
- Windows amd64 published executable: `--version` returned SSH KeyFlow v0.9.2;
  empty-home key/destination JSON, guide help, and TUI help passed with zero home
  writes. Local audit tools: Windows amd64, Go 1.26.2, Python 3.12.10.
- Source-hosted install.sh/install.ps1 fetched from main match commit
  `50ee1ba49b1378c8b68d42c9ec080a795ed19a76` exactly. Their SHA256 values are in the JSON
  evidence. PowerShell 7.6.10 production installer passed
  latest install, exact v0.9.2 update, and uninstall with isolated home/install
  paths and an explicit temporary profile. Installed bytes/notices and receipt
  hashes match the release. Seeded dummy SSH state/test key/config and profile
  remained byte-identical; persistent user PATH was unchanged. Temporary native
  homes were automatically removed after checking their resolved temp boundaries.
- Standards self-review: preparation integration preceded candidate/tag selection;
  tag/release/binary metadata agree; asset identities remained immutable; source
  installer verification names its own main commit. No unresolved standards finding.
- Specification self-review: exact asset set/hashes/archive metadata, native version,
  installer preservation, stable/latest status, prepared notes, and limits are
  recorded. The publication is complete; this audit record awaits its PR integration.
- Limits: no native execution on the other five targets, POSIX installer production
  run, live SSH server, elevated service setup, OSC52 delivery, or end-to-end
  interactive tutorial. Crossbuild and embedded-tag evidence are not all-target
  runtime verification. This audit changes records only; advancing main with it
  does not require another version or moving the immutable release tag.

## 2026-11-12: host key pinning

- Source commit: `d7efe69b225a6122356298c4306bf9418f630ee3` on
  `feat/host-key-pinning` (feature `62a7df7`, review fixes `06ad3ef`, Linux
  test fix `9f38bb1`, second review fixes `c5e489a`, helper refactor
  `d7efe69`). Base
  `main` `cdbc0ae7a9eda8b0bbb508e0cdced4d3f20c42ad`. Specification:
  `.scratch/host-key-pinning/spec.md`; issues 01–05; ADR 0006 (accepted).
- Environment: Windows amd64, Go 1.26.2, Git for Windows sh with OpenSSH
  10.2p1, PowerShell 7.6.10, Python 3.12.10. `go test ./... -count=1`,
  `go vet ./...` (also `GOOS=linux` and `GOOS=darwin`), gofmt on changed files,
  `scripts/check_docs.py` and `scripts/test_docs.py` passed on the commit above.
  GitHub Actions on `9f38bb1` passed `test` on ubuntu, macos and windows,
  `crossbuild` and `docs`; the first push failed one CLI test on ubuntu, which
  expected four guide steps although a linux override on a linux computer
  adds the two pin steps. The fixed CLI tests also passed on WSL Ubuntu 22.04.
- OpenSSH behaviour, checked before implementation with throwaway keys and an
  unprivileged sshd on `127.0.0.1` only
  ([spike](../.scratch/host-key-pinning/openssh-spike.md)): Linux OpenSSH
  9.6p1 and 9.2p1 clients and servers, Win32-OpenSSH 9.5p2 and Git for Windows
  10.2p1 clients through a loopback forward. The pinned form connects with no
  prompt on a non-22 port, a wrong pin is refused, `HostKeyAlias` ignores the
  port, quoted absolute Windows paths work with forward or escaped
  backslashes, and `IdentityFile <x>.pub` with `IdentitiesOnly yes` selects
  only the matching agent key. Standard Windows users can't read
  `%ProgramData%\ssh\ssh_host_*_key.pub`, so the guide's Windows print step
  requires Run as Administrator. No user SSH files were changed; all spike
  processes, files and the agent key were removed.
- Unit tests cover strict pin file decoding (CRLF, tabs, comments, markers,
  duplicates, symlinks; never rewritten), atomic writes keeping other
  destinations, parser accept/reject and limits, a newline-free paste stopping
  at 64 KiB, every pasted byte counted toward the limit (an exact 64 KiB paste
  accepted, endless blank lines refused), a BOM after leading whitespace,
  replacement at a split destination's first entry, no pin steps when the pin
  file path can't be written literally, identical re-pin without writing, refusal without `--replace`,
  `--yes` never replacing, `hosts unpin` requiring a real terminal, list text
  and JSON (`host_key_pins` only for managed destinations, omitted when the
  file is unreadable), the removal notice, guide steps until pinned and the
  pinned lines afterwards (unpinned snippet equals the pinned one minus the
  three lines), public-only snippets, native `ssh -G` parsing of the pinned
  snippet, and the TUI `t`/`T` handoff reading paste and confirmation from
  native input.
- Built binary smoke test in a temporary home: `--version`, `hosts pin` /
  `hosts unpin` help, empty `hosts list` (text and JSON) created no storage
  directory; an unknown destination exited 2 with usage.
- Standards review (separate pass): missing verification record (added);
  unchecked "Ignored line" write (fixed); duplicated path quoting and unused
  JSON tags (fixed). Accepted judgement calls: `pinHost`/`unpinHost` keep
  parallel lock flows like removal; operation and OS names stay strings as in
  the existing code; the public-only snippet change stays in this PR because
  the spec and ADR 0006 include it. Ticket `Status: ready-for-human` follows
  the repo's use for implemented work awaiting human integration.
- Specification review (separate pass): unbounded newline-free paste (fixed);
  `T` missing from the list footer (fixed); removed destinations listed in
  `host_key_pins` (fixed, contract updated); misleading lock comment (fixed);
  whitespace field separation and pin steps omitted with the SSH config step
  recorded in the spec. The `host-keys` report kind is internal, like other
  report kinds.
- Second review on `9f38bb1` (standards and specification passes): pin steps
  offered for a non-literal pin file path, a paste limit one byte stricter
  than `--file` and not counting blank lines, a BOM after whitespace kept, a
  `SavePins` comment overstating byte-for-byte preservation, a missing
  split-position test, and this record's stale commit were fixed in
  `c5e489a`. The judgement calls were then fixed in `d7efe69`: shared
  `literalConfigPath` and `PinStatus` helpers, one way to group pins
  (`HostKeyPins`), `SavePins` validating before it builds the file,
  `hostKeyPinning` without the extra boolean, and `Request.ReadsPaste` in
  place of the TUI executor's operation-name check. The removal notice for an
  unreadable pin file is now in the spec.
- Limits: macOS clients and servers and the 1Password agent were not tested.
  No real interactive paste through `hosts pin` in a native terminal was run
  in this session; the terminal flow is covered by tests with scripted input.
  The TUI was exercised through model tests, not a live terminal. Not
  released: v0.10.0 awaits integration and release authorization.
