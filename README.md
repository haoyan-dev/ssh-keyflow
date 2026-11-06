# SSH KeyFlow

自分の端末への SSH 接続に使う鍵、公開鍵登録、接続設定と登録履歴を管理する Go CLI です。コマンド名は `ssh-keyflow`、Windows では `ssh-keyflow.exe` です。

## 起動と基本操作

インストール後に `ssh-keyflow` を実行すると、SSH関連コマンドと保存先を確認し、対話メニューを開きます。不足するコマンドがあれば導入手順を表示し、導入後に再チェックできます。ソースから試す場合は、リポジトリ内で `go run ./cmd/ssh-keyflow` を実行します。

| メニュー | 操作 |
| --- | --- |
| 1 | 接続先の追加と公開鍵登録 |
| 2 | 接続先、鍵、登録状態と管理対象外の一覧 |
| 3 | 既存の接続先への別の公開鍵の登録 |
| 4 | 接続先からの公開鍵登録解除 |
| 5 | Ed25519鍵の新規作成、既存鍵の明示的な取り込み、作成の再開 |
| 6 | ローカル鍵ファイルの削除 |
| 0 | 終了 |

最初は `1` から別名、ホスト名またはIPアドレス、ユーザー名、ポート、接続先OSを入力します。鍵は新規作成か、取り込み済みの鍵の再利用を選びます。接続先の保存後、そのままパスワード・既存秘密鍵による自動登録、または手動手順を選べます。自動登録では接続確認とSSH設定の生成まで続けて進みます。Tailscaleの端末名・IPも接続先として指定できます。通常のSSHサーバーに接続するため、Tailscale SSH固有の認可やポリシーは変更しません。

パスワードと鍵のパスフレーズはOpenSSHの端末入力で扱います。自動登録できない場合は、選択した接続先OSに対応するSSH有効化と公開鍵登録のコピペ手順に従ってください。途中で中断した接続先追加は `1` の再開から、保存済みの接続先と鍵を使って続行します。

自動登録では、既存の秘密鍵またはパスワードによる新規接続を選びます。初回のホスト鍵はOpenSSHが確認を求め、記録と異なる場合は停止します。接続先のOS・ユーザー・標準登録先を確認してから変更し、Windows管理者用の共有ファイルには影響範囲の承認を求めます。登録後は指定鍵だけで接続を確認します。公開鍵しか取り込んでいない場合は接続未確認として残します。

専用SSH設定は `~/.ssh/ssh-keyflow/config` に生成します。通常の `~/.ssh/config` には、差分を表示して承認を得た場合だけIncludeを追加し、元の内容をバックアップします。既存別名の衝突や管理対象設定の変更を検出した場合は停止します。自動操作はユーザーのSSH設定を読み込まない直接接続を使うため、ProxyJump等が必要な環境では表示された手動手順を確認してください。

鍵の取り込みは `5` から行います。秘密鍵と公開鍵のペア、または公開鍵だけを指定でき、ファイルは元の場所に残ります。一覧で発見しただけの鍵やSSH設定は「管理対象外」として表示し、明示的に取り込むまで変更しません。登録解除とローカル削除は別の操作です。

登録解除は `4` から接続先と鍵を選びます。対象鍵と同じ指紋の鍵を除く別の秘密鍵、またはパスワードで新規接続を確認し、解除対象を表示した最終承認後に実行します。接続できない場合は手動解除手順を表示し、ローカル鍵を残します。Linux/macOS向けの解除スクリプトにはPerlが必要です。未対応のサーバー設定や権限は自動処理を止め、手動確認に切り替えます。解除前に元ファイル全体を接続先の同じディレクトリへ `authorized_keys.keyflow-backup-*` として保存し、完成済みファイルへ置き換えます。元の所有者・権限・ACLと無関係な内容を保持し、バックアップは復旧用に残します。中断後にロックや一時ファイルが残った場合は、接続先を確認してから手動で復旧してください。

ローカル削除は `6` から別途承認します。記録された登録先がすべて解除済みであること、鍵ファイルと管理対象設定が記録と一致することを確認します。途中で中断した削除は保存した意図から再開し、削除後も固定IDと登録履歴を残します。このツールが把握する登録先だけを確認するため、別の方法で登録した端末の有無は削除前に確認してください。

