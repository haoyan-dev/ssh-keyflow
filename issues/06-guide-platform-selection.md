Status: ready-for-human
Blocked by: none

# 06 Explicit guide platforms

## Objective

Let users indicate both the local and destination platforms when preparing
copy-paste scripts, including a terminal whose OS differs from the running app.

## Scope

Shared guide options, CLI flags, TUI guide form, regression tests and domain docs.
Specification: `.scratch/guide-platform-selection/spec.md`.

## Acceptance criteria

- [x] Local and destination platforms are visible and editable in the TUI.
- [x] CLI accepts equivalent guide-only overrides; defaults remain compatible.
- [x] Cross-platform local setup requires the private-key path in that terminal.
- [x] Invalid platforms/account/path inputs produce no copyable scripts.
- [x] Saved state, per-step copying and refresh invalidation remain intact.
- [x] Applicable checks, separate standards/spec reviews and verification records pass.

## Comments

- 2026-11-09: Refreshed origin/main; base
  `c226ffb5302fed539f36aa7400d8732ddd55e935`. The current local script uses
  runtime.GOOS; destination scripts use the saved host OS/account. Implement on
  `feat/06-guide-platform-selection`. No new release is requested.
