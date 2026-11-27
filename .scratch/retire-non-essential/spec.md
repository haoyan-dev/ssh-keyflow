# Retire non-essential functions (v0.11.0)

Decision record: `docs/adr/0007-key-pairs-only.md`.
Base: `origin/main` `666d1ea`. Branch: `refactor/retire-non-essential`.

## Objective

Keep only the functions that serve the user's real requirement: SSH key pairs on
this machine, managed locally, with only public keys leaving it. Remove features
built for per-destination copy-paste setup, which a follow-up multi-machine
design (separate work, recorded in its own ADR when decided) will replace.

## Decisions (confirmed by the user, 2026-11-28)

1. Retire every `hosts` command (add / list / remove / associate / dissociate),
   the copy-paste guide including the ssh-agent step, and host key pin / unpin.
2. Drop the TUI; the tool is CLI only. No-argument launch prints help.
3. Clean break to schema v3. On the first save, copy the old `state.json` to a
   new `state.v<N>-<UTC>.json` archive, then save only key pairs and unfinished
   creations.
4. Keep pair import only; retire public-only import.

## Scope

In: `internal/cli`, `internal/app`, `internal/state`, `cmd/ssh-keyflow`,
`go.mod`/`go.sum`, third-party notices, `scripts/test_portable.py`,
`scripts/check_docs.py`, the contract, glossary, README, guides, tutorial and
ADRs in ja / en / zh-CN.

Out: installers, release workflow, packaging, historical records
(`docs/verification.md` entries, old ADR bodies, `.scratch/` history), the fleet
design itself, and any release (tagging is a separate, explicitly authorized step).

## Acceptance criteria

- CLI: `keys list [--json]`, `keys public`, `keys create`, `keys import`
  (`--name`, `--public`, `--private` all required), `keys remove`, `resume`.
  Retired commands and flags return usage errors (exit 2) without side effects.
- `keys list --json` returns exactly `{"creations":[],"keys":[]}`-shaped output.
- v3 is strict; v1 / v2 load read-only and stay strict (unknown fields, wrong
  types and trailing content are rejected). A notice listing every non-empty
  record type that will not carry over (and each unfinished SSH config edit)
  goes to stderr in `keys list` and before the confirmation of every changing
  command. The first save writes the archive with O_EXCL, refuses to change
  state if archiving fails, and a retry reuses an identical archive.
- No TUI dependencies remain; notices cover exactly the shipped modules of all
  six targets.
- `go vet`, `go test ./...`, release / installer / portable Python tests, and
  `scripts/check_docs.py` / `scripts/test_docs.py` pass.
- Three-language docs describe only the retained functions and the migration.

## Version

0.x breaking change → `v0.11.0` (previous release `v0.10.0`). Release is pending
explicit user authorization.
