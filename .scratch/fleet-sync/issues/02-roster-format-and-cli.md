# 02 — Roster format and phase-1 CLI surface

Status: needs-triage
Blocked by: 01

Criteria 2 and 7. Also waits for the ADR 0008 gate in the spec. Specify the
roster JSON, its canonical byte form, the version and time fields, the common
account and per-node fields, and strict validation. The command names are decided
(spec, user decisions); settle the remaining flags and the automation-facing JSON before
code, since both become public contract.

## Comments

- 2026-11-28: User accepted ADR 0008, the client role name and the proposed
  command names.
