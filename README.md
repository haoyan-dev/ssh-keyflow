# SSH KeyFlow

[简体中文](README.zh-CN.md) · [English](README.en.md) · 日本語

この端末のSSH鍵ペアを作成・取り込み・管理し、公開鍵をコピーできる形で表示するGo製のCLIです。

[はじめて使う](docs/tutorials/first-connection.md) · [ドキュメント](docs/README.md) · [コントリビューション](CONTRIBUTING.md) · [リリース](https://github.com/haoyan-dev/ssh-keyflow/releases)

## できること

- Ed25519鍵の作成、既存の鍵ペアの取り込み、中断した鍵作成の再開。
- 鍵の一覧（表形式・JSON）と、コピー用の公開鍵1行の表示。
- 鍵ファイルを残したままの管理からの削除。

アプリはSSH接続、リモート登録、SSH configの編集を行いません。公開鍵の配置は利用者が行います。管理からの削除は鍵ファイルの削除・SSHアクセスの失効とは別の操作です。v0.11.0で接続先・手動設定ガイド・ホスト鍵の固定・TUI・公開鍵のみの取り込みを廃止しました（[移行](docs/guides/data-and-recovery.md#v0100以前からの移行)）。

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
ssh-keyflow keys list
```

変更操作の確認には対話端末を使います。最初の鍵を用意して接続する手順は[はじめての接続設定](docs/tutorials/first-connection.md)、コマンド例は[CLIの使い方](docs/guides/cli.md)にあります。

## ドキュメント

| 目的 | 読むページ |
| --- | --- |
| 最初の鍵を用意して接続する | [チュートリアル](docs/tutorials/first-connection.md) |
| 手順・困ったときの対処を探す | [ドキュメント一覧](docs/README.md) |
| CLI・JSON・保存状態の約束を確認する | [アプリケーション契約](docs/application-contract.md) |
| 修正や文書の改善に参加する | [コントリビューション](CONTRIBUTING.md) |

## 安全条件とデータ

管理情報は `~/.ssh/ssh-keyflow/state.json`、生成鍵は同ディレクトリの `keys/` に保存します。v0.10.0以前の状態は最初の変更時に `state.v<N>-<UTC時刻>.json` へ書き出してから移行します。旧版からの移行・ダウングレード前には[データ・移行・復旧](docs/guides/data-and-recovery.md)を確認してください。`state.json.bak` は直前状態の更新式バックアップで、旧版へ戻すための恒久保管ではありません。

## Portable配布の取得と実行

[リリース](https://github.com/haoyan-dev/ssh-keyflow/releases)のOS / CPU別アーカイブと `SHA256SUMS` を使えます。[検証と展開の手順](docs/guides/installation.md#portable配布の取得と実行)を参照してください。

## Portableの手動更新・PATH・本体削除

[手動での更新・PATH設定・削除](docs/guides/installation.md#portableの手動更新path本体削除)を参照してください。実行ファイルを移動しても保存先はホーム配下のままです。

## v0.1.0のインストーラー利用者

[旧インストーラーからの移行](docs/guides/installation.md#v010のインストーラー利用者)を参照してください。所有記録・PATH・手動差し替えの扱いを説明しています。

## 開発とリリース契約

開発環境とチェックは[コントリビューション](CONTRIBUTING.md)、配布と公開は[リリースガイド](docs/maintaining/releases.md)を参照してください。公開の承認範囲と完了条件は[AGENTS.md](AGENTS.md#tags-and-releases)に従います。

依存ライブラリのライセンスと著作権表示は[THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)にあります。
