Status: ready-for-human
Blocked by: none

# 02: Make entity navigation searchable, readable, and target-bound

## Problem and intended behavior

Lists mix current records with saved jobs/history, details are raw JSON, and visible details can trigger actions using a hidden list selection. Establish stable entity navigation and readable context before adding forms and recovery.

Specification: [AC01 and AC02](../spec.md#ac01-categories-identity-search-and-empty-states), relevant AC05/AC11. Evidence: [F03, F06, F08-F10](../review.md).

## Scope and seams

- Refactor `internal/tui/tui.go` navigation state only as needed: category, selected fixed ID, filter, scroll, visible entity, and origin. Use existing list/viewport components.
- Present Destinations, Keys, Registrations, Tasks categories. 05 supplies Tasks projection/content; do not ship an empty placeholder as completed recovery.
- Entity labels join names/login/fingerprint with current evidence. Move history and tombstones to explicit read-only views. Preserve inspectable full values/IDs and discovery ownership boundaries.
- Actions derive from the visible target; eligibility/blockers are advisory with backend revalidation. Do not introduce destination rename/edit, live SSH probes, or a generic router framework.
- Fresh-home/search-empty explanations and readable status labels must work with the size contract.

## Acceptance criteria

- [x] AC01 search/category state restoration and refresh fallback work by fixed ID, including duplicate-looking names and removed targets.
- [x] AC02 detail joins separate registration, per-destination connection, config evidence, and history; missing references require review.
- [x] Result/discovery/history screens cannot silently act on an underlying row; historical records expose no mutation.
- [x] Unsupported contextual shortcuts explain blockers or route to eligible selection, never blank ID forms.
- [x] Read-only startup/navigation remains storage- and SSH-probe-free. Empty Tasks copy makes no remote verification claim.
- [x] Relevant AC05 keyboard/help origin and AC11 text/size constraints hold; 04 completes draft-specific navigation.

## Verification

Public-entry event tests: search then details/help/back, category switch and restore, refresh reorder/removal, Unicode/long labels, duplicate-looking records, empty query results, missing joins, context target different from underlying selection, and read-only historical/discovery views. Assert requests do not execute during navigation. Record the tested commit and native-size limitations.

## Handoff

Navigation contract is implemented and stable for03. See [handoff-02.md](../handoff-02.md) for fixed-ID action seeds, explicit visible targets, page origins, tests, and ownership. Tasks is an honest interim shell;05 owns its projection/results.03 owns full selectors/staged forms and re-resolution;04 owns retained drafts/native outcomes. Avoid concurrent edits to tui.go.

## Comments

- 2026-11-07: Draft for human review; Tasks content is explicitly dependent work in 05.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.

- 2026-11-07: Implemented navigation, joined details, category/related search, fixed-ID refresh, explicit history/discovery, read-only result identity, advisory actions, help origin, and compact/minimum-size behavior. Supported criteria checked; ready-for-human records completed local implementation awaiting human review, with03 unblocked by the stable contract. Missing saved joins fail the existing strict read-only load boundary. Full Tasks and draft-specific behavior remain downstream by contract. Separate standards/specification reviews and verification conditions are recorded in [handoff-02.md](../handoff-02.md). Final commit SHA and post-commit verification are in the completion report.

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
