# CLIの使い方

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

`hosts guide` の `--local-os` と `--remote-os` は `linux` / `macos` / `windows` を指定できます。`--windows-account` は接続先Windows用です。省略時は実行元OS・保存済み接続先OSとアカウントを使います。`--identity-file` はローカル端末から見た秘密鍵パスで、実行元と異なるローカルOSを選んだ鍵ペアでは必須です。指定OSの絶対パスまたは `~/path` を使ってください。Windows版からWSLへ貼り付ける場合は `--local-os linux` を選び、WSLから見える対応する鍵のパスを指定します。アプリはそのパスの鍵を検証・転送せず、ドライブ対応も推測しません。これらの指定はガイドだけに適用し、保存済み接続先や管理済み鍵のパスを変更しません。

## JSONと終了コード

JSONは標準出力、診断は標準エラーへ分けます。`keys list --json` は `keys` と `creations`、`hosts list --json` は `hosts` と `associations` を返します。終了コードは成功0、実行失敗・取消1、引数・使用法エラー2です。非端末の引数なし起動はヘルプを表示して0で終了します。

手順に沿って試す場合は[はじめての接続設定](../tutorials/first-connection.md)、正確な入力・出力仕様は[アプリケーション契約](../application-contract.md#cli)を参照してください。
