# リリースと公開

[简体中文](../zh-CN/maintaining/releases.md) · [English](../en/maintaining/releases.md) · 日本語

[ドキュメント一覧](../README.md) · [コントリビューション](../../CONTRIBUTING.md)

このページは配布の概要とチェックの入口です。承認範囲と各段階の完了条件は[AGENTS.md](../../AGENTS.md#tags-and-releases)を正とし、公開前に[アプリケーション契約](../application-contract.md)と[release.yml](../../.github/workflows/release.yml)も確認してください。文書だけの変更を統合するために、アプリケーションのリリースタグを作る必要はありません。

## 開発とリリース契約

Goは `go.mod` の指定、梱包テストはPython 3.12の標準ライブラリを使います。TUIはBubble Tea / Bubbles / Lip Gloss、CLIはCobraを使い、ローカル鍵処理と状態管理を共通化します。

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
python scripts/package_release.py --version v0.0.0-check --output .tmp/dist-check
```

梱包は `CGO_ENABLED=0`、`-buildvcs=false`、`-trimpath` と `-ldflags "-s -w -X main.version=<tag>"` で6対象をビルドします。開発版は `dev`。上の `v0.0.0-check` はローカル検証用の版注入例で、公開は行いません。既存ファイルがある出力先への上書きは拒否します。既存バイナリの梱包には `--binaries DIRECTORY` を指定し、`DIRECTORY/linux_amd64/ssh-keyflow` 等の6対象から読み込みます。

| OS | CPU | アセット名 |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

アーカイブ内部は `ssh-keyflow`（Windowsは `ssh-keyflow.exe`）と `THIRD_PARTY_NOTICES.txt`。Unix実行権限は0755です。トップレベルの配布物は6アーカイブと `SHA256SUMS` の7件だけです。manifestは6アーカイブについて、64桁SHA256・半角スペース2個・ベース名・LFで記録します。

CIは3 OSでGoテスト、Portable生成物テスト、インストーラーテストを実行し、6対象の実クロスビルドとLinux amd64版の版注入を確認します。`v*` タグのpushでCI成功後に7件を `GITHUB_REPOSITORY` へ公開します。インストーラーは `main` のソースから別途配信し、7件のリリース配布物は変更しません。プレリリースタグはlatest正式版にしません。

リリースは、準備PRの統合、最終候補の検証、タグ作成・公開、公開後の配布物検証と記録PRの統合の順で進めます。準備記録には検証したソースのコミットを使い、最終タグのコミットはタグ・リリース情報と公開後の記録に残します。公開後の記録でmainがタグより先へ進むことは通常の運用で、記録だけの変更による再リリースは不要です。各段階の完了条件と承認範囲は [AGENTS.mdのTags and releases](../../AGENTS.md#tags-and-releases) を参照してください。

公開ジョブはタグをまたいで直列化し、その中で現在のlatestを取得して数値のバージョン順で比較します。旧系列の修正版は公開できますが、新しいlatestを置き換えません。latestが404の場合は全リリースの一覧取得に成功し、公開済み正式版がないと確認できた場合だけ初回公開として扱います。認証・通信・API応答の異常や、正式版があるのにlatestが取得できない場合は公開を停止します。`scripts/test_release.py` は外部公開なしでこれらの判定と公開コマンドを検証します。

テストは一時ホーム・配置先・配布物を使い、実ユーザーのSSHデータ、永続PATH、プロファイル、接続先を変更しません。クロスビルドは各OS / CPUでの実機実行の証拠とは区別します。確認済み範囲は [docs/verification.md](../verification.md)、公開インターフェースは [docs/application-contract.md](../application-contract.md) を参照してください。
