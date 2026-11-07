# Post-review fixes handoff

Status: ready-for-human
**Current handoff (AC11 follow-up):** 最新の製品・テスト / testedSHA / buildSHA は **`26269049b747db02988ad00db2b62dcff22119f0`**。integration `087964f7f1aa2a8ab0dab5f20474e9df171b63df` は取込済み。末尾の AC11 追補が現在の結果であり、以下の ba0f69d 証跡は前回提出時の履歴として残す。

## Source and ownership

2026-11-08 JST。単一の fresh post-review implementer が S1 と P1–P4 を修正した。再帰 agent は使用していない。専用 worktree は `C:\Users\haoyan-ts\repo\ssh-keyflow-ux-review`、branch は `fix/tui-ux-review`。

開始時の clean integration は `c8ce050fa7fae0108173abec736efe733c759856`。その後、integration `db2da488bfd413898f13d34d88ae614efa2f8277` を fast-forward で取り込んだ。製品・テスト・現行 runtime contract の確定コミット、全チェックと全 package の **testedSHA/buildSHA は `ba0f69d81c3345ed2bd78062521bcd63adda7c69`**。handoff は後続の証跡専用コミットに置く。

元 checkout `C:\Users\haoyan-ts\repo\ssh-keyflow` は clean な `main`、`89340aebbc0b3ab7f781b822aaafc2130ae76f73` のまま。リモート更新・push・main merge・tag・publication は行っていない。ローカル integration を基点とする明示指示に従った。

指摘原文は [final-review.md](final-review.md)、仕様は [spec.md](spec.md)、設計は [design.md](design.md)。過去の [coverage-06.md](coverage-06.md) / [handoff-06.md](handoff-06.md) は当時の記録として維持した。

## Resolution: Standards S1

`usesRemoteOptions()` を field construction / validation / request construction で共有した。Add の step 2 だけに表示する条件は `buildFields` 側に残した。既存の条件付き form・local resume journey と全 TUI suite が PASS。挙動不良を主張しない review judgement のため、独立した製品 RED は主張しない。

## Resolution: Specification P1

保存済み onboarding と新 snapshot の一致に加え、linked creation の ID / reserved KeyID / 保存内容の一致を検証する。変更のない `key_pending` onboarding が未完了 creation を指す場合に限り、まだ存在しない managed key の要求を省く。既存 key、host、通常の drift 検査、実行側 guard は維持した。

`TestReviewFixPendingOnboardingReachesReviewAndHandsOffSavedIntent` は schema-valid な `keys: []` と pending creation / onboarding を保存し、公開 `tui.NewProgram` で Tasks → details → s → Ctrl+S → Review → native handoff を通す。保存済み名前・path が original-terminal prompt に届くこと、no 応答だけで Runner 未実行・state 不変・key 未作成であることを確認する。

`TestReviewFixPendingOnboardingRejectsLinkedCreationDrift` は Review 前と handoff 前の linked creation drift を別々に拒否し、native prompt / Runner に到達しないことを確認する。

RED t5h: 有効な reserved identity が `Managed key target key-reserved changed or is missing` となり Review に進めない。GREEN t5j: PASS 1.087s。追加 drift tests を含む最終 focused t62 と固定 SHA の全 suite も PASS。

## Resolution: Specification P2

`PasteMsg` と textinput の返信を active form / list search / picker search に渡し、caret・draft・query・filter を更新する。clipboard / cursor command の返信には発行元 input の generation を付け、focus / form build / Review / search / overlay 遷移後の古い返信を破棄する。running と最小サイズ未満の guard は入力前に適用する。fixed fields / controls、Review、Help は paste で編集・実行されない。

公開 `tui.NewProgram` の regression tests:

