# SSH KeyFlow

ローカルのSSH鍵と接続先を管理し、公開鍵の配置と接続設定をコピペで支援するGo製のTUI / CLIです。

[はじめて使う](docs/tutorials/first-connection.md) · [ドキュメント](docs/README.md) · [コントリビューション](CONTRIBUTING.md) · [リリース](https://github.com/haoyan-dev/ssh-keyflow/releases)

## できること

- Ed25519鍵の作成、既存の鍵ペア・公開鍵の取り込み、中断した鍵作成の再開。
- Keys / DestinationsのTUIとCLIによる鍵・接続先・関連付けの管理。
- Linux / macOS / Windows向けの公開鍵配置、SSH config追記、接続手順の表示・コピー。
- 接続先で表示したホスト公開鍵の貼り付けによる固定。初回接続の確認を省き、他のホスト鍵を拒否します。
- FIDO鍵等、SSHエージェントにある秘密鍵を公開鍵ファイルで選んで使う設定。

アプリはSSH接続やリモート登録を実行しません。関連付けはローカルの管理情報で、接続成功を意味しません。生成したスクリプトを利用者が対象端末で実行して設定します。管理からの削除は鍵ファイルの削除・SSHアクセスの失効とは別の操作です。

## 1行でインストール

Windows / Linux / macOSのamd64（x64）/ arm64に対応します。Go・管理者権限は不要で、OSの `ssh` と `ssh-keygen` が必要です。

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

Windows（PowerShell）:

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

スクリプトは `main` から取得し、最新の正式リリースとチェックサムを検証して配置します。先に内容を確認したい場合は保存して読んでから実行してください。PATH追加は確認に承認した場合だけ行います。詳細は[インストール・更新・アンインストール](docs/guides/installation.md)を参照してください。

## 起動と操作

新しい端末で次を実行します。PATHを追加しなかった場合は、インストーラーが表示したフルパスを使ってください。

```sh
ssh-keyflow --version
ssh-keyflow
```

TUIは対話端末で使います。最初の鍵と接続先を用意する手順は[はじめての接続設定](docs/tutorials/first-connection.md)、操作キーは[TUIの操作](docs/guides/terminal-ui.md)、コマンド例は[CLIの使い方](docs/guides/cli.md)にあります。

## ドキュメント

| 目的 | 読むページ |
| --- | --- |
| 最初の鍵と接続先を用意する | [チュートリアル](docs/tutorials/first-connection.md) |
| 手順・困ったときの対処を探す | [ドキュメント一覧](docs/README.md) |
| CLI・JSON・保存状態の約束を確認する | [アプリケーション契約](docs/application-contract.md) |
| 修正や文書の改善に参加する | [コントリビューション](CONTRIBUTING.md) |

## 安全条件とデータ

管理情報は `~/.ssh/ssh-keyflow/state.json`、生成鍵は同ディレクトリの `keys/`、固定したホスト公開鍵は `known_hosts` に保存します。旧版からの移行・ダウングレード前には[データ・移行・復旧](docs/guides/data-and-recovery.md)を確認してください。`state.json.bak` は直前状態の更新式バックアップで、旧版へ戻すための恒久保管ではありません。

## Portable配布の取得と実行

[リリース](https://github.com/haoyan-dev/ssh-keyflow/releases)のOS / CPU別アーカイブと `SHA256SUMS` を使えます。[検証と展開の手順](docs/guides/installation.md#portable配布の取得と実行)を参照してください。

## Portableの手動更新・PATH・本体削除

[手動での更新・PATH設定・削除](docs/guides/installation.md#portableの手動更新path本体削除)を参照してください。実行ファイルを移動しても保存先はホーム配下のままです。

## v0.1.0のインストーラー利用者

[旧インストーラーからの移行](docs/guides/installation.md#v010のインストーラー利用者)を参照してください。所有記録・PATH・手動差し替えの扱いを説明しています。

## 開発とリリース契約

開発環境とチェックは[コントリビューション](CONTRIBUTING.md)、配布と公開は[リリースガイド](docs/maintaining/releases.md)を参照してください。公開の承認範囲と完了条件は[AGENTS.md](AGENTS.md#tags-and-releases)に従います。

依存ライブラリのライセンスと著作権表示は[THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)にあります。
