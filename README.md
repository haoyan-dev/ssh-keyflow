# SSH KeyFlow

自分の端末への SSH 接続に使う鍵、公開鍵登録、接続設定と登録履歴を管理する Go 製の TUI / CLI です。コマンド名は `ssh-keyflow`、Windows では `ssh-keyflow.exe` です。

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

## 起動と操作

端末で引数なし、または `ssh-keyflow tui` を実行するとTUIが開きます。Tab / Shift+Tabで Destinations（接続先）・Keys（鍵）・Registrations（登録）・Tasks（保存済みの要対応事項）を切り替え、矢印キーで選択、Enterで詳細を表示します。別名・ログイン先・鍵名・指紋を手がかりに選べ、固定IDと完全なパス・指紋は詳細で確認できます。登録、接続確認、接続設定、履歴はそれぞれ独立した保存証拠として表示します。

| キー | 一覧・詳細での操作 |
| --- | --- |
| `/` / `x` | 名前・ログイン先・指紋・固定IDで検索 / 検索を解除 |
| `a` | 接続先追加 |
| `c` / `i` | Ed25519鍵の作成 / 既存鍵の取り込み |
| `r` / `u` | 表示中の対象への公開鍵登録 / 登録解除。関連する鍵・接続先・登録を必要に応じて選択 |
| `d` | Keysの管理対象鍵をローカル削除。未解除の記録があれば停止 |
| `s` | Tasksなどに表示した保存済み操作を継続 |
| `F5` | 保存状態だけを再読込。操作の再実行やSSHの接続確認はしない |
| `h` / `g` | 読み取り専用の履歴 / 管理対象外の鍵・設定と要確認事項 |
| `?` / Esc | ヘルプ / 元の画面へ戻る |
| `q` / Ctrl+C | 入力・ヘルプ・実行中以外では終了 |

検索中はEnterで確定、Escで検索語を残して編集を終了します。カテゴリごとに検索・選択位置を保持します。詳細・指示・エラーの全文は矢印、PgUp / PgDn、Home / Endで縦にスクロールできます。

接続先追加は Destination → Key → Access method/options → Review の順に進みます。OSは明示して選択し、Windowsだけアカウント種別を指定します。管理対象鍵の再利用か新しい鍵の作成を選べます。既存ファイルの取り込みは先に `i` で行ってください。認証は手動案内、SSH設定生成は「しない」が既定です。秘密鍵で認証するときだけ認証鍵のパスを入力します。パスワード・パスフレーズの入力欄はありません。

フォームではTab / Shift+Tabで項目とContinue / Help / Cancelを移動し、Enterで進みます。Ctrl+Sは全項目を検証してReviewを開き、最初の入力不備があればその項目へ戻ります。ReviewのEditまたはEscで入力位置を保って編集できます。Escは一つ前の段階へ戻り、最初の段階のEsc・Cancel・Ctrl+Cは変更済みの入力を捨てるか確認します。既定のKeep editingは入力を保持します。フォームのHelpからも入力位置へ戻れます。入力欄の `q`・`?`・操作文字は通常の文字として扱います。

ReviewのContinue to terminalで実行へ進むと、共通処理が対象を再検証し、必要な承認を端末で別途求めます。パスワードとパスフレーズはOSのOpenSSHへ直接入力します。終了・拒否・失敗・中断後はTUIへ戻り、保存できた変更と今回未確認の事実、残る手順を結果に表示します。既に保存した変更はキャンセルで巻き戻りません。別名用の専用設定生成と通常のSSH設定へのInclude追加は別承認で、Includeを断っても生成済み別名は残ります。

結果のTab / Shift+TabとEnterで継続・View task・View entity・Refresh・Exit・Helpを選べます。`s` / `t` / `v` はその結果の保存済み操作・Task・対象に対応します。保存状態の再読込に失敗した場合はstaleを表示し、Refreshに成功するまで変更操作を止めます。診断と手動指示は結果を離れるまで参照できます。

