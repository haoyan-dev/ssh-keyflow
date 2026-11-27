# SSH KeyFlow — Fleet sync design (proposal)

Status: ready-for-human (proposal; the decision is ADR 0008, also a proposal).
Date: 2026-11-28. Written against 666d1ea (v0.10.0); refreshed against
`origin/main` 1938fa9 (v0.11.0, ADR 0007).

ADR 0007 (v0.11.0) retired ADR 0002, 0003, 0005 and 0006, partly updated 0001
and 0004, and narrowed the tool to local key pairs. ADR 0008 also reverses
ADR 0001's exclusion of remote registration and SSH config/Include ownership,
and ADR 0004's exclusion of automatic remote revocation. It names this fleet direction as follow-up work to be recorded in its own
ADR. The proposal conflicts with these parts of ADR 0007, which the new ADR must
supersede explicitly:
- "key pairs only" scope: no network, no other nodes;
- the existing `~/.ssh/ssh-keyflow/known_hosts`, SSH config and Includes are
  never edited;
- state schema v3 holds only `keys` and `creations`, so node, roster and
  fleet-root data need schema v4 and a migration.

## 1. Requirements (as stated by the user)

1. Many PCs, VMs and VPS servers.
2. Setting up SSH on every new environment is repetitive and has many steps.
3. A tool keeps key pairs aligned across those environments.
4. Environment types: host + remote, host-only, remote-only.
5. Only public keys may travel over the Internet.
6. The tool manages local keys.
7. On a new environment the only manual step is a shell one-liner that downloads
   the tool; everything else happens inside the tool.
8. P2P or self-hosted server are both acceptable; must be secure but convenient.

