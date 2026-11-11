# SSH KeyFlow

ローカルのSSH鍵と接続先を管理し、公開鍵の配置と接続設定をコピペで支援するGo製のTUI / CLIです。コマンド名は `ssh-keyflow`、Windowsでは `ssh-keyflow.exe` です。

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

端末で引数なし、または `ssh-keyflow tui` を実行するとTUIが開きます。Keys（鍵）とDestinations（接続先）から、鍵の作成・取り込み、接続先への関連付け、公開鍵と手動設定ガイドの表示・コピーを行います。接続先には作成済み・取り込み済みの鍵を選びます。新しい鍵が必要な場合は先に独立した鍵作成を行ってください。操作キーは画面内のヘルプを参照してください。

| キー | 操作 |
| --- | --- |
| Tab / Shift+Tab | Keys / Destinationsを切り替え |
| ↑ / ↓、Enter | 選択、詳細を表示 |
| `/` / `x` | 検索 / 検索解除 |
| `c` / `i` | Keysで鍵作成 / 取り込み |
| `a` / `l` | 接続先追加 / 鍵と接続先を関連付け |
| `d` / `u` | 選択した鍵・接続先を管理から削除 / 既存の関連付けを解除 |
| `p` / `g` | 公開鍵 / 配置・解除・接続ガイドを表示 |
| `y` | 公開鍵、またはガイドの選択中ステップの実行用スクリプトをコピー |
| ← / →、`b` / `n` | ガイドの前 / 次のステップ |
| `s` | 選択した未完了のローカル鍵作成を再開 |
| `v` / F5 | 読み取り専用の技術詳細・旧履歴 / 保存状態の再読込 |
| `?` / Esc / `q` | ヘルプ / 戻る / 終了 |

コピーは端末のOSC52対応が必要です。使えない場合はCLIの `keys public --key ID` / `hosts guide --host ID --key ID` の出力を、エディターやOSのクリップボードへ取り込んでください。公開鍵は1行を維持し、TUI表示の折り返しを改行として貼り付けないでください。長い本文は矢印、PgUp / PgDn、Home / Endで読み進められます。最小端末サイズは32列×12行です。フォームではTab / Shift+Tabで移動、Ctrl+Sで続行し、Esc / Ctrl+Cで入力を取り消します。

TUIはナビゲーション、一覧・本文、検索・フォーム入力、メッセージ、メニューを枠で区切ります。一覧の項目間には空行を置き、選択中の項目・入力欄と注意・入力エラーを色や記号で示します。フォームのContinue / Cancel / Helpは下部メニューに固定します。小さい端末では表示件数を減らし、通知がないときのメッセージ枠を省きます。色を使わない場合は、起動前に環境変数 `NO_COLOR=1` を設定してください。

ガイドは、対応する端末へそのまま貼り付けて実行できるスクリプトをステップごとに表示します。`g` で接続先と鍵を選ぶと、ローカルと接続先それぞれのOS（linux / macos / windows）を確認・変更するフォームが開きます。既定はアプリ実行元OSと保存済み接続先OSです。Windows接続先ではアカウント種別も指定します。Ctrl+Sでガイドを生成し、← / →または `b` / `n` でステップを選び、`y` でそのスクリプトだけをコピーしてください。説明文や他の端末向けのスクリプトはコピーに含みません。CLIの `hosts guide` は全ステップを表示します。

