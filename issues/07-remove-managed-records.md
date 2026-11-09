Status: ready-for-human

# 07: Remove managed keys, destinations, and associations

Specification: [.scratch/remove-managed-records/spec.md](../.scratch/remove-managed-records/spec.md)

Blocked by: none

Implement the three confirmed metadata operations across the common backend,
CLI, and TUI. Preserve files and legacy history; verify selective removal,
cancellation, failure paths, identity reuse, and existing behavior.

## Comments

- 2026-11-09: Started on `feat/07-remove-managed-records` in an isolated worktree
  from refreshed `origin/main` c226ffb5302fed539f36aa7400d8732ddd55e935.
- Removal archives identities rather than deleting historical references.
- 2026-11-09: Implemented and reviewed at
  `fa9442bbd74a30abc4797abb4abd13cdb9f597fa`. CLI and TUI support confirmed
  key/destination removal and selective association removal, preserving files
  and legacy history. Standards and specification reviews have no unresolved
  findings.
- Verification: Go tests and vet, 5 portable tests, 50 installer tests,
  30 installer-profile tests, six-target packaging, archive/checksum validation,
  and Windows amd64 version/help smoke checks passed. See
  [verification record](../docs/verification.md#2026-11-09-managed-keydestination-removal-and-dissociation)
  for environment, source SHA, and verification limits.
- Ready for human review. Changes are local; no push, merge, or release.
