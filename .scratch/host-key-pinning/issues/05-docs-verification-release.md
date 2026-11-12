# 05 — Documentation, verification and v0.10.0 release

Status: ready-for-human
Blocked by: 02, 03, 04

Accept ADR 0006, update the application contract, guides, tutorial, README and
glossary, record verification, then release v0.10.0 (MINOR).

## Acceptance

- ADR 0006 accepted and listed in `docs/adr/README.md`.
- `docs/verification.md` records the commit, environments, spike results and
  limits (macOS unverified).
- Release follows AGENTS.md "Tags and releases" and needs explicit
  authorization after the PR is integrated.

## Comments

- 2026-11-12: Documentation and verification done on `feat/host-key-pinning`.
  Release not started: awaiting integration and release authorization.

- 2026-11-12: Integrated by PR #33 as `dc7c84d`; required CI and both reviews passed. Release tracked in `.scratch/release-v0.10.0/`.
