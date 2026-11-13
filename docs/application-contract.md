# Application contract

[简体中文](zh-CN/reference/application-contract.md) · [English](en/reference/application-contract.md) · 日本語

## Source-hosted one-line installers

`install.sh` (POSIX sh on Linux/macOS) and `install.ps1` (Windows PowerShell 5.1
or PowerShell 7) are served separately from `main` through
`raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/`. They consume the existing
Portable release archives, including v0.2.0, without changing release assets.
The one-line entry points are `curl -fsSL <install.sh URL> | sh` and
`irm <install.ps1 URL> | iex`. Source delivery follows `main`; version selection
controls the Portable release downloaded by the installer.

The default repository is `haoyan-dev/ssh-keyflow`. `--repository` / `-Repository`
or `SSH_KEYFLOW_REPOSITORY` override it. Latest resolution requires a stable,
nondraft, nonprerelease version; explicit `--version` / `-Version` may select a
prerelease. OS/CPU selection supports the same six targets as Portable archives.
Download the archive and SHA256SUMS, require one matching checksum, and reject
unexpected members, links and nonregular archive entries before installation.
Both executable and `THIRD_PARTY_NOTICES.txt` are installed.

Default locations are `$HOME/.local/bin` and
`$env:LOCALAPPDATA/Programs/ssh-keyflow`. Both installers retain the v0.1.0
ownership formats and add notice ownership (`notices-hash` in the POSIX receipt
directory, `noticesHash` in Windows receipt JSON). An intact legacy receipt with
no notices upgrades directly. Unowned, changed, or missing owned files stop
installation/removal. A failed replacement or receipt write restores previous
executable, notices and ownership hashes.

PATH changes require terminal consent or explicit `--add-to-path` /
`-AddToPath`. Piped source is never read as an answer. Noninteractive execution
without consent leaves profiles unchanged and prints a full launch path and
manual PATH command. PowerShell also treats host refusal to prompt under
`-NonInteractive` as declined consent. PowerShell argument parsing and
installation run in a child scope, including under `Invoke-Expression`, to
preserve caller variables, strict mode and error preferences.
Profile changes use same-filesystem staging and atomic replacement, preserving
existing metadata and recovery copies until ownership is recorded. A PATH setup
failure restores the profile and the previous executable, notices and receipt.
PATH stanzas remain idempotent and byte-owned; uninstall removes only the exact,
unique owned stanza, executable, notices and receipt. Existing v0.1.0 stanzas
remain owned across upgrades. SSH keys, state, SSH configurations and remote
registrations are never modified. Application shutdown is required before
update/removal; PATH profile persistence takes effect in a new shell.

Isolated tests use loopback HTTP through `SSH_KEYFLOW_RELEASE_URL` and
`SSH_KEYFLOW_API_URL`. Normal defaults use HTTPS GitHub endpoints.

## Portable release

`scripts/package_release.py --version <tag> [--output <empty-directory>]` builds Windows/Linux/macOS amd64/arm64 with CGO disabled, VCS stamping disabled, trimpath and injected `main.version`. Existing nonempty output is refused. `--binaries <directory>` instead packages six prebuilt OS_ARCH binaries for artifact tests. Assets are staged before publication to the output directory.

The release contains exactly six archives and `SHA256SUMS` (seven assets), with no installer. Windows archives are zip; Linux/macOS archives are tar.gz. Each contains only its platform executable and `THIRD_PARTY_NOTICES.txt`. Unix executable mode is 0755, notice mode is 0644, tar uid/gid are zero. The manifest has six unique archive basenames with SHA256, two spaces, and LF. Notices retain dependency license/copyright text including platform-specific modules and Unicode data; collection evidence is in `docs/third-party-notices.md`.

Portable means freely placed executables. Storage remains `<Home>/.ssh/ssh-keyflow`; moving/replacing/removing the executable neither migrates data nor changes SSH keys, state, config, remote registrations, PATH, profiles or installer receipts. End users need OS OpenSSH, not Go. They exit the application before manual replacement/removal and retain the notice document when redistributing the binary.

Old v0.1.0 installers require one archive member and therefore reject the new archives. Use the new source-hosted installer for intact v0.1.0 installations. Manual binary replacement also invalidates the old ownership digest, so updates/removal may be rejected. The installation guide (`docs/guides/installation.md`) documents manual maintenance and optional manual cleanup of the exact owned PATH stanza; no automatic repair of changed receipts/profiles is added.