- `TestReviewFixPasteEditsAtCaretAndRetainsDirtyDraftThroughHelp`: caret insertion、q/?/action letters の literal 入力、Help 往復、dirty draft 保持。
- `TestReviewFixPasteFiltersListAndPickerAtCaret`: list / picker の検索・caret・selection、literal query と Help 保持。
- `TestReviewFixPasteClearsHiddenChoicesAndCannotEditFixedControls`: hidden authentication identity / Windows account type の clearing、fixed target 保持、Review paste による実行なし。
- `TestReviewFixPasteBelowMinimumRetainsFormAndSearch`: 31×11 中の paste / Ctrl+S / Enter を無視し、80×24 に戻った draft / query を保持。
- `TestReviewFixPasteDoesNotActivateFormControls`: Continue / Help / Cancel 上の paste が画面を変更せず、操作を実行しない。
- 既存 `TestHandoffAcceptsContinueOnlyOnceBeforeNativeReturn` に running 中の paste を加え、Continue が一度だけであることを維持。

RED t5k: form は `ac` のまま、list / picker は対象7でなく対象1に進み、paste query も消失。GREEN t5n: 初期回帰 PASS 1.250s。その後の拡張も最終 focused / full suite で PASS。現行 `docs/application-contract.md` の既知 paste 制限はこの検証後に更新した。履歴資料は書き換えていない。

## Resolution: Specification P3

Resume Review は registration に加えて creation / deletion / onboarding を解決する。creation は new Ed25519 intent、保存済み名前・reserved ID・既知の private/public path と存在時の saved fingerprint を示す。deletion は `recordedKey` で label / full fingerprint を解決する。onboarding は host / login と linked creation または key を示す。local deletion / onboarding の expected effects も具体化した。

`TestReviewFixResumeReviewShowsEverySavedIdentity` の creation / deletion / onboarding-new / onboarding-existing が PASS。RED t5q は creation の名前、onboarding-new の host、onboarding-existing の key / full fingerprint の欠落を再現した。最初の deletion failure は integrity row を選んだ fixture 問題であり、製品 RED として扱わない。task selector と長い一時 path の表示幅を修正した最終 table は t5t PASS 1.252s。

pending deletion の schema は active key を要求する。completed tombstone は履歴であり continuation 対象にしない。tombstone を参照できる既存 `recordedKey` を表示解決に再利用したが、schema / continuation / backend guard は緩めていない。

## Resolution: Specification P4

Details の Refresh は underlying tab でなく明示 target kind / ID に対応する category から再解決する。result origin と viewport を保持する。対象が削除された場合も origin を消さず、ID 付きの read-only removed 表示を残す。snapshot 読み込み失敗・再失敗・回復時も状態と origin を正直に表示する。

`TestReviewFixRefreshKeepsExplicitResultDetailsAndOrigin` は Result → View entity (`v`) と View task (`t`) の両方で、Destinations tab のまま異なる種類の details を表示し、F5 の target / scroll 保持、stale / repeated failure / recovery / removal、Esc の result 復帰を検証する。

RED t5v: 両方とも F5 で Managed destinations に戻る。修正後、最終 focused t62 と固定 SHA の全 suite で PASS。coordinator の native baseline / fixed recheck は重複実施していない。

## Harness / fixture corrections

初期編集時の compile error、task selection / path wrapping の fixture error、画面外の文字に対する assertion は修正したが製品 RED に数えない。t5w / t5x / t5y の timeout は、除去 fixture が `Registrations: nil` を保存し schema-invalid な null となったことと、filter 内 `t.Fatal` 後の cleanup 待機が原因だった。fixture は有効な空配列にし、削除した registration の History / Onboardings も整合させた。

`startHandoff` は `program.Run` の完了通知を defer で送信し、cleanup の受信待機を2秒に制限した。実際の assertion を無期限 cleanup が隠さない。最終 focused t62 は timeout なしで PASS 7.738s。

## Verification on exact product SHA

Windows amd64、Go 1.26.2、PowerShell 5.1、既存 CPython 3.12.13。すべて専用 hc background terminal、`powershell.exe -NoProfile -ExecutionPolicy Bypass` による process-only policy。永続 ExecutionPolicy は変更していない。build 開始時と inspection 時に clean tree / exact HEAD を確認した。

