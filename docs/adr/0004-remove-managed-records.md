# ADR 0004: Remove local management metadata while retaining history

- Status: accepted
- Date: 2026-11-09
- Specification: `.scratch/remove-managed-records/spec.md`

Users need to remove managed keys and destinations and disconnect individual
associations. ADR 0001 excludes physical key deletion and automatic remote
revocation; those boundaries continue to apply.

Add `remove-key`, `remove-host`, and `dissociate` to the common backend and expose
them through CLI and TUI. Removal clears all active associations for the selected
record. Dissociation clears only the selected pair. Each operation requires the
existing explicit confirmation and runs under the state lock with atomic saving
and backup.

Move removed key/destination metadata into optional v2 `removed_keys` and
`removed_hosts` arrays. Historical IDs remain unique and available to validate
creation results, registrations, onboarding, and configuration history. Archived
fingerprints, file paths, and destination aliases do not reserve active names or
files, so users may reimport or add the destination again with a fresh ID.
Associations may reference only active records. A pending legacy key deletion
continues to require manual review before removing that key.

Metadata cleanup does not read key files or require OpenSSH. Public/private key
files, local configuration, remote authorized keys, and legacy records remain
unchanged. Removing metadata does not revoke SSH access. The archive stores only
the metadata already held in state, never private key contents.

The optional archive fields extend v2. Older binaries reject a state containing
them rather than silently losing history. Keep a separate state backup before
downgrading; `state.json.bak` remains a rolling backup.