1. **接続先の端末**：選択した接続先OSに応じた公開鍵配置スクリプトを実行します。Linux/macOSはログインユーザーのPOSIXシェル、Windows標準アカウントは本人のPowerShell、Windows管理者アカウントは管理者として開いたPowerShellを使います。既存の公開鍵を保持して追記し、必要な権限を設定します。同じ公開鍵があれば再追記しません。
2. **ローカルの端末**：秘密鍵のある鍵では、選択したローカルOS向けのSSH config追記スクリプトを実行します。WindowsはPowerShell、Linux/macOSはPOSIXシェルです。アプリ実行元と異なるOSを選ぶ場合は、その端末から見た対応する秘密鍵パスを指定してください。既存内容を保持し、同じ明示的な別名があれば停止します。Include・ワイルドカード・Match・優先順位は実行前に自分で確認してください。
3. **ローカルの端末（パスフレーズ付きの鍵）**：ssh-agentを自動起動するステップです。秘密鍵の形式ヘッダーからパスフレーズの有無を判定し、パスフレーズ付きまたは判定できない鍵で表示します（判定できない場合はその旨を示します）。SSH configには `AddKeysToAgent yes` を加え、最初の接続で鍵をagentに読み込み、agentが止まるまで保持します。Windowsは管理者のPowerShellで `ssh-agent` サービスを自動起動に設定します。macOSはlaunchdのagentを確認し、`UseKeychain yes` でパスフレーズをキーチェーンに保存します。Linuxはagentへの到達を確認するだけで、keyring・systemdユーザーサービス・シェルプロファイルから選ぶ手順を案内します。
4. **ローカルの端末**：最後のステップの `ssh 別名` を実行します。

公開鍵のみの場合は配置ステップだけを表示し、対応する秘密鍵を別途用意する案内を示します。ガイドには手動解除案内も含みます。SSHサーバーとネットワークは事前に利用可能にし、カスタム構成の接続先では実際の `AuthorizedKeysFile` / `Match` を確認してください。Windowsの追記スクリプトはUTF-8の既存ファイルを扱い、UTF-16などは書き換えず停止します。

鍵と接続先の関連付けはローカルのメモです。保存・表示・コピーをしても登録や接続の成功を意味せず、手動作業後の成功を記録する機能もありません。アプリはスクリプトやSSH接続・登録・解除を実行せず、SSH設定やIncludeも直接編集しません。コピーしたスクリプトを利用者が実行したときに公開鍵やローカルconfigを追記します。鍵ファイルの削除は利用者が必要性を確認して手動で行います。