管理情報は `~/.ssh/ssh-keyflow/state.json`、生成した鍵は同ディレクトリの `keys/` に保存します。登録状態、指定鍵の接続確認、処理の進捗を区別するため、登録済みでも接続確認が失敗または未確認の場合があります。検証環境と確認済みの範囲は [docs/verification.md](docs/verification.md) を参照してください。

配布物とインストーラーは [GitHub Releases](https://github.com/haoyan-dev/ssh-keyflow/releases/latest) から取得できます。ソースからは `go run ./cmd/ssh-keyflow` で起動できます。

## インストールと更新

Windows、Linux、macOS の x64 / ARM64 に対応します。インストーラーは OS と CPU を判定し、既定で GitHub の最新正式リリースを取得します。明示したタグはプレリリースも指定できます。32 bit とその他の CPU は対象外です。

Linux/macOS は POSIX `sh`、`curl`、`tar`、`awk`、`sed`、`grep`、`dd`、`cmp`、通常の POSIX ファイル操作コマンド、および `sha256sum` または `shasum` が必要です。Windows は Windows PowerShell 5.1 または PowerShell 7 と標準の .NET API を使います。Go、Python、Git はエンドユーザーのインストールには不要です。本体の SSH 操作には `ssh` と `ssh-keygen` が必要です。

Linux/macOS:

```sh
repository='haoyan-dev/ssh-keyflow'
curl -fL "https://github.com/$repository/releases/latest/download/install.sh" -o install.sh
sh install.sh --repository "$repository"
ssh-keyflow --version
```

Windows PowerShell:

```powershell
$repository = 'haoyan-dev/ssh-keyflow'
Invoke-WebRequest -UseBasicParsing "https://github.com/$repository/releases/latest/download/install.ps1" -OutFile install.ps1
& .\install.ps1 -Repository $repository
ssh-keyflow --version
```

ダウンロードしたスクリプトを確認してから実行してください。Windows の実行ポリシーで拒否された場合は組織のポリシーに従ってください。インストーラーはポリシー変更も権限昇格も行いません。

指定版のインストールと、同じコマンドによる更新:

```sh
sh install.sh --repository 'haoyan-dev/ssh-keyflow' --version v1.2.3
```

```powershell
& .\install.ps1 -Repository 'haoyan-dev/ssh-keyflow' -Version v1.2.3
```

配布元は `--repository` / `-Repository` のほか、環境変数 `SSH_KEYFLOW_REPOSITORY` でも指定できます。既定の配置先は Linux/macOS では `~/.local/bin`、Windows では `%LOCALAPPDATA%\Programs\ssh-keyflow` です。`--install-dir` / `-InstallDir` でユーザーが書き込める絶対パスを指定できます。同じ配置先を指定して更新・削除してください。

取得したアーカイブは同じリリースの `SHA256SUMS` の一致する唯一のエントリーで検証し、期待する実行ファイル一つだけを取り出します。検証が終わるまで既存の実行ファイルを変更せず、同じファイルシステムで置き換えます。通常の置換・記録保存エラーでは旧版へ戻します。SHA256 は配布物の整合性チェックで、署名検証ではありません。実行ファイルを手動で差し替えた場合や、所有記録を失った場合は更新・削除を止めます。

## PATH の確認

永続化の対象と追加内容を表示し、`y` / `yes` の入力があった場合だけプロファイルへ追記します。既定は拒否です。拒否した場合や未知のシェルの場合は手動の PATH コマンドを表示します。PATH に配置先があれば追記しません。繰り返し実行・プロファイルの再読み込みで重複しない専用 stanza を記録します。

Linux/macOS は `SHELL` に応じて `.bashrc`（macOS は `.bash_profile`）、`.zshrc`、`.profile` を選びます。`--profile /absolute/path` でも指定できます。Windows の対象は現在の PowerShell の CurrentUserAllHosts プロファイルで、`-ProfilePath C:\absolute\profile.ps1` でも指定できます。Windows のレジストリ PATH は変更しません。Windows PowerShell と PowerShell 7 は別のプロファイルを使うため、使うシェル上でインストーラーを実行してください。

承認後は新しいシェルを開いてください。既存セッションで使う場合は表示された手動コマンドを実行します。`ssh-keyflow --version` は PATH の反映後に実行します。未知のシェル用に Bash の stanza を転用せず、そのシェルの構文で PATH を設定してください。

## アンインストール

```sh
sh install.sh --uninstall
```

```powershell
& .\install.ps1 -Uninstall
```

実行ファイル、インストーラーの所有記録、記録と完全一致する専用 PATH stanza だけを取り除きます。前後の設定や同じディレクトリの他のファイルは保持します。stanza が変更・複製されていれば自動削除を止めます。ロックが残っていれば、別のインストーラーが動いていないことと配置先を確認してから対処してください。

`~/.ssh` の鍵、`~/.ssh/ssh-keyflow/state.json`、SSH 設定、接続先の公開鍵登録は保持します。アンインストールは公開鍵の登録解除ではありません。現在のシェルの PATH はセッション終了まで残ることがあります。

## 開発時の検証

Go のバージョンは `go.mod` に従います。配布のテスト・梱包には Python 3.12（標準ライブラリのみ）を使います。

```sh
go test ./...
python scripts/test_installers.py -v
python scripts/package_release.py --version v1.2.3
```

Windows のシェルテストは `C:/Program Files/Git/bin/bash.exe` を使います。WSL の bash は使いません。Windows のテストには PowerShell 7 と Windows PowerShell 5.1 が必要です。テストは Git で除外する `.tmp/installer-tests/` 内の一時 home、配置先、プロファイル、配布物とループバック HTTP サーバーを使い、終了時に一時ディレクトリを削除します。実ユーザーの SSH データ、永続 PATH、プロファイルを変更せず、GitHub の公開リリースも取得しません。

`SSH_KEYFLOW_API_URL` と `SSH_KEYFLOW_RELEASE_URL` は配布 API / ダウンロードのベース URL を変更するテスト・ミラー用の明示設定です。通常は未設定で使ってください。テストの latest メタデータは正式版・指定版・チェックサムの挙動を本物のリリースから独立して検証します。梱包だけを確認する場合は `--binaries DIRECTORY` を指定し、`DIRECTORY/linux_amd64/ssh-keyflow` などの6対象の配置から読み込みます。

## リリース契約

`scripts/package_release.py` は `CGO_ENABLED=0`、`-buildvcs=false` と `-trimpath` を使い、`./cmd/ssh-keyflow` を6対象へビルドします。Git の信頼設定や作業ツリーの状態に依存する VCS stamp を含めず、バージョンは `-ldflags "-s -w -X main.version=<tag>"` で注入します。開発時の `main.version` は `dev` です。出力先は既定で `dist`、`--output DIRECTORY` で変更でき、既存ファイルがあれば上書きを拒否します。

| OS | CPU | アセット名（例） | 内部ファイル |
| --- | --- | --- | --- |
| Linux | x64 | `ssh-keyflow_v1.2.3_linux_amd64.tar.gz` | `ssh-keyflow` |
| Linux | ARM64 | `ssh-keyflow_v1.2.3_linux_arm64.tar.gz` | `ssh-keyflow` |
| macOS | x64 | `ssh-keyflow_v1.2.3_darwin_amd64.tar.gz` | `ssh-keyflow` |
| macOS | ARM64 | `ssh-keyflow_v1.2.3_darwin_arm64.tar.gz` | `ssh-keyflow` |
| Windows | x64 | `ssh-keyflow_v1.2.3_windows_amd64.zip` | `ssh-keyflow.exe` |
| Windows | ARM64 | `ssh-keyflow_v1.2.3_windows_arm64.zip` | `ssh-keyflow.exe` |

同じリリースに `install.sh`、`install.ps1`、`SHA256SUMS` も添付します。manifest は `64桁のSHA256`、半角スペース2個、アセットのベース名、LF の形式で、6アーカイブと2インストーラーを含みます。

CI は3 OS で Go テストとローカル配布物のテストを実行し、6対象のクロスビルドと Linux x64 の `--version` 注入を確認します。`v*` タグを push するとリリースワークフローが CI 成功後に9ファイルを公開します。公開先は実行環境の `GITHUB_REPOSITORY` です。`v1.2.3-rc.1` のようなタグはプレリリースにし、latest 正式版として扱いません。

新しいバージョンを公開するときは、CIが成功したコミットに対してタグを作成・push してください。クロスビルド成功は各 OS / CPU の実機動作確認を意味しません。fixture による対象選択・配置確認も、ARM64 実機でのバイナリ実行とは区別してください。
