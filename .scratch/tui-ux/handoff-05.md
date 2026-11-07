# Ticket05 Tasks/results handoff

Status: ready-for-human
Date: 2026-11-07
Branch: feat/05-tasks-results-recovery

製品ソース・検証 SHA: `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`。clean 開始点は `36e61375a651846654644b41c564fd9a54aa9543`。ローカル統合 `15fbfe9ca70fddabf3227d7c194deca1442b216f` を製品コミット前に取り込み済み。凍結後の最新統合 `4b67385a9098ddebde9b4af335a1c5db13c4d656` は native05 証拠だけの更新で、05完了時にマージする。以後の05変更は本書とチケットの完了記録だけ。最終 SHA は完了報告に記す。remote 更新・push・main merge は行っていない。

## 純粋投影と再開契約

`internal/tui/tasks.go` の公開境界 `ProjectTasks(app.View) []Task` は I/O・変更・依存プローブを行わない。`Task` は ID/Title、HostID/KeyID/RegistrationID、SourceIDs、Reasons、Priority、ResumeID を持つ。新しい保存 schema や workflow ID は追加しない。

- ペア内の onboarding/creation/current registration を集約。ID は辞書順先頭の保存 onboarding ID、なければ registration ID。SourceIDs はソート・重複除去した保存 ID／設定パス。継続先が変わっても同じグループの ID を保つ。優先順は requires-review=0、unknown/interrupted=1、verification=3、同順位は ID。保存状態だけでは manual と unknown を区別できないため neutral Outcome unknown を使い、manual 専用順位は捏造しない。
- 作成未完了なら、その CreationID を実際に参照する onboarding の ResumeID を選ぶ。先頭 onboarding の ID を機械的に再開しない。複数の未完了参照、異なる CreationID、欠落／矛盾リンクは requires-review・再開なし。作成完了だけでは onboarding を完了扱いしない。作成完了後に current registration があればその登録 ID、なければ onboarding を使う。
- current unregister が古い onboarding を上書きする。未完了／unknown／failed は保存 registration ID で Continue unregister。removed/complete なら remote follow-up なし。private-key の失敗／unknown／unverified は Continue registration で照合・検証も含む。Verify-only は追加しない。public-only 完了登録は verification unavailable とし、検証だけの永続 Task を作らない。
- standalone creation は保存 creation ID のローカル継続のみ。pending deletion は保存 deletion ID ごとに独立し、一般 Review があっても専用の元のガード付きローカル継続へ到達できる。認証フォームを出さず、path/digest/ownership/登録解除/config 等の backend 検査は維持する。完了削除・tombstone・履歴は実行 Task にしない。
- config pending digest／未完了 Include は requires-review。保存 binding が一つで親ペアのリンクが安全、かつ保存 Configs が単一の通常 `.ssh/ssh-keyflow/config` を指す場合だけ、そのファイルと対応する `.ssh/config` の理由・パスを親へ集約する。既存の登録／解除 ResumeID は保持。完了 verified/public-only ペアにも明示的 pending 証拠はレビュー Task として残すが、新たな resume は作らない。共有 binding、複数／不明パス、欠落 binding は read-only `config:<path>` 行に残す。一般 Review は `state.json` 行にも保持する。
- `resumeChoices` は同じ投影へ委譲。Tasks の s は表示 Task の正確な ResumeID を開く。結果の複数候補は picker で明示選択する。読み取り専用 config の汎用 resume、Mark done、dismiss、remote 自動再試行はない。

## 結果とアクション契約

`results.go` は01の型付き reports と独立した Snapshot から表示する。04の request/explicit target/reports/error/diagnostics/reloadErr をすべて保持し、native 実行・signal・確認コードは変更していない。エラー／stdout 文字列を結果分類に使わず、nil error や saved pending checkpoint を完了根拠にしない。

| 証拠 | 主表示と継続 |
| --- | --- |
| manual/required、nil error | Manual action required。今回の not-checked と以前の登録／verified 証拠を分離。新規保存 intent がなければ No saved recovery task、開いている間は指示を保持し、終了／再起動後は Register again。manual unregister は登録解除の保存継続で、Register again は出さない。 |
| 登録保存＋verification failed | Registered; connection check failed。元のペアの Continue registration。 |
| alias approval denied | Stopped after registration; managed alias not generated this time。以前の binding／設定ファイル／Include 証拠は別に保持。 |
| 登録＋alias 保存後の Include denial/failure | 両保存効果と残りの Include setup を表示。denial は今回 main config unchanged。failure は実際の typed checkpoint／エラー／診断を保持。intent 未保存なら Task を捏造せず、config requested の Register/Add が照合・登録を繰り返すことと config-only がないことを説明。 |
| authoritative NoChanges denial | Cancelled before changes。unknown がある場合はこの断定をしない。 |
| エラー／unknown | Operation stopped と不確実性、残る保存証拠、照合が必要な継続。 |
| 適用可能な要求すべてに終端の型付き証拠 | Completed。Configure 要求は alias 保存＋Include applied/reconciled が必要。public-only は verification unavailable と alias/Include の両 skipped が明示されたときだけ例外。単なる skipped／Snapshot／空 Review は成功根拠にしない。 |
| reload failure | 全 envelope と以前の view を保持し stale。成功した Snapshot-only Refresh まで mutation 禁止。Refresh は実行を再送しない。 |