CI runs Go and Portable artifact tests on three OSes plus actual six-target crossbuilds. Release workflow uploads the seven staged assets after CI; implementation itself does not tag/push/publish. Fixture artifact tests and crossbuilds do not establish native runtime behavior on every target.

## 現行機能の範囲（v0.5.0以降）

SSH KeyFlowはローカルの鍵管理とコピペによる接続設定支援に絞る。この契約は旧仕様の自動登録・解除、SSH接続確認、設定編集、鍵削除、番号メニュー、Tasks、再帰的設定探索に関する記述を置き換える。v0.5.0で導入する意図的な非互換変更であり、v0.4.0以前のバイナリに遡って適用しない。配布・インストーラー・公開手順は維持する。

公開鍵の配置・解除は利用者の手動作業とする。アプリはSSHプロセス、リモート認証・プローブ、サーバー導入・起動・ファイアウォール変更用スクリプトを実行・生成しない。ADR 0002により、利用者が対象端末へ貼り付けて実行する公開鍵配置・ローカルconfig追記スクリプトは生成する。新しい鍵・関連付け・ガイドは登録・接続について未確認とし、手動成功の記録操作も設けない。過去の成功記録は履歴として保持し、現在の接続可否へ読み替えない。

## CLI

`cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner, RunTUI}) int` が入口となり、`cmd/ssh-keyflow` が終了コードを返す。Homeの解決は必要な操作まで遅延する。

| コマンド | 入力 |
| --- | --- |
| `keys list` | 任意の `--json`。鍵と未完了のローカル作成IDを表示 |
| `keys create` | 必須 `--name`。Ed25519鍵を作成 |
| `keys import` | 必須 `--name`、`--public`。任意の `--private` で鍵ペアを検証。省略時は公開鍵のみ |
| `keys public` | 必須 `--key`。公開鍵のアルゴリズムとblobを表示し、ファイルのコメントは含めない |
| `keys remove` | 必須 `--key`。管理から鍵と全関連付けを削除し、鍵ファイルを保持 |
| `hosts list` | 任意の `--json`。接続先と新しい関連付け、ホスト鍵の固定状態を表示 |
| `hosts add` | 必須 `--alias`、`--hostname`、`--user`、`--os`、既存鍵の `--key`。`--port` は22が既定。Windowsでは `--windows-account standard\|administrator` が必要 |
| `hosts associate` | 必須 `--host`、`--key`。ローカルの関連付けを保存 |
| `hosts dissociate` | 必須 `--host`、`--key`。指定した既存の関連付けだけを解除 |
| `hosts remove` | 必須 `--host`。管理から接続先と全関連付けを削除 |
| `hosts guide` | 必須 `--host`、`--key`。任意の `--local-os`、`--remote-os`、`--windows-account`、`--identity-file`。読み取り専用の配置・解除・接続設定ガイド |
| `hosts pin` | 必須 `--host`。任意の `--file`、`--yes`、`--replace`。接続先で表示したホスト公開鍵を固定（ADR 0006） |
| `hosts unpin` | 必須 `--host`。固定したホスト公開鍵を解除 |
| `resume` | 必須 `--id`。未完了のローカルCreation IDのみ |
| `tui` | 追加引数なし。入力・出力とも実端末が必要 |

`--os` は `linux` / `macos` / `windows` を明示する。接続先は固定IDまたは大文字小文字を区別しない別名、鍵は固定IDまたは一意の完全一致管理名で解決する。曖昧・不在の対象は拒否する。管理済み接続先の別名重複は拒否するが、SSH config内の別名は探索しない。接続先追加は既存鍵を要求し、同時に新規鍵を作成しない。

`register`、`unregister`、`registrations list`、`keys delete`、`--new-key`、`--auth`、`--identity`、`--configure` は提供しない。旧リモート・削除IDの `resume` は実行せず手動確認を求める。

ADR 0004により管理メタデータの削除・関連付け解除を追加する。`remove-key` / `remove-host` / `dissociate` は共通バックエンドのOperationで、実端末と完全一致 `yes` の確認、ロック下での対象再解決、原子的保存・直前状態バックアップを使う。鍵・接続先の削除はその全関連付けを除き、解除は指定した1組だけを除く。存在しない組は引数エラー。鍵ファイルの検証や外部プロセスを必要とせず、欠損した鍵の管理情報も整理できる。未完了の旧鍵削除がある鍵は手動確認を要求し、管理からの削除を拒否する。

