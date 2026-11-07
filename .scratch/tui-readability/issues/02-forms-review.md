# 02 — Readable drafts, review and discard

Status: ready-for-human
Blocked by: none (01 completed)
Spec: ../spec.md AC03, AC04, AC08

Fix input width budget and focus visibility. Group review by readable target, options and effects with technical references below. Fit keyboard controls to terminal size and retain exact draft/caret.

## Acceptance

- [x] Forms show prior fields while space permits; focus/errors remain visible.
- [x] Long values remain scrollable; no prompt overflow creates blank rows.
- [x] Review is readable and editable at all target sizes.
- [x] Native approvals, secrets and exact target revalidation remain authoritative.

## Comments

- 2026-11-08: Capture evidence establishes excessive blank rows and lost focus context.
- 2026-11-08: Completed at `280ccebb61d539590c55c019595d4e2029c5d4be`.
  Long-value access, resized overlay focus, draft/caret restoration and native
  approval boundaries verified. [Preview](../preview.md),
  [review](../review-resolutions.md), [verification](../verification.md).