CLIでも同じ操作を直接指定できます。各コマンドの詳細は `--help` を参照してください。

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list --json
ssh-keyflow hosts list --json
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub
ssh-keyflow keys public --key laptop
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts associate --host my-server --key existing
ssh-keyflow hosts dissociate --host my-server --key existing
ssh-keyflow hosts remove --host my-server
ssh-keyflow keys remove --key existing
ssh-keyflow hosts guide --host my-server --key laptop
ssh-keyflow hosts guide --host my-server --key laptop --local-os linux --identity-file /home/alice/.ssh/id_ed25519 --remote-os windows --windows-account standard
ssh-keyflow resume --id creation-example
ssh-keyflow tui
```

`keys import` に `--private /absolute/path/id_ed25519` を追加すると鍵ペアを検証して取り込みます。省略すると公開鍵のみです。パスは自分の環境の絶対パスに置き換えてください。`hosts add` の `--port` は既定で22、`--os` は `linux` / `macos` / `windows`。Windowsでは `--windows-account standard` または `--windows-account administrator` も指定します。`resume` のIDは `keys list` に表示される未完了のローカル鍵作成IDに置き換えてください。

`keys remove` / `hosts remove` は確認後に管理対象とその全関連付けを削除します。`hosts dissociate` は指定した組の関連付けだけを解除し、鍵と接続先は残します。TUIでは一覧・詳細で `d`、関連付けの解除は `u` を押して既存の組を選びます。いずれも鍵ファイル・SSH設定・接続先の公開鍵登録を変更せず、SSHアクセスを失効させません。ファイルが見つからない鍵も管理から削除できます。未完了の旧鍵削除がある場合は手動確認が必要です。作成結果と旧履歴を保つため削除したメタデータは状態内の `removed_keys` / `removed_hosts` に保持し、再取り込み・同じ別名の接続先追加では新しいIDを割り当てます。これらの配列を含む状態を旧バイナリは読めないため、旧版へ戻す場合は事前に別途保管した状態を使ってください。

接続先は固定IDまたは別名、鍵は固定IDまたは一意の管理名で指定します。曖昧な鍵名を自動選択しません。`hosts guide` は接続先と鍵を明示する読み取り専用操作で、関連付けを保存しません。変更操作には確認用の実端末が必要です。パスフレーズはOSの `ssh-keygen` に直接入力します。`list`・`keys public`・`hosts guide` はリダイレクトでき、読み取りだけで保存領域を作成・移行しません。

`hosts guide` の `--local-os` と `--remote-os` は `linux` / `macos` / `windows` を指定できます。`--windows-account` は接続先Windows用です。省略時は実行元OS・保存済み接続先OSとアカウントを使います。`--identity-file` はローカル端末から見た秘密鍵パスで、実行元と異なるローカルOSを選んだ鍵ペアでは必須です。指定OSの絶対パスまたは `~/path` を使ってください。Windows版からWSLへ貼り付ける場合は `--local-os linux` を選び、WSLから見える対応する鍵のパスを指定します。アプリはそのパスの鍵を検証・転送せず、ドライブ対応も推測しません。これらの指定はガイドだけに適用し、保存済み接続先や管理済み鍵のパスを変更しません。

JSONは標準出力、診断は標準エラーへ分けます。`keys list --json` は `keys` と `creations`、`hosts list --json` は `hosts` と `associations` を返します。終了コードは成功0、実行失敗・取消1、引数・使用法エラー2です。非端末の引数なし起動はヘルプを表示して0で終了します。

## 安全条件とデータ

管理情報は `~/.ssh/ssh-keyflow/state.json`、生成鍵は同ディレクトリの `keys/` に保存します。取り込みは明示したファイルを参照し、元ファイルは移動・書き換えません。公開鍵の指紋と秘密鍵ファイルのダイジェストを検証し、不一致や読み取り不能は要確認として止めます。秘密鍵の内容やパスフレーズは状態JSON・引数へ保存しません。

鍵作成はネイティブの `ssh-keygen` を使い、既存パスを上書きしません。作成が中断した場合は `keys list` の作成IDで `resume` してください。保存済みのパス・指紋・ダイジェストとファイルを照合し、部分ファイルを自動削除せず再開します。未完了の作成がある間は別の鍵ペアを新規作成しません。状態変更は排他ロックで保護し、検証済みの直前状態を `state.json.bak` に保存してから原子的に置き換えます。不明なロックや不正な状態は自動修復しません。

v0.5.0の簡素化は意図的な非互換変更です。従来の `register`・`unregister`・`registrations list`・`keys delete` と、`--new-key`・`--auth`・`--identity`・`--configure` は使えません。旧番号メニュー、Tasks、再帰的なSSH設定探索も廃止します。既存の `Host` / Include / ワイルドカード設定との重複・優先順位は、スニペットを貼り付ける前に利用者が確認してください。

既存のv1状態は読み取り時には変更せず、最初の状態保存でv2になります。新しい `associations` は旧登録履歴から独立しています。過去の登録・接続確認・設定編集・削除の履歴と中断記録は保持し、現在の成功・完了へ書き換えません。旧リモート操作は読み取り専用で、`resume` は引き継ぎません。旧削除が途中の鍵は手動確認が必要なため使用を拒否しますが、他の鍵の操作は妨げません。

**旧版へ戻す可能性がある場合は、v0.5.0以降で最初の変更操作を行う前に、アプリを終了してv1の `state.json` を別の場所に保管してください。** `state.json.bak` は直前状態を保存する更新式バックアップであり、永久にv1を残すものではありません。初回の鍵作成だけでも複数回保存するため、同じ操作中にv2へ更新され得ます。旧版はv2を読めないため、ダウングレードには保管済みv1状態からの手動復元と、移行後に追加した鍵・関連付け等の確認が必要です。読み取りだけで移行用コピーを自動作成しません。

過去に生成した `~/.ssh/ssh-keyflow/config`、通常の `~/.ssh/config`、Include、既存バックアップは変更しません。残っているIncludeは従来の設定を引き続き読み込み得ます。必要なら利用者が内容を確認してください。過去の検証結果は [検証記録](docs/verification.md)、現在の仕様は [アプリケーション契約](docs/application-contract.md) を参照してください。

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

旧配置先は既定でLinux/macOSの `~/.local/bin`、Windowsの `%LOCALAPPDATA%\Programs\ssh-keyflow` です。本体と旧所有記録を変更していなければ、新しいインストーラーで更新できます。既存のPATH追記を保持し、新たに同梱ライセンス文書も管理します。v0.2.0のアーカイブは実行ファイル1個だけを要求する旧インストーラーには対応しないため、古いスクリプトを再利用しないでください。

手動差し替え後は本体のハッシュと旧所有記録が一致せず、旧スクリプトの更新・削除も拒否され得ます。以後の更新・削除は手動で行ってください。旧所有記録（POSIXの `.ssh-keyflow-install/`、Windowsの `.ssh-keyflow-install.json`）と既存PATHは自動移行・自動編集しません。記録はそのまま残せます。

旧PATHの追記を外す場合は所有記録が示すプロファイルと専用の `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` 範囲を確認し、必要な範囲だけ手動で編集してください。プロファイル全体やSSHデータは削除しません。Windowsの旧PATHはレジストリではなく実行したPowerShellのプロファイルにあり、Windows PowerShellとPowerShell 7では対象が異なります。

## 開発とリリース契約

Goは `go.mod` の指定、梱包テストはPython 3.12の標準ライブラリを使います。TUIはBubble Tea / Bubbles / Lip Gloss、CLIはCobraを使い、ローカル鍵処理と状態管理を共通化します。

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
python scripts/package_release.py --version v0.0.0-check --output dist-check
```