削除した鍵・接続先はv2の任意配列 `removed_keys` / `removed_hosts` に保持する。旧登録・作成結果・オンボーディング・設定履歴の参照はアーカイブ先のIDでも検証し、履歴自体を書き換えない。現行の関連付けは管理対象の鍵・接続先だけを参照できる。アーカイブの指紋・パス・別名は再利用でき、再取り込みと接続先再追加は新しいIDを作る。list・公開鍵・ガイド・選択候補は削除済みレコードを対象としない。鍵ファイル・SSH設定・リモート登録は変更せず、アクセス失効を意味しない。旧バイナリはアーカイブ配列を含む状態を拒否するため、ダウングレードには事前に保管した状態を使う。TUIは一覧・詳細の `d` で削除、`u` で選択対象の既存関連付けだけを候補にして解除し、既存の実端末確認へ引き渡す。

変更操作はストレージ初期化・プロセス実行より前に実端末入力を要求し、共通の `app.Run` へ明示的なRequestと確認関数を渡す。前後の空白を除いた完全一致 `yes` のみ承認する。`ReadConfirmation` は1行を1バイトずつ読み、後続のネイティブパスフレーズ入力を先読みしない。秘密値や無条件承認のフラグは設けない。唯一の例外は `hosts pin --yes` で、異なる鍵が固定されていない場合だけ確認を省き、置き換えは承認しない。

終了コードは成功0、実行失敗・取消1、引数・使用法エラー2。曖昧・不在の対象も引数エラーで、stderrに該当コマンドのusageを出す。実行時診断にはusageを付けない。`--help` と `--version` はHome解決・依存コマンド確認・保存を行わない。版表示は `SSH KeyFlow <version>`。

引数なしで入出力のいずれかが非端末ならヘルプを出して0で終了する。両方が実端末ならTUIを開く。明示的な `tui` の端末要件違反は2を返す。list・公開鍵・ガイドはリダイレクトでき、書き込み可能性やSSHコマンドのプローブを行わない。

listの通常出力は表形式。JSONは改行で終わる1オブジェクトで、`keys list` は `keys` / `creations`、`hosts list` は `hosts` / `associations` / `host_key_pins` を返す。`host_key_pins` は管理対象の接続先ID → `[{type, fingerprint}]` で削除済み接続先の固定は含めず、固定ファイルを読めない場合は省略し理由をstderrに出す。通常出力の `HOST KEYS` 列は `pinned (N)` / `not pinned` / `unknown` を示す。配列はnullでなく空配列とし、フィールド名は保存スキーマに一致する。要確認情報はstderrだけに出し、検証できた状態レコードは読み取り可能とする。不正な状態は1を返してJSONを出さず元ファイルを保持する。公開鍵・ガイドはコピー用のテキストをstdoutへ出す。出力失敗は実行失敗とする。

## 共通バックエンドと手動ガイド

`app.Request` のOperationは `list` / `create` / `import` / `add` / `associate` / `dissociate` / `remove-key` / `remove-host` / `guide` / `public` / `resume` / `pin-host` / `unpin-host`。`app.Run` は明示的なRequestを要求し、旧番号メニューを起動しない。変更対象はロック下でも再解決・再検証する。

`app.Snapshot(Options)` は `View{State, Review}` を返し、保存領域の初期化・依存コマンドのプローブ・旧記録の変更を行わない。`app.PublicKey(home, keyRef)` と `app.Guide(home, hostRef, keyRef)` は対象鍵の同一性を検証してテキストを返す。`app.PrepareGuide` は同じ検証を通した `SetupGuide{Text, Steps}` を返し、各 `GuideStep` の `Title` / `Instructions` / `Script` をTUIで使う。ガイド表示は保存済みの関連付けを前提とせず、新しい関連付けも保存しない。

