# Remove keys, destinations, and associations

Objective: let users remove managed keys and destinations, and disconnect an
existing key–destination association through both CLI and TUI.

Scope: local management metadata only. Preserve public/private key files, SSH
configuration, remote authorized keys, creation results, and legacy history.
No new dependencies, remote operations, physical key deletion, push, or release.

Acceptance criteria:

- `keys remove --key ID|NAME` removes the active key and all its associations.
- `hosts remove --host ID|ALIAS` removes the active destination and all its associations.
- `hosts dissociate --host ID|ALIAS --key ID|NAME` removes only that association.
- TUI `d` removes the visible key/destination; `u` selects an existing association
  from either category. Empty/unrelated selections do not start an operation.
- All changes use existing terminal confirmation, locking, validation, atomic
  saves, backups, and saved reports. Decline/errors before save preserve state.
- Missing files do not prevent metadata cleanup. Ambiguous/missing identities
  and nonexistent associations are rejected. Pending legacy key deletion stays
  quarantined for manual review.
- Removed identities are retained in `removed_keys` / `removed_hosts` for
  referential integrity. They are absent from active lists and cannot be used.
  Reimport and destination alias reuse allocate fresh IDs.
- Tests cover selective/cascaded removal, failure and cancellation paths,
  history retention, generated keys, identity reuse, CLI validation, and TUI routing.

Dependencies: state/history rules → common app operations → CLI/TUI → tests/docs.
Go verification and documentation review can run independently.

Open questions: none. “Remove key” means remove from management, consistent with
the existing file preservation contract; physical file deletion stays manual.

Base: refreshed `origin/main` c226ffb5302fed539f36aa7400d8732ddd55e935.
Existing branch `feat/06-guide-platform-selection` is separate and preserved.
