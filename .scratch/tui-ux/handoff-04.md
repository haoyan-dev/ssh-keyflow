# Ticket04 review and native handoff

Status: ready-for-human
Date: 2026-11-07
Branch: feat/04-review-native-handoff

Clean starting base and latest merged local integration: `cdd0c5b67cfd9c3df909d8ed6901079482c20569` (`implement/tui-ux`). Product commits: `f125818092c6b220bc54d3d715d152e9d21e0117`, then `6fdc973d6908fd32e4b3d73ae9272225bb753569`. The latter is the exact final verified product source. The completion commit adds only this artifact and the checked ticket; its SHA is supplied in the completion response. No remote refresh, transfer, push, main merge or release was performed.

## Implementation and draft contract

Review has keyboard controls Continue to terminal, Edit, Cancel and Help, including32x12. e/Esc and Edit restore the form's retained textinput models, caret, focus, step and viewport. Review/help/picker round trips retain the draft and caller category/query/selection. Help restores search editing as well as the query. q/?/action letters stay literal in text fields and search. Category shortcuts cannot abandon a form. The picker list's built-in q quit binding is disabled. Draft detection traverses the whole overlay origin stack, so Discovery over a picker cannot quit or drop the retained form.

`rememberDraft` records applicable initial values and the fixed request seed after form entry. Cancel/Ctrl+C compares against that entry baseline. Changed values or choices open Keep editing / Discard draft, with Keep editing selected. Escape/Ctrl+C there keeps the draft. Clean cancellation returns directly. Discard unwinds overlays and form to the caller without execution. Help closes before Ctrl+C can cancel a draft. First-step Esc invokes Cancel; later add steps go back. Explicit Cancel follows Help in the form's keyboard controls.

`rememberEdit` preserves the exact editing viewport before review; `editReview` restores it without rebuilding fields.03's `draft`, `formTarget`, validated `request`, `saveFields`, `clearHidden` and `resolveFormTargets` contracts remain. Exact fixed targets are re-resolved immediately before handoff. Label drift does not retarget a request; changed/missing saved targets return a retained-draft error. Review lists expected effects and runtime uncertainties; no storage write or command runs before native Continue.

## Native execution and interruption

`tea.Exec` retains the release/restore boundary and original terminal input/output. A running guard rejects queued keypresses until the completed message is processed, preventing double Enter from launching two operations. Output is copied into diagnostics; original stdin is never recorded or wrapped for the SSH Runner. Existing backend exact prompts, validation, locks, safety checks and request semantics remain authoritative. There are no password/passphrase fields or new stored workflow records.

Coordinator baseline evidence: exact02 binary `dce6c7c` on Windows Herdr t30 reached the manual unregister Type yes prompt and Ctrl+C killed the whole TUI with exit1, `program was killed: program was interrupted`. The disposable saved-state SHA256 stayed `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0`. This baseline is a failure reproduction, not04 acceptance evidence.

Bubble Tea v2.1.1 runs Exec synchronously, then restores input and returns its completion message. Its ordinary OS signal handler can enqueue InterruptMsg during Exec; the event loop filter runs before InterruptMsg terminates the runtime.04 owns a lifecycle-scoped signal listener instead of replacing caller WithFilter. `NewProgram` disables the library handler, starts the listener through Init, and unregisters it after Program.Wait. SIGINT during an active handoff cancels only that handoff context; outside it the listener sends ordinary InterruptMsg. SIGTERM retains quit behavior. Caller filters and terminal input/output options remain composed.

Windows confirmation uses the original app.ReadConfirmation input: console Ctrl+C aborts the native ReadFile, and context is checked before/after. Non-Windows cancellation uses the already installed cancelreader only around the exact approval read on an os.File, joins cancellation cleanup, and leaves child credential stdin untouched. No read-ahead input goroutine or replacement credential reader was introduced. Native confirmation callback remains distinct from informational reports.

Coordinator native04 PASS on exact clean source `6fdc973d6908fd32e4b3d73ae9272225bb753569`, Windows amd64, Go1.26.2, Herdr0.1.18,80x24. Coordinator built integration `.tmp/native-handoff/ssh-keyflow-04.exe` and used the disposable imported static public-only key/registered-pair fixture (example.test; no private credentials). u, Ctrl+S, Enter reached the exact native manual-unregister prompt. Native Ctrl+C returned to a usable Result with Error EOF, confirmation unknown and execution stopped; the TUI process stayed live. In that same process, Esc, u, Ctrl+S, Enter reached the prompt again; no + Enter returned Result with confirmation declined. q then exited0. State SHA256 remained `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0` throughout. Coordinator removed its t3f/t3g terminals. No approval was granted and no remote call was made. This proves restoration and repeat denial at this native confirmation boundary; it does not establish OpenSSH credential-prompt interruption or other platforms. The observed EOF is retained truthfully rather than rewritten as a more specific cancellation error.

