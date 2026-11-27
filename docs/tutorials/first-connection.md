# はじめての接続設定

[简体中文](../zh-CN/tutorials/first-connection.md) · [English](../en/tutorials/first-connection.md) · 日本語

[ドキュメント一覧](../README.md)

鍵を1つ用意し、その公開鍵を接続先へ自分で配置して、SSH接続を確認します。SSH KeyFlowが行うのは鍵の作成・取り込みと公開鍵の表示までで、接続先の設定やSSH接続は行いません。

## 事前に用意するもの

- [インストール済みのSSH KeyFlow](../guides/installation.md)と対話端末。
- ローカル端末の `ssh` と `ssh-keygen`。
- SSHサーバーが利用可能で、自分が設定してよい接続先。公開鍵をまだ登録していない場合も、その端末へ別の方法でログインできること。
- 接続先のホスト名とユーザー名。以下の `server.example` と `alice` は、自分の接続先に置き換えます。

旧版から利用している場合は、変更操作の前に[状態の移行](../guides/data-and-recovery.md#v0100以前からの移行)を確認してください。

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

既存の鍵ペアを使う場合は、代わりに次を実行します。2つのパスを自分の鍵ペアの絶対パスに置き換えてください。Windowsでは `C:\Users\alice\.ssh\id_ed25519` などを使います。

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

取り込みは元の鍵ファイルを移動・書き換えません。

```sh
ssh-keyflow keys list
```

管理名 `laptop`、指紋、公開鍵のパスを確認します。秘密鍵のパスは `keys list --json` の `private_path` で確認できます。作成が中断した場合は[復旧ガイド](../guides/troubleshooting.md#鍵作成が中断した)へ進みます。

## 3. 公開鍵を接続先へ配置する

ローカル端末で公開鍵を表示し、1行のままコピーします。

```sh
ssh-keyflow keys public --key laptop
```

**接続先の端末**（別の方法でログインした端末）で、ログインユーザーの `authorized_keys` に追記します。Linux / macOSの接続先では次のとおりです。`<公開鍵>` をコピーした1行に置き換えます。

```sh
mkdir -p ~/.ssh && chmod 700 ~/.ssh
printf '%s\n' '<公開鍵>' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Windowsの接続先では、標準アカウントは `%USERPROFILE%\.ssh\authorized_keys`、管理者アカウントは `%ProgramData%\ssh\administrators_authorized_keys` に追記します。管理者用のファイルは書き込み権限をSYSTEMとAdministratorsに限定する必要があります。既存の行は消さないでください。

## 4. 接続する

ローカル端末で、秘密鍵を指定して接続します。`<秘密鍵パス>` は手順2で確認したパスです。

```sh
ssh -i <秘密鍵パス> -o IdentitiesOnly=yes alice@server.example
```

初回接続では、接続先のホスト鍵の指紋を信頼できる方法で照合してから `yes` を入力してください。毎回の指定を省くには、`~/.ssh/config` に次のようなHostブロックを自分で追加します。既存のHost / Include / Matchと重複しないか確認してください。

```text
Host my-server
  HostName server.example
  User alice
  IdentityFile <秘密鍵パス>
  IdentitiesOnly yes
```

## 5. 結果と次の操作

自分のSSH端末で接続できたことを確認します。SSH KeyFlowは接続を検証せず、成功の記録操作も提供しません。接続できない場合は[トラブルシューティング](../guides/troubleshooting.md)を参照してください。

管理からの削除は[CLIガイド](../guides/cli.md#管理からの削除)を参照してください。SSHアクセスは失効しません。接続先から公開鍵を外す場合は、別のログイン手段を確保したうえで、`authorized_keys` から該当する1行を自分で取り除いてください。