Tasksは保存済みの記録から復旧候補をまとめた一覧です。空でも接続確認済みという意味ではありません。手動Registerが新しい意図を保存しない場合、結果はManual action required / No saved recovery taskと表示し、再起動後のTaskを作りません。結果のRegister againを使うか、指示を確認してから終了し、必要なら再起動後にもう一度Registerしてください。登録解除の継続は解除を引き継ぎ、古い接続先追加を再開して再登録する案内は出しません。接続確認だけ、設定だけを実行する操作はありません。ローカル鍵の作成・削除の継続ではリモート認証を求めません。

対応する最小端末は32列×12行です。これより小さくすると入力を保持してサイズ案内と終了操作を表示し、変更操作を開始しません。端末を広げると元の画面・フォーカスへ戻ります。現在の検証範囲と未完了の受入項目は [検証記録](docs/verification.md) を参照してください。

CLIでは操作と対象を直接指定できます。変更操作の承認・認証には端末が必要です。各コマンドの `--help` で全フラグと対象IDの指定方法を確認してください。

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow hosts list --json
ssh-keyflow keys list --json
ssh-keyflow registrations list --json
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub
ssh-keyflow register --host my-server --key laptop --auth password
ssh-keyflow resume --id <saved-operation-id>
```

接続先追加は `hosts add`、登録解除は `unregister --registration <id>`、ローカル削除は `keys delete --key <id-or-name>` です。接続先は固定IDまたは別名、鍵は固定IDまたは一意の管理名で指定します。再開・登録解除は固定IDを使います。曖昧な鍵名を自動選択しません。JSONは標準出力、診断は標準エラーへ分けます。終了コードは成功0、実行失敗・取消1、引数・使用法エラー2です。非端末の引数なし起動はヘルプを表示して0で終了します。

接続先のOSは明示します。Tailscaleの名前・IPも通常のSSH経路として使えますが、Tailscale SSHのポリシーは変更しません。自動登録できない場合はOS別の手動案内を使います。公開鍵のみの取り込みは接続未確認として残り、登録記録・指定鍵による接続確認・操作進捗は別に表示します。

## 安全条件とデータ

既存鍵のペアまたは公開鍵だけを明示的に取り込めます。元ファイルは移動しません。発見しただけの鍵やSSH設定は管理対象外で、自動では変更しません。秘密値は引数・状態JSONへ保存しません。

管理情報は `~/.ssh/ssh-keyflow/state.json`、生成鍵は同ディレクトリの `keys/`、専用SSH設定は `config` に保存します。本体の配置場所から独立しており、v0.1.0の状態を継続利用します。通常の `~/.ssh/config` へのInclude追加は差分・バックアップ・承認を伴います。既存別名の衝突や所有設定の変更を検出すると停止します。自動SSH操作はユーザー設定を読まない直接接続で、ProxyJump等が必要な場合は手動手順を使います。

登録解除は別の秘密鍵またはパスワードで新規接続を確認した後、対象を表示して最終承認を求めます。対象鍵と同じ指紋の鍵を代替認証には使えません。到達不能なら手動解除案内を表示してローカル鍵を残します。Linux/macOSの自動解除は接続先にPerlが必要です。解除前のファイルを接続先にバックアップし、無関係な内容・所有者・権限・ACLを保持して置き換えます。中断したロック等は実状態を確認して復旧してください。

ローカル削除は別操作です。記録した登録先がすべて解除済みで、鍵・所有設定が記録と一致する場合に限り実行し、固定IDと履歴は残します。ツール外で登録した接続先の有無も利用者が確認してください。結果不明・中断した操作は保存した意図と実状態を照合して再開します。

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

更新前にアプリを終了し、新しい配布物のチェックサムを検証して別のディレクトリへ展開します。その後、既存の実行ファイルと同梱ライセンス文書を新しいものへ差し替え、`--version` を確認してください。本体は移動しても従来のホーム配下の状態・鍵・生成設定を使います。

PATHは必要な場合だけ自分で設定します。現在のセッションだけなら、POSIXシェルでは `export PATH="/absolute/path/ssh-keyflow-portable:$PATH"`、PowerShellでは `$env:PATH = 'C:\absolute\path\ssh-keyflow-portable;' + $env:PATH` を実行します。永続化する場合は使うシェルのプロファイルやOSのPATH設定を自分で編集してください。本体はPATH・プロファイル・レジストリを自動変更しません。

本体を削除するときはアプリを終了し、実行ファイルと同梱ライセンス文書を削除します。自分で追加したPATHは必要に応じて取り除いてください。鍵、状態、SSH設定、登録履歴、接続先の公開鍵登録は保持されます。本体削除は登録解除やローカル鍵削除とは別です。

## v0.1.0のインストーラー利用者

旧配置先は既定でLinux/macOSの `~/.local/bin`、Windowsの `%LOCALAPPDATA%\Programs\ssh-keyflow` です。本体と旧所有記録を変更していなければ、新しいインストーラーで更新できます。既存のPATH追記を保持し、新たに同梱ライセンス文書も管理します。v0.2.0のアーカイブは実行ファイル1個だけを要求する旧インストーラーには対応しないため、古いスクリプトを再利用しないでください。

手動差し替え後は本体のハッシュと旧所有記録が一致せず、旧スクリプトの更新・削除も拒否され得ます。以後の更新・削除は手動で行ってください。旧所有記録（POSIXの `.ssh-keyflow-install/`、Windowsの `.ssh-keyflow-install.json`）と既存PATHは自動移行・自動編集しません。記録はそのまま残せます。

旧PATHの追記を外す場合は所有記録が示すプロファイルと専用の `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` 範囲を確認し、必要な範囲だけ手動で編集してください。プロファイル全体やSSHデータは削除しません。Windowsの旧PATHはレジストリではなく実行したPowerShellのプロファイルにあり、Windows PowerShellとPowerShell 7では対象が異なります。

## 開発とリリース契約

Goは `go.mod` の指定、梱包テストはPython 3.12の標準ライブラリを使います。TUIはBubble Tea / Bubbles / Lip Gloss、CLIはCobra、SSH処理と状態管理は既存実装を再利用します。

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
python scripts/package_release.py --version v0.2.0 --output dist
```

