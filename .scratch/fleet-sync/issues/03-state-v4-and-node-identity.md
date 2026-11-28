# 03 — State schema v4 and node identity

Status: ready-for-human
Blocked by: 02

Criteria 5 and 6. Add the node record (id, name, roles, login account override,
addresses) and fleet data to state v4, with an archived migration from v3
following the v3 rules. A client lists a managed key. A remote reads its sshd
host public keys read-only, from the `HostKey` entries in `sshd_config`, falling
back to the platform defaults (`/etc/ssh`, Windows `%ProgramData%\ssh`).

## Comments

2026-11-28 — Implemented on `feat/state-v4-node-identity` (base origin/main
1ea5aaa), uncommitted pending review.

- Scope: state schema v4 (`node`, `fleet` with roots, root key and accepted
  roster), the read-only v3 → v4 conversion with archive on first save,
  `node set` / `node show [--json]`, and `keys remove` refusing the node's
  client key. Field rules come from roster-format.md and live in
  `internal/roster`; sshd host-key reading is `internal/sshd`.
- Left to later tickets: `node export` and `fleet add` (05); `fleet init`,
  root key creation and signing (04). The `fleet` fields are validated but no
  command writes them yet.
- v0.11.0 and earlier cannot read v4; the migration notice says so and the v3
  archive is the downgrade path.
- Glossary updated in all three locales (pin, node description, node
  description file, client accepts imported keys, archive covers v3).
- Application contract and locale docs are deferred to the phase-1 release
  (spec criterion 8).
- State-only fields beyond the roster (layout left to this ticket by
  roster-format 6): `node.sshd_config`, `node.updated_at`,
  `fleet.roots[].pinned_at`, `fleet.root_key` (fingerprint and paths, never
  the private key) and `fleet.accepted.accepted_at`.
- Behaviour the contract must document at the phase-1 release: `keys remove`
  refuses the node's client key with exit 1; `node set --dry-run` prints the
  migration notice on stderr, while a confirmed `node set` prints it before
  the prompt as the other changing commands do.
- Left for 04/05: drop `login_account` when it equals the fleet's common
  account (needs a fleet; do it at signing), and check that
  `fleet.accepted.version` and `fleet.id` match the stored roster's `version`
  and `fleet_id` once the roster parser exists.
- Review (standards and spec, separate): fixed — pinned roots must be
  Ed25519; `--roles`/`--addresses` take exactly one comma-separated value;
  IPv4-mapped IPv6 is allowed, as 1.5 allows any RFC 5952 canonical literal;
  `node show` reports an unreadable key instead of hiding it; the dry-run
  notice goes to stderr; the unused `sshd.DefaultConfig` was removed. Not
  changed: remote-field checks stay in both `app` (usage errors) and `state`
  (integrity); sshd forbids `HostKey` in `Match`, so stopping the scan there
  is intended.
- Verified: `go vet ./...` and `go test -count=1 ./...` pass on Windows
  (uncommitted tree on 1ea5aaa). Linux and macOS run in CI once pushed.
