# 01 — Shared panels and browsable pages

Status: ready-for-human
Blocked by: none

Implement cell-aware panels and selection geometry for lists, pickers and
empty states. Preserve references, search and exact-target paging.

## Acceptance

Follow spec.md. Verify panel widths, CJK/ANSI, three supported sizes and
PageDown/selection agreement.

## Comments

- 2026-11-08: User selected A. Implementation started on feat/tui-grouped-panels.
- 2026-11-08: Completed in `b27b6ac` and `fbb2c37`. Cell-aware borders,
  selection geometry, counts, paging and distinct references verified at all
  three supported sizes. Review findings resolved; see
  [verification](../verification.md).
