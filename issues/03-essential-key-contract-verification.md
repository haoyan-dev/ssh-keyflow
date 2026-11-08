Status: ready-for-human
Blocked by: none

# 03 契約とドメイン文書の更新・最終検証

Spec: [.scratch/essential-key-utility/spec.md](../.scratch/essential-key-utility/spec.md)

README、application-contract、用語集、設計判断を新しい範囲に合わせる。
StandardsとSpecのレビューを別々に行い、指摘を解決する。
必要なGo/Python/配布チェックと対象コミット・条件をverificationへ記録する。

## Comments

- 2026-11-08: チケット作成。公開・マージ・リリースは今回の作業範囲外。
- 2026-11-08: 実装・検証完了。README・契約・用語集・ADRを更新し、未使用依存を1件除去した。全Goテスト/vet、Portable 5件、インストーラー50件、プロファイル30件が成功した。
- ソース `22275867764d74f5dc60c4fc8d48a70d141e856b` から6対象を実クロスビルドし、7配布物・全ハッシュ・メンバー・Unix権限・版とビルド情報・ライセンス対応を確認した。Windows amd64配布版の起動・ネイティブ鍵作成・TUIを確認。独立したStandards / Astra Specレビューの指摘は解消した。
- mainへの統合と公開は未実施。残る実機検証範囲と実行条件は [検証記録](../docs/verification.md#必須機能への縮小dev2026-11-08) を参照。

- 2026-11-08: main統合・版タグ作成のユーザー承認を受領。v0.5.0候補 `6a09684180e1731aae9d75a834759283f9228ee1` の全ローカル検証と3 OS hosted CIが成功。macOSのテスト用ホーム正規化だけを追加し、実行コードは不変。候補配布物と移行案内を再確認。タグpushによる公開は具体的な版・SHAへの承認待ち。
- 2026-11-08: 上記版・SHAの公開承認を受領し、main統合済みソースの注釈付き `v0.5.0` をpushした。Release workflow 42484308830が成功し、正式版・latestとして公開。公開済み7配布物を取得して全SHA256/API digest・アーカイブ内容・ビルド情報・ライセンス・Windows amd64の版と読み取り非変更性を確認した。main配信インストーラーのソース一致、PowerShell版の実配布物による新規導入・同版更新・削除・既存SSHデータとプロファイル保持も成功。詳細と実機未検証範囲は [公開後の検証記録](../docs/verification.md#v050公開後の検証2026-11-08-jst) を参照。