梱包は `CGO_ENABLED=0`、`-buildvcs=false`、`-trimpath` と `-ldflags "-s -w -X main.version=<tag>"` で6対象をビルドします。開発版は `dev`。上の `v0.0.0-check` はローカル検証用の版注入例で、公開は行いません。既存ファイルがある出力先への上書きは拒否します。既存バイナリの梱包には `--binaries DIRECTORY` を指定し、`DIRECTORY/linux_amd64/ssh-keyflow` 等の6対象から読み込みます。

| OS | CPU | アセット名 |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

アーカイブ内部は `ssh-keyflow`（Windowsは `ssh-keyflow.exe`）と `THIRD_PARTY_NOTICES.txt`。Unix実行権限は0755です。トップレベルの配布物は6アーカイブと `SHA256SUMS` の7件だけです。manifestは6アーカイブについて、64桁SHA256・半角スペース2個・ベース名・LFで記録します。

CIは3 OSでGoテスト、Portable生成物テスト、インストーラーテストを実行し、6対象の実クロスビルドとLinux amd64版の版注入を確認します。`v*` タグのpushでCI成功後に7件を `GITHUB_REPOSITORY` へ公開します。インストーラーは `main` のソースから別途配信し、7件のリリース配布物は変更しません。プレリリースタグはlatest正式版にしません。

リリースは、準備PRの統合、最終候補の検証、タグ作成・公開、公開後の配布物検証と記録PRの統合の順で進めます。準備記録には検証したソースのコミットを使い、最終タグのコミットはタグ・リリース情報と公開後の記録に残します。公開後の記録でmainがタグより先へ進むことは通常の運用で、記録だけの変更による再リリースは不要です。各段階の完了条件と承認範囲は [AGENTS.mdのTags and releases](AGENTS.md#tags-and-releases) を参照してください。

公開ジョブはタグをまたいで直列化し、その中で現在のlatestを取得して数値のバージョン順で比較します。旧系列の修正版は公開できますが、新しいlatestを置き換えません。latestが404の場合は全リリースの一覧取得に成功し、公開済み正式版がないと確認できた場合だけ初回公開として扱います。認証・通信・API応答の異常や、正式版があるのにlatestが取得できない場合は公開を停止します。`scripts/test_release.py` は外部公開なしでこれらの判定と公開コマンドを検証します。

テストは一時ホーム・配置先・配布物を使い、実ユーザーのSSHデータ、永続PATH、プロファイル、接続先を変更しません。クロスビルドは各OS / CPUでの実機実行の証拠とは区別します。確認済み範囲は [docs/verification.md](docs/verification.md)、公開インターフェースは [docs/application-contract.md](docs/application-contract.md) を参照してください。
