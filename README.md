# SSH KeyFlow

自分の端末への SSH 接続に使う鍵、公開鍵登録、接続設定と登録履歴を管理する Go CLI です。コマンド名は `ssh-keyflow`、Windows では `ssh-keyflow.exe` です。

**公開先の GitHub owner/repository は未設定です。** 以下の `OWNER/REPO` を実際の公開先に置き換えてください。リリースが公開されるまではダウンロード手順は利用できません。

## インストールと更新

Windows、Linux、macOS の x64 / ARM64 に対応します。インストーラーは OS と CPU を判定し、既定で GitHub の最新正式リリースを取得します。明示したタグはプレリリースも指定できます。32 bit とその他の CPU は対象外です。

Linux/macOS は POSIX `sh`、`curl`、`tar`、`awk`、`sed`、`grep`、`dd`、`cmp`、通常の POSIX ファイル操作コマンド、および `sha256sum` または `shasum` が必要です。Windows は Windows PowerShell 5.1 または PowerShell 7 と標準の .NET API を使います。Go、Python、Git はエンドユーザーのインストールには不要です。本体の SSH 操作には `ssh` と `ssh-keygen` が必要です。

Linux/macOS:

```sh
repository='OWNER/REPO'
curl -fL "https://github.com/$repository/releases/latest/download/install.sh" -o install.sh
sh install.sh --repository "$repository"
ssh-keyflow --version
```

Windows PowerShell:

```powershell
$repository = 'OWNER/REPO'
Invoke-WebRequest -UseBasicParsing "https://github.com/$repository/releases/latest/download/install.ps1" -OutFile install.ps1
& .\install.ps1 -Repository $repository
ssh-keyflow --version
```

ダウンロードしたスクリプトを確認してから実行してください。Windows の実行ポリシーで拒否された場合は組織のポリシーに従ってください。インストーラーはポリシー変更も権限昇格も行いません。

指定版のインストールと、同じコマンドによる更新:

```sh
sh install.sh --repository 'OWNER/REPO' --version v1.2.3
```

```powershell
& .\install.ps1 -Repository 'OWNER/REPO' -Version v1.2.3
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

公開先と権限を設定した保守者が、公開するコミットに対してタグを作成・push してください。この作業自体ではタグ push やリリース公開を行っていません。クロスビルド成功は各 OS / CPU の実機動作確認を意味しません。fixture による対象選択・配置確認も、ARM64 実機でのバイナリ実行とは区別してください。
