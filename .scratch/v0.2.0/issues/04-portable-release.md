# 04: インストーラー廃止とPortable配布への統一

Status: ready-for-agent
Resolution: completed
Blocked by: 02, 03

## What to build

[spec.md](../spec.md) に従い、現行の梱包処理を修正して6対象のPortableアーカイブを配布する。インストーラー、関連テスト・案内を整理し、CI・リリース契約を更新する。

## Acceptance criteria

- [x] 6対象アーカイブとSHA256SUMSを生成し、インストーラーを添付しない。必要な依存ライセンス文書を同梱する。
- [x] manifestは6アーカイブを対象とし、アーカイブの内容・実行権限・チェックサム・バージョン注入をテストする。
- [x] 展開実行、アプリ終了後の差し替え、手動PATH設定、本体削除をREADMEに記載する。
- [x] 本体の移動・差し替え・削除で既存の鍵・状態・設定・リモート登録を変更しない。
- [x] 旧インストーラー利用者向けに、既存PATH・所有記録を自動変更しないことと、手動差し替え後は旧スクリプトによる更新・削除が拒否され得ることを説明する。
- [x] installer専用テストを整理し、Portableの梱包テスト、Goテスト、6対象クロスビルドへCIを更新する。
- [x] README、application-contract、verificationとリリース定義を最終実装へ揃え、実機未検証の対象を明記する。
- [x] このチケットの完了を理由にタグ作成・push・公開を行わない。

## Resolution

completed (2026-11-06): 既存梱包処理を6対象のPortableアーカイブとSHA256SUMSへ統一した。各アーカイブに実行ファイルとTHIRD_PARTY_NOTICES.txtだけを同梱し、旧インストーラー2本と専用HTTP/配置テストを削除。CI・リリース定義・README・公開契約・検証記録を現行の7アセット契約へ更新した。

- TDD: 生成物境界の最初のテストでインストーラー2本が残る失敗を確認し、梱包処理を最小変更して成功した。
- `python scripts/test_portable.py -v`: 5件成功。6アーカイブ・同梱物・権限・manifest、欠損時の未公開、既存出力と元バイナリ保持、版注入したWindows実バイナリの展開・移動・差し替え・削除と一時ホームの保持を検証。
- `go test -buildvcs=false ./...`: 全件成功（app 65.426秒、ssh 25.301秒、cli 1.909秒、tui 0.879秒）。`go vet -buildvcs=false ./...` 成功。
- `python scripts/package_release.py --version v0.2.0 --output .tmp/release-ticket04-final`: CGO無効の6対象実クロスビルド成功。実アーカイブ全6件の許可メンバー、Unix0755、同梱文書bytes、SHA256全6件を検算し、Windows amd64の展開後 `--version` は `SSH KeyFlow v0.2.0`。
- `go list -buildvcs=false -deps` で全6対象の実行用依存（各22、和集合23モジュール）と版を照合。Goと依存のLICENSE/PATENTSに加え、生成Unicodeテーブルの参照条件も収録。出典と元bytesのハッシュはdocs/third-party-notices.sources.json。
- 旧インストーラーのアーカイブ形式拒否と差し替え後の所有ハッシュ不一致を案内し、PATH・プロファイル・旧記録の自動編集を追加していない。
- Windows amd64以外のネイティブ実行・端末、実SSHサーバー、昇格したWindows管理者共有対象は未確認。CI定義を更新したがホストCIは未実行。実ユーザーのSSHデータを変更せず、タグ・push・公開なし。

## Comments

2026-11-06: v0.2.0方針の合意に基づき作成。独立した配布基盤への置き換えは行わない。

2026-11-06: to-specに従って仕様を再整理。生成アーカイブを境界とし、クロスビルドと実機確認を区別する検証方針をユーザー確認済み。
