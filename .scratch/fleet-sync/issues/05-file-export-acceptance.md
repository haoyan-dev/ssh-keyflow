# 05 — File export, acceptance and verification

Status: ready-for-human
Blocked by: 04

Criteria 4, 7 and 8. Export the signed roster and a node's public description as
files. Acceptance verifies against the pinned root keys, pins the root on first
acceptance after fingerprint confirmation, rejects bad or unknown signatures,
lower versions and same-version different content, and warns on age. Supports
`--dry-run`. Tests on all three CI OSes; contract and ja/en/zh-CN docs at
release.

## Comments

2026-11-28 — Implemented on `feat/fleet-export-accept` (base origin/main
aca97e4), uncommitted pending review.

- Commands (roster-format 2, 3, 3.1): `node export --out <dir>`,
  `fleet export --out <dir>`, `fleet add <dir> [--dry-run]` and
  `fleet accept <dir> [--max-age <days>] [--dry-run]`. `fleet add` reuses the
  `fleet sign` path (ticket 04); `roster.NodeFile` shares the roster's strict,
  non-normalizing parser.
- Defaults chosen here (not user decisions):
  - Exports need no confirmation; `--out` is created when absent, must
    otherwise be an empty plain directory (a non-empty or linked one fails
    with exit 1), and files are created exclusively with mode 0644. A failed
    export removes the files it wrote.
  - `node export` keeps the node's login account as set; `fleet add` drops it
    when it equals the roster's common account.
  - `fleet export` works on any node with an accepted roster, so a member can
    re-export it; it writes the stored signature and needs no passphrase.
  - A missing or unreadable `<dir>` is a usage error checked before the
    terminal check. A wrong file set, an unreadable file, a bad root file and
    every verification failure exit 1 and change nothing.
  - `fleet add` needs a terminal before planning (passphrase). `fleet accept`
    plans first and needs a terminal only when it will prompt, so re-accepting
    the same roster succeeds without one (roster-format 2 rule 5).
  - An admin node whose `fleet init` is unfinished refuses `fleet accept` and
    is told to rerun `fleet init`.
  - `fleet-root.pub` must be one `ssh-ed25519 <base64>` line without a
    comment (the pinned roots are Ed25519). A CRLF signature file is accepted
    and stored with LF; the roster bytes are never normalized.
  - Verification failures are worded with `-Y find-principals`: "does not
    match its signature" (the root signed something else) versus "is not a
    valid signature by root".
  - `fleet accept` verifies in a new directory under the system temporary
    directory, not `fleet/`, so `--dry-run` initializes no storage; only
    public data is written there (roster-format 3.2 updated). The user chose
    this over a `fleet/` workspace on 2026-11-28, and spec criterion 7 now
    names the exception.
  - Accepting the accepted version with the same bytes prints "already
    accepted" with no prompt and no write, also in a dry run.
  - The plan lists the changes against the accepted roster (common account,
    added, replaced, removed nodes). A first pin prints the root fingerprint
    and fleet id, and the prompt names the fingerprint.
- Tests drive the real ssh-keygen: export → add → export → accept end to end
  across two homes; each rejection (unknown root, bad signature, modified
  roster, rollback, same version with other bytes, other fleet, CRLF roster,
  extra file, commented root) in dry and confirmed runs with no state change;
  age and future warnings on stderr only; declined first pin pins nothing;
  `fleet add` refusals (own id, name or key in use, CRLF node.json); CLI exit
  codes. Verified locally on Windows OpenSSH_for_Windows_9.5p2; CI runs Linux
  and macOS.
- Contract and ja/en/zh-CN docs follow at the phase-1 release (spec
  criterion 8), as in tickets 03 and 04.
