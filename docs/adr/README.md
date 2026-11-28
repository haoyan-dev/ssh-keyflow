# 設計判断

[ドキュメント一覧](../README.md) · [用語集](../../GLOSSARY.md)

ADRは判断時点の理由と影響を保存します。後続ADRが過去の判断を更新する場合があります。現在の外部仕様は[アプリケーション契約](../application-contract.md)で確認してください。

| ADR | 判断 |
| --- | --- |
| [0001](0001-essential-key-utility.md) | ローカル鍵管理とコピペ支援への集約 |
| [0002](0002-platform-terminal-snippets.md) | 端末で実行できるステップ別スクリプト |
| [0003](0003-explicit-guide-platforms.md) | ガイドのローカル・接続先OSの明示 |
| [0004](0004-remove-managed-records.md) | 履歴を保持した管理メタデータ削除 |
| [0005](0005-ssh-agent-autostart-step.md) | パスフレーズ付き鍵のssh-agent設定支援 |
| [0006](0006-host-key-pinning.md) | 接続先ホスト公開鍵のアプリ所有known_hostsへの固定 |
| [0007](0007-key-pairs-only.md) | 鍵ペアの管理だけを残し、接続先・ガイド・TUIを廃止（0002・0003・0005・0006を廃止、0001・0004を更新） |
| [0008](0008-signed-fleet-roster.md) | 署名付きフリート名簿で端末間の公開鍵を揃える（0001・0004・0007を更新、未実装） |
