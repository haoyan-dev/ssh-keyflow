# 05: インストール・更新・アンインストール・6対象リリースビルド

Status: ready-for-human
Resolution: completed
Blocked by: None (can start immediately)

## What to build

インストール・更新・アンインストール・6対象リリースビルドを対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [x] ユーザー専用の場所へ最新正式版または指定版を取得し、OSとCPUを判定する
- [x] 更新と削除で鍵・状態・設定・リモート登録を保持する
- [x] PATH変更は重複せず明示確認し、削除はインストーラー所有の変更だけに限定する
- [x] テスト用配布物でインストール・更新・削除をTDDで検証する
- [x] Windows/Linux/macOSのx64/ARM64をビルドでき、READMEとCI・リリースワークフローがある
- [x] 公開先未設定を明示し、実機未検証をクロスビルド成功と区別する

## Resolution

completed (2026-11-04): インストーラー、配布境界のテスト、6対象梱包、README、CI・リリース定義を実装し、以下のローカル検証を完了した。公開先設定・実際の公開・GitHub Actions の実行と、Linux/macOS/ARM64 の実機検証は今回の完了証拠に含めない。

- 開始時の共通ベースは `9ed731f`。統合先 `implement/ssh-keyflow` の foundation `3fba832` をこのチケットの実ブランチへ fast-forward merge し、最終コミット前に docs 更新の統合 tip `9505e33` も merge した。本体の Go ファイルは編集していない。
- インストーラー境界で、未実装のインストール、更新・削除、PATH確認、チェックサム異常、記録保存失敗時のロールバック、改変 stanza の保護などを failing test → 最小実装 → passing test で進めた。
- `python scripts/test_installers.py -v`: **24 tests / 68.079s / OK**。Git Bash は `C:/Program Files/Git/bin/bash.exe`、Windows は PowerShell 7 と Windows PowerShell 5.1 を実行した。6対象選択は fixture の OS/CPU 入力で検証し、ARM64 実機実行とは区別した。
- fixture はループバック HTTP の latest メタデータ、アーカイブ、SHA256 manifest を使用する。`.tmp/installer-tests/` に一時 home・配置先・プロファイルを作成して終了時に削除し、各テスト後に鍵・状態・既存と管理用の SSH 設定が byte 単位で変わらないことを確認した。実ユーザーの SSH データ・プロファイル・永続 PATH と本物の GitHub releases は使用していない。
- `python scripts/package_release.py --version v0.0.0-ticket05 --output .tmp/distribution-release-v0.0.0-ticket05`: **6対象の実 Go クロスビルド・梱包成功**。`-buildvcs=false -trimpath` と `-ldflags "-s -w -X main.version=v0.0.0-ticket05"` を使用。初回は管理 worktree の VCS stamp 取得で失敗したため、タグ注入に一本化して Git の信頼設定に依存しない配布ビルドへ修正した。
- 生成した9アセット、manifest の8エントリーの SHA256、各アーカイブの単一メンバーと Unix 実行権限を検査して成功。実 Windows amd64 アーカイブをローカル fixture server からインストールし、`--version` が **`SSH KeyFlow v0.0.0-ticket05`** を出力すること、更新・削除と SSH データ保持も確認した。
- `go test ./...`: **PASS** (`cmd/ssh-keyflow`, `internal/app`; `internal/state` は直接の test files なし)。シェル構文、PowerShell parser、Python AST・一文字変数/コードコメント検査、2 workflow の YAML parse も成功した。
- 配布契約は `ssh-keyflow_<tag>_<linux|darwin|windows>_<amd64|arm64>.tar.gz`（Windows は `.zip`）、中身は `ssh-keyflow`（Windows は `.exe`）、加えて `install.sh`、`install.ps1`、`SHA256SUMS`。manifest は6アーカイブと2インストーラーを含む。公開先は引数/環境変数で明示必須で、workflow では `GITHUB_REPOSITORY` を使う。

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。

2026-11-04: 配布スクリプトとローカル配布物のテストは本体実装から独立しているため、01への依存を解除して並行実装する。本体のクロスビルド確認は統合後に行う。

2026-11-04: 統合後の6対象実ビルドと Windows amd64 配布物のローカル動作確認を完了。生成 fixture を `scripts/` から既存の ignore 対象 `.tmp/` へ移した。Python cache も ignore し、既存 ignore 規則を保持した。公開先は引き続き未設定で、リリースの公開は行っていない。

2026-11-13: v0.1.0（`ee3a00f`）で公開済み。インストーラーはv0.2.0で廃止され、v0.2.1でソース配置の1行インストーラーとして再導入された。6対象リリースビルドは現行版に残る。
