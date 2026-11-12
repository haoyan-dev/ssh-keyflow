# 02 — `hosts pin` / `hosts unpin` and list status

Status: ready-for-human
Blocked by: 01

Add the `pin-host` / `unpin-host` operations and CLI commands, and pinned
status in `hosts list`, as in spec.md "CLI".

## Acceptance

- Terminal paste flow, `--file`, stdin with `--yes`; fingerprints shown before
  a typed `yes`; identical re-pin writes nothing.
- Different pinned keys need `--replace` and typed `yes` in a real terminal;
  `--yes` never replaces.
- `hosts unpin` needs a real terminal, like `hosts remove`.
- `hosts list` shows `HOST KEYS`; JSON adds `host_key_pins`, omitted with a
  stderr reason when the pin file is unreadable.
- Removing a pinned destination says to unpin first and keeps the pin file.

## Comments

- 2026-11-12: Implemented in `internal/app/pins.go`, `internal/app/removal.go`
  and `internal/cli/cli.go`; tests in `internal/cli/pins_test.go`. `unpin
  --yes` was dropped during docs review to keep the no-unconditional-approval
  contract.