| Check | Command / evidence | Result |
| --- | --- | --- |
| Final focused TUI | `go test -buildvcs=false ./internal/tui -count=1 -timeout=90s` (t62、commit と同一 source) | PASS 7.738s |
| Full uncached Go | `go test -buildvcs=false ./... -count=1` (t63) | PASS: cmd 1.874s / app 41.732s / cli 2.437s / ssh 23.273s / tui 9.048s; state has no test files |
| Vet | `go vet -buildvcs=false ./...` (t64) | PASS exit 0 |
| Native dev build | `go build -buildvcs=false -o .tmp/ux-review-verification/ssh-keyflow.exe ./cmd/ssh-keyflow`, `--version` (t66) | PASS `SSH KeyFlow dev` |
| Six-target package | `python -X utf8 scripts/package_release.py --version v0.2.0 --output .tmp/ux-review-verification/package` (t65) | PASS exit 0; six archives + SHA256SUMS |
| Archive / metadata / extracted smoke | task-owned assert-based inspector `.tmp/ux-review-verification/inspect.py` (t68) | PASS exit 0; details below |

Native dev executable SHA256: `1275fabe720b9a7ffbe861fd74d9d1bf734f1183d947ae95818e8ce8e596b020`。

Portable5 / installer50 / profile30 は今回の TUI 変更の影響外で再実施しなかった。script / installer / packaging / distribution file は変更なし。先行実施の exact testedSHA は `284a00bcf7078042f072804a5ea36da725a4fd11`、結果は [coverage-06.md](coverage-06.md)。ユーザーの指定どおり unaffected checks の反復を避けた。

## Local package evidence

buildSHA: `ba0f69d81c3345ed2bd78062521bcd63adda7c69`。`v0.2.0` はローカル smoke の埋め込み版だけで tag / release はない。

manifest は6行、lowercase SHA256 / two spaces / unique basename / LF / no CR、全 digest を再計算した。全 archive は executable と source-identical `THIRD_PARTY_NOTICES.txt` の2ファイルだけ。tar は regular files、mode 0755 / 0644、uid/gid 0、空 owner/group。ZIP CRC 検査も PASS。各 target の ELF64 / Mach-O64 / PE signature と CPU を検査し、抽出 binary の `go version -m` で Go1.26.2 / GOOS / GOARCH / CGO_ENABLED=0 / trimpath=true / main package を確認した。

| Asset | Archive bytes | Binary bytes | SHA256 |
| --- | ---: | ---: | --- |
| ssh-keyflow_v0.2.0_darwin_amd64.tar.gz | 2970699 | 10125648 | `5db027cf44925d8282e4b89c5a46f944b5826936a298f9bfd51b50c3a8d50ad7` |
| ssh-keyflow_v0.2.0_darwin_arm64.tar.gz | 2771638 | 9697058 | `77294c2d895fc6b8d1ebe25d29f38b07e5afeaf68653549287fc83e2f7b5cb5d` |
| ssh-keyflow_v0.2.0_linux_amd64.tar.gz | 2933443 | 9978018 | `d9f7e2074fe714046eba9ded1ebd495c06603fda331240cc391d317156345dde` |
| ssh-keyflow_v0.2.0_linux_arm64.tar.gz | 2660806 | 9568418 | `fdee1d1e9ad6840ad5e8a56e1817052e02abcdfc74cef7d0049ba4c1c62f2275` |
| ssh-keyflow_v0.2.0_windows_amd64.zip | 3052747 | 10315776 | `b04a42cc5a8e99e5a87b8db76df21ab091651892f5e12ef92aedd76a7122e6aa` |
| ssh-keyflow_v0.2.0_windows_arm64.zip | 2730825 | 9702400 | `8139b50e59561019737df3af1a49a08d622f907e7600c6d7e53333956e5602a9` |

