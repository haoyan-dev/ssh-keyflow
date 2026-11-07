Status: ready-for-human
Blocked by: 01, 02, 04

# 05: Surface follow-up work and render honest operation results

## Problem and intended behavior

Recovery is scattered and stale onboarding looks unfinished forever; generic completion hides manual and partial outcomes. Provide one derived Tasks surface and results that identify saved facts, uncertainty, and the next supported action.

Specification: [AC08 and AC09](../spec.md#ac08-results-and-stale-state), relevant AC02/AC07/AC10/AC11. Backend constraints: [review](../review.md#backend-constraints-that-shape-the-redesign).

## Scope and seams

- Project Tasks read-only from existing state using AC09 precedence. Group onboarding/creation/registration and preserve standalone local work. Retain stable source identity and deterministic ordering; no state schema or workflow engine.
- Choose safe resume IDs/operation forms. Current unregister supersedes stale onboarding. Explain when a continuation includes registration rather than standalone verification/config work.
- Show config/integrity review without inventing a config resume ID or disabling guarded pending-deletion recovery. Do not auto-clear/manual-mark-done records.
- Use 01 reports plus refreshed Snapshot for AC08 results, preserving per-registration connection facts, operation effects, diagnostics, and target-bound links. Missing durable facts remain not recorded.
- Failed reload locks stale-state mutations behind Refresh/Exit; Refresh only loads state. Full diagnostics/manual instructions remain accessible.

## Acceptance criteria

- [x] AC09 table is covered: one row for linked work, correct continuation ID, completed onboarding suppression by evidence, and no accidental re-registration after unregister.
- [x] Public-only verification is unavailable without perpetual verification-only task; failed private-key check remains supported follow-up.
- [x] Config pending/edit evidence shows honest recovery limits; incomplete deletion stays resumable despite general review warnings; history/tombstones remain read-only.
- [x] AC08 manual nil-error says Manual action required; manual Register for unrecorded/already verified pairs explains when no new Task was saved, preserves prior evidence, and behaves honestly after restart. Include denial before intent persistence also creates no fabricated Task.
- [x] Registration-success/verification-failure and stopped-after-saved-effects remain distinct. Alias-generation denial versus later Include denial/failure name exactly which file/binding effects remain, including main-config unchanged on denial.
- [x] Unknown outcomes require reconciliation; output strings/absence of error do not classify completion.
- [x] Load/reload error never renders empty Tasks or silently retains mutable stale state; Refresh never replays execution.
- [x] AC02 target-bound actions and AC10 unregister/delete safeguards remain intact. Screens meet relevant AC11 sizes.

## Verification

Table-driven projection cases for every AC09 row, combined reasons/links, public-only/private key, multiple onboarding references, current unregister, removed/complete, missing joins, pending config paths and Include edits, interrupted deletion, and historical records. Event journeys test result -> exact entity/task, manual diagnostics, partial failure/cancellation, and reload fail -> refresh. Assert no writes/Runner calls from projection and no blind remote retry. Use authoritative app fixtures for checkpoints; 06 performs integrated/native acceptance.

## Handoff

Starts after 01/02/04. Provide the completed outcome matrix, task precedence cases, and any remaining native/cross-platform limits to 06. Do not introduce new operation types to paper over unsupported recovery routes.

## Comments

- 2026-11-07: Proposed ticket; Tasks and result wording remain ready-for-human until design approval.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.
- 2026-11-07: 05実装完了、ready-for-human。製品／検証 SHA `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`、統合 `15fbfe9ca70fddabf3227d7c194deca1442b216f` 取り込み済み。純粋 Tasks、正確な再開対象、型付き結果、manual/restart、config 両拒否／不確実性、stale Snapshot-only Refresh を公開境界で検証。追加指摘の複数 onboarding owner と安全な binding→config 親集約も解決。全 Go suite／vet／Windows build・version／diff は同 SHA で PASS。詳細は [handoff-05](../handoff-05.md)。coordinator の同 SHA Windows native05 smoke も PASS（統合 `4b67385a9098ddebde9b4af335a1c5db13c4d656` の native-verification.md）。cross-journey／出荷 docs／packaging は06、既知 paste defect は指定の fresh fixer。local のみ、push/main merge なし。

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
