Status: ready-for-agent

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
  Acceptance and verification results will be recorded before handoff.
