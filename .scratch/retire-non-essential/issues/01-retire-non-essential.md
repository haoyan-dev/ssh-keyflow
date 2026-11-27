# 01 — Retire hosts, guide, pinning, TUI and public-only import

Status: ready-for-human
Blocked by: none

Implement spec.md: reduce the CLI to key-pair management, move state to schema
v3 with an archived migration, remove the TUI and its dependencies, and update
the three-language documentation.

## Acceptance

See spec.md "Acceptance criteria".

## Comments

- 2026-11-28: Code retirement done on `refactor/retire-non-essential`
  (uncommitted). `go vet` clean; `go test ./...` passes; Python release (17),
  installer (50), installer profile (30) and portable (5) tests pass.
- 2026-11-28: Notices pruned to Go + 5 shipped modules (union of 6 targets via
  `go list -deps`); Unicode license kept for Go's `unicode` tables. Japanese
  docs, ADR 0007 and statuses of ADR 0001–0006 updated; `terminal-ui.md`
  removed in all languages. en / zh-CN translation in progress.
- 2026-11-28: en / zh-CN translations done. The standards and spec reviews ran
  separately, and all findings are resolved. Each changing command now shows
  the migration notice before its confirmation. The dropped list is complete,
  including unfinished SSH config edits. Legacy decoding is strict again, a
  retry reuses an identical archive, and tests cover all of this. The docs
  are updated in all three languages. All Go, Python and docs checks pass on
  base `666d1ea` plus the uncommitted tree; see `docs/verification.md`.
  Remaining: commit, PR and merge, and the v0.11.0 release. Each needs the
  user's approval.
