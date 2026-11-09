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
- 2026-11-09: Implemented and verified source
  `b292eb1cdfb017d7c86081ea9ee60f7a708da284`. Full Go tests, vet and native
  Windows dev build/help checks passed. Independent standards review and
  separate specification review found no unresolved findings. Verification
  environment, matching source inputs and limits are in `docs/verification.md`.
  Ready for review on the local feature branch; no push, merge or new tag.
- 2026-11-10: The user accepted updating the branch from current `main`,
  resolving conflicts, testing, and merging through a PR after CI passes.
  Integrated `87f58ba68afe1c42827f3205ebe25e71be8ffc0f` in merge commit
  `9db8806143c6f2d3880853573d52fa381cc0187f`, preserving v0.7.0 removal and
  dissociation. Uncached full Go tests, vet, native Windows dev/help smoke,
  formatting, and separate standards/specification reviews passed. See
  `docs/verification.md` for conditions, evidence and limits. Publication of a
  new version is outside this integration task.
- 2026-11-10: Portable packaging (5), installers (50), and installer profiles
  (30) also passed on the integrated source. Hosted three-OS tests and
  six-target packaging remain the final merge gate.