SHA256SUMS 自体の SHA256: `df9ec8fe40f9515f7e9af18e2f89b9bb33587244f3d79b9eeb82c5e6d811f7b9`。
THIRD_PARTY_NOTICES.txt の SHA256: `db7d61f9a20b112009d2d3da9e4df8cb4dd681fd3a9f8096c6cb3f1ae07d8289`。

抽出 Windows amd64 の `--version` は `SSH KeyFlow v0.2.0`、`--help` と `keys list --json` は exit 0 / stderr なし、keys は空。isolated HOME / USERPROFILE は完全に空のまま。残り5 target は cross-build / metadata 検査だけで native 実行していない。trimpath build の Go metadata は ldflags を出さないため、embedded version の実行確認は Windows amd64 のみ。archive timestamp 依存があり、別 build の同一 hash を保証しない。

## Separate self-review

Standards: AGENTS / RTK / domain / tracker / glossary、既存 code conventions、公開 test seam、単一 worktree、最小の修正範囲、exact-SHA 証跡、validation / error handling の維持を確認した。S1 の eligibility 重複を除去した。dependency / persisted state schema / stdout parsing / backend approval の変更はない。自己確認で追加の standards 指摘はない。独立 Standards recheck は coordinator が行う。

Specification: P1–P4 の各失敗を公開 journey と実出力で確認し、上記回帰を追加した。Review / execution の分離、drift guard、conditional hidden clearing、readonly controls、running / belowminimum guard、result origin を再確認した。runtime contract は現在の入力 behavior に限定して更新した。自己確認で追加の spec 指摘はない。native acceptance と独立 Specification recheck は coordinator が行う。

## Integration, cleanup and remaining gates

最終時点で integration `db2da488bfd413898f13d34d88ae614efa2f8277` を再 merge し、Already up to date を確認した。後続の変更はこの handoff だけで、製品 source は testedSHA と同一。work branch はコミット済み clean の状態で fresh merger に渡す。

証跡保存後、自分の `.tmp/ux-review-verification`（binary / package / extracted inspection / isolated smoke home / inspector / evidence JSON）と、自分の終了済み22端末 `t5h,t5j,t5k,t5m,t5n,t5p,t5q,t5r,t5t,t5v,t5w,t5x,t5y,t5z,t60,t61,t62,t63,t64,t65,t66,t68` を削除した。coordinator の fixtures / terminals / native evidence は保持した。handoff 内の local Markdown links、named tests、差分 whitespace、clean branch / integration ancestry / 元 main 不変を最終確認した。

[native-verification.md](native-verification.md) は coordinator 所有で編集していない。固定版の実際の paste / resize / child / P1 / P3 / P4 の最終 native 証跡、fresh merger、独立両軸 recheck、final-review resolution、spec / ticket06 の最終 gate closure は coordinator が担当する。この handoff は全仕様の完了承認ではない。

native Linux/macOS / Windows arm64、real remote SSH / server 操作、elevated Windows administrator shared-file 実行は未検証。source / binary transfer、real keys / servers、push / main merge / release は行っていない。
## AC11 follow-up: cursor responses preserve form scroll

独立 SPEC recheck は ba0f69d の P1–P4 解決を確認したが、cursor-only `inputResponse` が `updateForm` に届くたび無条件 `GotoTop()` により viewport が戻る AC11 regression を新たに指摘した。同じ post-review worker が継続して修正し、新 implementer は起動していない。

integration `087964f7f1aa2a8ab0dab5f20474e9df171b63df` を clean work branch に fast-forward で取り込んでから修正した。製品・テストの追加コミットは **`26269049b747db02988ad00db2b62dcff22119f0`** (`fix(tui): retain form scroll through cursor responses`)。変更は `internal/tui/forms.go` と `internal/tui/review_fixes_test.go` の2ファイルだけ。focused PASS 後に exact SHA を coordinator へ即時共有した。