梱包は `CGO_ENABLED=0`、`-buildvcs=false`、`-trimpath` と `-ldflags "-s -w -X main.version=<tag>"` で6対象をビルドします。開発版は `dev`。既存ファイルがある出力先への上書きは拒否します。既存バイナリの梱包には `--binaries DIRECTORY` を指定し、`DIRECTORY/linux_amd64/ssh-keyflow` 等の6対象から読み込みます。

| OS | CPU | アセット名 |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

アーカイブ内部は `ssh-keyflow`（Windowsは `ssh-keyflow.exe`）と `THIRD_PARTY_NOTICES.txt`。Unix実行権限は0755です。トップレベルの配布物は6アーカイブと `SHA256SUMS` の7件だけです。manifestは6アーカイブについて、64桁SHA256・半角スペース2個・ベース名・LFで記録します。

CIは3 OSでGoテスト、Portable生成物テスト、インストーラーテストを実行し、6対象の実クロスビルドとLinux amd64版の版注入を確認します。`v*` タグのpushでCI成功後に7件を `GITHUB_REPOSITORY` へ公開します。インストーラーは `main` のソースから別途配信し、7件のリリース配布物は変更しません。プレリリースタグはlatest正式版にしません。

テストは一時ホーム・配置先・配布物を使い、実ユーザーのSSHデータ、永続PATH、プロファイル、接続先を変更しません。クロスビルドは各OS / CPUでの実機実行の証拠とは区別します。確認済み範囲は [docs/verification.md](docs/verification.md)、公開インターフェースは [docs/application-contract.md](docs/application-contract.md) を参照してください。