Terminology in this document (the user's original terms). ADR 0008 and the
glossary rename the "host" role to **client** (接続元); "remote" and "node" stay.

- **Client** (the user's "host"): a machine you SSH *from* (needs a user key pair).
- **Remote**: a machine you SSH *into* (runs sshd, needs `authorized_keys`).
- **Node**: any enrolled machine; its roles are `client`, `remote`, or both.

> Terminology: v0.11.0 retired the `hosts` commands, and the glossary no longer
> defines "Host". OpenSSH still uses "host" for the destination (`Host` blocks,
> host keys, `known_hosts`), so "host = machine you SSH from" will collide in docs
> and output. Settle the terms in GLOSSARY.md (all three locales) before code.

## 2. The real security problem

Public keys are not secret, so requirement 5 is easy. The hard property is
**integrity and authenticity**: if an attacker can swap a public key in
transit or on the server, they either gain login (fake user key in
`authorized_keys`) or can MITM you (fake host key in `known_hosts`).
So the design needs a trust anchor that every node pins once, and every
key distributed afterwards must be verifiable against it.

## 3. Research summary (primary sources)

| Option | What it gives | Limits for this use case |
| --- | --- | --- |
| OpenSSH user certificates (`ssh-keygen -s`, `TrustedUserCAKeys`) | One CA key trusted by every remote; new clients need no change on remotes | Revocation needs `RevokedKeys` (KRL) in `sshd_config` = root. Without root, only expiry works, so certs must be short-lived and the CA must be online to renew. |
| `cert-authority,principals=` line in `~/.ssh/authorized_keys` | Per-user CA trust **without root** (sshd(8)) | Same revocation gap: no user-level KRL. |
| OpenSSH host certificates / `@cert-authority` in known_hosts | Clients trust every remote via one line | Remote needs `HostCertificate` in `sshd_config` (root). |
| `ssh-keygen -Y sign/verify`, `allowed_signers` | Standard, auditable signatures with SSH keys and namespaces | Only a primitive; we still need a roster format. |
| magic-wormhole (SPAKE2 PAKE) | Short one-time human code → strong mutually authenticated channel; attacker (incl. malicious mailbox server) gets one guess per code; mailbox and transit relay are self-hostable | Both sides must be online at the same moment. |
| Tailscale SSH | No key management at all; WireGuard identity + ACL | Replaces SSH keys with a third-party/IdP identity; **SSH server is Linux/macOS only** (no Windows remotes); port 22 only. Headscale exists for self-hosting. |
| GitHub `https://github.com/<user>.keys` | Zero-infrastructure public key distribution | User keys only (no host keys), trust = GitHub account security, no roles. |

Sources: sshd(8) and ssh-keygen(1) at man.openbsd.org;
magic-wormhole.readthedocs.io (welcome/design); tailscale.com/kb/1193/tailscale-ssh.

## 4. Proposed architecture: signed fleet roster

```
            (approve, sign)                     untrusted, dumb
 [admin node] ── roster vN + sig ──────▶ [relay: HTTPS / static store]
      ▲                                         │  pull + verify sig
      │ PAKE pairing (short code)               ▼
 [new node] ◀───────────── roster vN ─── [every node: render local SSH files]
```

### 4.1 Keys

- Each node generates its own keys locally. Private keys never leave the node.
  - `client` role: Ed25519 user key under the tool's directory (optionally
    `ed25519-sk` FIDO or OS keystore later).
  - `remote` role: the tool *reads* the existing sshd host public keys; it does
    not create them.
- **Fleet root key**: an Ed25519 signing key held on one or more *admin nodes*,
  passphrase-protected (FIDO optional). Its public key is the only trust anchor,
  pinned by every node at enrollment. Recommend two admin keys (daily + offline
  backup) so losing one device is not fatal.

### 4.2 Roster

A versioned document listing nodes: id, name, roles, login account(s),
addresses, user public keys, host public keys, and which clients may log into
which remotes (default: all clients → all remotes for the owner's common account,
with a per-remote account override).
It is signed by a fleet root key (`ssh-keygen -Y` compatible signature,
namespace `ssh-keyflow-roster`). Nodes reject:

- bad or unknown signatures,
- a version lower than the highest one they have already seen (rollback),
- a roster older than a configurable maximum age (freeze attack), reported as a warning.

### 4.3 Applying the roster on a node (no root required)

- **remote**: a delimited, tool-owned block in `~/.ssh/authorized_keys`
  (Windows: the user file, or `administrators_authorized_keys` for admin
  accounts, with the ACL handling already researched for the guides). Lines
  outside the block are never touched. With root, `AuthorizedKeysCommand`
  can be offered as an option instead of file edits.
- **client**: an app-owned known_hosts file and an app-owned SSH config included
  once from `~/.ssh/config`, providing `ssh <node-name>` aliases with
  `HostKeyAlias` and `StrictHostKeyChecking yes`. v0.11.0 removed the old pin
  code. Users may still have `~/.ssh/ssh-keyflow/known_hosts` and
  `~/.ssh/ssh-keyflow/config` from v0.10.0 guides, so use new file names (or an
  explicit adoption step) rather than silently taking over those files.
- Every apply shows or logs a diff, is atomic, and supports `--dry-run`.

### 4.4 Enrollment of a new environment

```sh
curl -fsSL https://…/install.sh | sh     # verifies SHA256SUMS (signature: see §6)
ssh-keyflow join --relay https://keys.example.net
#  → asks roles; generates keys; prints code "7-crossbow-oxygen"
```

On any admin node: `ssh-keyflow approve 7-crossbow-oxygen`.
The two run a SPAKE2 exchange through the relay's mailbox. The new node sends
its public keys; the admin sends the fleet root public key and relay URL.
The admin shows the new node's fingerprints and roles, then signs roster vN+1.
The new node verifies it, applies it, and installs a periodic `sync`
(systemd user timer / launchd / Task Scheduler) or relies on `sync` at login.

Fallback without a relay: the same exchange as a file or QR
(`join --offline`), confirmed by comparing fingerprints.

### 4.5 Transport choice

Recommend a **self-hosted relay as the default** and P2P as an optional add-on:

- VMs and laptops are often offline; a store-and-forward relay lets every node
  catch up whenever it wakes. Pure P2P needs both peers online and NAT
  traversal.
- The relay is **untrusted**: it stores signed blobs and runs the PAKE
  mailbox. It can withhold or delay updates but cannot forge keys. It can
  reject writes not signed by the fleet root (anti-spam).
- Ship it as `ssh-keyflow relay` (one binary on one of your VPS) and also support
  a static backend (any HTTPS/S3/git URL) for read-only distribution.
- Over Tailscale/LAN, nodes can additionally pull the roster from each other.

### 4.6 Revocation and rotation

- Revoke a node: admin removes it → new signed roster → each remote drops its
  key on next sync. Latency = sync interval (show "last synced" per node).
- Key rotation: node generates a new key and requests approval like a join.
- Fleet root rotation: old root signs a roster that introduces the new root.

## 5. Why not certificates as the core

Certificates scale better, but without root you cannot use KRLs, so
revocation depends on short expiry. Short expiry means the CA private key must
be online to renew, and that conflicts with keeping the most sensitive key
offline. The signed roster keeps the signing key offline and revocation explicit.
It also works the same on Windows. A CA mode can be added later for
root-managed Linux remotes and carried in the same roster.

## 6. Residual risks / open items

- Fleet root compromise = whole fleet. Mitigations: passphrase/FIDO, backup key,
  rotation procedure, approval requires showing fingerprints.
- Installer trust: `curl | sh` trusts GitHub + TLS. Add signed checksums
  (minisign or GitHub artifact attestations) and have the installer verify them.
- Remote-only nodes still need sshd installed; on Windows this needs admin.
  The tool should detect and guide this.
- The tool now modifies `authorized_keys`, which ADR 0007 still leaves to the
  user ("manual public key placement"). A bug there can lock you out: keep one
  break-glass key outside the managed block.

## 7. Comparison with the current design (v0.11.0)

| Aspect | Current (ADR 0007) | Proposed |
| --- | --- | --- |
| Scope | Local key-pair manager | Fleet-wide key alignment |
| Network | None | Relay sync + PAKE pairing |
| Tool on remotes | Not involved | Required on every node |
| `authorized_keys` | User appends `keys public` output | Tool-owned managed block |
| known_hosts | Not touched | App-owned file, filled from roster |
| SSH config | Not touched | App-owned Include file |
| Trust | User's eyes per step | Fleet root signature + PAKE |
| Revocation | Manual per remote | One roster change, propagates |
| Effort per new env | N steps × M machines | install one-liner + `join` + approve |
| Attack surface | Minimal | Relay, daemon/timer, signing key |

Reusable as-is: Ed25519 creation via native ssh-keygen, the state store (lock,
atomic save, backup, archived schema migration as done for v3), one-line
installers and the release pipeline. Reusable from git history (v0.10.0, retired
in v0.11.0): the `HostKeyAlias` pinning approach and the Windows
`authorized_keys` path/ACL knowledge from the guides. The TUI and copy-paste guide
are gone; for machines that cannot run the tool (NAS, routers, locked-down
servers) the fallback is manual placement of `keys public` output.

## 8. Recommendation

Evolve the existing codebase into the fleet design, but don't rewrite it. The current
design does not meet requirements 2, 3 and 7: every new environment still costs
manual steps on every other machine. The local-safety foundations it built are
exactly what the fleet design needs. Record ADR 0008 superseding the
"key pairs only / never edit known_hosts or SSH config" parts of ADR 0007 and
the matching exclusions in ADR 0001 and 0004, and
settle the node/role terms in the glossary.

Phased delivery (each phase usable on its own):

1. Node identity, roles, fleet root key, roster format, sign/verify;
   transport by file (`export` / `import`). This proves the trust model offline.
2. `apply`: managed authorized_keys block, known_hosts, config Include, with dry-run.
3. `join` / `approve` with SPAKE2 pairing.
4. `ssh-keyflow relay` + static backend + scheduled `sync`.
5. Revocation UX, rotation, admin backup key, signed installer checksums.

Questions to decide before phase 1:

- Do you have root on most remotes? (If yes, a CA mode becomes attractive later.)
- Are any remotes Windows? (Rules out Tailscale SSH as the alternative.)
- Same login account on all remotes, or per-node accounts?
- Which VPS will host the relay, and is a 15-minute revocation latency acceptable?
- Would you rather adopt Tailscale SSH / Headscale and drop key management for
  Linux/macOS remotes entirely? (Buy-vs-build check.)
