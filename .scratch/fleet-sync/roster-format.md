# Fleet sync phase 1: roster format and CLI surface (ticket 02)

Status: decided 2026-11-28 (the user's answers A–D below); revised after the
standards and spec reviews. Base: `origin/main` 33b1e5c.
Inputs: `spec.md` (criteria 2 and 7, open questions), ADR 0008,
`openssh-spike.md` (ticket 01), `docs/application-contract.md` (v0.11.0
conventions this draft keeps).

This becomes public contract when phase 1 ships (MINOR). Until then it is the
target for tickets 03–05. Items marked **Decided** record the user's answers.
JSON shown with `…` is schematic; the byte layout is defined only by 1.3.

## 1. Roster document

### 1.1 Files

A roster travels as three files in one directory:

| File | Content |
| --- | --- |
| `roster.json` | The roster, in canonical bytes (1.3) |
| `roster.json.sig` | `ssh-keygen -Y sign -n ssh-keyflow-roster` signature over those bytes |
| `fleet-root.pub` | The signing fleet root public key, one line `<type> <base64>`, no comment |

`fleet-root.pub` is a convenience for the first acceptance only. Trust comes
from the fingerprint the user confirms, then from the pinned fleet root keys in
state; a later `fleet-root.pub` that equals none of them is rejected.

### 1.2 Fields

Example (canonical layout, placeholder values):

```json
{
  "format": "ssh-keyflow-roster",
  "schema": 1,
  "fleet_id": "fleet-0f3c9a7e5b2d4c1f8e6a0b9d7c5e3f21",
  "version": 7,
  "created_at": "2026-11-28T09:15:00Z",
  "login_account": "user",
  "nodes": [
    {
      "id": "node-5a1e0c3b7d9f4e2a8b6c1d0e9f7a3b5c",
      "name": "laptop",
      "roles": [
        "client",
        "remote"
      ],
      "login_account": "admin",
      "addresses": [
        "192.0.2.10",
        "laptop.lan"
      ],
      "port": 2222,
      "client_keys": [
        "ssh-ed25519 AAAA…"
      ],
      "host_keys": [
        "ecdsa-sha2-nistp256 AAAA…",
        "ssh-ed25519 AAAA…"
      ]
    }
  ]
}
```

| Field | Rule |
| --- | --- |
| `format` | Exactly `ssh-keyflow-roster` |
| `schema` | Integer, exactly `1`. A higher value fails (exit 1) with "upgrade SSH KeyFlow", not a generic parse error. Kept separate from the product version |
| `fleet_id` | `fleet-` + 32 lowercase hex (128-bit random), fixed at `fleet init` |
| `version` | Integer, 1 to 2^53−1. Each signed roster is the previous version + 1 |
| `created_at` | UTC RFC 3339 with seconds and `Z`, no fraction: the signing time |
| `login_account` | The fleet's common login account (1.4) |
| `nodes` | 1 to 1000 nodes, sorted by `id`, unique `id` and unique `name` |
| node `id` | `node-` + 32 lowercase hex, generated once on the machine by `node set` |
| node `name` | Lowercase DNS label: `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`, so it can later serve as an `ssh <name>` alias |
| `roles` | Non-empty subset of `client`, `remote`, in that order |
| node `login_account` | Remote only, optional; omitted when equal to the common account |
| `addresses` | Remote only, required, 1–16 unique entries (1.5) |
| `port` | Remote only, optional integer 1–65535; omitted when 22, so `22` never appears |
| `client_keys` | Client only, required, 1–16 unique public keys. Phase 1 `node set` writes exactly one |
| `host_keys` | Remote only, required, 1–16 unique public keys |

A public key is `<type> <base64>` without a comment; the type must match the
type inside the blob. Allowed types: `ssh-ed25519`, `ecdsa-sha2-nistp256/384/521`,
`ssh-rsa` (host keys only, at least 2048 bits), `sk-ssh-ed25519@openssh.com`,
`sk-ecdsa-sha2-nistp256@openssh.com`. `ssh-dss` and certificates are rejected.
A key may appear on only one node. Fields a role does not allow are rejected,
not ignored.

### 1.3 Canonical bytes

**Decided (D):** the signature covers the exact bytes of `roster.json`, and the
canonical form is the only accepted form; the tool never normalizes it:

- UTF-8 without BOM, LF line endings, one trailing LF, no tabs, at most 1 MiB.
- Each object member on its own line as `"name": value`, one space after the
  colon, a comma directly after every member but the last; fields in the order
  of the table above. Absent optional fields are omitted, never `null`.
- Nesting indents by 2 spaces per level. Arrays are never empty, so `[` ends
  its line, each element is on its own line, and `]` is on its own line, like
  objects. String arrays are in byte order, except `roles` (fixed order);
  `nodes` are in `id` order.
- Integers are plain decimal: no sign, leading zero, fraction or exponent.
- No accepted value contains `"`, `\`, a control character or a non-ASCII
  character, so the form never needs an escape; a value that would is rejected.

This is the output of Go's `json.MarshalIndent(v, "", "  ")` plus a trailing LF
for these value sets, but the rules above are the contract.

On acceptance the tool verifies the signature over the received bytes, parses
them strictly (unknown, duplicate or case-variant fields, trailing data, wrong
types and the rules above all fail), re-serializes and requires byte equality.
Any difference is a rejection ("roster is not in canonical form"), so a CRLF
checkout or a reformatting editor fails loudly instead of being repaired.

### 1.4 Account names

`^[A-Za-z0-9_][A-Za-z0-9._-]{0,63}$`. This covers the usual Unix and local
Windows account names. Windows domain accounts (`DOMAIN\user`, `user@domain`)
and names with spaces are out of phase 1 and need a schema change.

### 1.5 Addresses

Each entry is exactly one of:

- an IPv4 literal in dotted decimal without leading zeros;
- an IPv6 literal in the RFC 5952 canonical form (lowercase, no brackets, no
  zone);
- a lowercase DNS name: 1–127 labels as in node `name`, dot-separated, at most
  253 characters, no trailing dot, and not all-numeric.

No port and no brackets; the port is `port`.

## 2. Acceptance rules

`fleet accept <dir>` runs every check before asking anything, then asks once,
and writes state only after `yes`. Any failure changes nothing.

1. The directory holds exactly the three files of 1.1, each a regular file.
   A missing or unreadable `<dir>` is a usage error (exit 2).
2. Choose the fleet root key: with pinned roots, `fleet-root.pub` must equal
   one of them; without, it is the candidate for the first pin.
3. Verify with `ssh-keygen -Y verify` against an `allowed_signers` file the
   tool writes from the chosen key(s): `fleet-root namespaces="ssh-keyflow-roster" <key>`.
   Failure is any non-zero exit (the built-in Windows OpenSSH returns -1,
   others 255). `-Y find-principals` is used only to word the error.
4. Strict parse and canonical form (1.3); with a pinned fleet, `fleet_id` must
   equal the pinned one.
5. Rollback: reject `version` below the highest accepted; reject the same
   `version` with a different SHA-256 of the bytes. The same version and bytes
   is a no-op success with no prompt.
6. Age: warn when `created_at` is older than `--max-age` (default 30 days), or
   more than 5 minutes in the future. Warnings go to stderr only (contract) and
   do not change the exit code.
7. Confirm: print the version, `created_at`, node count and changes against
   the accepted roster. On a first pin, also print the fleet root key's SHA256
   fingerprint and the `fleet_id`, and tell the user to compare them with
   `fleet show` on the admin node. Then ask for a terminal `yes`.
8. After `yes`, save in one state write: the pinned root and `fleet_id` (first
   pin only), the highest accepted version with its SHA-256, and the accepted
   roster bytes. State v4 (ticket 03) defines the layout.

## 3. CLI surface

Conventions kept from v0.11.0:

- Exit 0 success, 1 runtime failure or cancellation, 2 usage. An absent or
  ambiguous target (`--key`, `<dir>`) is a usage error with the command's
  usage on stderr. Invalid state exits 1, prints no JSON and keeps the file.
- Commands that change state (`node set`, `fleet init`, `fleet sign`, `fleet add`,
  `fleet accept`) ask for a terminal `yes` before any side effect and support
  `--dry-run`, which prints the plan and exits 0 without asking or writing.
  No unconditional-approval or secret-value flags.
- Export commands change no state, so they need no confirmation. Their
  `--out` directory must be absent or empty, and every file is created
  exclusively.
- List flags take one comma-separated value (`--roles`, `--addresses`).

| Command | Flags | Effect |
| --- | --- | --- |
| `node set` | `--name`, `--roles client,remote`, `--key <id\|name>` (client), `--addresses a,b` (remote), `--port` (remote), `--login-account` (remote override), `--sshd-config <path>` (remote), `--dry-run` | Creates or updates this machine's node description in state. The client key is any managed key, created or imported. Host keys are read from `HostKey` lines in `sshd_config`, falling back to the platform defaults |
| `node show` | `--json` | Prints this machine's node description |
| `node export` | `--out <dir>` | Writes `node.json` (3.1) for the admin node to add |
| `fleet init` | `--login-account <name>`, `--dry-run` | Admin only. Requires `node set` first. Creates the fleet root key with native `ssh-keygen` (refuses an empty passphrase), pins it, and signs roster version 1 holding this node |
| `fleet sign` | `--dry-run` | Admin only. Signs the next version from the accepted roster with this node's current description; refreshes `created_at` even when nothing changed (ticket 04) |
| `fleet add` | `<dir>`, `--dry-run` | Admin only. Adds the node in `<dir>/node.json` and signs the next version (3.1) |
| `fleet show` | `--json`, `--max-age <days>` (1-36500) | Prints the accepted roster, pinned root fingerprints, version and age; warnings on stderr |
| `fleet export` | `--out <dir>` | Writes the three files of 1.1 for the current roster |
| `fleet accept` | `<dir>`, `--max-age <days>`, `--dry-run` | Section 2 |

**Decided (A):** add `node export` and `fleet add`, new names beyond the five
already decided; they answer the spec's open question on adding nodes without
`join`. Removing a node is not in phase 1 (revocation UX is phase 5).

**Decided (B):** `--max-age` is a per-run flag with a 30-day default, not a
stored setting. A stored setting would need its own command.

### 3.1 Node description file and `fleet add`

`node.json` follows the canonical rules of 1.3 and holds `format`
(`ssh-keyflow-node`), `schema` (`1`) and `node` (one node object from 1.2). It
is unsigned. `fleet add` shows the node's name, roles and every key
fingerprint, and the admin compares them with `node show` on that machine.

- A new id with a name already used by another id is rejected.
- A known id replaces that node: `fleet add` shows old and new values side by
  side, and the prompt names it a replacement.
- The admin node's own id is rejected; its own changes reach the roster
  through the signing flow of ticket 04.

### 3.2 Files and processes

All files are under `~/.ssh/ssh-keyflow/fleet/` or a user-given `--out`:

- the fleet root key `root_ed25519` (+ `.pub`); the private key is never
  copied, printed or stored in state, only its path and fingerprint;
- temporary files for `ssh-keygen -Y`: the `allowed_signers` file of step 3,
  and a copy of the roster bytes to sign (`-Y sign` writes `<file>.sig` next
  to its input). Each is created exclusively in a new temporary directory
  there and removed afterwards;
- exports: `roster.json`, `roster.json.sig`, `fleet-root.pub` and `node.json`.

No SSH file is written. Reads beyond state are the managed client public key,
`sshd_config` and the host public keys it names. Signing lets `ssh-keygen`
prompt on the inherited terminal; the tool sets no `SSH_ASKPASS`.

## 4. Automation-facing JSON

One object ending with a newline on stdout; arrays are never `null`; field
names follow the roster names. Warnings are not part of JSON (contract: notices
go to stderr); automation can compare `roster.created_at` itself.

`node show --json`: `{"node": <node object>, "is_admin": false}`; before
`node set`, `{"node": null, "is_admin": false}`.

`fleet show --json`:

```json
{
  "fleet": {
    "roots": [
      {
        "fingerprint": "SHA256:…"
      }
    ],
    "sha256": "…",
    "roster": "…roster object from 1.2…"
  }
}
```

Before any acceptance, `{"fleet": null}`. `fleet_id`, `version` and
`created_at` are read from `roster`, not repeated. `roots` always has one
element in phase 1; it is an array so the backup root key of phase 5 adds no
breaking change.

## 5. Minimum OpenSSH

**Decided (C):** require OpenSSH 8.4 or later. 8.2 would suffice for
`find-principals`, and phase 1 does not use `SSH_ASKPASS_REQUIRE`; 8.4 leaves
room for a non-interactive signer later. The lowest build tested is 9.5p2.

The tool runs the first `ssh-keygen` on PATH, as the key commands do today, and
uses that same binary for every call within one command. On Windows the
built-in Win32-OpenSSH and Git for Windows' build may both be on PATH, so the
tool prints the resolved path with the version in `--dry-run` output and in
every signing or verification error, and stops below the minimum.

## 6. Not decided here

- State v4 layout and migration (ticket 03).
- Root key creation details and the signing flow, including how the admin
  node's own changes are signed (ticket 04).
- Export/accept file handling and tests (ticket 05).
- Glossary: "pin" and "node description file" are new terms, and the client
  entry says "a key pair made by the tool" while `--key` accepts imported keys.
  Update all three locales with ticket 03, before code.
- Later phases: backup root keys, root rotation, node removal (phase 5);
  storing rosters with `-text` in a git static backend (phase 4); more than
  one port per remote and per-client access lists (later schema).
