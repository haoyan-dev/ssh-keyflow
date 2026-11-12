# 01 — Pin file storage and host key parser

Status: ready-for-human
Blocked by: none

Add the app-owned pin file `<Home>/.ssh/ssh-keyflow/known_hosts` and the parser
for pasted host key output, as in spec.md "Storage" and "Parsing pasted output".

## Acceptance

- Strict decoding: three single-space fields, prefix, clean ID, canonical
  base64 whose inner algorithm matches the type, no duplicates, LF only.
  Malformed files are refused and never rewritten.
- Writes hold the state lock, use atomic replace and symlink refusal, and keep
  other destinations' lines in order.
- Parser: allowed types only, 64 KiB, 16 keys, duplicates removed, ignored
  lines reported sanitized, only parsed fields written.

## Comments

- 2026-11-12: Implemented in `internal/state/pins.go` and `internal/app/pins.go`
  with tests in `internal/state/pins_test.go` and `internal/app/pins_test.go`.

- 2026-11-12: Integrated by PR #33 as `dc7c84d`; required CI and both reviews passed. Release tracked in `.scratch/release-v0.10.0/`.
