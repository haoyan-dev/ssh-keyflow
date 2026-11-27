# CLIの使い方

[简体中文](../zh-CN/guides/cli.md) · [English](../en/guides/cli.md) · 日本語

[ドキュメント一覧](../README.md)

## コマンドの例

SSH KeyFlowはCLIだけで操作します。各コマンドの詳細は `--help` を参照してください。引数なしで起動するとヘルプを表示します。

例の名前・パスは自分の環境へ置き換えてください。以下は目的別の例で、全ブロックを続けて実行する手順ではありません。初回は[チュートリアル](../tutorials/first-connection.md)を使ってください。

### ヘルプ・版・一覧

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list
ssh-keyflow keys list --json
```

### 鍵の作成・取り込み・公開鍵表示

作成と取り込みは必要な方を選びます。取り込みには公開鍵と秘密鍵の両方のパスが必要です。

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
ssh-keyflow keys public --key laptop
```

`keys public` は公開鍵を1行で標準出力へ出します。ファイルのコメントは含めません。リダイレクトやクリップボードへのパイプに使えます。

```sh
ssh-keyflow keys public --key laptop > laptop.pub
ssh-keyflow keys public --key laptop | pbcopy            # macOS
ssh-keyflow keys public --key laptop | Set-Clipboard     # Windows PowerShell
```

### 中断した鍵作成の再開

```sh
ssh-keyflow resume --id creation-example
```

IDは `keys list` に表示される未完了のローカル鍵作成IDに置き換えてください。

## 管理からの削除

```sh
ssh-keyflow keys remove --key existing
```

確認後に鍵を管理対象から外します。鍵ファイル・SSH設定・接続先の公開鍵登録は変更せず、SSHアクセスを失効させません。ファイルが見つからない鍵も管理から削除できます。同じ鍵ペアは `keys import` で再び管理できます（新しいIDになります）。

鍵は固定IDまたは一意の管理名で指定します。曖昧な名前を自動選択しません。変更操作（create / import / remove / resume）には確認用の実端末が必要です。パスフレーズはOSの `ssh-keygen` に直接入力します。`keys list` と `keys public` はリダイレクトでき、読み取りだけで保存領域を作成・移行しません。

## 廃止したコマンド

v0.11.0で `hosts`（add / list / remove / associate / dissociate / guide / pin / unpin）、`tui`、`keys import --public-only` を廃止しました。指定すると使用法エラー（終了コード2）になります。公開鍵の配置とSSH configの設定は自分で行ってください。手順の例は[チュートリアル](../tutorials/first-connection.md)にあります。

## JSONと終了コード

JSONは標準出力、診断は標準エラーへ分けます。`keys list --json` は `keys` と `creations` だけを返します。終了コードは成功0、実行失敗・取消1、引数・使用法エラー2です。

正確な入力・出力仕様は[アプリケーション契約](../application-contract.md#cli)を参照してください。
