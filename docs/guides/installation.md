# インストール・更新・アンインストール

[简体中文](../zh-CN/guides/installation.md) · [English](../en/guides/installation.md) · 日本語

[ドキュメント一覧](../README.md)

対象はWindows / Linux / macOSのamd64 / arm64です。OSのOpenSSH（`ssh` と `ssh-keygen`）を用意してください。以下の固定バージョンは指定方法の例です。通常は版指定なしで正式版を取得し、過去の版へ戻す前に[データと復旧](data-and-recovery.md)を確認してください。

## 1行でインストール

次のコマンドで、最新の正式リリースをインストールできます。

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

**Windows（PowerShell）**

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

OSとCPU（x64 / ARM64）を判定し、最新の正式リリースと `SHA256SUMS` を取得します。チェックサムとアーカイブの内容を検証した後、本体とライセンス文書をユーザー専用の場所へ配置します。Goと管理者権限は不要です。OSの `ssh` と `ssh-keygen` は別途必要です。取得するスクリプトを先に確認したい場合は、保存して内容を読んでから実行してください。

既定の配置先はmacOS/Linuxの `~/.local/bin`、Windowsの `%LOCALAPPDATA%\Programs\ssh-keyflow` です。PATHへの追加は対話端末で確認し、承認した場合だけシェルのプロファイルに保存します。新しい端末を開いて、次を実行してください。

```sh
ssh-keyflow --version
ssh-keyflow
```

PATHの追加を断った場合や、非対話環境で実行した場合は、インストーラーが表示する本体のフルパスで起動できます。WindowsのPATH保存は、実行したPowerShellのプロファイルだけに適用されます。別の種類のPowerShellやコマンドプロンプトでは、表示されるPATHの手動設定案内を使ってください。

同じコマンドを再実行すると更新できます。更新前にSSH KeyFlowを終了してください。管理するのは本体・ライセンス文書・インストーラー自身のPATH追記だけです。鍵、状態、SSH設定、接続先の公開鍵登録は変更しません。所有記録がない既存ファイルや改変されたファイルは上書きしません。

版や配置先を指定する場合やアンインストールする場合は、スクリプトを保存して実行します。

```sh
# macOS / Linux：v0.2.1を指定、明示的にPATHへ追加
curl -fsSLo install.sh https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh
sh install.sh --version v0.2.1 --add-to-path
# 本体を終了してから、所有する本体・ライセンス文書・PATH追記だけを削除
sh install.sh --uninstall
```

```powershell
# Windows：v0.2.1を指定、明示的にPATHへ追加
Invoke-WebRequest -Uri https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version v0.2.1 -AddToPath
# 本体を終了してから削除
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
```

上記のWindows例は、そのプロセスだけ実行ポリシーを指定します。PowerShell 7で管理する場合は `powershell` を `pwsh` に置き換えてください。配置先は `--install-dir /absolute/path` / `-InstallDir C:\absolute\path`、対象プロファイルは `--profile /absolute/path` / `-ProfilePath C:\absolute\profile.ps1` で指定します。独自の配置先を使った場合は更新・削除でも同じ指定を使ってください。別のリポジトリを使う場合は `--repository OWNER/REPO` / `-Repository OWNER/REPO` を指定できます。

## Portable配布の取得と実行

Windows / Linux / macOS の amd64（x64）/ arm64 用アーカイブを提供します。Windowsはzip、Linux/macOSはtar.gzです。Goやインストーラーは不要で、OS側の `ssh` と `ssh-keygen` が必要です。TUIは対話端末で使います。

配布元は [GitHub Releases](https://github.com/haoyan-dev/ssh-keyflow/releases) です。対象タグの6アーカイブと `SHA256SUMS` のうち、自分のOS / CPUのアーカイブとmanifestを取得してください。

例：v0.2.0をLinux amd64で使う場合、次のように検証して展開します。

```sh
sha256sum --ignore-missing -c SHA256SUMS
mkdir -p ./ssh-keyflow-portable
tar -xzf ssh-keyflow_v0.2.0_linux_amd64.tar.gz -C ./ssh-keyflow-portable
./ssh-keyflow-portable/ssh-keyflow --version
./ssh-keyflow-portable/ssh-keyflow
```

macOSでは `shasum -a 256 ssh-keyflow_v0.2.0_darwin_arm64.tar.gz` 等の結果をmanifestの該当する唯一の行と照合してください。Windows PowerShellでは次のように照合します。

```powershell
Get-FileHash .\ssh-keyflow_v0.2.0_windows_amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS
# 対象ファイル名の唯一の行とハッシュが一致した後に展開
Expand-Archive .\ssh-keyflow_v0.2.0_windows_amd64.zip .\ssh-keyflow-portable
& .\ssh-keyflow-portable\ssh-keyflow.exe --version
& .\ssh-keyflow-portable\ssh-keyflow.exe
```

SHA256は整合性確認であり署名検証ではありません。各アーカイブは実行ファイルと `THIRD_PARTY_NOTICES.txt` を含みます。配置先は利用者が書き込める任意の場所を選び、配布物の移動・共有時には同梱ライセンス文書も保持してください。Portableは実行ファイルの配置を自由に選べる意味で、ユーザー間で鍵や管理データを持ち運ぶ仕組みは含みません。

## Portableの手動更新・PATH・本体削除

更新前にアプリを終了し、新しい配布物のチェックサムを検証して別のディレクトリへ展開します。その後、既存の実行ファイルと同梱ライセンス文書を新しいものへ差し替え、`--version` を確認してください。本体は移動しても従来のホーム配下の状態・鍵を使います。過去の生成設定もそのまま残ります。

PATHは必要な場合だけ自分で設定します。現在のセッションだけなら、POSIXシェルでは `export PATH="/absolute/path/ssh-keyflow-portable:$PATH"`、PowerShellでは `$env:PATH = 'C:\absolute\path\ssh-keyflow-portable;' + $env:PATH` を実行します。永続化する場合は使うシェルのプロファイルやOSのPATH設定を自分で編集してください。本体はPATH・プロファイル・レジストリを自動変更しません。

本体を削除するときはアプリを終了し、実行ファイルと同梱ライセンス文書を削除します。自分で追加したPATHは必要に応じて取り除いてください。鍵、状態、SSH設定、登録履歴、接続先の公開鍵登録は保持されます。本体削除は登録解除やローカル鍵削除とは別です。

## v0.1.0のインストーラー利用者

旧配置先は既定でLinux/macOSの `~/.local/bin`、Windowsの `%LOCALAPPDATA%\Programs\ssh-keyflow` です。本体と旧所有記録を変更していなければ、新しいインストーラーで更新できます。既存のPATH追記を保持し、新たに同梱ライセンス文書も管理します。v0.2.0以降のアーカイブは実行ファイル1個だけを要求する旧インストーラーには対応しないため、古いスクリプトを再利用しないでください。

手動差し替え後は本体のハッシュと旧所有記録が一致せず、旧スクリプトの更新・削除も拒否され得ます。以後の更新・削除は手動で行ってください。旧所有記録（POSIXの `.ssh-keyflow-install/`、Windowsの `.ssh-keyflow-install.json`）と既存PATHは自動移行・自動編集しません。記録はそのまま残せます。

旧PATHの追記を外す場合は所有記録が示すプロファイルと専用の `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` 範囲を確認し、必要な範囲だけ手動で編集してください。プロファイル全体やSSHデータは削除しません。Windowsの旧PATHはレジストリではなく実行したPowerShellのプロファイルにあり、Windows PowerShellとPowerShell 7では対象が異なります。
