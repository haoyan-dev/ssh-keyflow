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

Authoritative section: [現行機能の範囲（v0.11.0以降）](../../application-contract.md#現行機能の範囲v0110以降) (Japanese).

## CLI

The entry point is `cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner}) int`, and `cmd/ssh-keyflow` returns its exit code. Home resolution is deferred until an operation needs it.

- Commands: `keys list` (optional `--json`; shows managed keys and unfinished local creations), `keys public` (required `--key`; prints the public key algorithm and blob on one line without the file comment), `keys create` (required `--name`; creates an Ed25519 key pair), `keys import` (required `--name`, `--public` and `--private`; verifies an existing key pair and manages it without moving the files), `keys remove` (required `--key`; removes it from management and keeps the key files) and `resume` (required `--id`; resumes only unfinished local creations).
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

- Storage is `<Home>/.ssh/ssh-keyflow`, using `state.json`, `keys/` for generated keys, the rolling `state.json.bak`, a temporary `state.lock`, and `state.v<N>-<UTC time>.json` when migrating from an old schema. With no state, an empty v3 state is returned and no file is created on read.
- v3 is the only schema this version writes. It has only `version`, the required array `keys` and the optional array `creations`. A Key saves `id`, `name`, `fingerprint`, `public_path`, `private_path`, `private_sha256`, `origin`, and `created_at` (created) or `imported_at` (imported). A Creation saves `id`, `key_id`, `name`, `private_path`, `public_path`, `created_at`, `stage` (`planned` / `pending`) and optional `private_sha256` / `fingerprint`.
- Key IDs are fixed random `key-<128-bit hex>`, fingerprints are the SHA256 form of the OpenSSH public blob, and paths are absolute. Import saves `origin: imported` and a nonzero UTC RFC3339Nano `imported_at`; creation saves `origin: created` and `created_at`. The private key digest detects replacement without asking for a passphrase and does not store private key contents. A completed creation is removed from `creations` in the same save that adds the key.
- Loading v3 is strict: invalid or trailing JSON, duplicate or case-variant fields, unknown fields, unsupported versions, missing `keys`, invalid metadata, and duplicate IDs/fingerprints/paths are rejected, without automatic restore or overwrite.

Authoritative section: [保存スキーマと互換性](../../application-contract.md#保存スキーマと互換性) (Japanese).

### Migrating from v1 / v2

- v1 and v2 (v0.10.0 and earlier) are converted to v3 in memory on load. Loading alone does not change the file. Loading the old schema is also strict: unknown fields, wrongly typed arrays, trailing JSON content and missing required arrays (`keys` / `hosts` / `registrations`) are rejected. The conversion carries over only keys that have a private key path and unfinished local creations. It does not carry over destinations, associations and archived removal records (`hosts` / `associations` / `removed_keys` / `removed_hosts`); the records and history of legacy registrations, connection checks, config, SSH config edits, onboardings and deletions (`registrations` / `history` / `configs` / `config_edits` / `onboardings` / `deleted_keys` / `deletions`); public-only keys and keys with an unfinished legacy key deletion; or legacy fields such as a key's `connection_check` and a destination's `config_key_id`.
- The notice gives a count for each non-empty record type. Each unfinished SSH config edit is listed with its path and a request to check that file manually. Before migrating, `keys list` prints this notice on stderr. Changing commands (create / import / remove / resume) show the same notice after taking the lock and before the confirmation prompt, so the confirmation also approves the migration. The first successful state save exclusively writes the bytes of the original `state.json` to a new `state.v<N>-<UTC time>.json` in the same directory (with a numeric suffix on collision), then updates `.bak` and saves v3. If creating the archive fails, the state is not changed. If an earlier attempt already left an archive with the same bytes, a retry reuses it instead of adding an identical archive. The archive is never overwritten or deleted automatically, so records that are not carried over remain on disk.
- Older binaries cannot read v3. To downgrade, exit the app and manually restore the archive as `state.json`. Keys created or imported after the migration are not included in the restored state.

Authoritative section: [v1 / v2からの移行](../../application-contract.md#v1--v2からの移行) (Japanese).

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