Additional coordinator native PASS on the same6fdc973 binary: at32x12, c, q, a, r, Tab, Tab, Enter opened labelled Form help. Ctrl+C returned to the form; Ctrl+S opened Review; Esc, Ctrl+C showed Keep editing by default, and Enter restored Help focus. Actual Herdr core resize32x12 ->31x11 through an hc attach terminal showed the minimum-size guard: Ctrl+S/Enter/a neither executed nor changed the input. Actual resize to60x18 restored the same Help focus; Shift+Tab twice revealed original input qar unchanged. Ctrl+C, Tab, Enter discarded without execution. Saved-state hash remained the same. Coordinator removed its t3h/t3j/t3k terminals. This is actual native size/focus evidence, separate from injected WindowSizeMsg tests.06 will repeat critical smoke against final integration.

Event Ctrl+C and a fake Runner returning context.Canceled are separate evidence and do not prove native OS signal restoration.06 owns broader native integrated acceptance and final-state resize verification.

## Contract for05: operation return and results

`execution.go` supplies `completed{result operationResult}`. Preserve these independent fields when replacing the interim renderer:

| Field | Meaning |
| --- | --- |
| request app.Request | Exact validated request passed to app.Run. |
| target app.ReportTarget | Explicit host/key/registration/onboarding/creation/deletion IDs, seeded from the request and loaded saved resume relationships, then enriched by nonempty report IDs. Never infer a result action from a hidden list row. |
| reports []app.Report | Ordered01 authoritative facts, collected synchronously. Saved means that checkpoint was persisted, including pending/unknown; it is not whole-operation success. Allocated IDs alone do not prove persistence. |
| err error | Original app.Run/native execution error, retained even after saved effects. |
| diagnostics string | Original native output/instructions retained for scrolling; ANSI sequences are stripped for display. It is not an outcome classifier or keyboard input. |
| reloadErr error | Independent Snapshot failure on return or later Refresh. |

The collector also calls the caller's existing Options.Report sink. Sinks may run under an operation lock: collect facts only, do not call an operation or mutate storage. Native Run is the only executor. The refreshed `m.view` remains separate from current-operation facts. Missing facts mean unknown/not recorded; nil error is not proof of completed requested work. Manual required plus verification not-checked remains manual follow-up, even if no new registration/task was saved. Preserve prior recorded evidence separately. A later error/unknown fact must not erase earlier saved checkpoints or imply rollback.

05 owns replacing `renderOperationResult` with full AC08 classification, target-bound entity/task links, primary next actions, and AC09 Tasks projection.04's renderer is deliberately factual interim content: request/explicit IDs, each typed outcome and Saved checkpoint, error and scrollable diagnostics; no false generic Operation complete. Do not carry its neutral execution-returned heading forward as final product classification. No stdout/error-text parsing, fabricated Task, Mark done or operation replay belongs in results.

Completion attempts a read-only Snapshot. Failure retains the envelope and prior view, marks stale and keeps mutation guards active. F5 on result retries Snapshot only; it retains facts/error/diagnostics/target and viewport, and clears stale only after success. Refresh cannot replay app.Run or imply remote verification. Keep this behavior when adding result links.02's visibleTarget isolation and03's locked-target re-resolution remain required.05 rendering work should leave native signal ownership and confirmation input handling intact.

## Red/green and final verification

Agreed seam: public `tui.NewProgram`, key/window messages and rendered View; real app.Run/Snapshot on disposable fixtures; fake Runner only at external command boundary. No private model accessor, recursive agent, live server, credential or user SSH state. Each listed red is a behavioral assertion failure, not a compiler/tooling failure.

| Slice | Red | Green |
| --- | --- | --- |
| Dirty cancel | ux04-red-cancel: Ctrl+C dropped changed draft. | ux04-green-cancel2: Keep editing default, retained caret produces abc, discard writes nothing. |
| Review controls/edit | ux04-red-review: compact review lacked accessible Edit/Cancel/Help. | ux04-green-review: all controls reachable, exact viewport/focus retained. |
| Return facts | ux04-red-envelope3: manual nil-error said Operation complete and lost typed facts. | ux04-green-envelope: manual/required and verification/not-checked, explicit target and instructions survive. |
| Duplicate Continue | ux04-red-duplicate: deterministic two public Enters with running guard temporarily disabled launched two native approvals. | ux04-green-duplicate: one approval, no external command on denial. |
| Nested overlay | ux04-red-overlay: q quit Discovery above a retained picker draft. | ux04-green-overlay: draft survives q, Ctrl+C offers Keep editing. Full TUI suite PASS3.359s. |

