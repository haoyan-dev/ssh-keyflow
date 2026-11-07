Status: ready-for-human
Blocked by: none

# 01: Report authoritative operation outcomes without changing CLI behavior

## Problem and intended behavior

A nil error can mean manual instructions were prepared, while a later error/cancellation can follow successful saved effects. The TUI cannot honestly render these outcomes from error plus captured text. Add the smallest typed informational report boundary so results can distinguish these cases.

Specification: [AC07 and AC08](../spec.md#ac07-additive-outcome-facts). Evidence: [F01 and backend constraints](../review.md).

## Scope and seams

- Add an optional report sink to `app.Options`; retain `app.Run(ctx, Options) error`, existing CLI text/exit behavior, request types, and state schema.
- Emit typed facts from existing authoritative branches/checkpoints in `internal/app`: creation/import, onboarding, registration, unregister, configuration, deletion, and confirmation/uncertainty paths. Emit persisted effects only after the save succeeds.
- Report resolved IDs, saved effects, manual follow-up, destination-specific verification/unavailability, config outcome, and stopped/uncertain evidence. Omit secrets and avoid deriving facts from stdout/error strings.
- Preserve existing confirmation callbacks. Reporting is not an execution plan, approval engine, or new operation API. The consumer supplements reports with Snapshot, retaining diagnostics and errors.

## Acceptance criteria

- [x] AC07 passes with and without a consumer; public `Run` compatibility remains intact.
- [x] Explicit manual register and unregister reporting says follow-up required even when `Run` returns nil. Manual Register with no prior pair record and with an already verified pair preserves the existing persistence behavior and reports the distinction; no task is fabricated.
- [x] Registration success followed by failed verification reports both facts and skipped configuration.
- [x] Report alias generation and main-config Include independently. Declining alias generation preserves registration; declining Include preserves registration plus managed alias/binding, leaving main config unchanged and naming manual setup. Test later Include failure as well. Denial before any effects remains distinguishable.
- [x] Ambiguous/interrupted execution and failed saves do not report complete/no changes without evidence.
- [x] Report-disabled existing CLI callers retain output, exit semantics, approvals, locks, and safety behavior.

## Verification

Completed focused shared-app tests through Run/Options and Snapshot using temporary saved fixtures, fake Runner and confirmation; no real keys, secrets, or SSH. Full Go suite, vet, build/version, and diff checks passed on integration merge b9cf0b9c0b1e96a48a8bca68a6de0f03a7a02bcd (implementation f22e6d8596072114e3c2c56374490b737c2b8f7e). Commands ran under process-only powershell.exe -NoProfile -ExecutionPolicy Bypass; no persistent settings changed. Exact commands, package results and Herdr references are in [handoff](../handoff-01.md#verification).

## Handoff

Implemented on feat/01-operation-outcomes. See [handoff-01.md](../handoff-01.md) for the exact report contract, outcome coverage, tested source SHA, execution conditions, and limits for 04/05. Ticket-owned changes are confined to internal/app and these ticket/handoff documents. Integration dce6c7c was merged before final checks.

## Comments

- 2026-11-07: Proposed implementation ticket. Human review of the UX/spec remains required; no implementation has started.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.

- 2026-11-07: Implementation complete and ready-for-human. Optional typed reports distinguish saved checkpoints, current manual follow-up, pair verification, alias/Include outcomes and stopped/unknown effects. TDD red/green regressions and separate standards/spec reviews completed; findings resolved. Latest integration dce6c7c merged and full Go suite/vet/build passed. Native/cross-platform TUI acceptance remains with ticket 06.

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
