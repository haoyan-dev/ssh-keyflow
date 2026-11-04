# 最終レビュー

2026-11-04。`code-review` に従い、Standards と Spec を別の読み取り専用エージェントで並行確認した。

- 基点：`a8f67ba9f04123297056fd915ac5fb3c25a3aebe`（main の確定仕様）。
- レビュー対象：`cda1cb8c02976b5c5b980e2750e35b00170f7e64`（全5チケット統合済み本体）。
- その後の `f17cca0` と `4d034ca` は README・検証記録・チケット状態の更新のみ。
- 全指摘の修正は単一実装エージェントが `ticket/review-fixes` で行う。

## Standards

| 重大度 | 指摘 | 根拠 | 修正後確認 |
| --- | --- | --- | --- |
| P2 | 公開鍵と秘密鍵のパスを正規化前の文字列で比較し、同じファイルを表す別表記を許可する | `internal/state/state.go:251`、application-contract の重複パス拒否 | 修正中 |
| P2 | app が OS 別の読取スクリプトを生成し、ssh の責務を持つ | `internal/app/registration.go:276`、`internal/app/revocation.go:188`、plan の構成 | 修正中 |
| P3・設計判断 | 発見表示と別名の事前確認に再帰 Include 探索が重複する | `internal/app/discovery.go:78`、`internal/app/alias.go:23`、Duplicated Code のヒューリスティック | 修正中 |

初回は3件。最も重大な指摘はP2。P3は規約違反の断定ではなく、表示と変更前確認の解釈ずれを減らす設計判断。

## Spec

| 重大度 | 指摘 | 根拠 | 修正後確認 |
| --- | --- | --- | --- |
| P1 | 登録解除で既存 authorized_keys を直接書き直し、書込中断時に無関係な行を壊す可能性がある | `internal/ssh/revocation.go:145`、`:314`、spec の他の登録保持 | 修正中 |
| P2 | 新規接続先追加後に手動手順で止まり、自動登録・指定鍵検証・別名設定まで続けられない | `internal/app/onboarding.go:158`、requirements の登録の流れ | 修正中 |

初回は2件。最も重大な指摘はP1。スコープ逸脱の指摘なし。実SSHサーバー・GitHub公開先など、明示済みの外部検証制約は実装不備に含めていない。
