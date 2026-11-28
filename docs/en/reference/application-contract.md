# Application contract (overview)

[简体中文](../../zh-CN/reference/application-contract.md) · English · [日本語](../../application-contract.md)

> This page is an English overview. The authoritative contract is [docs/application-contract.md](../../application-contract.md) (mostly in Japanese); if they differ, that page wins. Its first two sections are already written in English there, so this overview only points to them.

[Documentation index](../README.md) · [Glossary](glossary.md)

Each section below summarizes one section of the authoritative contract, in the same order, and ends with a link to that section.

## Source-hosted one-line installers

This section of the authoritative contract is written in English; read it there rather than a restatement here. In short: `install.sh` and `install.ps1` are served separately from `main` and install a verified Portable release (latest stable by default) into a per-user location, change PATH only with consent, and never modify SSH keys, state, SSH configurations or remote registrations.

Authoritative section (English): [Source-hosted one-line installers](../../application-contract.md#source-hosted-one-line-installers).

## Portable release

This section of the authoritative contract is also written in English; read it there. In short: each release has exactly six archives plus `SHA256SUMS` (seven assets), each archive holds only the platform executable and `THIRD_PARTY_NOTICES.txt`, storage stays at `<Home>/.ssh/ssh-keyflow` wherever the executable is placed, and old v0.1.0 installers reject the new archives.

Authoritative section (English): [Portable release](../../application-contract.md#portable-release).

## Current feature scope (v0.11.0 and later)

- SSH KeyFlow is a CLI limited to managing the SSH key pairs on this machine. It provides Ed25519 key creation, import of existing key pairs, resuming interrupted creation, removal from management, listing, and showing public keys for copying. Per ADR 0007, destinations (`hosts`) and associations, the manual setup guide, snippets and ssh-agent step, host key pinning, the terminal UI and public-only import were retired in v0.11.0. This is an intentional incompatible change and does not apply retroactively to v0.10.0 or earlier binaries. Distribution, installers and the publishing procedure are kept.
- The app runs no SSH processes, remote authentication or probes, remote registration or script generation. It does not read, edit, regenerate or delete `~/.ssh/config`, `~/.ssh/known_hosts`, Include files or the past dedicated config. The `<Home>/.ssh/ssh-keyflow/known_hosts` (host key pin file) created by v0.10.0 and earlier, and SSH config lines the user appended from the guide, are also left as they are. Deploying a public key to a destination is manual work for the user, who copies the output of `keys public`.
- `main` includes ADR 0008 fleet sync phase 1 (`node` / `fleet` commands, state schema v4) as an unreleased feature. Its scope and rules are in [Fleet sync phase 1 (unreleased)](#fleet-sync-phase-1-unreleased) and do not apply to released versions. Phase 1 also makes no SSH connections or remote registrations and writes no SSH files.

Authoritative section: [現行機能の範囲（v0.11.0以降）](../../application-contract.md#現行機能の範囲v0110以降) (Japanese).

## CLI

The entry point is `cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner}) int`, and `cmd/ssh-keyflow` returns its exit code. Home resolution is deferred until an operation needs it.

- Commands: `keys list` (optional `--json`; shows managed keys and unfinished local creations), `keys public` (required `--key`; prints the public key algorithm and blob on one line without the file comment), `keys create` (required `--name`; creates an Ed25519 key pair), `keys import` (required `--name`, `--public` and `--private`; verifies an existing key pair and manages it without moving the files), `keys remove` (required `--key`; removes it from management and keeps the key files) and `resume` (required `--id`; resumes only unfinished local creations).
- For the `node` / `fleet` commands on `main`, see [Fleet sync phase 1 (unreleased)](#fleet-sync-phase-1-unreleased). `keys remove` rejects a key used as the client key of the node description as an execution failure (1), and tells the user to change it with `node set` first.
- A key resolves by fixed ID or unique exact managed name. Ambiguous or missing targets are rejected. `hosts`, `tui`, `register`, `unregister`, `registrations list`, `keys delete`, `--public-only`, `--new-key`, `--auth`, `--identity` and `--configure` are not provided; specifying them is a usage error.
- Changing operations (create / import / remove / resume) require real-terminal input before storage initialization or any process execution, and pass an explicit Request and confirmation function to the shared `app.Run`. Only an exact `yes` (surrounding whitespace trimmed) approves. `ReadConfirmation` reads one line byte by byte and does not read ahead into the following native passphrase input. There are no flags for secrets or unconditional approval.
- Exit codes: 0 success, 1 execution failure or cancellation, 2 argument/usage error. Ambiguous or missing targets are argument errors too, and the command's usage is written to stderr; runtime diagnostics carry no usage. `--help` and `--version` do not resolve Home, check dependency commands or save anything. The version output is `SSH KeyFlow <version>`. Launching with no arguments prints help and exits with 0, whatever kind of terminal it runs in.
- Lists and public key output can be redirected and do not probe writability or SSH commands. List output is a table by default. JSON is a single object ending with a newline that has only the two fields `keys` and `creations`. Arrays are empty arrays, not null, and element field names match the storage schema. Items needing review and migration notices go only to stderr, and verifiable state records remain readable. Invalid state returns 1, prints no JSON and keeps the original file. `keys public` writes copyable text ending with a newline to stdout. An output failure is an execution failure.

Authoritative section: [CLI](../../application-contract.md#cli) (Japanese).

## Shared backend

- `app.Request{Operation, Name, PublicPath, PrivatePath, Key, Resume}` Operations are `create` / `import` / `remove` / `resume`. An unknown Operation is an argument error. `app.Run` requires an explicit Request and re-resolves and re-validates change targets under the lock. `remove` does not check for `ssh-keygen`, so management data for missing key files can also be cleaned up.
- `app.Snapshot(Options)` returns `View{State, Review}` without initializing storage, probing dependency commands or migrating. Review includes missing or replaced key files and a notice that state in an old schema will be migrated by the next change. `app.PublicKey(home, keyRef)` rechecks the target key's fingerprint and returns `<type> <base64>`.
- Removal from management, after a real terminal and exact `yes` confirmation, removes only the key's record from the state. Key files, SSH configuration and public key registrations on destinations are not changed, and removal does not mean access is revoked. Removed records are not kept. The same key pair can be managed again under a new ID by re-importing it.

Authoritative section: [共通バックエンド](../../application-contract.md#共通バックエンド) (Japanese).

## Storage schema and compatibility

- Storage is `<Home>/.ssh/ssh-keyflow`, using `state.json`, `keys/` for generated keys, the rolling `state.json.bak`, a temporary `state.lock`, and `state.v<N>-<UTC time>.json` when migrating from an old schema. With no state, an empty state is returned and no file is created on read.
- v3 is the only schema v0.11.0 writes. It has only `version`, the required array `keys` and the optional array `creations`. `main` (phase 1, unreleased) writes only v4. v4 adds the optional `node` and `fleet` to the v3 fields, with `version` 4 (see [State schema v4](#state-schema-v4)). A Key saves `id`, `name`, `fingerprint`, `public_path`, `private_path`, `private_sha256`, `origin`, and `created_at` (created) or `imported_at` (imported). A Creation saves `id`, `key_id`, `name`, `private_path`, `public_path`, `created_at`, `stage` (`planned` / `pending`) and optional `private_sha256` / `fingerprint`.
- Key IDs are fixed random `key-<128-bit hex>`, fingerprints are the SHA256 form of the OpenSSH public blob, and paths are absolute. Import saves `origin: imported` and a nonzero UTC RFC3339Nano `imported_at`; creation saves `origin: created` and `created_at`. The private key digest detects replacement without asking for a passphrase and does not store private key contents. A completed creation is removed from `creations` in the same save that adds the key.
- Loading v3 is strict: invalid or trailing JSON, duplicate or case-variant fields, unknown fields, unsupported versions, missing `keys`, invalid metadata, and duplicate IDs/fingerprints/paths are rejected, without automatic restore or overwrite. Loading v4 follows the same rules.

Authoritative section: [保存スキーマと互換性](../../application-contract.md#保存スキーマと互換性) (Japanese).

### Migrating from v1 / v2

- v1 and v2 (v0.10.0 and earlier) are converted to v3 in memory on load. Loading alone does not change the file. Loading the old schema is also strict: unknown fields, wrongly typed arrays, trailing JSON content and missing required arrays (`keys` / `hosts` / `registrations`) are rejected. The conversion carries over only keys that have a private key path and unfinished local creations. It does not carry over destinations, associations and archived removal records (`hosts` / `associations` / `removed_keys` / `removed_hosts`); the records and history of legacy registrations, connection checks, config, SSH config edits, onboardings and deletions (`registrations` / `history` / `configs` / `config_edits` / `onboardings` / `deleted_keys` / `deletions`); public-only keys and keys with an unfinished legacy key deletion; or legacy fields such as a key's `connection_check` and a destination's `config_key_id`.
- The notice gives a count for each non-empty record type. Each unfinished SSH config edit is listed with its path and a request to check that file manually. Before migrating, `keys list` prints this notice on stderr. Changing commands (create / import / remove / resume) show the same notice after taking the lock and before the confirmation prompt, so the confirmation also approves the migration. The first successful state save exclusively writes the bytes of the original `state.json` to a new `state.v<N>-<UTC time>.json` in the same directory (with a numeric suffix on collision), then updates `.bak` and saves v3. If creating the archive fails, the state is not changed. If an earlier attempt already left an archive with the same bytes, a retry reuses it instead of adding an identical archive. The archive is never overwritten or deleted automatically, so records that are not carried over remain on disk.
- Older binaries cannot read v3. To downgrade, exit the app and manually restore the archive as `state.json`. Keys created or imported after the migration are not included in the restored state.

Authoritative section: [v1 / v2からの移行](../../application-contract.md#v1--v2からの移行) (Japanese).

## Fleet sync phase 1 (unreleased)

- This section describes the implementation on `main`. It is not part of any published version until a MINOR release includes phase 1. The design decision is [ADR 0008](../../adr/0008-signed-fleet-roster.md) (Japanese). Phase 1 covers handing a roster signed by the fleet root key to another node as files, and that node verifying it with its pinned root public key. It does not write `authorized_keys`, known_hosts or SSH config, and has no network enrollment or sync, revocation or key rotation.

Authoritative section: [フリート同期フェーズ1（未リリース）](../../application-contract.md#フリート同期フェーズ1未リリース) (Japanese).

### Commands

- `node set` (required `--name` and `--roles`, one or more of `client` / `remote` as one comma-separated value; a client requires `--key`; a remote requires `--addresses` (one comma-separated value) and accepts optional `--port`, `--login-account` and `--sshd-config`; optional `--dry-run`) creates or replaces this machine's node description, generating the node ID the first time and keeping it after that. `node show` (optional `--json`) shows the node description and whether this is the admin node. `node export` (required `--out <dir>`) writes the node description file `node.json`. On the admin node, `fleet init` (required `--login-account`, optional `--dry-run`) creates and pins the fleet root key and signs roster version 1 containing only this node; `fleet sign` (optional `--dry-run`) signs the next version with the current node description, updating `created_at` even with no changes; `fleet add <dir>` (optional `--dry-run`) adds or replaces the node in `<dir>/node.json` and signs the next version. `fleet show` (optional `--json`, `--max-age <days>`, 1 to 36500, default 30) shows the accepted roster, pinned root fingerprint, version and age. `fleet export` (required `--out <dir>`) writes the accepted roster, signature and root public key as three files; it works on any node and asks for no passphrase. `fleet accept <dir>` (optional `--max-age <days>`, default 30, and `--dry-run`) verifies a roster and accepts it after confirmation, pinning the root public key the first time.
- List flags (`--roles`, `--addresses`) take one comma-separated value; repeating them is a usage error. A node name is one lowercase DNS label (63 characters or fewer); an account name is up to 64 letters, digits, `.`, `_` or `-`; an address is a lowercase DNS name or a canonical IP address (no port or brackets, 1 to 16, no duplicates). `--key` is required for the client role and not allowed otherwise. `--addresses`, `--port`, `--login-account` and `--sshd-config` are allowed only for the remote role. `--port 22` is the same as omitting it. The client key can be any managed key, created or imported. A remote's host public keys are read from the `HostKey` lines and Includes of `sshd_config` (default `/etc/ssh/sshd_config` on Unix, `%ProgramData%\ssh\sshd_config` on Windows), or, with no `HostKey` lines, from `ssh_host_{rsa,ecdsa,ed25519}_key.pub` in the platform default directory (`/etc/ssh` or `%ProgramData%\ssh`, regardless of `--sshd-config`). Relative Include paths also resolve against the default directory. Missing public keys are skipped and shown in the plan; if none can be read, the command fails. Private host keys are never opened. A specified `--sshd-config` that does not exist is a failure.

Authoritative section: [コマンド](../../application-contract.md#コマンド) (Japanese).

### Terminal, confirmation and exit codes

- Exit codes are the same as v0.11.0: 0 success, 1 execution failure or cancellation, 2 usage error. A missing or unreadable `<dir>`, invalid flag values as above, and a missing or ambiguous `node set --key` are usage errors.
- Changing commands (`node set`, `fleet init`, `fleet sign`, `fleet add`, `fleet accept`) show a plan, then require an exact `yes` on a real terminal, and cause no side effects before approval. `--dry-run` does all planning except confirmation (including the `ssh-keygen` version check, and for `fleet accept` the signature verification and rejection checks) and shows the result, without confirmation, writing or storage initialization. It exits with 0 when planning succeeds and 1 when a check rejects, and needs no real terminal. `fleet init`, `fleet sign` and `fleet add` require a real terminal before the plan for the `ssh-keygen` passphrase input, and `node set` for confirmation. `fleet accept` requires one after the plan and only when confirmation is needed, so accepting the same roster again succeeds without a terminal.
- Exports (`node export`, `fleet export`) do not change the state and ask for no confirmation. `--out` must be a missing or empty regular directory; a non-empty directory or a link fails without writing anything. Files are created exclusively (mode 0644), and written files are deleted if a later step fails.
- Reads (`node show`, `fleet show`, exports, `fleet accept --dry-run`) do not initialize storage or migrate state. Migration notices and warnings go only to stderr.

Authoritative section: [端末・確認・終了コード](../../application-contract.md#端末確認終了コード) (Japanese).

### Roster and files

- A roster is a JSON document with `format` (`ssh-keyflow-roster`), `schema` (1), `fleet_id`, `version` (1 or more, increasing by 1 with each signing), `created_at` (UTC with second precision and `Z`), the shared `login_account`, and `nodes` in `id` order (1 to 1000). Each node has `id`, `name`, `roles`; a remote has an optional `login_account` (omitted when equal to the shared account), `addresses`, an optional `port` (omitted when 22) and `host_keys`; a client has `client_keys`. Public keys are `<type> <base64>` without a comment.
- The signature covers the roster bytes as they are, and only the canonical form is accepted: UTF-8 without BOM, LF line endings, one trailing newline, no tabs, two-space indentation, one member (`"name": value`) per line, the field order above, and 1 MiB or less. Absent optional fields are omitted, not written as `null`. String arrays are in byte order (`roles` in the order `client`, `remote`). The app does not normalize, so CRLF or reformatting is rejected. Unknown, duplicate and case-variant fields are rejected too. A roster with `schema` 2 or higher fails and asks for an app update.
- `fleet export` writes `roster.json`, `roster.json.sig` (an `ssh-keygen -Y sign` signature in the `ssh-keyflow-roster` namespace) and `fleet-root.pub` (one line `ssh-ed25519 <base64>` without a comment). `node export` writes an unsigned `node.json` (`format` `ssh-keyflow-node`, `schema` 1, one `node`). `fleet add` omits a node's account when it equals the shared account.

Authoritative section: [名簿とファイル](../../application-contract.md#名簿とファイル) (Japanese).

### Admin node operations

- `fleet init` requires `node set` first, and creates a passphrase-protected Ed25519 fleet root key at `~/.ssh/ssh-keyflow/fleet/root_ed25519` (and `.pub`) with the native `ssh-keygen`. An empty passphrase is rejected and the created key is discarded. Existing files are never reused or replaced. The state saves only the root key's paths and fingerprint; private key contents never appear in the state or output.
- If signing version 1 fails after the key is pinned, running `fleet init` again signs version 1 with the same key. Until then, `fleet sign` and `fleet accept` are rejected and `fleet show` warns. `fleet init` runs only once and is rejected on a node that has accepted a roster.
- `fleet sign` and `fleet add` build the next version from the accepted roster, and `ssh-keygen` asks for the passphrase directly on the terminal. The app does not set `SSH_ASKPASS`. After signing, the result is verified with the pinned root, then the signature, roster and SHA-256 are saved as accepted in one save. On failure nothing is saved.
- `fleet add` shows the name, roles and each key's fingerprint, and asks the user to compare them with `node show` on the target machine. A known ID is a replacement: old and new values are shown side by side, and the confirmation wording says replacement. The admin node's own ID, a name used by another ID, and a key identical to another node's are rejected. Phase 1 has no node removal.

Authoritative section: [管理ノードの操作](../../application-contract.md#管理ノードの操作) (Japanese).

### Acceptance rules

`fleet accept` performs every check before confirmation, and no failure changes the state.

1. `<dir>` contains only the three files above, each a regular file.
2. If a root is pinned, `fleet-root.pub` must match one of the pinned roots; otherwise it is the candidate for the first pin.
3. The signature is verified with `ssh-keygen -Y verify`. Temporary files hold only public data in a new directory inside the system temporary directory and are deleted on exit. CRLF in the signature file is saved as LF.
4. The roster is parsed strictly and must be canonical. For a pinned fleet, `fleet_id` must match.
5. A version lower than the accepted one, or the same version with different content (SHA-256), is rejected. The same version with the same bytes succeeds without confirmation and writes nothing.
6. A `created_at` older than `--max-age`, or more than 5 minutes in the future, gives a warning on stderr. The exit code does not change.
7. The version, creation time, node count and changes from the accepted roster (shared account; added, replaced and removed nodes) are shown. The first time, the root public key fingerprint and fleet ID are also shown, the user is asked to compare them with `fleet show` on the admin node, and the confirmation wording includes the fingerprint.
8. After `yes`, the first root pin and fleet ID, and the accepted version, SHA-256, roster and signature are saved at once.

An admin node with an unfinished `fleet init` does not accept a roster and is told to run `fleet init` again.

Authoritative section: [受け入れの規則](../../application-contract.md#受け入れの規則) (Japanese).

### JSON for automation

- One object ending with a newline goes to stdout. Warnings are not included in the JSON.
- `node show --json`: `{"node": <node>, "is_admin": <bool>}`. The node uses the same field names as a roster node, and `client_keys` holds the client key's public key. Before `node set` it is `{"node":null,"is_admin":false}`.
- `fleet show --json`: `{"fleet": {"roots": [{"fingerprint": "SHA256:…"}], "sha256": "<hex SHA-256 of the roster bytes>", "roster": <roster object>}}`. Read `fleet_id`, `version` and `created_at` from `roster`. In phase 1 `roots` always has one element. Before acceptance it is `{"fleet":null}`.

Authoritative section: [自動処理向けJSON](../../application-contract.md#自動処理向けjson) (Japanese).

### OpenSSH requirement

- Signing and verification require OpenSSH 8.4 or later. The version is checked with `ssh -V` from the same directory as the first `ssh-keygen` on PATH, and the command fails if it is too old. One command uses the same `ssh-keygen` throughout, and `--dry-run` output and signing or verification errors show its path and version.
- This applies only to `fleet init`, `fleet sign`, `fleet add` and `fleet accept`; the requirements of the key management commands are unchanged.

Authoritative section: [OpenSSHの要件](../../application-contract.md#opensshの要件) (Japanese).

### State schema v4

- v4 adds the optional `node` and `fleet` to the v3 `version`, `keys` and `creations`. A Node saves `id`, `name`, `roles`, `updated_at` and optional `login_account`, `addresses`, `port`, `client_key_ids`, `host_keys` and `sshd_config`. A Fleet saves `id`, `roots` (`public_key`, `fingerprint`, `pinned_at`), `root_key` on the admin node only (`fingerprint`, `public_path`, `private_path`), and optional `accepted` (`version`, `sha256`, `roster`, `signature`, `accepted_at`).
- `client_key_ids` are managed key IDs; the public keys themselves are read from the key files. `accepted.roster` keeps the accepted roster bytes as they are, and `fleet export` writes them with the signature. A key used as the node's client key cannot be removed with `keys remove`.
- Migration: v3 state is not changed by loading alone. `keys list`, `node show`, `fleet show`, exports and `--dry-run` of changing commands print a notice on stderr that the next change migrates to v4, and changing commands show the same notice before confirmation. The first successful save exclusively writes the original `state.json` to `state.v3-<UTC time>.json`, under the same rules as v1 / v2, and then saves v4. All v3 records are carried over. `main` also migrates v1 / v2 state directly to v4, carrying over the same scope as [Migrating from v1 / v2](#migrating-from-v1--v2).
- Downgrade: v0.11.0 and earlier cannot read v4. To go back, exit the app, move the current `~/.ssh/ssh-keyflow/` to another location and restore the archive as `state.json`. The node description, fleet, accepted roster and keys created or imported after the migration are not in the restored state, and the root key files in `fleet/` are no longer referenced by the state. Running `fleet init` again in v4 with those files left in place is rejected because the files exist. `state.json.bak` is the rolling previous-state backup: it is v3 right after the migration but is replaced with v4 by the next change, so use the archive to go back.
- Fleet root key backup: the fleet root private key cannot be recreated; if it is lost, the next version of the same fleet cannot be signed. On the admin node, keep `~/.ssh/ssh-keyflow/fleet/` together with the state in an access-restricted location. Phase 1 has no backup root key or root rotation.

Authoritative section: [状態スキーマv4](../../application-contract.md#状態スキーマv4) (Japanese).

## Local key safety conditions and storage

- Creation uses the native `ssh-keygen -t ed25519` and allows an empty passphrase. The target is `keys/id_ed25519_<安全化した管理名>_<UTC時刻>` (sanitized managed name and UTC time) with a numeric suffix on collision. Both paths and saved reservations are checked, and checked again after confirmation. Existing files are never overwritten or adopted automatically.
- Before any side effect, the creation intent, fixed ID and paths are saved as `planned`, and generation is recorded as `pending`. Resume uses the same paths: with no files it generates; with only a private key it derives the missing public key with the native `ssh-keygen -y` and creates it exclusively; with a pair it verifies without regenerating. Public-key-only, empty or nonregular files, changed recorded fingerprints/digests, or disappearance of observed files stop for review. Partial files are never deleted automatically, and no other key pair is newly created while a creation is unfinished.
- Import verifies the specified public key with `ssh-keygen -l -E sha256` and requires it to match the fingerprint of the public key derived from the private key with `ssh-keygen -y`. Original files are not moved or rewritten. Fingerprints and the private key digest are rechecked before saving, and missing or replaced files are not treated as success. There is no API for deleting local key files or automatic cleanup.
- All changes are protected by the exclusive `state.lock` through `state.WithLock`, held during confirmation, creation and saving. LockedStore is not used outside the lock, and locks are not nested. Existing locks are never deleted automatically, and writability is checked only when changing. After validation, state is written and synced to a temporary file in the same directory, the previous valid state is atomically saved to `.bak`, and then `state.json` is replaced. If the backup fails, the current state is not replaced. Users must not edit target key paths concurrently during an operation.

Authoritative section: [ローカル鍵の安全条件と保存](../../application-contract.md#ローカル鍵の安全条件と保存) (Japanese).

## Verification and publishing

- The Go tests and vet in CONTRIBUTING, the Portable / installer tests and the six-target builds apply. Tests use a temporary home and substitute processes and do not change the real user's SSH data, PATH, profiles or destinations. Covered checks include unchanged reads of old schemas with the notice, the archive and the reduced migration, an exact previous-state backup, read-only behavior of reads, key creation resume with identity verification, and usage errors for retired commands.
- Results, target commits, environments and unverified scope are recorded in `docs/verification.md`. Past verification records of automatic registration, the guide, the TUI and host key pinning are history, not the contract for current features. Cross-builds are distinct from evidence of running native key creation on each OS.
- Publishing uses an annotated `vMAJOR.MINOR.PATCH` tag on a commit merged into and verified on main. [Tags and releases in AGENTS.md](../../../AGENTS.md#tags-and-releases) is authoritative for the steps, completion criteria and approval scope from merging the preparation PR, verifying the final candidate, creating and publishing the tag, to merging the post-release record PR. The preparation record references the verified source commit; the final tag's commit is kept in the tag, release information and post-release record. Even if main moves ahead of the tag with audit documents, the published tag keeps pointing at its build source. Pushing a `v*` tag starts automatic publication after CI. Prereleases are never made latest, and the stable latest advances to newer stable versions. Published tags and binary assets are immutable; fixes ship as new versions. After publishing, distributions, checksums, internal version, tag and source commit are checked to match. Development builds continue to show `dev`.

Authoritative section: [検証と公開](../../application-contract.md#検証と公開) (Japanese).