Tab/Shift+Tab と Enter で primary action／View task／View entity／Refresh／Exit／Help を操作。s/t/v/r は同じ対象固定の導線。result の entity は明示 registration、key、host の順に実在を確認し、裏のリスト行は参照しない。Task 関連は保存 source ID、同じ非空ペア、report の正確な config パスで照合。診断・手動指示・ファイル・full fingerprint は縦スクロールでき、Help は元の位置へ戻る。stale は Refresh/Help/Exit のみ許可する。

## 公開境界の検証

承認済み seam は `tui.ProjectTasks`、`tui.NewProgram` の公開イベント／rendered View、実 `app.Run`／Snapshot。使い捨て fixture と外部 Runner のみを fake にし、内部 model accessor・実サーバー・秘密情報は使わない。

- `projection_test.go` と `tasks_test.go`: AC09 各行の優先順／literal ID／再開先、入力非変更と順序反転の決定性、複数 onboarding 参照、実際の未完了作成所有者、矛盾リンク、public/private、current unregister、完了／履歴、独立 deletion、config 同パス／異パス／安全な親／共有 binding／public-only pending を検証。
- `tasks_test.go` と `results_test.go`: manual 未記録・以前 verified の即時結果と再起動、remote uncertainty、verified/public-only/verification failure/alias denial/Include denial/Include conflict/configured の実 app 分岐、以前の binding と Include を持つ alias denial、結果から正確な entity／resume、manual unregister 再開、一般整合性警告下の pending deletion を実行境界まで検証。ファイル変更の有無と保存証拠は Snapshot 等で確認する。
- `handoff_test.go`: 保存 creation interruption、元の native input、NoChanges denial、reports/error/diagnostics を保持する stale reload と成功 Refresh、Runner／report の再送なし。`tasks_test.go`／`results_test.go` で32x12/60x18/80x24 の結果／Tasks controls、Help／scroll、31x11 guard 後の対象保持も確認。
- 代表 red/green: unknown remote が Manual の一語に隠れる失敗(t42)→stopped/unknown 表示(t43)、複数 creation owner／曖昧参照(t46→t47)、config 親の重複行(t4c→t4d)。config 集約後の TUI 全体(t4f) PASS7.739s。初回 pending deletion journey は検索が理由にも一致する review 行を選んだため、正確な deletion 行の選択を fixture に明示して実経路を確認した。

## 凍結 SHA の確認結果

Windows amd64、Go1.26.2、Herdr0.1.18、2026-11-07。すべて製品 `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`、文書変更前の clean tree から実行。

| 確認 | 結果 |
| --- | --- |
| `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'` (t4g) | PASS exit0。cmd1.917s、app41.098s、cli2.438s、ssh22.631s、tui8.262s、state は test file なし。実行プロセスだけの policy で永続変更なし。 |
| `go vet -buildvcs=false ./...` (t4h) | PASS exit0。 |
| `go build -buildvcs=false -o .tmp/ux05-checks/ssh-keyflow.exe ./cmd/ssh-keyflow` と `--version` (t4j) | PASS exit0、SSH KeyFlow dev。 |
| `git diff --check 15fbfe9ca70fddabf3227d7c194deca1442b216f HEAD`、統合 ancestor | PASS。 |

検証ログを確認後、所有・終了状態を確認した `ux05-*` terminal 27件と `.tmp/ux05-checks/ssh-keyflow.exe` および空になった専用ディレクトリを削除済み。coordinator の t4k／native terminals／fixtures は保持した。文書のみの完了差分に `git diff --cached --check` とリンク・チケット形式の検査を適用し、製品テストの再実行は不要と判断した。

## 規約自己点検

仕様点検と独立に AGENTS／RTK／domain／tracker／triage／TDD／ponytail を照合。既存 list/viewport/forms と backend validation・保存操作を再利用。app production、state schema、依存、CLI、native signal／確認処理に変更なし。指定の05ファイルと05記録だけを変更。追加した型は要求された純粋投影境界だけ。standards の未解決指摘なし。

## 仕様自己点検と引き継ぎ範囲

AC08/09、関連 AC02/07/10/11 を別に照合し、05の全 acceptance criteria に上記の自動証拠を対応付けた。binding→config 親集約の指摘も実装・テスト済みで、既知の05仕様 gap は残していない。native の合格を自動テストから推定しない。

Coordinator が同じ凍結 SHA で Windows native results/Tasks smoke PASS を確認し、統合 `4b67385a9098ddebde9b4af335a1c5db13c4d656` の [native-verification.md](native-verification.md#ticket05-native-results-restart-and-saved-task-continuation) に記録した。80x24 の manual Register 未記録／以前 verified・指示スクロール・再起動後の Tasks 空／保存意味不変、32x12 の unknown registration 正確な ID で Review／Help／取消、60x18 の本物 ssh-keygen 中断由来 pending creation のローカル専用 Review／取消が対象。後二者の state hash 不変も確認済み。coordinator の native terminals は本人がすべて削除済みで、05側は操作していない。既知 PasteMsg defect は指示通り05に混ぜず、06後の単一 fresh fixer に残す。native Linux/macOS・実 SSH／サーバー効果は未検証。最終 cross-journey、出荷 docs、repository/package/Python の統合確認は06の責務。
