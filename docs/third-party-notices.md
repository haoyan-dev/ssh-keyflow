# 同梱ライセンスの照合

2026-11-06、Go 1.26.2、v0.2.0の本体に対して確認した。`CGO_ENABLED=0` と6通りの `GOOS` / `GOARCH` を指定し、`go list -buildvcs=false -deps` で実行ファイルに含まれるモジュールを取得した。各対象22モジュール、和集合23モジュールについて、固定バージョンと `THIRD_PARTY_NOTICES.txt` の収録内容を照合し、不足はなかった。テスト・ツール専用の全module graphは配布対象の根拠にしていない。

Goランタイム・標準ライブラリと、直接・間接の依存モジュールのLICENSE類を転載した。Cobraとunisegの `LICENSE.txt`、Go・x/sys・x/syncの `PATENTS`、Windows専用のmousetrapとx/windowsも含む。プロジェクト自体に新しいライセンスを設定するものではない。

使用パッケージのソース先頭と生成テーブルの注記も確認した。unisegのUnicode 15.0由来テーブルはUnicodeのライセンスを参照しており、GoとUnicode分割・文字幅依存が使用するデータのため、Unicode公式の `https://www.unicode.org/license.txt` から取得した全文も収録した。モジュール単位のライセンス名だけで済ませず、著作権・許諾本文を含めた。

[third-party-notices.sources.json](third-party-notices.sources.json) は元ファイル名、モジュール・版、取得元バイト列のSHA256を記録する。モジュール文書はGoキャッシュの該当バージョンから取得し、本文の改行と文末空白を整えて連結した。Unicode文書は上記公式URLから2026-11-06に取得した。ハッシュは整形前の元ファイルに対するもの。開発用スキャナーや実行時の追加依存は導入していない。

依存やGoツールチェーンを更新するときは、6対象の実行用パッケージ、関連する著作権・追加ライセンス注記と同梱本文を再照合する。
