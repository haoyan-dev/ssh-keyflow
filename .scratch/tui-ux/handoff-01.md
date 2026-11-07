# Ticket 01 handoff

Status: ready-for-human

Implementation: f22e6d8596072114e3c2c56374490b737c2b8f7e. Integration merge/tested source: b9cf0b9c0b1e96a48a8bca68a6de0f03a7a02bcd, containing implement/tui-ux dce6c7c6b9c8d71c92e7ed71fdb77b2445cce5c3. Started clean on feat/01-operation-outcomes at integration bb20d99b997bed2b42e6afbb1e2649f79d560a79; no remote refresh/push or main merge.

## Contract for 04/05

- Set optional `app.Options.Report func(app.Report)`; keep `app.Run(ctx, Options) error`, returned error, diagnostics, and a separate fresh `app.Snapshot`. The sink runs synchronously, sometimes under the operation lock: collect facts only; do not mutate storage or invoke operations inside it. Definitions: [reports.go](../../internal/app/reports.go); hookup: [app.go](../../internal/app/app.go).
- `Report.Kind` separates target, import, creation, onboarding, registration, verification, alias, Include, manual, confirmation, and execution. Target embeds fixed host/key/registration/onboarding/creation/deletion IDs. Allocated IDs can appear on a failed save: validate persisted recovery identities with Saved and Snapshot.
- `Saved` means that checkpoint was saved successfully, including pending/unknown checkpoints; it never means the whole request completed. Registration facts carry saved Status, Stage, Operation, and destination-specific ConnectionCheck. Paths identify local file/config evidence. No credentials, key content, native output, or error text enter facts.
- Preserve independent facts and earlier saved checkpoints. A later unknown/stopped fact does not erase them. A failed save emits unknown with Saved=false and omits the intended status/stage. Missing facts remain unknown/not recorded. There is deliberately no overall success event.
- Structured confirmation denial has NoChanges=true only before any saved checkpoint or reported uncertainty in that Run. False gives no guarantee. Execution/stopped means Run returned an error, including interruption or denial; retain the error to explain it. Legacy menu operations emit branch facts but handle their errors in the existing menu.

| Fact | Interpretation for results |
| --- | --- |
| manual/required + verification/not-checked | Follow-up required even when Run returned nil. Existing saved registration remains separate; an empty registration ID promises no task. |
| registration/applied + verification/failed | Registration saved, specified-key check failed, configuration skipped. |
| verification/unavailable | Public-only key, saved connection remains unverified. |
| alias/applied then Include/declined | Registration and managed alias/binding saved; main config unchanged by this request; manual Include setup remains. No edit intent is fabricated. |
| Include/pending Saved=true then unknown | Saved edit intent; preserve exact journal/backup and show uncertain checkpoint, never claim rollback. |
| Include/reconciled | Exact active top-level Include observed (Saved=false), or interrupted edit reconciled and completion saved (Saved=true). |
| Include/requires-review | Scoped/ordered Include or preflight conflict; retain diagnostics. May accompany nil error. |
| creation/deletion pending then unknown | Preserve saved IDs and file checkpoint evidence; use existing guarded resume. |

## Coverage and review

[reports_test.go](../../internal/app/reports_test.go) uses Run/Options, Snapshot, temporary saved fixtures, fake Runner and confirmation. Covers manual Register with no pair and prior verified pair; manual unregister; registration/check success/failure/public-only; both config denials; pre-intent conflict and post-intent Include failure; exact/scoped/recovered Includes; failed import/registration/alias/deletion saves; native generation interruption and same-ID resume; unregister removal/absence/uncertainty; onboarding links; stopped target identity; and failed Snapshot reload without report replay. Explicit sink-on/off comparisons preserve output, errors, prompts, commands and registration state.

TDD used vertical red/green slices. Final review regressions first failed on missing denial/resume targets, a falsely pending existing completed registration, and lost allocated identity after failed save; each passed after its corresponding reporting-only fix. Standards review and spec review were performed separately. All findings resolved: no new dependency/schema, output/error/approval changes, safety bypass, or task-owned TUI edit.

## Verification

Windows amd64, Go 1.26.2, 2026-11-07. Run through Herdr from this worktree, using process-only `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command '<command>'`. No persistent policy changes. Plain-policy fixture failure predates this work; the coordinator supplied a passing unchanged-source baseline under the same process-only condition.

- `go test -buildvcs=false ./internal/app -run TestReports -count=1`: PASS, 5.364s (Herdr t1b, f22e6d8 source).
- `go test -buildvcs=false ./... -count=1`: PASS, exit 0 (Herdr t1d, b9cf0b9). cmd 1.848s; app 40.801s; cli 2.254s; ssh 22.306s; state no tests; tui 2.245s.
- `go vet -buildvcs=false ./...`: PASS, exit 0 (Herdr t1f, b9cf0b9).
- `go build -buildvcs=false -o $verificationBinary ./cmd/ssh-keyflow`, then `& $verificationBinary --version`: PASS, exit 0, SSH KeyFlow dev (Herdr t1g, b9cf0b9). Temporary binary removed.
- `git diff --check`: PASS. Integration ancestor check: PASS.

Limits: fake SSH proves branch/command behavior, not real-server behavior. No real keys, SSH servers, native interactive handoff, Linux/macOS runtime, or TUI acceptance checks here; ticket 06 owns cross-platform/native acceptance. Reports are session facts; persisted task projection still follows Snapshot only. A reload failure must leave facts/diagnostics intact and views stale until Refresh succeeds.
