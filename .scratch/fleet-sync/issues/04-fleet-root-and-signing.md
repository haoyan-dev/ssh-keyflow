# 04 — Fleet root key and roster signing

Status: ready-for-human
Blocked by: 02, 03

Criterion 3. Create the passphrase-protected Ed25519 fleet root key with native
ssh-keygen, store only its path and fingerprint, and sign rosters under
namespace `ssh-keyflow-roster`. The private key never appears in state or output.

## Comments

2026-11-28 — Decisions from the user (roster-format 6 left the signing flow to
this ticket):

- **`fleet sign`** (new command, `--dry-run`): signs version N+1 from the
  accepted roster with this node's current description, and refreshes
  `created_at` even when nothing changed, so the admin can renew a roster
  before `--max-age`. `fleet add` (05) reuses the same signing path.
- **`fleet init --login-account` is required**; omitting it is a usage error.
- **`fleet show [--json] [--max-age]` is built here**, with `fleet init`.
- **Empty passphrase:** the root key is generated in a new temporary
  directory under `fleet/` and moved to `root_ed25519` only after
  `ssh-keygen -y -P ""` fails (the key is protected). An unprotected key is
  discarded with its temporary directory; nothing is pinned.

Defaults chosen here (not user decisions):

- The confirmed `fleet init` saves the pinned root and root key as soon as the
  key exists, then signs. If signing fails, rerunning `fleet init` signs
  version 1 with the existing key instead of creating another.
- An existing `fleet/root_ed25519` (or `.pub`) without a fleet in state is
  refused for review; the tool never reuses or replaces it.
- The signature is stored with the accepted roster in state
  (`fleet.accepted.signature`), so `fleet export` (05) needs no passphrase.
- Every signature is verified with `ssh-keygen -Y verify` against the pinned
  key before it is saved.
- The ssh-keygen version comes from `ssh -V` next to the resolved
  `ssh-keygen` (ssh-keygen has no version flag); below 8.4 stops.
- From ticket 03: a node's `login_account` equal to the common account is
  omitted when the roster is built, and state checks that
  `fleet.accepted.version` and `fleet.id` match the stored roster.

2026-11-28 — Implemented on `feat/fleet-root-signing` (uncommitted; awaiting
review):

- `internal/roster/roster.go`: `Roster`/`Node`, `Validate`, canonical
  `Marshal`, `Parse` (strict, never normalizes; newer schema asks for an
  upgrade), `ValidSignature`, `AgeWarning`.
- State v4: `fleet.accepted.signature`; validation parses the stored roster
  and checks its version and `fleet_id` against the record.
- `fleet init --login-account <name> [--dry-run]`, `fleet sign [--dry-run]`,
  `fleet show [--json] [--max-age <days>]` (`internal/app/fleet.go`,
  `internal/cli/fleet.go`). One resolved ssh-keygen per command; its path and
  version appear in dry runs and signing/verification errors.
- Tests run the real ssh-keygen with an `SSH_ASKPASS` helper in the test
  runner only: init end to end, empty passphrase discarded, wrong passphrase
  saves nothing, resume after a failed signature, `fleet sign` to version 2,
  independent `-Y verify` of every accepted signature. Verified locally on
  Windows OpenSSH_for_Windows_9.5p2; not run on Linux/macOS OpenSSH.

2026-11-28 — Review fixes (standards and spec axes run separately):

- Spec: `fleet show` text prints the roster's age; `--max-age` is limited to
  1-36500 days (usage error otherwise; prevents duration overflow); the root
  pair is placed with hard links, which never replace an existing file; the
  empty-passphrase check accepts only ssh-keygen's "incorrect passphrase"
  failure as proof of protection, and its errors name the ssh-keygen path and
  version.
- Standards: `fleet init` and `fleet sign` share one plan → confirm → apply
  flow; shared existing-path and migration-notice helpers (also used by
  `node set`); unreadable keys are shown, not skipped; `CheckLoginAccount`
  rename; `state` calls `roster.ValidID` directly; `fleet sign` added to
  roster-format 3.
- Contract and ja/en/zh-CN docs (`docs/application-contract.md`) are not
  updated here: they follow at the phase-1 release (spec criterion 8), as in
  ticket 03.
- Kept as is: `FleetOptions` mixes request flags and wiring (unlike
  `NodeOptions`); `roster.AgeWarning` words CLI flags; the port-22 default is
  repeated in older code. Kept the "Admin node" line in the text view to help
  compare with the admin node.