`app.PrepareGuideWithOptions(home, hostRef, keyRef, GuideOptions)` は同じ検証に加え、ガイド専用の `LocalOS` / `RemoteOS` / `WindowsAccount` / `IdentityFile` を受け取る。`app.Request.GuideOptions` も同じ選択を共通バックエンドに渡す。OSは `linux` / `macos` / `windows`、省略時は実行元OS・保存済み接続先OSとアカウントを使う。異なる接続先OSへ変更すると元のWindowsアカウント種別は引き継がない。Windowsではstandard / administratorを必須とし、非Windows宛てのアカウント指定は拒否する。TUIでは接続先と鍵の選択後、これらの既定値をフォームに表示し変更できる。選択は保存しない。管理済み秘密鍵は形式ヘッダーだけを読み、パスフレーズ付きまたは判定不能なら `AddKeysToAgent yes` とローカルOS向けssh-agentステップ（config追記と接続の間）を加える。`IdentityFile` 指定時は検査せず判定不能として扱う（ADR 0005）。

ガイドは公開鍵のアルゴリズムとblob、指紋、配置先、手動解除案内、ローカルファイルパスを含む。選択した接続先OS・アカウント種別に応じて、端末へ貼り付ける公開鍵配置スクリプトを最初のステップにする。Linux/macOSはPOSIXシェル、WindowsはPowerShell。既存の鍵の内容を保持し、同じ公開鍵がすでに含まれる場合は追記しない。改行のない既存の最終行とも分離して追記する。シンボリックリンク・reparse pointは拒否する。POSIXはログインユーザー名、Windows標準アカウントはユーザー名、Windows管理者アカウントは昇格済み端末を確認する。標準OpenSSH構成の配置先は次のとおり。

| 接続先 | 配置先と確認事項 |
| --- | --- |
| Linux / macOS | ログインユーザーの `~/.ssh/authorized_keys`。既存行を保持し、ユーザー所有の `.ssh` は700、ファイルは600 |
| Windows標準アカウント | `%USERPROFILE%\.ssh\authorized_keys`。既存行を保持し、書き込み権限を当人・SYSTEM・Administratorsに限定 |
| Windows管理者アカウント | `%ProgramData%\ssh\administrators_authorized_keys`。管理者ログイン間で共有するため、他の鍵を保持しSYSTEM・AdministratorsのACLを確認 |

カスタム構成では実際の `AuthorizedKeysFile` / `Match` を利用者が確認する。SSHサーバーとネットワークは事前に利用可能であることを前提とする。解除時は別の有効なアクセス手段を残し、同じファイルから一致する公開鍵の行だけを手動で取り除く。解除の成否は検証しない。

秘密鍵パス、または公開鍵のみの鍵では保存済みの公開鍵パスを使い、クライアント用の `HostName` / `User` / `Port` / `IdentityFile` / `IdentitiesOnly yes` を含むHostスニペットを `~/.ssh/config` に追記するスクリプトを、配置ステップの次（ホスト鍵の表示・固定ステップがあればその次）に置く。選択したローカルOSに従い、WindowsはPowerShell、Linux/macOSはPOSIXシェルを使う。実行元と異なるローカルOSでは、対応する秘密鍵の端末内パスをIdentityFileで必須とする。指定OSの絶対パスまたは ~/pathを受け付け、制御文字・相対パス・異なるOSのパスは拒否する。指定パスのファイルは読み取らず、鍵の転送・パス変換・一致の保証はしない。管理済み元ファイルの同一性検証は維持し、公開鍵のみの鍵をIdentityFile指定で鍵ペアへ変更しない。既存内容を保持し、グループ化されたHost行を含め、同じ明示的な別名があれば停止する。Include・ワイルドカード・Matchの解決や優先順位の保証は行わない。PowerShellの既存ファイルはUTF-8を要求し、UTF-16・不正なUTF-8は追記前に拒否する。標準アカウント用のWindows公開鍵配置では `.ssh` ディレクトリと鍵ファイルのDACLを本人・SYSTEM・Administratorsに限定し、管理者用では共有ファイルだけのDACLをSYSTEM・Administratorsに限定する。所有者と監査情報を保持し、Windows PowerShell 5.1 / PowerShell 7それぞれの.NET ACL APIを使う。

最後のステップは、前のステップを実行した後にローカル端末で使う `ssh 別名` 接続コマンドとする。接続コマンドはPOSIXシェル / PowerShell共通とし、秘密鍵パスをコマンド引数へ重複展開しない。OpenSSH自身の展開も考慮し、`%`はIdentityFile内でエスケープする。`${...}`を含むパスではリテラル参照を保証できないため、公開鍵の配置案内を維持し、理由を示してconfig追記・接続ステップを省略する。公開鍵のみの鍵では、OpenSSHがSSHエージェント（FIDO鍵等）から対応する鍵だけを使うよう公開鍵ファイルを `IdentityFile` にし、対応する秘密鍵をエージェントに読み込む必要がある旨を示す。`AddKeysToAgent` とssh-agentステップは加えない。実行元と異なるローカルOSでは `--identity-file` の指定を求め、未指定なら配置ステップだけを示す。どのスクリプト・コマンドもアプリは実行しない。