Additional public journeys cover repeated edits and literal q?cirauds, explicit clean Cancel, dirty picker search/Help/Keep, caller Keys query retention, review Help scroll,80x24/60x18/32x12 and undersize input guards. Handoff tests cover manual facts; denial plus failed Snapshot, failed/successful F5 with no replay; saved creation checkpoint plus simulated interrupted child, explicit allocated IDs, original Runner stdin identity, ANSI diagnostic rendering and usable returned Help.03's existing real handoff tests retain key/password command intent and clear stale identity arguments. Simulated interruption is explicitly not native signal acceptance.

Exact product `6fdc973d6908fd32e4b3d73ae9272225bb753569`, Windows amd64, Go1.26.2. Long checks ran in task-owned Herdr terminals using process-only PowerShell execution policy; no persistent policy change:

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'`: PASS exit0, ux04-full-final/t3b. cmd1.941s, app41.112s, cli2.406s, ssh22.319s, tui4.209s; state has no tests.
- `go vet -buildvcs=false ./...`: PASS exit0, ux04-vet-final/t3c.
- `go build -buildvcs=false -o <temporary executable> ./cmd/ssh-keyflow`, then `--version`: PASS, SSH KeyFlow dev. Same exact source cross-built with CGO_ENABLED=0 for linux/darwin/windows × amd64/arm64: all six PASS, ux04-build-final/t3d.
- `git diff --check cdd0c5b`: PASS. Latest local integration is the recorded base and is already an ancestor.

Earlier full suite/vet/build at f125818 also passed. Final6fdc973 adds only the nested-overlay guard and related Help/footer fix with its red/green test; final full suite/vet/all builds were repeated afterward. Later changes are completion documentation only. All23 finished worker check terminals and the verified task-owned `.tmp/ux04-checks` directory were removed after evidence collection; coordinator binaries/fixtures and other workers were left untouched. The final local integration merge again reported Already up to date at cdd0c5b.

## Changed files

- Review/draft/forms: `internal/tui/review.go`, `internal/tui/forms.go`, `internal/tui/actions.go`.
- Event routing, origin/stale refresh and native return: `internal/tui/tui.go`, `internal/tui/navigation.go`, `internal/tui/execution.go`.
- Signal/approval handling: `internal/tui/signals.go`, `internal/tui/confirmation_windows.go`, `internal/tui/confirmation_other.go`.
- Public journeys: `internal/tui/review_test.go`, `internal/tui/handoff_test.go`, `internal/tui/tui_test.go`.
- Completion artifacts: this handoff and `.scratch/tui-ux/issues/04-review-edit-native-handoff.md`.

## Standards self-check

Reviewed separately against AGENTS.md, RTK, domain/tracker/triage rules, TDD and ponytail. GLOSSARY vocabulary and01/02/03 extension contracts are retained; no relevant ADR directory exists. Existing textinput/list/viewport, backend validators and installed cancelreader are reused. No CLI/backend/report implementation, schema, secret field, new dependency version or safety approval was changed. Tests use the authorized public seam and literal behavioral expectations. Caller WithFilter and report sink composition, original credential stdin, context/listener cleanup, stale-state mutation guards and target identity were inspected. No unresolved standards finding remains in04 scope.

## Specification self-check and limits

Reviewed AC05/06 separately from standards, with relevant AC08/10/11. Lossless review/edit, dirty default Keep, clean cancel, explicit controls, text contexts, Help origins, no pre-handoff mutation, double-Enter rejection, typed return envelope and stale Snapshot-only reload have public evidence. Final spec review found the nested Discovery draft escape; a red/green regression fixed it. Backend unregister/delete safeguards are preserved and full regression tests pass.05 still owns the full results/Tasks presentation;06 owns native final acceptance and runtime documentation.

Coordinator native Windows manual-unregister interruption/denial passed at the exact product SHA, as recorded above. No claim is made for actual OpenSSH password/passphrase interruption, Linux/macOS native execution, other native hosts, actual remote effects, release packaging or published runtime docs. Cross-build success proves compilation only. No remote transfer was attempted. Event and fake-command checks do not prove these native behaviors.
