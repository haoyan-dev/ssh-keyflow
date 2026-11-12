# 03 — Guide steps and pinned / public-only snippets

Status: ready-for-human
Blocked by: 01

Add the print and pin steps for unpinned destinations, the pinning lines for
pinned ones, and the public-only key snippet, as in spec.md "Guide flow" and
"Snippet changes".

## Acceptance

- Steps and lines only when the guide's local OS is this computer.
- Windows print step requires Run as Administrator.
- Unpinned snippets are unchanged; unreadable pin file shows `unknown` and
  adds nothing.
- Public-only keys get `IdentityFile "<public_path>"`, `IdentitiesOnly yes`,
  agent guidance, no `AddKeysToAgent` or agent step; `--identity-file` still
  overrides; another local OS without it keeps placement only.
- Native `ssh -G` parses the pinned snippet.

## Comments

- 2026-11-12: Implemented in `internal/app/guide.go` and `internal/app/pins.go`;
  existing guide tests updated for the new steps.
