# 04 — TUI pin and unpin

Status: ready-for-human
Blocked by: 02

Add `t` / `T` on destinations, handing off to the native terminal, and the
pinned status in destination details, as in spec.md "TUI".

## Acceptance

- `t` / `T` on a key only sets a status message.
- `t` hands off `pin-host` with `Replace`; paste and confirmation are read from
  the native input and can be cancelled like other handoffs.
- Details show `Host keys: pinned (N)` / `not pinned` / `unknown`; footers and
  help list the keys.

## Comments

- 2026-11-12: `k` / `K` collided with list navigation, so the binding is
  `t` / `T`. Implemented in `internal/tui`; tests in `internal/tui/pins_test.go`.

- 2026-11-12: Integrated by PR #33 as `dc7c84d`; required CI and both reviews passed. Release tracked in `.scratch/release-v0.10.0/`.
