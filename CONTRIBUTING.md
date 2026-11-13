# コントリビューション

[简体中文](CONTRIBUTING.zh-CN.md) · [English](CONTRIBUTING.en.md) · 日本語

[README](README.md) · [ドキュメント一覧](docs/README.md) · [作業ポリシー](AGENTS.md)

不具合修正、機能提案、チュートリアルや文書の改善を受け付けます。1つの変更で解決する課題と、確認できる完了条件を明確にしてください。

## 報告・提案

[GitHub Issues](https://github.com/haoyan-dev/ssh-keyflow/issues)で既存の報告を確認し、版、OS / CPU、端末、再現手順、期待結果と実際の結果を記録してください。機能提案では、解決したい作業と既存機能で不足する点を説明します。秘密鍵・パスフレーズ・認証トークンを添付せず、ログ・パス・接続先情報も公開前に確認してください。

実装の仕様とチケットは[ローカルMarkdown](docs/agents/issue-tracker.md)で管理します。`.scratch/<feature>/spec.md` に目的・範囲・完了条件・判断を、`issues/01-<task>.md` からの連番で作業を記録します。各チケットの `Status:` と `Blocked by:` は[状態ラベル](docs/agents/triage-labels.md)と依存関係に合わせて更新します。GitHub上の報告やPRとは相互に参照してください。

## 開発環境

- Goは [go.mod](go.mod) の指定に従います。
- 検証スクリプトはPython 3.12の標準ライブラリを使います。
- ネイティブ鍵操作の検証には `ssh` と `ssh-keygen` を用意します。
- TUIの手動確認は対話端末で行います。

```sh
git clone https://github.com/haoyan-dev/ssh-keyflow.git
cd ssh-keyflow
go run ./cmd/ssh-keyflow --help
```

開発ビルドの版表示は `dev` です。TUIや変更操作を試す場合は、実際に使う鍵と混ざらない専用ユーザー環境を用意してください。自動テストは一時ホームを使います。

## 変更の進め方

1. [AGENTS.md](AGENTS.md)を読み、作業ツリーとブランチを確認します。最新の `origin/main` から `<type>/<short-kebab-case-description>` の作業ブランチを作ります。外部コントリビューターはfork上のブランチからPRを送れます。
2. 調査し、仕様とチケットを作ります。設計やコード調査の前には[用語集](GLOSSARY.md)と[関連ADR](docs/adr/README.md)を確認します。
3. 変更を実装し、必要な文書を同じPRで更新します。文書の配置と所有する情報は[文書の保守](docs/maintaining/documentation.md)を参照してください。
4. 影響するチェックを実行し、規約への適合と仕様への適合を別々にレビューします。指摘の解消を確認します。
5. PRに課題、変更内容、検証した完全なコミットSHA、実行環境、結果、未検証範囲を記録します。必要なCI・レビューを通してから統合します。

## 検証

文書のリンク・見出し参照・言語リンク・翻訳の充足:

```sh
python scripts/test_docs.py -v
python scripts/check_docs.py
```

実装・配布に関わる変更の確認:

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
```

文書だけの変更ではリンクチェックと手順・コマンドの確認を行います。README・ガイドなど翻訳対象のページを変更したら、同じPRで[英語版と中国語版](docs/maintaining/documentation.md#言語別の配置)も更新するか、更新待ちとして明示してください。実装・依存・梱包・インストーラーを変更した場合は対応するテストを実行し、梱包入力が変わる場合は[リリースガイド](docs/maintaining/releases.md)の6対象ビルドも適用してください。CIの必須チェックは変更種別にかかわらず通す必要があります。

継続して参照する検証結果は [docs/verification.md](docs/verification.md)に対象SHA・環境・結果・限界を記録します。クロスビルド、実機実行、実際のSSH接続確認はそれぞれ区別してください。

## レビューと公開

PRは1つの目的に絞り、利用者に見える変更と互換性・移行への影響を説明してください。文書だけのPRもレビュー後に `main` へ統合します。

アプリの公開は[リリースガイド](docs/maintaining/releases.md)から、準備PR、最終候補の検証、タグと自動公開、配布物監査と記録PRの順で進めます。タグのpushは公開を開始するため、[AGENTS.mdの承認範囲](AGENTS.md#tags-and-releases)を確認してください。