アプリ自身は通常の `~/.ssh/config`、過去の専用config、Include、バックアップを編集・再生成・削除しない。コピーしたconfig追記スクリプトを利用者が実行した場合だけ通常のconfigを追記する。再帰的探索、ワイルドカード・Include優先順位の解析は提供しない。利用者が貼り付け先と既存Hostの重複・優先順位を確認する。既存Includeが過去の専用configを有効にしている場合もそのまま残る。

## ホスト鍵の固定

ADR 0006により、接続先のホスト公開鍵をアプリ所有の `<Home>/.ssh/ssh-keyflow/known_hosts` に固定する。利用者は接続先自身の端末で `cat /etc/ssh/ssh_host_*_key.pub`（POSIX）または管理者として開いたPowerShellで `Get-Content "$env:ProgramData\ssh\ssh_host_*_key.pub"`（Windows）を実行し、出力を貼り付ける。アプリはネットワーク接続を開かず、`~/.ssh/known_hosts`・SSH config・`sshd_config` を編集しない。

`hosts pin` は実端末では表示コマンドを示して空行までの貼り付けを読み、非端末では標準入力、または `--file` を読む。入力は64 KiB・16鍵まで。`<type> <base64> [comment]` 形式で、`ssh-ed25519` / `ecdsa-sha2-nistp256` / `ecdsa-sha2-nistp384` / `ecdsa-sha2-nistp521` / `ssh-rsa` の正規base64かつblob内のアルゴリズムが一致する行だけを受け付ける。コメントは捨て、その他の行は無視した行として表示し、有効な鍵を妨げない。各鍵の種類とSHA256指紋を表示し、完全一致 `yes` で保存する。同じ鍵の組の再固定は書き込まない。非端末入力には `--yes` が必要で、`--yes` は異なる鍵が固定されていない場合だけ確認を省く。異なる鍵が固定済みなら新旧の指紋と警告を示し、`--replace` と実端末での `yes` を要求する。`hosts unpin` は実端末での確認後に、その接続先の行だけを除く。いずれも管理対象の接続先だけを対象とし、接続先削除は固定を残すため、削除確認で先に `hosts unpin` するよう案内する。

固定ファイルは1鍵1行の `ssh-keyflow-<host-id> <type> <base64>` でLF終端とする。書き込みは状態と同じロック・原子的置換・シンボリックリンク拒否・権限検査を使い、他の接続先の行の順序と内容を保つ。読み込みは厳格で、不正な行・CRLF・タブ・重複・コメント・マーカーを含むファイルでは固定・解除を止め、自動修復しない。状態スキーマは変更しない。

ガイドは生成時に固定状態を読む。ローカルOSが実行元と同じ場合だけ、未固定なら配置ステップの後に接続先の「ホスト鍵の表示」とローカルの「ホスト鍵の固定」（`ssh-keyflow hosts pin --host <id>`）を加え、固定済みならHostスニペットの `IdentitiesOnly yes` の後に `HostKeyAlias ssh-keyflow-<host-id>`、スラッシュ区切りの絶対パスを引用した `UserKnownHostsFile`、`StrictHostKeyChecking yes` を加える。パスはIdentityFileと同じエスケープを使い、制御文字・`${` を含む場合は行を生成しない。固定ファイルを読めない場合は `Host keys: unknown` と示し、固定に関する行もステップも加えない。未固定の接続先のスニペットは変えない。既存のHostには利用者が3行を手動で加える。

## TUI

