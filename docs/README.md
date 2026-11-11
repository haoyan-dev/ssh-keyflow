# SSH KeyFlow ドキュメント

[プロジェクト概要](../README.md) · [コントリビューション](../CONTRIBUTING.md)

このディレクトリは `main` の現行実装に対応する文書です。公開済みバージョンの動作を調べる場合は、[対象タグ](https://github.com/haoyan-dev/ssh-keyflow/tags)の同じ文書と[リリースノート](https://github.com/haoyan-dev/ssh-keyflow/releases)を参照してください。古い版へ現行仕様を遡って適用しません。

## よく使う手順

| やりたいこと | 読むページ |
| --- | --- |
| 最初の鍵・接続先を用意し、接続設定を進める | [はじめての接続設定](tutorials/first-connection.md) |
| インストール・更新・本体の削除を行う | [インストールガイド](guides/installation.md) |
| TUIの操作キーやコピー先を確認する | [TUIの操作](guides/terminal-ui.md) |
| CLIで鍵・接続先・関連付けを管理する | [CLIの使い方](guides/cli.md) |
| 状態の保管、移行、旧版へ戻す際の注意を確認する | [データ・移行・復旧](guides/data-and-recovery.md) |
| 起動・コピー・鍵・接続の問題を調べる | [トラブルシューティング](guides/troubleshooting.md) |

## 仕様と背景

| 文書 | 役割 |
| --- | --- |
| [アプリケーション契約](application-contract.md) | CLI、JSON、終了コード、状態互換性、インストール・配布の公開契約 |
| [用語集](../GLOSSARY.md) | 鍵、接続先、関連付け、未確認などのドメイン用語 |
| [設計判断の一覧](adr/README.md) | 採用した判断と、それを更新・置換する判断 |
| [検証記録](verification.md) | 対象コミット・実行条件・結果・限界を含む時点ごとの証拠 |
| [第三者ライセンスの収集方法](third-party-notices.md) | 配布物へ含めるライセンス文書と収集根拠 |

## 開発と保守

- [コントリビューション](../CONTRIBUTING.md)：環境構築、変更の進め方、チェック、レビュー。
- [文書の保守](maintaining/documentation.md)：文書種別、所有する情報、更新時の確認。
- [リリースと公開](maintaining/releases.md)：配布物、検証コマンド、公開手順の入口。
- [ローカルチケット](agents/issue-tracker.md)と[状態ラベル](agents/triage-labels.md)：仕様・作業の記録。
- [ドメイン文書の運用](agents/domain.md)：用語と設計判断の参照先。
- [作業・ブランチ・公開ポリシー](../AGENTS.md)：開発時の規則と承認範囲。

## 過去の調査・レビュー

次の文書は記録時点の調査と検証です。現行仕様はアプリケーション契約を参照してください。

- [SSH実装調査](ssh-implementation-notes.md)
- [初期実装の最終レビュー](final-review.md)
- [v0.2.0レビュー](v0.2.0-review.md)
- [バージョン運用ポリシー監査](versioning-policy-audit.md)

`.scratch/` は機能ごとの研究・仕様・チケット・作業記録です。古い仕様は後続のADRや現行契約で置き換えられている場合があります。
