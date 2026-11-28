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

## 現行機能の範囲（v0.11.0以降）

SSH KeyFlowはこの端末のSSH鍵ペアの管理に絞ったCLIである。Ed25519鍵の作成、既存の鍵ペアの取り込み、中断した作成の再開、管理からの削除、一覧、コピー用の公開鍵表示を提供する。ADR 0007により、接続先（`hosts`）と関連付け、手動設定ガイド・スニペット・ssh-agentステップ、ホスト鍵の固定、端末UI、公開鍵のみの取り込みをv0.11.0で廃止した。意図的な非互換変更であり、v0.10.0以前のバイナリに遡って適用しない。配布・インストーラー・公開手順は維持する。

アプリはSSHプロセス、リモート認証・プローブ、リモート登録、スクリプト生成を行わない。`~/.ssh/config`、`~/.ssh/known_hosts`、Include、過去の専用configを読まず、編集・再生成・削除しない。v0.10.0以前が作った `<Home>/.ssh/ssh-keyflow/known_hosts`（ホスト鍵の固定ファイル）と、利用者がガイドから追記したSSH configの行も、そのまま残す。公開鍵を接続先へ配置するのは利用者の手動作業で、`keys public` の出力をコピーして使う。

`main` には、ADR 0008のフリート同期フェーズ1（`node` / `fleet` コマンド、状態スキーマv4）が未リリースの機能として含まれる。範囲と規則は[フリート同期フェーズ1](#フリート同期フェーズ1未リリース)に記載し、リリース済みの版には適用しない。フェーズ1もSSH接続・リモート登録を行わず、SSHファイルを書き込まない。

## CLI

`cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner}) int` が入口となり、`cmd/ssh-keyflow` が終了コードを返す。Homeの解決は必要な操作まで遅延する。

| コマンド | 入力 |
| --- | --- |
| `keys list` | 任意の `--json`。管理対象の鍵と未完了のローカル作成を表示 |
| `keys public` | 必須 `--key`。公開鍵のアルゴリズムとblobを1行で表示し、ファイルのコメントは含めない |
| `keys create` | 必須 `--name`。Ed25519鍵ペアを作成 |
| `keys import` | 必須 `--name`、`--public`、`--private`。既存の鍵ペアを検証して管理対象にする。ファイルは移動しない |
| `keys remove` | 必須 `--key`。管理対象から外し、鍵ファイルを保持 |
| `resume` | 必須 `--id`。未完了のローカル作成だけを再開 |

`main` の `node` / `fleet` コマンドは[フリート同期フェーズ1](#フリート同期フェーズ1未リリース)を参照。`keys remove` は、ノード記述の接続元鍵に使われている鍵を実行失敗（1）として拒否し、先に `node set` で変更するよう案内する。

鍵は固定IDまたは一意の完全一致管理名で解決する。曖昧・不在の対象は拒否する。`hosts`、`tui`、`register`、`unregister`、`registrations list`、`keys delete`、`--public-only`、`--new-key`、`--auth`、`--identity`、`--configure` は提供せず、指定すると使用法エラーになる。

変更操作（create / import / remove / resume）はストレージ初期化・プロセス実行より前に実端末入力を要求し、共通の `app.Run` へ明示的なRequestと確認関数を渡す。前後の空白を除いた完全一致 `yes` のみ承認する。`ReadConfirmation` は1行を1バイトずつ読み、後続のネイティブパスフレーズ入力を先読みしない。秘密値や無条件承認のフラグは設けない。

終了コードは成功0、実行失敗・取消1、引数・使用法エラー2。曖昧・不在の対象も引数エラーで、stderrに該当コマンドのusageを出す。実行時診断にはusageを付けない。`--help` と `--version` はHome解決・依存コマンド確認・保存を行わない。版表示は `SSH KeyFlow <version>`。引数なしの起動は端末の種類にかかわらずヘルプを出して0で終了する。

list・公開鍵はリダイレクトでき、書き込み可能性やSSHコマンドのプローブを行わない。listの通常出力は表形式。JSONは改行で終わる1オブジェクトで、`keys` と `creations` の2フィールドだけを持つ。配列はnullでなく空配列とし、要素のフィールド名は保存スキーマに一致する。要確認情報と移行の予告はstderrだけに出し、検証できた状態レコードは読み取り可能とする。不正な状態は1を返してJSONを出さず元ファイルを保持する。`keys public` は末尾に改行を付けたコピー用テキストをstdoutへ出す。出力失敗は実行失敗とする。

## 共通バックエンド

`app.Request{Operation, Name, PublicPath, PrivatePath, Key, Resume}` のOperationは `create` / `import` / `remove` / `resume`。未知のOperationは引数エラーとする。`app.Run` は明示的なRequestを要求し、変更対象をロック下でも再解決・再検証する。`remove` は `ssh-keygen` を確認せず、欠損した鍵ファイルの管理情報も整理できる。

`app.Snapshot(Options)` は `View{State, Review}` を返し、保存領域の初期化・依存コマンドのプローブ・移行を行わない。Reviewには鍵ファイルの欠落・置換と、旧スキーマの状態を次の変更で移行する旨の予告を含める。`app.PublicKey(home, keyRef)` は対象鍵の指紋を再検査し、`<type> <base64>` を返す。

管理からの削除は実端末と完全一致 `yes` の確認後、鍵のレコードだけを状態から除く。鍵ファイル・SSH設定・接続先の公開鍵登録は変更せず、アクセス失効を意味しない。削除したレコードは保持しない。同じ鍵ペアは再取り込みで新しいIDとして管理できる。

## 保存スキーマと互換性

保存領域は `<Home>/.ssh/ssh-keyflow`。`state.json`、生成鍵用 `keys/`、更新式の `state.json.bak`、一時的な `state.lock`、旧スキーマから移行したときの `state.v<N>-<UTC時刻>.json` を使う。状態がない場合は空状態を返すだけで、読み取り時にファイルを作成しない。

v0.11.0が書き込む唯一のスキーマはv3で、`version`、必須配列 `keys`、任意配列 `creations` だけを持つ。`main`（フェーズ1、未リリース）はv4だけを書き込む。v4はv3のフィールドに任意の `node` と `fleet` を加え、`version` は4とする（[状態スキーマv4](#状態スキーマv4)）。

| レコード | 保存フィールド |
| --- | --- |
| Key | `id`, `name`, `fingerprint`, `public_path`, `private_path`, `private_sha256`, `origin`, `created` なら `created_at`、`imported` なら `imported_at` |
| Creation | `id`, `key_id`, `name`, `private_path`, `public_path`, `created_at`, `stage`（`planned` / `pending`）, 任意の `private_sha256` / `fingerprint` |

鍵IDは固定のランダム `key-<128-bit hex>`、指紋はOpenSSH公開blobのSHA256形式、パスは絶対パス。インポートは `origin: imported` と非ゼロのUTC RFC3339Nano `imported_at`、作成は `origin: created` と `created_at` を保存する。秘密鍵ダイジェストはパスフレーズを求めず置換を検出するためのもので、秘密鍵の内容を保存しない。完了した作成は、鍵の追加と同じ保存で `creations` から除く。

v3の読み込みは厳格で、不正・後続付きJSON、重複・大文字小文字違いのフィールド、未知フィールド、未対応バージョン、`keys` の欠落、不正メタデータ、重複したID・指紋・パスを拒否し、自動復元・上書きしない。v4の読み込みも同じ規則に従う。

### v1 / v2からの移行

v1とv2（v0.10.0以前）は読み込み時にメモリ上でv3へ変換する。読み込みだけではファイルを変更しない。旧スキーマの読み込みも厳格で、未知フィールド、型の誤った配列、後続付きJSON、必須配列（`keys` / `hosts` / `registrations`）の欠落を拒否する。変換では秘密鍵パスを持つ鍵と未完了のローカル作成だけを引き継ぎ、次のものは引き継がない。

- 接続先・関連付け・アーカイブ済みの削除レコード（`hosts` / `associations` / `removed_keys` / `removed_hosts`）
- 旧登録・接続確認・設定・SSH config編集・オンボーディング・削除の記録と履歴（`registrations` / `history` / `configs` / `config_edits` / `onboardings` / `deleted_keys` / `deletions`）
- 公開鍵のみの鍵と、未完了の旧鍵削除がある鍵
- 鍵の `connection_check`、接続先の `config_key_id` などの旧フィールド

予告には、空でない記録の種類ごとの件数を挙げる。未完了のSSH config編集は対象パスを個別に挙げ、そのファイルを手動で確認するよう案内する。`keys list` は移行前にこの予告をstderrへ出す。変更コマンド（create / import / remove / resume）は、ロック取得後、確認プロンプトの前に同じ予告を表示するため、確認は移行も含めた承認になる。最初の成功した状態保存は、元の `state.json` のバイト列を同じディレクトリの新しい `state.v<N>-<UTC時刻>.json`（衝突時は数値suffix）へ排他的に書き出してから、`.bak` 更新とv3の保存を行う。アーカイブの作成に失敗した場合は状態を変更しない。前回の試行で同じバイト列のアーカイブが既にあれば、再試行はそれを再利用し、同一のアーカイブを増やさない。アーカイブは上書き・自動削除しないため、引き継がない記録もディスク上に残る。

旧バイナリはv3を読めない。ダウングレードする場合は、アプリを終了し、アーカイブを `state.json` として手動で戻す。移行後に作成・取り込みした鍵は戻した状態に含まれない。

## フリート同期フェーズ1（未リリース）

この節は `main` の実装を記述し、フェーズ1を含むMINORリリースまでどの公開版にも含まれない。設計判断は[ADR 0008](adr/0008-signed-fleet-roster.md)。フェーズ1は、フリートルート鍵で署名した名簿をファイルとして別のノードへ渡し、受け取ったノードが固定したルート公開鍵で検証するところまでを扱う。`authorized_keys`・known_hosts・SSH configの書き込み、ネットワーク経由の参加・同期、失効、鍵の更新は含まない。

### コマンド

| コマンド | 入力 | 効果 |
| --- | --- | --- |
| `node set` | 必須 `--name`、`--roles`（`client` / `remote` の1つ以上をカンマ区切りの1値で）。接続元は必須 `--key`。接続先は必須 `--addresses`（カンマ区切りの1値）、任意 `--port`、`--login-account`、`--sshd-config`。任意 `--dry-run` | この端末のノード記述を作成・置換する。初回にノードIDを生成し、以後は保持する |
| `node show` | 任意 `--json` | この端末のノード記述と、管理ノードかどうかを表示 |
| `node export` | 必須 `--out <dir>` | ノード記述ファイル `node.json` を書き出す |
| `fleet init` | 必須 `--login-account`、任意 `--dry-run` | 管理ノード用。フリートルート鍵を作成・固定し、このノードだけを含む名簿の版1に署名する |
| `fleet sign` | 任意 `--dry-run` | 管理ノード用。この端末の現在のノード記述で次の版に署名する。変更がなくても `created_at` を更新する |
| `fleet add <dir>` | 任意 `--dry-run` | 管理ノード用。`<dir>/node.json` のノードを追加・置換し、次の版に署名する |
| `fleet show` | 任意 `--json`、`--max-age <日数>`（1〜36500、既定30） | 受け入れた名簿、固定したルートの指紋、版、経過日数を表示 |
| `fleet export` | 必須 `--out <dir>` | 受け入れた名簿・署名・ルート公開鍵の3ファイルを書き出す。管理ノード以外でも使え、パスフレーズを求めない |
| `fleet accept <dir>` | 任意 `--max-age <日数>`（既定30）、`--dry-run` | 名簿を検証し、確認後に受け入れる。初回はルート公開鍵を固定する |

リスト値のフラグ（`--roles`、`--addresses`）は1つのカンマ区切り値で指定し、繰り返すと使用法エラーとする。ノード名は小文字のDNSラベル1つ（63文字以下）、アカウント名は英数字・`.`・`_`・`-` の64文字以下、アドレスは小文字のDNS名か正規形のIPアドレス（ポート・角括弧なし、1〜16件、重複不可）。接続元の役割には `--key` が必須で、それ以外では指定できない。`--addresses`・`--port`・`--login-account`・`--sshd-config` は接続先の役割でだけ指定できる。`--port 22` は省略と同じに扱う。接続元鍵は作成・取り込みのどちらの管理対象の鍵でもよい。接続先のホスト公開鍵は `sshd_config`（既定はUnixで `/etc/ssh/sshd_config`、Windowsで `%ProgramData%\ssh\sshd_config`）の `HostKey` 行と Include から読み、`HostKey` 行がなければ既定のディレクトリ（`/etc/ssh` または `%ProgramData%\ssh`。`--sshd-config` の指定に関係しない）の `ssh_host_{rsa,ecdsa,ed25519}_key.pub` を読む。相対パスの Include も既定のディレクトリを基準とする。存在しない公開鍵は飛ばして計画に表示し、1つも読めなければ失敗する。秘密のホスト鍵は開かない。`--sshd-config` を指定した場合、そのファイルがなければ失敗する。

### 端末・確認・終了コード

終了コードはv0.11.0と同じで、成功0、実行失敗・取消1、使用法エラー2。存在しない・読めない `<dir>`、上記のフラグ値の誤り、不在・曖昧な `node set --key` は使用法エラーとする。状態を変更するコマンド（`node set`、`fleet init`、`fleet sign`、`fleet add`、`fleet accept`）は計画を表示してから実端末で完全一致 `yes` を求め、承認前には副作用を起こさない。`--dry-run` は確認以外の計画（`ssh-keygen` の版確認、`fleet accept` の署名検証と拒否の検査を含む）を行って結果を表示し、確認・書き込み・ストレージ初期化をしない。計画が成功すれば0、検査で拒否すれば1で終了する。実端末は不要。`fleet init`・`fleet sign`・`fleet add` は `ssh-keygen` のパスフレーズ入力のため、`node set` は確認のため、計画の前に実端末を要求する。`fleet accept` は計画の後、確認が必要な場合だけ実端末を要求するため、受け入れ済みと同じ名簿の再受け入れは端末なしで成功する。書き出し（`node export`、`fleet export`）は状態を変えないため確認しない。`--out` は存在しないか空の通常ディレクトリとし、空でない・リンクの場合は何も書かずに失敗する。ファイルは排他的に作成（モード0644）し、途中で失敗したら書いたファイルを削除する。

読み取り（`node show`、`fleet show`、書き出し、`fleet accept --dry-run`）はストレージを初期化せず、状態を移行しない。移行の予告と警告はstderrにだけ出す。

### 名簿とファイル

名簿とノード記述ファイルの形式は次のとおり。

- 名簿は `format`（`ssh-keyflow-roster`）、`schema`（1）、`fleet_id`、`version`（1以上、署名ごとに1増える）、`created_at`（秒精度のUTC、`Z`付き）、共通の `login_account`、`id` 順の `nodes`（1〜1000件）を持つJSON文書。各ノードは `id`、`name`、`roles`、接続先の任意 `login_account`（共通アカウントと同じなら省略）・`addresses`・任意 `port`（22なら省略）・`host_keys`、接続元の `client_keys` を持つ。公開鍵はコメントなしの `<type> <base64>`。
- 署名は名簿のバイト列そのものを対象とし、正規形だけを受け入れる。UTF-8（BOMなし）、LF改行、末尾の改行1つ、タブなし、2スペースのインデント、1行に1メンバー（`"name": value`）、上記のフィールド順で、1 MiB以下。省略した任意フィールドは `null` にせず出力しない。文字列配列はバイト順（`roles` は `client`、`remote` の順）。アプリは正規化しないため、CRLFや再整形は拒否される。未知・重複・大文字小文字違いのフィールドも拒否する。`schema` が2以上の名簿はアプリの更新を求めて失敗する。
- `fleet export` は `roster.json`、`roster.json.sig`（名前空間 `ssh-keyflow-roster` の `ssh-keygen -Y sign` 署名）、`fleet-root.pub`（コメントなしの1行 `ssh-ed25519 <base64>`）を書き出す。`node export` は署名のない `node.json`（`format` が `ssh-keyflow-node`、`schema` 1、`node` 1件）を書き出す。共通アカウントと同じノードのアカウントは `fleet add` が省略する。

### 管理ノードの操作

`fleet init` は先に `node set` を要求し、ネイティブ `ssh-keygen` でパスフレーズ付きEd25519のフリートルート鍵を `~/.ssh/ssh-keyflow/fleet/root_ed25519`（と `.pub`）に作成する。空のパスフレーズは拒否し、作った鍵を破棄する。既存のファイルは再利用・置換しない。状態にはルート鍵のパスと指紋だけを保存し、秘密鍵の内容を状態・出力に含めない。鍵を固定した後に版1の署名が失敗した場合は、`fleet init` を再実行すると同じ鍵で版1に署名する。未完了の間、`fleet sign` と `fleet accept` は拒否し、`fleet show` は警告する。`fleet init` は一度だけ実行でき、名簿を受け入れたノードでは拒否する。

`fleet sign` と `fleet add` は受け入れた名簿から次の版を作り、`ssh-keygen` がパスフレーズを端末で直接求める。アプリは `SSH_ASKPASS` を設定しない。署名後に固定したルートで検証してから、署名・名簿・SHA-256を1回の保存で受け入れ済みとする。失敗した場合は何も保存しない。`fleet add` は名前・役割・各鍵の指紋を表示し、対象の端末の `node show` と照合するよう求める。既知のIDは置換として新旧を並べ、確認の文言も置換とする。管理ノード自身のID、別IDで使用中の名前、別ノードと同じ鍵は拒否する。ノードの削除はフェーズ1にない。

### 受け入れの規則

`fleet accept` はすべての検査を確認より前に行い、どの失敗でも状態を変えない。

1. `<dir>` は前記の3ファイルだけを含み、それぞれ通常ファイルである。
2. 固定済みのルートがある場合、`fleet-root.pub` はそのいずれかと一致しなければならない。ない場合は初回の固定の候補とする。
3. 署名を `ssh-keygen -Y verify` で検証する。一時ファイルはシステムの一時ディレクトリ内の新しいディレクトリに公開データだけを書き、終了時に削除する。署名ファイルのCRLFはLFとして保存する。
4. 名簿を厳格に解析し、正規形を要求する。固定済みのフリートでは `fleet_id` が一致しなければならない。
5. 受け入れ済みより低い版と、同じ版で内容（SHA-256）が異なる名簿は拒否する。同じ版・同じバイト列は確認なしで成功し、何も書き込まない。
6. `created_at` が `--max-age` より古い場合と、5分を超えて未来の場合はstderrに警告する。終了コードは変えない。
7. 版・作成時刻・ノード数と、受け入れ済みの名簿からの変更（共通アカウント、追加・置換・削除されたノード）を表示する。初回はルート公開鍵の指紋とフリートIDも表示し、管理ノードの `fleet show` と照合するよう求め、確認の文言にも指紋を含める。
8. `yes` の後、初回のルートの固定とフリートID、受け入れた版・SHA-256・名簿・署名を1回で保存する。

`fleet init` が未完了の管理ノードは、名簿を受け入れずに `fleet init` の再実行を案内する。

### 自動処理向けJSON

改行で終わる1オブジェクトをstdoutへ出し、警告はJSONに含めない。

- `node show --json`：`{"node": <ノード>, "is_admin": <bool>}`。ノードは名簿のノードと同じフィールド名を持ち、`client_keys` には接続元鍵の公開鍵を出す。`node set` の前は `{"node":null,"is_admin":false}`。
- `fleet show --json`：`{"fleet": {"roots": [{"fingerprint": "SHA256:…"}], "sha256": "<名簿バイト列のSHA-256、16進>", "roster": <名簿オブジェクト>}}`。`fleet_id`・`version`・`created_at` は `roster` から読む。フェーズ1の `roots` は常に1要素。受け入れ前は `{"fleet":null}`。

### OpenSSHの要件

署名と検証にはOpenSSH 8.4以上が必要で、PATH上の最初の `ssh-keygen` と同じディレクトリの `ssh -V` で版を確認し、満たさなければ失敗する。1つのコマンドの中では同じ `ssh-keygen` を使い、`--dry-run` の出力と署名・検証のエラーにそのパスと版を表示する。これは `fleet init`・`fleet sign`・`fleet add`・`fleet accept` だけの要件で、鍵の管理コマンドの要件は変わらない。

### 状態スキーマv4

v4はv3の `version`・`keys`・`creations` に、任意の `node` と `fleet` を加える。

| レコード | 保存フィールド |
| --- | --- |
| Node | `id`, `name`, `roles`, `updated_at`, 任意の `login_account`, `addresses`, `port`, `client_key_ids`, `host_keys`, `sshd_config` |
| Fleet | `id`, `roots`（`public_key`, `fingerprint`, `pinned_at`）, 管理ノードだけの `root_key`（`fingerprint`, `public_path`, `private_path`）, 任意の `accepted`（`version`, `sha256`, `roster`, `signature`, `accepted_at`） |

`client_key_ids` は管理対象の鍵のIDで、公開鍵そのものは鍵ファイルから読む。`accepted.roster` は受け入れた名簿のバイト列をそのまま保持し、`fleet export` はこれと署名を書き出す。ノードの接続元鍵になっている鍵は `keys remove` で外せない。

v3の状態は読み込みだけでは変更しない。`keys list`・`node show`・`fleet show`・書き出しと変更コマンドの `--dry-run` は、次の変更でv4へ移行することをstderrに予告し、変更コマンドは確認の前に同じ予告を表示する。最初の成功した保存はv1 / v2と同じ規則で元の `state.json` を `state.v3-<UTC時刻>.json` へ排他的に書き出してからv4を保存する。v3のレコードはすべて引き継ぐ。`main` はv1 / v2の状態も直接v4へ移行し、引き継ぐ範囲は[v1 / v2からの移行](#v1--v2からの移行)と同じ。

v0.11.0以前はv4を読めない。戻す場合は、アプリを終了して現在の `~/.ssh/ssh-keyflow/` を別の場所へ保管し、アーカイブを `state.json` として戻す。移行後のノード記述・フリート・受け入れた名簿・作成や取り込みをした鍵は戻した状態に含まれず、`fleet/` のルート鍵ファイルは状態から参照されなくなる。そのファイルを残したままv4で `fleet init` を再実行すると、既存ファイルとして拒否する。`state.json.bak` は直前状態の更新式バックアップで、移行直後はv3だが次の変更でv4に置き換わるため、戻す用途にはアーカイブを使う。

フリートルート鍵の秘密鍵は再作成できず、失うと同じフリートの次の版に署名できない。管理ノードでは `~/.ssh/ssh-keyflow/fleet/` を状態と合わせてアクセス制限された場所へ保管する。予備のルート鍵とルートの更新はフェーズ1にない。

## ローカル鍵の安全条件と保存

作成はネイティブ `ssh-keygen -t ed25519` を使い、空のパスフレーズも許容する。生成先は `keys/id_ed25519_<安全化した管理名>_<UTC時刻>` と衝突時の数値suffix。両パスと保存済み予約を調べ、確認後にも再検査する。既存ファイルを上書き・自動採用しない。

副作用前に作成意図・固定ID・パスを `planned` として保存し、`pending` で生成を記録する。再開は同じパスを使い、ファイルなしなら生成、秘密鍵のみならネイティブ `ssh-keygen -y` で導出して欠けた公開鍵を排他的に新規作成する。ペアがあれば再生成せず照合する。公開鍵のみ、空・非通常ファイル、記録済み指紋・ダイジェストの変更、観測済みファイルの消失は要確認で停止する。部分ファイルを自動削除せず、未完了の作成があれば別の鍵ペアを新規作成しない。

取り込みは明示した公開鍵を `ssh-keygen -l -E sha256` で検証し、秘密鍵から `ssh-keygen -y` で導いた公開鍵の指紋と一致させる。元ファイルは移動・書き換えない。保存前に指紋と秘密鍵ダイジェストを再検査し、欠落・置換を成功と扱わない。ローカル鍵ファイルの削除や自動整理のAPIは設けない。

変更全体は `state.WithLock` の排他 `state.lock` で保護し、確認・作成・保存中も保持する。LockedStoreはロック外で使わず、ロックを入れ子にしない。既存ロックを自動削除せず、変更時だけ書き込み可能性を確認する。状態は検証後、同一ディレクトリの一時ファイルへ書き込み・同期し、直前の正当な状態を原子的に `.bak` へ保存してから `state.json` を置き換える。バックアップ失敗時は現在状態を置き換えない。利用者は操作中に対象鍵パスを並行編集しない。

## 検証と公開

CONTRIBUTINGのGoテスト・vet、Portable / インストーラーテスト、6ターゲットビルドを適用する。テストは一時ホームと代替プロセスを使い、実ユーザーのSSHデータ、PATH、プロファイル、接続先を変更しない。旧スキーマの非変更読込と予告、アーカイブと縮小した移行、正確な直前バックアップ、読み取りの非変更性、鍵作成再開と同一性検証、廃止したコマンドの使用法エラーを確認対象に含める。

結果・対象コミット・環境・未検証範囲は `docs/verification.md` に記録する。過去の自動登録・ガイド・TUI・ホスト鍵固定の検証記録は履歴であり、現行機能の契約ではない。クロスビルドは各OS実機でのネイティブ鍵作成の実行証拠と区別する。

公開はmainへ統合・検証済みのコミットに対する注釈付き `vMAJOR.MINOR.PATCH` タグを使う。準備PRの統合から、最終候補の検証、タグ作成・公開、公開後の検証記録PRの統合までの手順・完了条件・承認範囲は [AGENTS.mdのTags and releases](../AGENTS.md#tags-and-releases) を正とする。準備記録は検証したソースのコミットを参照し、最終タグのコミットはタグ・リリース情報と公開後の記録に残す。監査文書の追加でmainがタグより先へ進んでも、公開済みタグはビルド元を指したまま保持する。`v*` タグのpushはCI後の自動公開を開始する。プレリリースはlatestにせず、安定版latestはより新しい安定版へ進める。公開済みタグ・バイナリアセットは不変で、修正は新バージョンで出す。公開後に配布物・チェックサム・内部バージョン・タグ・ソースコミットの一致を確認する。開発ビルドは引き続き `dev` と表示する。
