# Fleet sync, phase 1: signed roster offline

Status: ready-for-agent
Decision: ADR 0008 (`docs/adr/0008-signed-fleet-roster.md`, accepted 2026-11-28); design
in `design.md`.
Base: `origin/main` 1938fa9 (v0.11.0).

## Decisions from the user (2026-11-28)

- Build the signed-roster design into SSH KeyFlow; do not adopt Tailscale SSH or
  Headscale.
- Some remotes are Windows, and the user has root/admin on most remotes. A CA
  mode stays a possible later add-on for root-managed remotes, outside phase 1.
- Mostly the same login account on all remotes: the roster default is that every
  client may log into one common account on every remote, with a per-remote
  account override.
- Start with ADR 0008, glossary terms and this spec on a docs branch.
- Accept ADR 0008; rename the user's "host" role to client (接続元); adopt
  the phase-1 CLI names `fleet init`, `fleet show`, `fleet export --out <dir>`,
  `fleet accept <dir>` and `node set --roles client,remote`.

## Gates

- No product code before the PR recording ADR 0008 is merged. Ticket 01 is a
  research spike with throwaway scripts only, so it may run earlier.
- Phases 2–5 in ADR 0008 are direction, not commitments; each gets its own spec.

## Objective and scope

Prove the trust model with no network and no SSH file edits. A node can create a
fleet, describe itself, and pass a signed roster to another node as files; the
receiving node verifies it against its pinned fleet root keys.

In scope: node identity and roles; fleet root key creation; roster format;
signing and verification with native `ssh-keygen -Y`; rollback and age checks;
file export and acceptance; state schema v4 with an archived migration from v3.

Out of scope (later phases): writing `authorized_keys`, known_hosts or SSH
config (`apply`); `join`/`approve` and SPAKE2; the relay and `sync`; revocation
UX, key rotation, the backup root key, signed installer checksums; CA mode.

## Acceptance criteria

1. `ssh-keygen -Y sign/verify` works with the OpenSSH versions shipped on
   Windows 11, current macOS and the supported Linux runners. The minimum
   version, passphrase handling without a TTY or through ssh-agent, and any
   platform gaps are recorded before the format is fixed.
2. The roster is a documented JSON document with a canonical byte form,
   a monotonically increasing version, a creation time, the fleet's common login
   account, and per-node id, name, roles (`client`/`remote`), optional login
   account override, addresses, client public keys and host public keys.
   Unknown fields and bad values are rejected.
3. An admin node creates a passphrase-protected Ed25519 fleet root key with
   native ssh-keygen and signs rosters under namespace `ssh-keyflow-roster`.
   The private key never appears in state or output.
4. Accepting a roster verifies the signature against the pinned root keys. It
   rejects unknown or bad signatures, versions lower than the highest seen, and
   a version equal to one already seen with different content. It warns when the
   roster is older than the configured maximum age. The first acceptance pins the
   root only after the user confirms its fingerprint.
5. A client node lists one of its managed keys. A remote node reads its sshd host
   public keys without changing them, from the `HostKey` entries in
   `sshd_config` with the platform defaults as fallback.
6. State moves to schema v4, with the v3 file written to a migration archive
   first, following the v3 migration rules (exclusive archive, no change on
   read, strict parsing).
7. Phase 1 writes no SSH file: only the state, the fleet root key and the
   roster files the user exports, all under new names in `~/.ssh/ssh-keyflow/`
   or a user-given output directory. v0.10.0's `known_hosts` and `config` there
   are not read or reused. Commands that change state support confirmation and
   `--dry-run`.
8. Tests cover the format, signature verification, rollback/age rejection and
   migration on Linux, macOS and Windows CI. The contract and ja/en/zh-CN docs
   describe the new commands, JSON and state when phase 1 is released (MINOR).

## Open questions

- CLI flags beyond the decided command names ("import" stays reserved for
  key-pair import). Settle in ticket 02.
- Default maximum roster age for the freeze warning; propose 30 days for offline
  phase 1.
- How the admin node adds other nodes in phase 1 without `join`: proposal is
  accepting a node's exported public description file after confirming its
  fingerprints.
- For phase 4 (not phase 1): which VPS hosts the relay, and what revocation
  latency (sync interval) is acceptable.

## Dependencies

01 OpenSSH `-Y` spike → 02 roster format and CLI → 03 state v4 and node identity
→ 04 fleet root and signing → 05 file export, acceptance and verification.
02 also waits for the ADR 0008 gate above.
