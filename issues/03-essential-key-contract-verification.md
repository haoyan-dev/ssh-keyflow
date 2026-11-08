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