`tui.Run(ctx, app.Options)` をCLIのTUI入口とし、Keys / Destinationsの2画面を中心にする。鍵作成・取り込み、既存鍵を使った接続先追加、関連付けと解除、管理対象からの削除、公開鍵とガイドの表示・コピーを提供する。新しい鍵は独立して作成する。旧Registrations / Tasks、リモート認証・設定編集のフォーム、鍵ファイルの削除操作は提供しない。Tabで画面切替、c/iで鍵作成・取り込み、aで接続先追加、lで関連付け、uで既存関連付け解除、dで管理対象からの削除、p/gで公開鍵・ガイド表示、sで選択したローカル作成を再開する。`t` / `T` で選択した接続先のホスト鍵を固定 / 解除し、ネイティブ端末へ引き渡して貼り付けと `yes` 入力を行う（固定はReplaceを指定し、異なる鍵があれば新旧の指紋を示す）。接続先詳細は `Host keys: pinned (N)` / `not pinned` / `unknown` を表示する。表示中のテキストはyでOSC52コピーを要求し、端末非対応時はCLIの公開鍵・ガイド出力を利用するよう案内する。公開鍵は1行のまま扱い、TUIの折り返しを改行としてコピーしない。vの技術詳細で旧記録を読み取り専用表示する。最小端末は32列×12行。キー操作の詳細は画面内ヘルプを正とする。

ガイドは第1ステップから表示し、← / →または `b` / `n` で切り替える。`y` は選択中の `Script` だけをコピーし、説明文・他の端末向けスクリプトを混ぜない。各ステップにローカル / 接続先、OS、端末、必要なアカウントと権限を明示する。Helpから戻った場合も選択ステップを保つ。F5再読込時は履歴内も含めてスクリプトを無効化し、ステップ切替でも古いコピーを復活させない。

CLIと同じ検証・保存処理を使い、読み取りやガイド表示で実行を開始しない。変更には対象と影響の確認を伴う。必要時はネイティブ `ssh-keygen` へ端末を引き渡し、パスフレーズをTUIや状態に保持しない。失敗・中断・取消を成功と表示せず、保存済み進捗を巻き戻したと扱わない。

## 保存スキーマと互換性

保存領域は `<Home>/.ssh/ssh-keyflow`。`state.json`、生成鍵用 `keys/`、更新式の `state.json.bak`、一時的な `state.lock`、最初の固定時に作るホスト鍵の固定ファイル `known_hosts` を使う。状態がない場合はv2の空状態を返すだけで、読み取り時にファイルを作成しない。

v1とv2を読む。v1は読み込みだけでは内容・バージョン・バックアップを変更せず、最初の成功した状態保存でv2になる。v2の `associations` は `{host_id, key_id}` の組で、有効な鍵と接続先を参照し、組の重複を認めない。旧 `registrations` / `history` を関連付けへ変換せず、その登録・接続状態も変更しない。

| レコード | 保存フィールド |
| --- | --- |
| Key | `id`, `name`, `fingerprint`, `public_path`, 任意の `private_path` / `private_sha256`, `connection_check`, `origin`, `imported_at`, 任意の `created_at` |
| Host | `id`, `alias`, `hostname`, `user`, `port`, `os`, 任意の `windows_account` / 旧 `config_key_id` |
| Association | `host_id`, `key_id` |
| Creation | `id`, `key_id`, `name`, `private_path`, `public_path`, `created_at`, `stage`, 任意の `private_sha256` / `fingerprint` |

必須配列は `keys` / `hosts` / `registrations`。任意の `creations` / `associations` / `removed_keys` / `removed_hosts` と、旧 `onboardings` / `history` / `configs` / `config_edits` / `deletions` / `deleted_keys` を保持する。過去の固定ID・登録履歴・設定ダイジェスト・中断意図・削除済み鍵情報と参照を維持し、新規操作の成功・完了へ書き換えない。旧履歴専用の閲覧コマンドは設けないが、保存データを残す。v2保存ではJSONを再整形するため、過去レコードの値を保つことと状態ファイルのバイト列不変は区別する。

鍵IDは固定のランダム `key-<128-bit hex>`、指紋はOpenSSH公開blobのSHA256形式、パスは絶対パス。インポートは `origin: imported` と非ゼロのUTC RFC3339Nano `imported_at`、作成は `origin: created` と `created_at` を保存する。秘密鍵ダイジェストはパスフレーズを求めず置換を検出するためのもので、秘密鍵の内容を保存しない。新しい鍵の `connection_check` は `unverified` とし、関連付けには成功状態のフィールドを設けない。

不正・後続付きJSON、重複・未知フィールド、未対応バージョン、必須配列の欠落、不正メタデータ、重複ID・別名・指紋・パス、矛盾や孤立した参照は読込時に拒否し、自動復元・上書きしない。旧リモート操作・設定編集の未完了記録は読み取り専用で自動再開しない。未完了の旧削除は対象鍵だけを使用不可にして手動確認を求め、その記録だけを理由に他の鍵の操作を止めない。