`updateForm` は textinput 更新前後の Value / Position を比較し、実際の編集または caret 移動の場合だけ `GotoTop()` を呼ぶ。cursor blink / unrelated response のみなら viewport を維持する。Tab / Enter / field rebuild 等の既存 navigation は従来の処理を維持する。`inputResponse` の generation 判定と失効箇所は変更していない。

`TestReviewFixScrolledFormSurvivesCursorReplies` は公開 `tui.NewProgram` の32×12 Import form で long public path の末尾まで PgDn を進め、自然に発行された textinput command の応答を公開 filter で opaque message のまま一時保持・配送する。private model / message の型や field は参照しない。同じ input の cursor reply 後の viewport、unrelated response 後の viewport、次の editable private-path field に移動した後の旧 input の delayed reply、実際の caret 移動 / paste edit で focused field を再表示することを検証する。Runner は未実行。

- RED t6r: PgDn 後には path 末尾と Field 2/6 が読めていたが、cursor reply 配送後に Import existing key / Existing public key path / Full value の先頭へ戻った。FAIL 2.285s。
- GREEN t6s: 同一 public regression PASS 2.279s (test 1.23s)。
- 初期 t6q は startup WindowSize による80×24上書きで fixture が scroll せず失敗した。public filter で32×12を固定してから上記 meaningful RED を得た。t6q は製品 RED に数えない。

### Exact-source verification for AC11 follow-up

2026-11-08 JST、Windows amd64、Go1.26.2、PowerShell5.1、CPython3.12.13。専用 hc background terminals / process-only `powershell.exe -NoProfile -ExecutionPolicy Bypass`。全固定-source checks は clean **26269049b747db02988ad00db2b62dcff22119f0** を対象とし、後続の変更はこの handoff だけ。

| Check | Command / evidence | Result |
| --- | --- | --- |
| Full focused TUI | `go test -buildvcs=false ./internal/tui -count=1 -timeout=90s` (t6t、commit と同一 source) | PASS 9.037s |
| Full uncached Go | `go test -buildvcs=false ./... -count=1` (t6v) | PASS: cmd 2.028s / app 41.354s / cli 2.343s / ssh 23.251s / tui 10.282s; state has no test files |
| Vet | `go vet -buildvcs=false ./...` (t6w) | PASS exit 0 |
| Native dev build | `go build -buildvcs=false -o .tmp/ux-review-ac11-verification/ssh-keyflow.exe ./cmd/ssh-keyflow`, `--version` (t6x) | PASS `SSH KeyFlow dev` |
| Six-target package | `python -X utf8 scripts/package_release.py --version v0.2.0 --output .tmp/ux-review-ac11-verification/package` (t6y) | PASS exit 0 |
| Archive / metadata / extracted smoke | task-owned `.tmp/ux-review-ac11-verification/inspect.py` (t6z) | PASS exit 0 |

Native dev executable SHA256: `8038c3a6537f128d4e69f083d39941f66f3d0562a3db2592bdc7153699342c1f`。

Portable5 / installer50 / profile30 の scripts は変更なしで今回も再実施しなかった。先行 exactSHA / PASS は前回記録と coverage-06 を参照する。

### Package evidence for 2626904

