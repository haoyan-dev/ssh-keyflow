# Ticket 01 application contract

## Public application boundary

Module: `ssh-keyflow` (Go 1.26; standard library only).

`app.Run(ctx context.Context, options app.Options) error` in `internal/app` is the application boundary for subsequent tickets and tests. `Options` accepts `Home` (required), `Input io.Reader`, `Output io.Writer`, and `Runner app.Runner`. Nil streams default to the process terminal streams; a nil runner selects `SystemRunner`. Tests use temporary homes and replace only the external process boundary. Extend this application package for tickets 02–04; keep behavior tests in `package app_test`, calling `Run` and observing CLI output and user-visible file effects.

`Runner` has `LookPath(string) (string, error)` and `Run(context.Context, Command) error`. `Command` carries `Name`, `Args`, `Stdin`, `Stdout`, `Stderr`, and `Interactive`. Execution uses argument arrays with no shell. Set `Interactive: true` for credential-bearing operations. `SystemRunner` requires a character-device input file for those commands and passes that same file to the child. Menu prompts read one byte at a time and retain no unread buffer. Keep credential input with ssh/ssh-keygen; do not add a scanner or buffered reader over the shared terminal. Capturing `ssh-keygen -y` stdout captures only the derived public key; its input and error stream remain attached to the supplied terminal streams. A successful derivation does not mark connection verification successful.

CLI menu choices are 1 add/register, 2 list, 3 register another public key, 4 unregister, 5 create/import, 6 delete local key, 0 exit. Ticket 01 implements 2 and the `i` submenu of 5. Other choices explicitly report unavailable behavior. Import prompts accept an existing public or private path, its companion (blank means public-only when a public key was selected, or `<private>.pub` when a private key was selected), a management name and exact `yes` confirmation. `~/` expands against `Options.Home`; other relative paths use the process working directory. Source files remain at their existing paths. Public-only imports record an empty private path. Imports do not register anything remotely.

`cmd/ssh-keyflow` starts this boundary without arguments. `--version` prints `SSH KeyFlow <version>` without discovering the home, checking dependencies or initializing storage. `main.version` defaults to `dev`; release builds inject it with `-ldflags "-X main.version=<version>"`. Other arguments fail with usage and exit status 2.

## Persisted schema

The managed directory is `<Home>/.ssh/ssh-keyflow`, with `keys/`, `state.json`, `state.json.bak` and transient `state.lock`. Empty state is version 1 with non-null empty `keys`, `hosts` and `registrations` arrays. It is saved on the first confirmed mutation. The generated SSH configuration belongs to later tickets.

`state.State`, `Key`, `Host`, and `Registration` are exported from `internal/state`. JSON field names:

| Record | Fields |
| --- | --- |
| Key | `id`, `name`, `fingerprint`, `public_path`, optional `private_path` and `private_sha256`, `connection_check`, `origin`, `imported_at` |
| Host | `id`, `alias`, `hostname`, `user`, `port`, `os` |
| Registration | `id`, `key_id`, `host_id`, `status`, `connection_check`, `operation`, `stage` |

Key IDs are random stable `key-<128-bit hex>` identities. Renaming a display name must preserve its ID. Fingerprints use the SHA256 OpenSSH public-blob format. Private file digests detect replacement without asking for a passphrase at startup. Paths are absolute. Import provenance is `origin: imported` plus a nonzero RFC3339Nano `imported_at` in UTC; ticket 02 must extend the schema and validation deliberately for created-key provenance. Only one record per fingerprint and one registration per key/host pair are permitted.

Current values are `connection_check`: `unverified`, `verified`, `failed`, `unknown`; host `os`: `unknown`, `windows`, `linux`, `macos`; registration `status`: `pending`, `registered`, `removed`, `unknown`, `failed`; `operation`: `register`, `unregister`; `stage`: `planned`, `pending`, `complete`, `unknown`, `failed`. These independent fields preserve the distinction between recorded registration history, connection verification and operation progress. Tickets implementing richer progress should extend the schema and tests together.

Load rejects malformed or trailing JSON, duplicate fields (including nested fields), unknown fields, missing arrays, unsupported versions, invalid metadata, duplicate identities/aliases/fingerprints/paths and orphaned or duplicate relationships. It never restores or overwrites a bad state automatically. Missing or replaced managed key files produce `requires review`; reads remain available and all imports stop. Fingerprints and private digests are rechecked before committing an import.

## Operation-wide exclusion and persistence

Use `state.WithLock(home, func(store *state.LockedStore) error { ... })` for the **whole mutation**, including user confirmation, key generation or remote side effects. `LockedStore.Load()` loads validated state and `LockedStore.Save(document)` persists a checkpoint without releasing the lock. A store is invalid outside its callback. Never nest `state.Update` or `WithLock` inside an existing locked operation. `state.Update` is a convenience for a single load/change/save operation only.

Follow the import flow: acquire once, load state, verify managed-file integrity, collect/confirm intent, save intent before new side effects, execute the external operation, and save progress/results while retaining that same lock. Ticket 01 import only reads external files and writes the confirmed final import; later tickets must add their intent journal before creating keys or changing a server. State-layer validation checks schema and relationships; application flows must also enforce file integrity and operation-specific safety conditions. The private `integrity` helper in `internal/app` is shared by the existing flows and can be used by later files in the same package.

Exclusive creation of `state.lock` blocks competing imports throughout validation and confirmation. A pre-existing lock is never automatically removed; users must determine ownership before recovery. Startup probes storage with a unique temporary file. Saves validate state, write and sync a same-directory temporary file, atomically replace the previous-valid backup, then replace `state.json`. Backup failure stops before replacing current state. Failed and cancelled imports retain all source files and never announce success.

## Discovery and verification limits

Discovery inspects key candidates directly in `~/.ssh/` and the managed `keys/` directory. It reads `~/.ssh/config` and recursive Include globs, resolving relative Include paths from `~/.ssh/`. Quoted paths are supported. Cycles, unmatched/unreadable Includes, malformed lines and unresolved `%`/environment/home-user expressions are annotated. A depth limit of 64 and file-visit limit of 4096 bound discovery. Host patterns, IdentityFile declarations and Match scopes are discovery evidence requiring review; they are not evaluated effective connection settings. `Match exec` is never executed. Discovered files are not imported, rewritten, moved or removed.

Automated tests exercise public CLI/application entry points with temporary homes and injected process behavior. The version test builds a temporary CLI executable with VCS stamping disabled, because the managed worktree requires command-local Git trust. Windows native public-only and PTY pair import smoke were independently reported by the parent task. Linux/macOS native execution, network registration, host verification, generated SSH config, key generation, unregister and deletion remain with their assigned tickets. Cross-builds are compilation evidence only.
