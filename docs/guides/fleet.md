# フリートの名簿を作成・受け入れる

[简体中文](../zh-CN/guides/fleet.md) · [English](../en/guides/fleet.md) · 日本語

[ドキュメント一覧](../README.md)

> このページは `main` の未リリース機能（フリート同期フェーズ1）を説明します。公開済みの版にはありません。

フリートは、同じフリートルート鍵を信頼の起点として固定した、自分の端末（ノード）の集まりです。管理ノードが名簿（ノードと公開鍵の一覧）に署名し、ほかのノードはファイルで受け取った名簿を、固定したルート公開鍵で検証してから受け入れます。フェーズ1は名簿を作って配るところまでで、`authorized_keys`・known_hosts・SSH configは書き込みません。用語は[用語集](../../GLOSSARY.md)を参照してください。

## 前提

- 署名と検証にはOpenSSH 8.4以上が必要です。PATH上の最初の `ssh-keygen` を使い、同じディレクトリの `ssh -V` で版を確認します。`fleet init --dry-run` などの `--dry-run` で、使われる `ssh-keygen` のパスと版を確認できます。
- 状態を変更するコマンドは、計画を表示してから実端末で `yes` を求めます。`--dry-run` を付けると計画だけを表示し、何も変更しません。
- `main` で初めて状態を変更すると（フリートのコマンドに限らず、鍵の作成なども含みます）、v0.11.0の `state.json`（スキーマv3）は `state.v3-<UTC時刻>.json` へ保管されてからv4で保存されます。v0.11.0以前はv4を読めません。事前に[データ・移行・復旧](data-and-recovery.md)を確認してください。

## 1. 各端末でノード記述を設定する

接続元（`client`）はSSHで接続する側、接続先（`remote`）は接続される側です。接続元には管理対象の鍵が必要です（`keys create` または `keys import`）。

```sh
# 接続元だけのノード
ssh-keyflow node set --name laptop --roles client --key laptop

# 接続元と接続先を兼ねるノード
ssh-keyflow node set --name desk --roles client,remote --key desk --addresses desk.lan,192.0.2.10
ssh-keyflow node show
```

接続先のホスト公開鍵は `sshd_config` の `HostKey` 行（既定はUnixで `/etc/ssh/sshd_config`、Windowsで `%ProgramData%\ssh\sshd_config`）から読みます。別の場所なら `--sshd-config` を指定します。ポートが22以外なら `--port`、共通のログインアカウントと違う場合は `--login-account` を指定します。リスト値はカンマ区切りの1つの値で指定します。

## 2. 管理ノードでフリートを作成する

名簿に署名する端末を1台選びます。

```sh
ssh-keyflow fleet init --login-account user --dry-run
ssh-keyflow fleet init --login-account user
```

`--login-account` は全接続先で使う共通のログインアカウントです。フリートルート鍵は `~/.ssh/ssh-keyflow/fleet/root_ed25519` に作られ、`ssh-keygen` がパスフレーズを直接求めます。空のパスフレーズは使えません。この鍵は再作成できないため、`~/.ssh/ssh-keyflow/` ごとアクセス制限された場所へ保管してください。

署名だけが失敗した場合は、同じコマンドを再実行すると作成済みの鍵で版1に署名します。管理ノード自身のノード記述を変えたら `fleet sign` で次の版に署名します。

## 3. ほかのノードを名簿に加える

追加する端末でノード記述ファイルを書き出し、管理ノードへコピーします。

```sh
ssh-keyflow node export --out ./desk-node
```

管理ノードで追加し、次の版に署名します。

```sh
ssh-keyflow fleet add ./desk-node
```

表示された名前・役割・指紋を、追加する端末の `node show` と照合してから `yes` と入力してください。同じノードを再度追加すると、新旧の値を並べた置換になります。

## 4. 名簿を配って受け入れる

管理ノードで名簿を書き出し、受け入れる端末へコピーします。

```sh
ssh-keyflow fleet export --out ./roster
```

`roster.json`・`roster.json.sig`・`fleet-root.pub` の3ファイルができます。`--out` は存在しないか空のディレクトリにしてください。受け入れる端末で次を実行します。

```sh
ssh-keyflow fleet accept ./roster --dry-run
ssh-keyflow fleet accept ./roster
ssh-keyflow fleet show
```

初回はルート公開鍵の指紋とフリートIDが表示されます。管理ノードの `fleet show` と一致することを確認してから `yes` と入力してください。この操作でルートが固定され、以後は固定したルートの署名だけを受け入れます。

`roster.json` は署名されたバイト列のまま扱います。Gitの改行変換やエディタで変更すると、正規形でない名簿として拒否されます。

## 拒否と警告

次の場合、`fleet accept` は何も変更せず終了コード1で終了します。

- 固定したルートと異なるルート、署名の不一致、変更された `roster.json`
- 受け入れ済みより低い版、同じ版で内容が異なる名簿、別のフリートの名簿
- 3ファイル以外のファイルがあるディレクトリ、正規形でない名簿

同じ名簿を再び受け入れても何も変更しません。名簿が `--max-age`（既定30日）より古い場合や作成時刻が5分を超えて未来の場合は、標準エラーに警告して処理を続けます。管理ノードで `fleet sign` を実行すると作成時刻が更新されます。

正確な規則・JSON・状態の形式は[アプリケーション契約](../application-contract.md#フリート同期フェーズ1未リリース)を参照してください。