buildSHA は `26269049b747db02988ad00db2b62dcff22119f0`。local smoke version `v0.2.0` のみで tag / publication はない。全6 digest を再計算し、manifest の unique names / digests、64hex / two spaces / LF / no CR、exact7 assets を検査した。archive は executable + source-identical notices だけ。tar regular files / 0755・0644 / uid・gid0 / empty owner・group、ZIP CRC、ELF64・Mach-O64・PE と各 CPU、抽出全 binary の Go1.26.2 / main package / GOOS・GOARCH / CGO_ENABLED=0 / trimpath=true を再検査した。
| Asset | Archive bytes | Binary bytes | SHA256 |
| --- | ---: | ---: | --- |
| ssh-keyflow_v0.2.0_darwin_amd64.tar.gz | 2970695 | 10129744 | `bca06feaaf7759f9150de141865d423eaa69161b1c473904593c89eb8b467ff6` |
| ssh-keyflow_v0.2.0_darwin_arm64.tar.gz | 2771908 | 9713570 | `2e4e7b0ea3f01a31df1a232471779a93563db1e51223081db9a120495444c786` |
| ssh-keyflow_v0.2.0_linux_amd64.tar.gz | 2933703 | 9978018 | `fb3f35250f3a3e4955c348039b555d23ba957424f64660d6afb51a6972e93276` |
| ssh-keyflow_v0.2.0_linux_arm64.tar.gz | 2661210 | 9568418 | `06fbc60305316156860132b3c8badcc6beaa361199c117c9ed99de0f8214bdb4` |
| ssh-keyflow_v0.2.0_windows_amd64.zip | 3052972 | 10319360 | `24c199b7b93ee6c0efe98514923b623d0da27e9385a077ccc3dbd12876cebf6e` |
| ssh-keyflow_v0.2.0_windows_arm64.zip | 2731141 | 9705472 | `3e974ad3156c5d6869f6afcc16b10658c4558db9db8897274981263dcd6a39b1` |
SHA256SUMS 自体の SHA256: `bb09d58504238ba6378ead702058a724cd8554e15ec4d2cacddb43e2ad8b8e61`。
notices SHA256 は `db7d61f9a20b112009d2d3da9e4df8cb4dd681fd3a9f8096c6cb3f1ae07d8289`。

抽出 Windows amd64 は `--version` = `SSH KeyFlow v0.2.0`、`--help` に Usage、`keys list --json` で keys[]、すべて exit0 / stderr なし。isolated HOME / USERPROFILE は空のまま。残り5 target は native 実行していない。trimpath metadata は ldflags を表示しないので埋め込み版の実行確認は Windows amd64 のみ。別 build の archive 同一 hash は保証しない。

### Follow-up self-review, cleanup and gates

Standards self-review: product は Value / Position の既存公開 API による focused な条件追加のみ。test は公開 program / filter / view / Runner 境界を利用し、自然な command reply をそのまま配送する。既存 generation / schema / dependency / runtime guard の変更や広範な refactor はない。diff whitespace と local links / named tests を確認し、追加の standards 指摘はない。

Specification self-review: AC11 の長い editable path の垂直アクセスを実 tick と非編集応答の後も維持し、旧 input の返信を次 input に誤配送せず、実 edit / caret navigation の再表示も確認した。P1–P4 回帰を含む全 TUI / full suite を再実施した。追加の spec 指摘はない。独立 recheck は coordinator 所有。

最新 integration `087964f7f1aa2a8ab0dab5f20474e9df171b63df` の再 merge は Already up to date。元 checkout は clean main `89340aebbc0b3ab7f781b822aaafc2130ae76f73` のまま。証跡保存後、自分の `.tmp/ux-review-ac11-verification` と終了済み9端末 `t6q,t6r,t6s,t6t,t6v,t6w,t6x,t6y,t6z` を削除した。coordinator の native fixtures / terminals / evidence は保持した。

coordinator から ba0f69d の prior native cases および実 ssh-keygen interruption / restart PASS の報告があったが、ここでは独自の native 証拠に置き換えていない。2626904 の affected native 再検証、independent rechecks、fresh merger、final-review resolution、spec / ticket06 gate closure は coordinator が担当する。[native-verification.md](native-verification.md) と過去 evidence は編集していない。push / main merge / release / source・binary transfer はない。

## Coordinator closure

The above worker handoffs are historical checkpoints. Final independent Standards and Spec rechecks at `26269049b747db02988ad00db2b62dcff22119f0` resolved S1/P1–P5 with no further material findings. Final affected native scroll check passed with matching binary SHA. Fresh merger integrated handoff d279463 at `c155e2adf6c0b0963fb68085c9d036286f188539` with identical tree and a passing exact-merge focused TUI run. Final coverage/review/native records and all six ready-for-human tickets supersede the earlier pending coordinator gates. Limits remain explicit; no external publication occurred.