`state.json.bak` は保存直前の検証済み `state.json` の正確なバイト列である。各保存で更新するため、移行後の同一操作内でもv2へ置き換わり得る。恒久的なv1移行バックアップではない。**ダウングレードが必要になり得る利用者は、v0.5.0以降で最初の変更操作を行う前に、アプリを終了してv1状態を別途保管する。** 旧アプリはv2を読めないため、保管したv1状態からの手動復元と移行後の追加データの確認が必要になる。読み取り時に移行用コピーは作らず、既存の鍵・config・Include・バックアップも移行に伴って書き換えない。

## ローカル鍵の安全条件と保存

作成はネイティブ `ssh-keygen -t ed25519` を使い、空のパスフレーズも許容する。生成先は `keys/id_ed25519_<安全化した管理名>_<UTC時刻>` と衝突時の数値suffix。両パスと保存済み予約を調べ、確認後にも再検査する。既存ファイルを上書き・自動採用しない。

副作用前に作成意図・固定ID・パスを保存し、`planned` / `pending` / `complete` の進捗を維持する。再開は同じパスを使い、ファイルなしなら生成、秘密鍵のみならネイティブ `ssh-keygen -y` で導出して欠けた公開鍵を排他的に新規作成する。ペアがあれば再生成せず照合する。公開鍵のみ、空・非通常ファイル、固定済み指紋・ダイジェストの変更、観測済みファイルの消失は要確認で停止する。部分ファイルを自動削除せず、未完了の作成があれば別の鍵ペアを新規作成しない。

取り込みは明示した公開鍵を `ssh-keygen -l -E sha256` でも検証し、秘密鍵指定時は `ssh-keygen -y` の公開鍵指紋と一致させる。元ファイルは移動・書き換えない。保存前に指紋と秘密鍵ダイジェストを再検査し、欠落・置換を成功と扱わない。ローカルファイル削除や自動整理のAPIは設けない。

変更全体は `state.WithLock` の排他 `state.lock` で保護し、確認・作成・保存中も保持する。LockedStoreはロック外で使わず、ロックを入れ子にしない。既存ロックを自動削除せず、変更時だけ書き込み可能性を確認する。状態は検証後、同一ディレクトリの一時ファイルへ書き込み・同期し、直前の正当な状態を原子的に `.bak` へ保存してから `state.json` を置き換える。バックアップ失敗時は現在状態を置き換えない。利用者は操作中に対象鍵パスを並行編集しない。

## 検証と公開

CONTRIBUTINGのGoテスト・vet、Portable / インストーラーテスト、6ターゲットビルドを適用する。テストは一時ホームと代替プロセスを使い、実ユーザーのSSHデータ、PATH、プロファイル、接続先を変更しない。v1非変更読込、v2移行と正確な直前バックアップ、旧履歴保持、影響鍵だけの削除ブロック、読み取りの非変更性、鍵作成再開と同一性検証を確認対象に含める。

結果・対象コミット・環境・未検証範囲は `docs/verification.md` に記録する。過去の自動登録・解除テスト記録は履歴であり、現行機能の契約や新実装の検証結果ではない。クロスビルドは各OS実機でのTUI・ネイティブ鍵作成の実行証拠と区別する。ガイド表示は実際の接続先設定・登録・解除・接続成功の検証ではない。

公開はmainへ統合・検証済みのコミットに対する注釈付き `vMAJOR.MINOR.PATCH` タグを使う。準備PRの統合から、最終候補の検証、タグ作成・公開、公開後の検証記録PRの統合までの手順・完了条件・承認範囲は [AGENTS.mdのTags and releases](../AGENTS.md#tags-and-releases) を正とする。準備記録は検証したソースのコミットを参照し、最終タグのコミットはタグ・リリース情報と公開後の記録に残す。監査文書の追加でmainがタグより先へ進んでも、公開済みタグはビルド元を指したまま保持する。`v*` タグのpushはCI後の自動公開を開始する。プレリリースはlatestにせず、安定版latestはより新しい安定版へ進める。公開済みタグ・バイナリアセットは不変で、修正は新バージョンで出す。公開後に配布物・チェックサム・内部バージョン・タグ・ソースコミットの一致を確認する。開発ビルドは引き続き `dev` と表示する。
