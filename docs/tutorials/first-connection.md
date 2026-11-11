# はじめての接続設定

[ドキュメント一覧](../README.md)

鍵を1つ作成し、接続先と関連付け、手動設定ガイドを表示します。最後のSSH接続は自分で実行して確認します。アプリ内に保存した関連付けだけでは、接続先の設定は変わりません。

## 事前に用意するもの

- [インストール済みのSSH KeyFlow](../guides/installation.md)と対話端末。
- ローカル端末の `ssh` と `ssh-keygen`。
- SSHサーバーが利用可能で、自分が設定してよい接続先。公開鍵をまだ登録していない場合も、その端末へ別の方法でログインできること。
- 接続先のホスト名・ユーザー名・OS。以下の `server.example` と `alice` は、自分の接続先に置き換えます。

旧版から利用している場合は、変更操作の前に[状態の保管と移行](../guides/data-and-recovery.md)を確認してください。

## 1. 起動を確認する

ローカルの端末で実行します。Windowsでも同じコマンドを使えます。

```sh
ssh-keyflow --version
ssh-keyflow keys list
```

版が表示され、鍵一覧を読めれば次へ進めます。鍵がなくても問題ありません。開発ビルドの版は `dev` です。一覧の読み取りだけでは保存領域を作成・移行しません。

## 2. 鍵を用意する

新しい鍵が必要な場合:

```sh
ssh-keyflow keys create --name laptop
```

表示された内容を確認し、確認要求に `yes` を入力します。パスフレーズはOSの `ssh-keygen` へ直接入力します。既存パスは上書きしません。

既存鍵を使う場合は、代わりに次を実行します。2つのパスを自分の鍵ペアの絶対パスに置き換えてください。Windowsでは `C:\Users\alice\.ssh\id_ed25519` などを使います。

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

取り込みは元の鍵ファイルを移動・書き換えません。`--private` を省略すると公開鍵のみとなり、この後のガイドにはローカルconfig・接続ステップが含まれません。

```sh
ssh-keyflow keys list
ssh-keyflow keys public --key laptop
```

管理名 `laptop` と公開鍵を確認します。同じ管理名が複数ある場合は一覧にある固定IDを使ってください。作成が中断した場合は[復旧ガイド](../guides/troubleshooting.md#鍵作成が中断した)へ進みます。

## 3. 接続先を保存する

以下はLinux接続先の例です。`my-server` は自分で選ぶ別名です。既存のSSH configと重複しない名前を選びます。

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts list
```

確認後、接続先と鍵の関連付けがローカルに保存されます。既定のSSHポートは22です。別のポートは `--port`、macOS接続先は `--os macos` を指定します。Windows接続先には `--os windows --windows-account standard`、または管理者アカウント用の `--windows-account administrator` が必要です。

## 4. 手動設定ガイドを開く

```sh
ssh-keyflow hosts guide --host my-server --key laptop
```

出力には、各ステップを実行する端末とスクリプトが示されます。表示だけでは設定を変更しません。

1. **接続先の端末**で、公開鍵配置ステップを確認して実行します。
2. **ローカルの端末**で、既存のHost / Include / Match / ワイルドカードと優先順位を確認した後、config追記ステップを実行します。同じ明示的な別名があると追記は停止します。
3. パスフレーズ付き、または判定できない鍵では、**ローカルの端末**向けのssh-agentステップを読み、必要な設定を行います。Windowsのサービス設定には管理者PowerShellが必要です。Linuxではagentの選択・設定を自分で行います。
4. **ローカルの端末**で、最後の接続ステップを実行します。この例では `ssh my-server` です。初回接続では接続先のホスト鍵を信頼できる方法で照合してください。

アプリ実行元とは別のOSへ貼り付ける場合は、[CLIのガイド用OS指定](../guides/cli.md)を先に確認してください。アプリは鍵を別端末へ転送せず、パスの対応も推測しません。

## 5. 結果と次の操作

自分のSSH端末で接続できたことを確認します。SSH KeyFlowは接続を検証せず、成功の記録操作も提供しません。接続できない場合は[トラブルシューティング](../guides/troubleshooting.md)を参照してください。

TUIで同じ管理を行う場合は `ssh-keyflow` を起動し、[TUIの操作](../guides/terminal-ui.md)を参照してください。ガイドでは表示中のステップだけを `y` でコピーできます。

関連付けの解除や管理からの削除は[CLIガイド](../guides/cli.md)を参照してください。どちらもSSHアクセスを失効させません。接続先から公開鍵を外す場合は、手動設定ガイドの解除案内を読み、別のログイン手段を確保したうえで作業してください。
