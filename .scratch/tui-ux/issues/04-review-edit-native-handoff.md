Status: ready-for-human
Blocked by: 01, 03

# 04: Preserve drafts through review/help and maintain native approvals

## Problem and intended behavior

Esc from review currently loses the draft. Introduce back-to-edit and explicit discard behavior, then return native execution facts to the TUI without changing credential input or confirmation safety.

Specification: [AC05 and AC06](../spec.md#ac05-back-edit-cancel-and-help), relevant AC08/AC10/AC11.

## Scope and seams

- Preserve draft values, current step/focus, query/selection, and scroll across review, edit, picker, and help. Make Cancel explicit; dirty discard defaults Keep editing.
- Distinguish text-input keys from global shortcuts, and make Help keyboard-accessible in forms. Follow the specified search/category/back/quit contexts.
- Review resolved labels, method/options, expected effects, and runtime uncertainties. Continue to terminal is orientation, not replacement approval.
- Retain native `tea.Exec`-style stdin/stdout handoff and all backend exact confirmations. Collect 01 outcome facts and diagnostics, prevent duplicate starts, refresh state on return.
- Do not promise rollback or capture credentials. 05 owns full result presentation, Tasks, and recovery actions.

## Acceptance criteria

- [x] AC05 Esc/Edit from review restores input and focus. Repeated edit/review is lossless.
- [x] Cancel/Ctrl+C on dirty draft offers Keep editing by default; Discard causes no operation. Clean cancel returns directly.
- [x] Help returns to origin; text `q`, `?`, and action letters remain text; category keys do not abandon forms.
- [x] AC06 review clearly distinguishes provisional effects from exact native checks. No pre-handoff mutation occurs.
- [x] Double Enter cannot start two operations; native credentials/confirmation/input route stays unchanged and interruption restores a usable TUI.
- [x] Return preserves typed facts, diagnostics, errors, and explicit target identity for 05; stale reload cannot be treated as empty success.

## Verification

Public-event tests cover the reproduced name-loss journey, validation/edit, help from each page, dirty/default discard decisions, literal shortcut characters, picker cancel, and duplicate continue. Use fake Runner/confirmation tests for no pre-review effects and preserved denial behavior. Native Windows PTY checks cover focus, raw input handoff, terminal restore, resize, and interruption; record any Linux/macOS gap for 06.

Final product source: `6fdc973d6908fd32e4b3d73ae9272225bb753569`. [04 handoff](../handoff-04.md) records public red/green journeys, exact final full-suite/vet/native-version/six-target cross-build results, separate standards/spec self-checks and native limits. Coordinator native smoke on this exact clean source passed: Windows amd64, Go1.26.2, Herdr0.1.18,80x24, disposable public-only manual unregister. Ctrl+C returned a usable Result and kept the process live; the same process then denied a second prompt with no + Enter and q exited0. State SHA256 stayed8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0. Additional native32x12 Help/review/default Keep and actual32x12 ->31x11 ->60x18 resize checks passed, retaining qar/focus and preventing undersize execution/input; discard left saved state unchanged. The handoff records exact EOF/unknown/stopped facts and limits;06 retains final integrated native acceptance.

## Handoff

Starts after 01's report API and 03's draft/selector contract. Supplies the execution/return envelope and native checks to 05/06. Keep a concrete record of any interruption semantics that cannot be proven by event-only tests.

## Comments

- 2026-11-07: Draft, ready for human design review. Exact native approvals remain part of the accepted safety contract.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.

- 2026-11-07: Implemented review/edit/cancel/help, duplicate handoff guard, native signal ownership, typed facts/error/diagnostic/target envelope and stale read-only reload. Final product6fdc973 passes full Go suite, vet, native version/build and six cross-builds. Detailed evidence and04-to05 contract are in the handoff. Coordinator native Ctrl+C/denial smoke passed at this exact clean source with saved-state hash unchanged; exact evidence and remaining native/platform limits are recorded separately from public event tests. Ready for human review.

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
