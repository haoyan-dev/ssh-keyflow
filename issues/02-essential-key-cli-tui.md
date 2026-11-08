Status: ready-for-human
Blocked by: none

# 02 CLIとTUIをKeys / Destinationsに統一

Spec: [.scratch/essential-key-utility/spec.md](../.scratch/essential-key-utility/spec.md)

自動登録/解除/削除/認証選択を操作面から削除し、公開鍵と案内の表示・コピーを中心に構成する。
CLIの終了コード、JSON、非端末での読み取り、TUIのキーボード操作・スクロール・端末引き渡しを検証する。

## Comments

- 2026-11-08: 01の公開APIを前提に並行実装する。
- 2026-11-08: 実装・検証完了。ソース `22275867764d74f5dc60c4fc8d48a70d141e856b`。Keys / Destinationsの2タブ、明示的な鍵操作・接続先操作、公開鍵・ガイドのOSC52コピー、旧履歴の読み取り表示を提供する。旧登録・解除・削除・認証選択・Tasksを削除した。
- CLIのJSON・終了コード・非端末読取、TUIの検索・スクロール・32×12・ネイティブ端末引き渡しを確認。ヘルプ経由のF5で古いコピーを復元できる問題と、フォームの `?` 入力を修正し、回帰テストを追加した。Windows実端末で配布版の起動・公開鍵表示・OSC52要求・終了を確認。端末側クリップボードの読戻しは未検証。証拠は [検証記録](../docs/verification.md#必須機能への縮小dev2026-11-08)。
