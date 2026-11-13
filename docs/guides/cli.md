# CLIの使い方

[简体中文](../zh-CN/guides/cli.md) · [English](../en/guides/cli.md) · 日本語

[ドキュメント一覧](../README.md)

## コマンドの例

CLIでも同じ操作を直接指定できます。各コマンドの詳細は `--help` を参照してください。

例の名前・ホスト名・パスは自分の環境へ置き換えてください。以下は目的別の例で、全ブロックを続けて実行する手順ではありません。初回は[チュートリアル](../tutorials/first-connection.md)を使ってください。

### ヘルプ・版・一覧

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list --json
ssh-keyflow hosts list --json
ssh-keyflow tui
```

### 鍵の作成・取り込み・公開鍵表示

作成と取り込みは必要な方を選びます。`existing` は取り込む公開鍵の管理名です。

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub
ssh-keyflow keys public --key laptop
```

### 接続先と関連付け

`laptop` と `existing` は作成・取り込み済みの鍵を指定します。

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts associate --host my-server --key existing
```

### 手動設定ガイド

```sh
ssh-keyflow hosts guide --host my-server --key laptop
ssh-keyflow hosts guide --host my-server --key laptop --local-os linux --identity-file /home/alice/.ssh/id_ed25519 --remote-os windows --windows-account standard
```

### ホスト鍵の固定

```sh
ssh-keyflow hosts pin --host my-server
ssh-keyflow hosts pin --host my-server --file host-keys.txt --yes
ssh-keyflow hosts pin --host my-server --replace
ssh-keyflow hosts unpin --host my-server
```

`hosts pin` は接続先で実行するコマンドを表示します。接続先自身の端末で出力したホスト公開鍵を貼り付け、空行でEnterを押します（Windowsは管理者として開いたPowerShellで表示します）。各鍵の種類とSHA256指紋を確認し、`yes` を入力すると `~/.ssh/ssh-keyflow/known_hosts` に保存します。`--file` は出力を保存したファイルを読み、パイプやファイルからの入力には `--yes` が必要です。`--yes` は異なる鍵が固定済みの場合は置き換えません。接続先のホスト鍵が変わったことを接続先自身で確認した場合だけ、実端末で `--replace` を指定してください。`hosts unpin` は実端末での確認後に固定を解除します。SSH configに追加した固定用の3行は自分で削除してください。

固定後に `hosts guide` を再度実行すると、Hostスニペットに `HostKeyAlias`・`UserKnownHostsFile`・`StrictHostKeyChecking yes` が加わります。既にSSH configにあるHostには、この3行を自分で追加してください。`hosts list` は固定状態を `HOST KEYS` 列に表示します。接続先を管理から削除しても固定は残るため、不要なら削除前に `hosts unpin` を実行します。

### 中断した鍵作成の再開

```sh
ssh-keyflow resume --id creation-example
```

`keys import` に `--private /absolute/path/id_ed25519` を追加すると鍵ペアを検証して取り込みます。省略すると公開鍵のみです。パスは自分の環境の絶対パスに置き換えてください。`hosts add` の `--port` は既定で22、`--os` は `linux` / `macos` / `windows`。Windowsでは `--windows-account standard` または `--windows-account administrator` も指定します。`resume` のIDは `keys list` に表示される未完了のローカル鍵作成IDに置き換えてください。

## 管理からの削除と関連付け解除

整理したい対象に応じて選びます。以下も独立した操作例です。

```sh
ssh-keyflow hosts dissociate --host my-server --key existing
ssh-keyflow hosts remove --host my-server
ssh-keyflow keys remove --key existing
```

`keys remove` / `hosts remove` は確認後に管理対象とその全関連付けを削除します。`hosts dissociate` は指定した組の関連付けだけを解除し、鍵と接続先は残します。TUIでは一覧・詳細で `d`、関連付けの解除は `u` を押して既存の組を選びます。いずれも鍵ファイル・SSH設定・接続先の公開鍵登録を変更せず、SSHアクセスを失効させません。ファイルが見つからない鍵も管理から削除できます。未完了の旧鍵削除がある場合は手動確認が必要です。作成結果と旧履歴を保つため削除したメタデータは状態内の `removed_keys` / `removed_hosts` に保持し、再取り込み・同じ別名の接続先追加では新しいIDを割り当てます。これらの配列を含む状態を旧バイナリは読めないため、旧版へ戻す場合は事前に別途保管した状態を使ってください。

接続先は固定IDまたは別名、鍵は固定IDまたは一意の管理名で指定します。曖昧な鍵名を自動選択しません。`hosts guide` は接続先と鍵を明示する読み取り専用操作で、関連付けを保存しません。変更操作には確認用の実端末が必要です。パスフレーズはOSの `ssh-keygen` に直接入力します。`list`・`keys public`・`hosts guide` はリダイレクトでき、読み取りだけで保存領域を作成・移行しません。

## ガイド用のOSと秘密鍵パス

`hosts guide` の `--local-os` と `--remote-os` は `linux` / `macos` / `windows` を指定できます。`--windows-account` は接続先Windows用です。省略時は実行元OS・保存済み接続先OSとアカウントを使います。`--identity-file` はローカル端末から見た秘密鍵パスで、実行元と異なるローカルOSを選んだ鍵ペアでは必須です。指定OSの絶対パスまたは `~/path` を使ってください。公開鍵のみの鍵では保存済みの公開鍵パスを `IdentityFile` にするため、実行元と異なるローカルOSでは `--identity-file` にその端末から見た公開鍵パスを指定します。ホスト鍵の固定に関するステップと行は、ローカルOSが実行元と同じ場合だけ表示します。Windows版からWSLへ貼り付ける場合は `--local-os linux` を選び、WSLから見える対応する鍵のパスを指定します。アプリはそのパスの鍵を検証・転送せず、ドライブ対応も推測しません。これらの指定はガイドだけに適用し、保存済み接続先や管理済み鍵のパスを変更しません。

## JSONと終了コード

JSONは標準出力、診断は標準エラーへ分けます。`keys list --json` は `keys` と `creations`、`hosts list --json` は `hosts`・`associations`・`host_key_pins`（接続先IDごとの種類と指紋）を返します。固定ファイルを読めない場合は `host_key_pins` を省き、理由を標準エラーに出します。終了コードは成功0、実行失敗・取消1、引数・使用法エラー2です。非端末の引数なし起動はヘルプを表示して0で終了します。

手順に沿って試す場合は[はじめての接続設定](../tutorials/first-connection.md)、正確な入力・出力仕様は[アプリケーション契約](../application-contract.md#cli)を参照してください。
