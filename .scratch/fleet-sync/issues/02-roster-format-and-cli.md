# 02 — Roster format and phase-1 CLI surface

Status: ready-for-human
Blocked by: 01

Criteria 2 and 7. Also waits for the ADR 0008 gate in the spec. Specify the
roster JSON, its canonical byte form, the version and time fields, the common
account and per-node fields, and strict validation. The command names are decided
(spec, user decisions); settle the remaining flags and the automation-facing JSON before
code, since both become public contract.

## Comments

- 2026-11-28: User accepted ADR 0008, the client role name and the proposed
  command names.
- 2026-11-28: Ticket 01 merged (PR #43). Draft in `roster-format.md`. The user
  decided: add `node export` and `fleet add` for adding nodes without `join`;
  `--max-age <days>` is a per-run flag with a 30-day default; minimum OpenSSH
  8.4; non-canonical roster bytes are rejected, never normalized.
- 2026-11-28: Revised after the standards and spec reviews (canonical layout
  fully defined, confirmation after all acceptance checks, warnings on stderr
  only, exit codes, file locations). Human step: review `roster-format.md`,
  especially the choices not yet confirmed (DNS-label node names, account
  name pattern, one port per remote, 1000-node and 16-key limits), and merge.
