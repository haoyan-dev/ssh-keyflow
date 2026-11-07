# SSH KeyFlow

自分の端末への SSH 接続に使う鍵、公開鍵登録、接続設定と登録履歴を管理する Go 製の TUI / CLI です。コマンド名は `ssh-keyflow`、Windows では `ssh-keyflow.exe` です。

## 起動と操作

端末で引数なし、または `ssh-keyflow tui` を実行すると一覧中心のTUIが開きます。Tab / Shift+Tabで接続先・鍵・登録履歴を切り替え、矢印キーで選択、Enterで詳細、`?` でヘルプを表示します。`g` は管理対象外の鍵・設定と要確認事項の読み取り専用一覧です。

| キー | 操作 |
| --- | --- |
| `a` | 接続先追加 |
| `c` / `i` | Ed25519鍵の作成 / 既存鍵の取り込み |
| `r` / `u` | 公開鍵の登録 / 登録解除 |
| `d` | ローカル鍵の削除 |
| `s` | 保存した操作の再開 |
| Esc | 入力・実行前確認から戻る |
| `q` | 終了 |

フォームに入力して実行前の対象と内容を確認します。実行に必要な承認は共通処理が別途求めます。パスワードとパスフレーズはOSのOpenSSHへ直接入力し、終了・失敗・中断後はTUIへ戻って保存状態を確認できます。既に完了した変更はキャンセルで巻き戻りません。

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

配布元は [GitHub Releases](https://github.com/haoyan-dev/ssh-keyflow/releases) です。v0.2.0の実装完了だけでは公開されません。対象タグの6アーカイブと `SHA256SUMS` のうち、自分のOS / CPUのアーカイブとmanifestを取得してください。

例：`v0.2.0` が公開されている場合、Linux amd64では次のように検証して展開します。

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

## 更新・PATH・本体削除

更新前にアプリを終了し、新しい配布物のチェックサムを検証して別のディレクトリへ展開します。その後、既存の実行ファイルと同梱ライセンス文書を新しいものへ差し替え、`--version` を確認してください。本体は移動しても従来のホーム配下の状態・鍵・生成設定を使います。

PATHは必要な場合だけ自分で設定します。現在のセッションだけなら、POSIXシェルでは `export PATH="/absolute/path/ssh-keyflow-portable:$PATH"`、PowerShellでは `$env:PATH = 'C:\absolute\path\ssh-keyflow-portable;' + $env:PATH` を実行します。永続化する場合は使うシェルのプロファイルやOSのPATH設定を自分で編集してください。本体はPATH・プロファイル・レジストリを自動変更しません。

本体を削除するときはアプリを終了し、実行ファイルと同梱ライセンス文書を削除します。自分で追加したPATHは必要に応じて取り除いてください。鍵、状態、SSH設定、登録履歴、接続先の公開鍵登録は保持されます。本体削除は登録解除やローカル鍵削除とは別です。

## v0.1.0のインストーラー利用者

旧配置先は既定でLinux/macOSの `~/.local/bin`、Windowsの `%LOCALAPPDATA%\Programs\ssh-keyflow` です。旧PATHを引き続き使う場合は、その場所の本体を終了後に手動で差し替えてください。v0.2.0のアーカイブにはライセンス文書もあるため、実行ファイル1個だけを要求する旧インストーラーでは更新できません。

手動差し替え後は本体のハッシュと旧所有記録が一致せず、旧スクリプトの更新・削除も拒否され得ます。以後の更新・削除は手動で行ってください。旧所有記録（POSIXの `.ssh-keyflow-install/`、Windowsの `.ssh-keyflow-install.json`）と既存PATHは自動移行・自動編集しません。記録はそのまま残せます。

旧PATHの追記を外す場合は所有記録が示すプロファイルと専用の `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` 範囲を確認し、必要な範囲だけ手動で編集してください。プロファイル全体やSSHデータは削除しません。Windowsの旧PATHはレジストリではなく実行したPowerShellのプロファイルにあり、Windows PowerShellとPowerShell 7では対象が異なります。

## 開発とリリース契約

Goは `go.mod` の指定、梱包テストはPython 3.12の標準ライブラリを使います。TUIはBubble Tea / Bubbles / Lip Gloss、CLIはCobra、SSH処理と状態管理は既存実装を再利用します。

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/package_release.py --version v0.2.0 --output dist
```

梱包は `CGO_ENABLED=0`、`-buildvcs=false`、`-trimpath` と `-ldflags "-s -w -X main.version=<tag>"` で6対象をビルドします。開発版は `dev`。既存ファイルがある出力先への上書きは拒否します。既存バイナリの梱包には `--binaries DIRECTORY` を指定し、`DIRECTORY/linux_amd64/ssh-keyflow` 等の6対象から読み込みます。

| OS | CPU | アセット名 |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

アーカイブ内部は `ssh-keyflow`（Windowsは `ssh-keyflow.exe`）と `THIRD_PARTY_NOTICES.txt`。Unix実行権限は0755です。トップレベルの配布物は6アーカイブと `SHA256SUMS` の7件だけです。manifestは6アーカイブについて、64桁SHA256・半角スペース2個・ベース名・LFで記録します。

CIは3 OSでGoテストとPortable生成物テストを実行し、6対象の実クロスビルドとLinux amd64版の版注入を確認します。`v*` タグのpushでCI成功後に7件を `GITHUB_REPOSITORY` へ公開します。プレリリースタグはlatest正式版にしません。今回の実装作業ではタグ・push・公開を行いません。

テストは一時ホーム・配置先・配布物を使い、実ユーザーのSSHデータ、永続PATH、プロファイル、接続先を変更しません。クロスビルドは各OS / CPUでの実機実行の証拠とは区別します。確認済み範囲は [docs/verification.md](docs/verification.md)、公開インターフェースは [docs/application-contract.md](docs/application-contract.md) を参照してください。
