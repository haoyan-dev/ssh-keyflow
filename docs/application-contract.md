# Application contract

## Source-hosted one-line installers

`install.sh` (POSIX sh on Linux/macOS) and `install.ps1` (Windows PowerShell 5.1
or PowerShell 7) are served separately from `main` through
`raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/`. They consume the existing
Portable release archives, including v0.2.0, without changing release assets.
The one-line entry points are `curl -fsSL <install.sh URL> | sh` and
`irm <install.ps1 URL> | iex`. Source delivery follows `main`; version selection
controls the Portable release downloaded by the installer.

The default repository is `haoyan-dev/ssh-keyflow`. `--repository` / `-Repository`
or `SSH_KEYFLOW_REPOSITORY` override it. Latest resolution requires a stable,
nondraft, nonprerelease version; explicit `--version` / `-Version` may select a
prerelease. OS/CPU selection supports the same six targets as Portable archives.
Download the archive and SHA256SUMS, require one matching checksum, and reject
unexpected members, links and nonregular archive entries before installation.
Both executable and `THIRD_PARTY_NOTICES.txt` are installed.

Default locations are `$HOME/.local/bin` and
`$env:LOCALAPPDATA/Programs/ssh-keyflow`. Both installers retain the v0.1.0
ownership formats and add notice ownership (`notices-hash` in the POSIX receipt
directory, `noticesHash` in Windows receipt JSON). An intact legacy receipt with
no notices upgrades directly. Unowned, changed, or missing owned files stop
installation/removal. A failed replacement or receipt write restores previous
executable, notices and ownership hashes.

PATH changes require terminal consent or explicit `--add-to-path` /
`-AddToPath`. Piped source is never read as an answer. Noninteractive execution
without consent leaves profiles unchanged and prints a full launch path and
manual PATH command. PowerShell also treats host refusal to prompt under
`-NonInteractive` as declined consent. PowerShell argument parsing and
installation run in a child scope, including under `Invoke-Expression`, to
preserve caller variables, strict mode and error preferences.
Profile changes use same-filesystem staging and atomic replacement, preserving
existing metadata and recovery copies until ownership is recorded. A PATH setup
failure restores the profile and the previous executable, notices and receipt.
PATH stanzas remain idempotent and byte-owned; uninstall removes only the exact,
unique owned stanza, executable, notices and receipt. Existing v0.1.0 stanzas
remain owned across upgrades. SSH keys, state, SSH configurations and remote
registrations are never modified. Application shutdown is required before
update/removal; PATH profile persistence takes effect in a new shell.

Isolated tests use loopback HTTP through `SSH_KEYFLOW_RELEASE_URL` and
`SSH_KEYFLOW_API_URL`. Normal defaults use HTTPS GitHub endpoints.

## v0.2.0 Portable release (ticket 04)

`scripts/package_release.py --version <tag> [--output <empty-directory>]` builds Windows/Linux/macOS amd64/arm64 with CGO disabled, VCS stamping disabled, trimpath and injected `main.version`. Existing nonempty output is refused. `--binaries <directory>` instead packages six prebuilt OS_ARCH binaries for artifact tests. Assets are staged before publication to the output directory.

The release contains exactly six archives and `SHA256SUMS` (seven assets), with no installer. Windows archives are zip; Linux/macOS archives are tar.gz. Each contains only its platform executable and `THIRD_PARTY_NOTICES.txt`. Unix executable mode is 0755, notice mode is 0644, tar uid/gid are zero. The manifest has six unique archive basenames with SHA256, two spaces, and LF. Notices retain dependency license/copyright text including platform-specific modules and Unicode data; collection evidence is in `docs/third-party-notices.md`.

Portable means freely placed executables. Storage remains `<Home>/.ssh/ssh-keyflow`; moving/replacing/removing the executable neither migrates data nor changes SSH keys, state, config, remote registrations, PATH, profiles or installer receipts. End users need OS OpenSSH, not Go. They exit the application before manual replacement/removal and retain the notice document when redistributing the binary.

Old v0.1.0 installers require one archive member and therefore reject the new archives. Use the new source-hosted installer for intact v0.1.0 installations. Manual binary replacement also invalidates the old ownership digest, so updates/removal may be rejected. README documents manual maintenance and optional manual cleanup of the exact owned PATH stanza; no automatic repair of changed receipts/profiles is added.

CI runs Go and Portable artifact tests on three OSes plus actual six-target crossbuilds. Release workflow uploads the seven staged assets after CI; implementation itself does not tag/push/publish. Fixture artifact tests and crossbuilds do not establish native runtime behavior on every target.

## v0.2.0 CLI (ticket 02)

`cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner, RunTUI}) int` is the command entry point. `cmd/ssh-keyflow` supplies the release version and exits with its result. Cobra v1.10.2 provides command parsing, flag validation, help and version output. `github.com/charmbracelet/x/term` v0.2.2 checks actual terminal descriptors, including rejecting nonterminal character devices. Home resolution is deferred until an operation actually needs it.

| Command | Inputs |
| --- | --- |
| `hosts list` | Optional `--json` |
| `keys list` | Optional `--json` |
| `registrations list` | Optional `--json` |
| `keys create` | Required `--name` |
| `keys import` | Required `--name`, `--public`; optional `--private` for a pair. Files remain at their existing paths. |
| `keys delete` | Required `--key` fixed ID or unique exact management name |
| `hosts add` | Required `--alias`, `--hostname`, `--user`, `--os`; `--port` defaults to 22; exactly one of `--key` or `--new-key`; Windows requires `--windows-account standard|administrator` |
| `register` | Required `--host` fixed ID or case-insensitive alias, and `--key` fixed ID or unique exact name |
| `unregister` | Required `--registration` fixed ID |
| `resume` | Required `--id` creation, onboarding, registration or pending local deletion ID shown by a list |
| `tui` | No operation arguments; terminal input and output required |

`hosts add`, `register`, `unregister`, and `resume` accept `--auth manual|password|key` (default `manual`) and `--identity` (required for `key`, path only). For unregister these mean fresh alternate authentication, never the key being revoked. `hosts add`, `register`, and `resume` also accept `--configure`, which offers managed configuration after successful registration and preserves separate Include consent.

All mutation commands require a real terminal before home initialization, storage writes or command execution. They use `app.Run` with a structured request, retaining the real input stream for OpenSSH. Confirmation reads one byte at a time without buffering ahead; only exact `yes` after trimming surrounding whitespace approves. No `--yes` or credential flags are provided.

Exit statuses are success 0, execution failure/cancellation 1 and usage/argument failure 2. Argument failures include incomplete/invalid options and ambiguous or missing managed targets (`app.ErrInvalidInput`). Usage errors include the selected command's usage on stderr. Runtime diagnostics have no usage text. Explicitly requested manual guidance succeeds while recorded registration/connection state remains unknown/unverified. Unexpected fallback from an automatic operation fails.

`--help` and `--version` resolve no home, check no SSH dependencies and initialize nothing. Version output is `SSH KeyFlow <version>`. No-argument invocation prints help and exits 0 when either input or output is nonterminal, including terminal stdin with redirected stdout; explicit `tui` requires both terminal descriptors and otherwise fails with status 2. When both descriptors are terminals, no-argument invocation and `tui` call `Options.RunTUI(ctx, app.Options)` with resolved home, original input/output and runner; the executable supplies `tui.Run` as that callback. No legacy menu is launched from the executable.

Lists call `app.Snapshot` without SSH checks or writable-storage probes. Plain output is a table of the selected records and saved progress. JSON output is one object followed by a newline:

- `hosts list`: `hosts`, `onboardings`, `discovery`.
- `keys list`: `keys`, `creations`, `deletions`, `deleted_keys`, `discovery`.
- `registrations list`: `registrations`, `history`.

Record arrays use the documented persisted field names and are empty arrays rather than null. `discovery` contains read-only unmanaged candidates and Include review notes; managed records are in their corresponding arrays. It is advisory text rather than an additional structured API. Ordinary `keys list` and `hosts list` also print this information under a labelled discovery section after their managed records. Record and file metadata can include private-key paths and digests, never private-key contents or credentials. Integrity review warnings go only to stderr while validated records remain readable. Invalid schema/state fails with status 1, no JSON, and preserves the source state.

## TUI interaction contract (TUI UX tickets 01–06)

`tui.Run(ctx, app.Options) error` is the terminal UI entry supplied to `cli.Options.RunTUI`. `tui.NewProgram(ctx, options, ...tea.ProgramOption)` exposes the same entry through Bubble Tea's public Program for embedding and event tests. Bubble Tea v2.1.1, Bubbles v2.2.1 and Lip Gloss v2.0.6 provide terminal lifecycle and UI components. Startup, navigation, discovery and Tasks use read-only `app.Snapshot`; they neither initialize storage nor probe SSH.

Lists are Destinations, Keys, Registrations and Tasks. Tab/Shift+Tab switch categories on lists/entity pages. Names, login targets and independent registration/connection/configuration evidence identify records; fixed IDs remain inspectable and are used in requests. `/` searches displayed identity, fingerprints and IDs, case-insensitively; Enter accepts, Esc leaves editing with the query retained, and x clears. Each category retains its query, selected fixed identity and position across details/help/category changes. F5 reloads saved state only, preserves identity across reorder and announces nearest-row selection when the selected record disappears. h opens history and g opens discovery; both are read-only. Missing relationships fail validated loading or yield requires-review information, never inferred mutation targets.

Actions c/i/a/r/u/d/s mean create/import/add/register/unregister/delete/continue saved work. Actions bind to the visible entity, result or task; hidden list selections are never a fallback. Related pickers show destination login or key fingerprint/private-key availability, support search and retain caller drafts on Esc. Ineligible actions explain blockers. Fixed IDs are re-resolved before Review and again before terminal handoff; changed/missing target identity retains the draft and stops continuation. Presentation eligibility remains advisory; locked backend validation decides execution safety.

Add uses Destination -> Key -> Access method/options -> Review. OS is explicit; Windows alone exposes account type. Existing managed key and new-key intent are exclusive. Import remains a separate form. Manual authentication and configure=no are defaults; only key authentication exposes an identity path. Local creation/deletion recovery has no remote authentication/configuration inputs. Hidden inapplicable values are cleared when controlling choices change. Step continuation and Ctrl+S validate through shared request/destination rules, retain valid values and focus the first invalid field. No password/passphrase input exists.

Tab/Shift+Tab traverse form/overlay controls. Enter advances; Ctrl+S validates all applicable steps and opens Review. Esc moves back one step; first-step Esc, Cancel and Ctrl+C leave clean drafts directly or offer Keep editing / Discard draft, defaulting to Keep editing. Review Edit/e/Esc restores the exact editing position; its keyboard controls are Continue to terminal, Edit, Cancel and Help. Labelled Help retains origin, draft, caret, query and viewport. q/?/action letters remain literal text in editable inputs. Outside drafts/help/native execution, q/Ctrl+C exits; help consumes Ctrl+C first. Search retains normal text semantics. Ordinary key input and bracketed paste update the active form or search at its caret. Text remains literal, including action letters; paste does not activate controls or execute operations. Native terminal acceptance is tracked separately in the implementation verification record.

Review describes the resolved action/target/login, full key fingerprint or creation intent, method, configuration request, expected effects and unresolved runtime checks. It performs no operation. Continue hands off once through `tea.Exec` to shared `app.Run`; its exact terminal approvals, locks and guards remain authoritative. `app.ReadConfirmation` is shared with CLI, reads only its line and never reads ahead into native credential input. OpenSSH receives original terminal input/output. Native terminal interruption returns to TUI with diagnostic/error and saved checkpoint evidence; cancellation never implies rollback. Native/control output is sanitized for diagnostic display and is not replayed as result-page keyboard actions.

An operation result retains typed current-session reports, returned error/diagnostics, request/target and a separately refreshed Snapshot. Registration, specified-key verification, managed alias generation and main-config Include are independent facts. Manual nil-error guidance means Manual action required; a public-only key means verification unavailable. Saved registration plus failed verification preserves registration; alias-generation denial and Include denial/failure name their distinct saved effects. Only authoritative no-effects denial permits Cancelled before changes. Missing reports/interruption mean unknown, not success. Old Snapshot evidence is labelled recorded and cannot prove this request completed. A failed Snapshot reload preserves facts and prior views as stale; only Snapshot Refresh, Help and Exit remain available until reload succeeds. Refresh never repeats execution or asserts live remote state.

Results expose keyboard-focusable continuation, View task, View entity, Refresh, Exit and Help where applicable. s/t/v use exact result identities; r Register again exists only for manual Register with saved destination/key. Diagnostics/manual commands remain vertically scrollable until leaving the result. Manual Register for an unrecorded or previously verified pair may save no intent; it neither fabricates a Task nor replaces prior verification. The result then says No saved recovery task and directs Register again after exit/restart. Include denial before an edit intent likewise may leave only session instructions alongside a saved managed alias. Configuration continuation uses supported Register/Add with explicit configuration request and explains repeated reconciliation/registration; there is no verify-only or config-only operation.

`tui.ProjectTasks(app.View) []Task` is a pure projection from one saved view. Task identities derive from saved operation IDs/config paths, with deterministic priority: requires-review, uncertain/interrupted, manual follow-up, verification follow-up. Related onboarding/creation/registration work is grouped, while independent pending deletion remains separate. Incomplete linked creation continues onboarding; an existing registration becomes the continuation after creation. Current unregister intent supersedes stale onboarding/verification, and completed removal never prompts re-registration. Complete verified/private or complete public-only registrations and completed creations are not perpetual tasks. Public-only verification unavailability remains in details. Incomplete config evidence is grouped only with a safely identified parent; otherwise guidance is read-only. Missing/contradictory links never receive a guessed resume ID. Deleted-key tombstones/history are read-only. Tasks do not persist workflow state, probe dependencies, mark done, dismiss, retry automatically or bypass deletion/alternate-authentication guards. Empty Tasks means no recorded follow-up, not universal verification.

Arrows/PgUp/PgDn and Home/End expose full wrapped values, errors and instructions without horizontal-only access. Layout reserves header/status/controls and keeps focused content usable at 80x24, 60x18 and minimum 32x12. Below minimum, only resize guidance and exit are available; drafts/focus remain and mutation/handoff cannot start. Resize restores the previous screen without execution. Labels supplement color. Native Windows evidence, automated event checks, exact source commits and remaining gates are recorded in `.scratch/tui-ux/coverage-06.md` and `docs/verification.md`. Native Linux/macOS and real remote SSH behavior remain unverified by these fixture tests.

## Optional operation reporting (TUI UX ticket 01)

`app.Options.Report func(app.Report)` adds synchronous informational facts to the existing `app.Run(ctx, Options) error` boundary. The sink receives authoritative target/checkpoint facts for import, creation, onboarding, registration, verification, alias, Include, manual guidance, confirmation and execution. It is not an approval callback or executor; consumers must not mutate/reenter operations from it, including while the operation lock is held. It contains identities, paths and typed evidence, never credentials/key contents/native input. `Saved` means successful checkpoint persistence, including pending/unknown evidence, not whole-operation completion. An allocated target identity before a failed save is not itself durable recovery evidence. Absent facts mean not recorded. Reports supplement returned errors/diagnostics and read-only Snapshot without parsing stdout. Without a sink, CLI output, exit behavior, requests, exact confirmations and version-1 state schema are preserved.

## v0.2.0 structured operations (ticket 01)

`app.Run(ctx, app.Options{Home, Input, Output, Runner, Request, Confirm})` is the common application entry for CLI and TUI. A non-nil `Request` selects one operation directly; it does not supply simulated menu answers. The legacy nil-Request menu remains temporarily available while the CLI and TUI tickets replace its entry point.

`Request.Validate()` checks argument completeness and supported authentication without accessing storage or invoking commands. Request fields:

| Operation | Inputs |
| --- | --- |
| `list` | No mutation inputs; prints the read-only discovery listing |
| `import` | `Name`, `PublicPath`, optional `PrivatePath`; the public path is explicit even for private-pair imports |
| `create` | `Name` |
| `add` | `Destination` (alias, hostname, user, port, OS, Windows account), exactly one of existing `Key` or new-key `Name`; port 0 defaults to 22 |
| `register` | `Host`, `Key` |
| `unregister` | `Registration` fixed ID |
| `delete` | `Key` |
| `resume` | `Resume` fixed creation, onboarding, registration or pending local deletion ID |

`Key` resolves a fixed ID first, otherwise a unique exact management name. Duplicate names require a fixed ID. `Host` resolves a fixed ID first, otherwise a case-insensitive alias. `Destination` is the existing `state.Host` shape; new host IDs are allocated by the application.

For `add`, `register`, `unregister` and `resume`, `Bootstrap` is `manual` (also the default), `password` or `key`. `Identity` is required for `key` and contains only a private-key path, never a secret. `Configure` offers generation of the managed SSH configuration after successful registration; Include changes require their own confirmation.

`Confirm func(message string) (bool, error)` handles approvals for structured operations. The output stream contains the target/preview printed immediately before each approval. Creation/onboarding resume previews include saved IDs, stages, key names and paths, the destination and available key fingerprint, and the work to resume before requesting consent. A missing callback fails closed. A false answer returns `app.ErrCancelled`; this is not success and does not imply that earlier completed steps were rolled back. Confirmation input is distinct from the original `Input` passed unchanged to native OpenSSH. Structured operation arguments are copied at entry so resume dispatch does not modify the caller's request.

`app.Snapshot(app.Options{Home}) (app.View, error)` exposes validated persisted `State`, read-only `Discovery` text and an optional `Review` integrity warning. It does not initialize storage, probe SSH commands or require a writable home. Schema errors fail rather than returning an empty view. Integrity warnings preserve read access and do not authorize mutation. JSON consumers should select their documented output fields from this view; it contains provenance and file metadata, never private-key contents.

All existing mutation-wide locks, intent checkpoints, remote reconciliation, alternate-authentication checks and ownership guards remain in the shared implementations. An explicitly requested manual registration/revocation returns guidance with unknown state preserved. Manual/default structured unregister first confirms the exact user, host, port, fingerprint and removal target before recording unregister intent/history. Missing or refused consent leaves persisted state unchanged; approved guidance records unknown remote outcome. Automatic fallback after a remote attempt continues to retain uncertainty evidence. An automatic operation that falls back to guidance returns its error to structured callers instead of reporting automatic success. If specified-key connection verification fails or does not return its success marker, the completed registration and failed connection evidence remain saved; the operation returns an error before configuration generation. CLI reports status 1 and TUI presents the stopped operation.

The shared-operation ticket needs no new dependency; CLI/TUI libraries will be added and pinned by the tickets that use them.

## Legacy regression entry and shared process boundary

Module: `ssh-keyflow` (Go 1.26). The shared application uses the standard library; the executable CLI adds the dependencies documented above. The nil-Request number menu described below remains only as a legacy regression entry; the executable uses the v0.2.0 CLI.

`app.Run(ctx context.Context, options app.Options) error` in `internal/app` is the application boundary for subsequent tickets and tests. `Options` accepts `Home` (required), `Input io.Reader`, `Output io.Writer`, and `Runner app.Runner`. Nil streams default to the process terminal streams; a nil runner selects `SystemRunner`. Tests use temporary homes and replace only the external process boundary. Extend this application package for tickets 02–04; keep behavior tests in `package app_test`, calling `Run` and observing CLI output and user-visible file effects.

`Runner` has `LookPath(string) (string, error)` and `Run(context.Context, Command) error`. `Command` carries `Name`, `Args`, `Stdin`, `Stdout`, `Stderr`, and `Interactive`. Execution uses argument arrays with no shell. Set `Interactive: true` for credential-bearing operations. `SystemRunner` requires a character-device input file for those commands and passes that same file to the child. Menu prompts read one byte at a time and retain no unread buffer. Keep credential input with ssh/ssh-keygen; do not add a scanner or buffered reader over the shared terminal. Capturing `ssh-keygen -y` stdout captures only the derived public key; its input and error stream remain attached to the supplied terminal streams. A successful derivation does not mark connection verification successful.

CLI menu choices are 1 add/register, 2 list, 3 register another public key, 4 unregister, 5 create/import, 6 delete local key, 0 exit. All listed choices are implemented. Choices 4 and 6 are separate, explicitly confirmed operations. Import (`5/i`) accepts an existing public or private path, its companion (blank means public-only when a public key was selected, or `<private>.pub` when a private key was selected), a management name and exact `yes` confirmation. `~/` expands against `Options.Home`; other relative paths use the process working directory. Source files remain at their existing paths. Public-only imports record an empty private path. Imports do not register anything remotely.

`5/c` confirms a management name and automatic Ed25519 path, then gives the original terminal to native `ssh-keygen -t ed25519 -f <path>`. Passphrases, including an empty passphrase, stay in the native process. `5/r` resumes a numbered creation after confirmation. `1/n` collects alias, hostname/IP/Tailscale name, login user, port (default 22), explicit Linux/macOS/Windows selection and Windows account type, then offers a managed key or a new key and confirms the destination. It immediately offers password/keyboard-interactive bootstrap, an existing private key, or manual guidance; automatic registration continues through target confirmation, specified-key verification and the configuration offer in the same operation. `1/r` resumes a numbered saved onboarding and offers bootstrap authentication or manual guidance again. Unicode display names, aliases and users are supported. `1/a` selects a saved onboarding for automatic registration. Menu 3 selects an existing host and managed key, confirms the fingerprint and destination, and uses the same registration flow without duplicating the host/onboarding. Manual guidance and creation resume remain available.

`main.version` defaults to `dev`; release builds inject it with `-ldflags "-X main.version=<version>"`. The executable's arguments and dispatch now follow the v0.2.0 CLI contract above.

## Persisted schema

The managed directory is `<Home>/.ssh/ssh-keyflow`, with `keys/`, `state.json`, `state.json.bak` and transient `state.lock`. Empty state is version 1 with non-null empty `keys`, `hosts` and `registrations` arrays. It is saved on the first confirmed mutation. The dedicated generated configuration is `config` in this directory; ownership is recorded by digest before it may be regenerated.

`state.State`, `Key`, `Host`, and `Registration` are exported from `internal/state`. JSON field names:

| Record | Fields |
| --- | --- |
| Key | `id`, `name`, `fingerprint`, `public_path`, optional `private_path` and `private_sha256`, `connection_check`, `origin`, `imported_at`, optional `created_at` |
| Host | `id`, `alias`, `hostname`, `user`, `port`, `os`, optional `windows_account` (`standard` or `administrator`), optional `config_key_id` |
| Registration | `id`, `key_id`, `host_id`, `status`, `connection_check`, `operation`, `stage` |
| Creation | `id`, `key_id`, `name`, `private_path`, `public_path`, `created_at`, `stage`, optional `private_sha256` and `fingerprint` |
| Onboarding | `id`, `host_id`, `key_id`, optional `creation_id`, `stage`, `created_at` |
| RegistrationEvent | `registration_id`, `at`, `status`, `connection_check`, `stage` |
| Config | `path`, optional `sha256`, `pending_sha256` |
| ConfigEdit | `path`, `before_sha256`, `after_sha256`, optional `backup_path`, `complete` |

Key IDs are random stable `key-<128-bit hex>` identities. Renaming a display name must preserve its ID. Fingerprints use the SHA256 OpenSSH public-blob format. Private file digests detect replacement without asking for a passphrase at startup. Paths are absolute. Import provenance is `origin: imported` plus a nonzero RFC3339Nano `imported_at` in UTC and no `created_at`. Creation provenance is `origin: created`, nonzero UTC `created_at`, empty `imported_at` and a private path. Only one record per fingerprint and one registration per key/host pair are permitted.

Optional top-level `creations` and `onboardings` arrays extend version 1 without rewriting old import-only documents. Creation stages are `planned`, `pending`, `complete`; complete records must match the created key's ID, paths, timestamp, digest and fingerprint. Onboarding stages are `key_pending`, `planned`, `manual`; foreign keys and creation identity are validated, and `manual` requires its registration pair. Local deletion retains registration evidence, stable references and complete key provenance in deletion tombstones; completed creation/onboarding records may reference those tombstones. Unknown fields and inconsistent history still fail closed.

Current values are `connection_check`: `unverified`, `verified`, `failed`, `unknown`; host `os`: `unknown`, `windows`, `linux`, `macos`; registration `status`: `pending`, `registered`, `removed`, `unknown`, `failed`; `operation`: `register`, `unregister`; `stage`: `planned`, `pending`, `complete`, `unknown`, `failed`. These independent fields preserve the distinction between recorded registration history, connection verification and operation progress. Tickets implementing richer progress should extend the schema and tests together.

Load rejects malformed or trailing JSON, duplicate fields (including nested fields), unknown fields, missing arrays, unsupported versions, invalid metadata, duplicate identities/aliases/fingerprints/paths and orphaned or duplicate relationships. It never restores or overwrites a bad state automatically. Missing or replaced managed key files produce `requires review`; reads remain available and all imports stop. Fingerprints and private digests are rechecked before committing an import.

## Operation-wide exclusion and persistence

Use `state.WithLock(home, func(store *state.LockedStore) error { ... })` for the **whole mutation**, including user confirmation, key generation or remote side effects. `LockedStore.Load()` loads validated state and `LockedStore.Save(document)` persists a checkpoint without releasing the lock. A store is invalid outside its callback. Never nest `state.Update` or `WithLock` inside an existing locked operation. `state.Update` is a convenience for a single load/change/save operation only.

Acquire once, load state, verify managed-file integrity, collect/confirm intent, save intent before new side effects, execute the external operation, and save progress/results while retaining that same lock. Ticket 01 import only reads external files and writes the confirmed final import. Ticket 02 saves creation intent and a pending checkpoint before native generation, with explicit onboarding identity when applicable. Standalone creation never infers ownership from the last onboarding. State-layer validation checks schema and relationships; application flows also enforce file integrity and operation-specific safety conditions. The private `integrity` helper in `internal/app` is shared by the flows and can be used by later files in the same package.

Exclusive creation of `state.lock` blocks competing imports throughout validation and confirmation. A pre-existing lock is never automatically removed; users must determine ownership before recovery. Startup probes storage with a unique temporary file. Saves validate state, write and sync a same-directory temporary file, atomically replace the previous-valid backup, then replace `state.json`. Backup failure stops before replacing current state. Failed and cancelled imports retain all source files and never announce success.

## Discovery and verification limits

Automatic key names use `keys/id_ed25519_<sanitized-name>_<UTC timestamp>` with a numeric collision suffix. Both paths and journal reservations are checked, including a recheck after confirmation; unmanaged files are never adopted during that confirmation. Resume keeps the recorded identity and path. No files means generation may retry at that path; private-only means derive with native interactive `ssh-keygen -y` and exclusively create the missing public file; a complete pair is derived/checked without regeneration. Public-only, empty/nonregular files, a changed pinned digest/fingerprint or a missing observed file require review. Partial files are never automatically deleted. An unfinished creation must be resumed/reviewed before another pair is allocated. The operation lock coordinates application instances; users must not concurrently edit reserved managed paths while native keygen owns the terminal.

New destinations reject case-insensitive literal alias collisions through recursive unmanaged Includes. Matching wildcard Host defaults produce review warnings and preserve host/manual registration; no Host or Include is written. Unreadable/unresolved configuration stops onboarding for review, and Match exec remains unexecuted.

`internal/ssh.StandardRegistration(system, windowsAccount, public)` returns `RegistrationPlan{Enable, Target, Script, Shared}` for manual guidance and automatic registration. Scripts embed only validated key type/blob, discard comments, preserve unrelated authorized_keys bytes, recognize option-prefixed matching identities and avoid duplicates. POSIX scripts check ownership/modes and symlinks; Windows scripts check reparse points and ACLs, preserve UTF-8 BOM/newlines, distinguish per-user versus shared administrator targets and require separate approval for the shared target. Unexpected permissions/configuration require review; existing ACLs are never reset. Windows script parameters default to `$HOME` and `%ProgramData%/ssh`; fixture tests supply isolated directories.

Onboarding saves registration `status: unknown`, `connection_check: unverified`, `operation: register`, `stage: pending` and onboarding `stage: planned` before offering bootstrap authentication. Choosing manual guidance, or falling back to it, saves onboarding `stage: manual` before printing copyable guidance. Printing or resuming guidance never claims successful registration/connection. Automatic registration reuses this unique pair and shared script, appending evidence to `history` rather than replacing earlier events. For interactive SSH, keep terminal Stdin unchanged; carry scripts in a safely quoted remote command payload, never a `strings.Reader` replacing terminal input. Automatic SSH invocation is described below.

Discovery inspects key candidates directly in `~/.ssh/` and the managed `keys/` directory. It reads `~/.ssh/config` and recursive Include globs, resolving relative Include paths from `~/.ssh/`. Quoted paths are supported. Cycles, unmatched/unreadable Includes, malformed lines and unresolved `%`/environment/home-user expressions are annotated. A depth limit of 64 and file-visit limit of 4096 bound discovery. Host patterns, IdentityFile declarations and Match scopes are discovery evidence requiring review; they are not evaluated effective connection settings. `Match exec` is never executed. Discovered files are not imported, rewritten, moved or removed.

## Automatic registration and connection evidence

The whole registration/config operation runs under one `WithLock`, including prompts and native processes. Bootstrap is explicitly password/keyboard-interactive or an existing regular private-key path. Every SSH command is interactive with the original `Options.Input`; scripts travel in a safely quoted `sh -c` argument or a UTF-16LE PowerShell encoded argument. Large PowerShell payloads use a gzip/base64 bootstrap inside that encoded argument to stay within Windows command-line limits; decompression preserves the script and terminating-error behavior. PowerShell errors terminate the operation. No credentials are stored or read by the application.

A read-only probe checks native OS, current login, Windows account type and effective default `sshd -T -C` configuration for the SSH connection. It requires standard AuthorizedKeysFile, public-key authentication, StrictModes, no AuthorizedKeysCommand and no DNS-dependent Match. The user confirms the exact observed target and that the running daemon uses its default configuration; a daemon launched with custom `-f/-o` requires manual review. This confirmation is necessary because querying the default configuration cannot prove the running daemon's startup flags. Windows administrator registration additionally requires explicit shared-file impact consent. Each remote read/write repeats the probe and compares the exact target. Shared script ownership/mode/ACL, symlink/reparse and encoding checks apply to both writing and read-only reconciliation. An unreadable or unsupported effective configuration stops automatic writes and prints manual guidance.

Direct SSH uses `-F none`, explicit hostname/user/port, `ControlMaster=no` and `ControlPath=none`. The first bootstrap uses `StrictHostKeyChecking=ask`; later steps require `yes`. `UserKnownHostsFile` explicitly names both `<Options.Home>/.ssh/known_hosts` and `known_hosts2`, quoted separately for spaces; global records retain native defaults. Existing records are never cleared or bypassed. Changed-key failures stop the operation. Paths containing control characters, double quotes or SSH expansion tokens `%/$` require manual setup. Configured ProxyJump, aliases, Match and other routing options are not loaded with `-F none`; failure guidance calls out that limitation instead of retrying through a different route.

The target-key check supplies only the selected managed private key, disables identities from config, agent and certificates, and permits only public-key authentication (password, keyboard-interactive, GSSAPI and hostbased are disabled). Multiplexed connections are disabled. It requires an exact remote success marker. Public-only keys remain `unverified`. A successful append/read reconciliation records `registered` independently of a failed verification; no bootstrap success implies target-key verification.

Before a remote write, save the existing registration ID with `operation: register`, `status: unknown`, `stage: pending` and `connection_check: unverified`, plus an event. A failed/lost response adds an `unknown` event. On resume, any existing event, registered status or unknown stage requires read-only reconciliation first. Present means no append; absent requires fresh explicit retry approval; unreadable/ambiguous results stop. Completed registration and verification each append events with UTC RFC3339Nano timestamps. Existing events retain their order and contents. `recordRegistration` is the application checkpoint helper; state validation checks event references and values, not arbitrary callers' append-only behavior. Registration and removal events append their operation (`register` or `unregister`); legacy events without operation remain readable. Removal retains prior events and registration identity.

## Managed config and Include consent

`configs`, `config_edits` and `history` are optional version-1 arrays. `Host.config_key_id` selects one managed private key per alias and must reference an existing registration pair. Config generation checks all selected aliases against recursively included unmanaged files; literal collisions stop and wildcard defaults warn. It never executes Match commands.

Generation requires CLI approval and a usable managed private key. A dedicated file without a Config record is unowned and preserved. A missing/changed owned file stops registration before SSH; a matching current or pending digest supports crash recovery. Before writing generated content, save `pending_sha256`, then atomically replace the file and save `sha256` while clearing the pending digest. Generated quoted IdentityFile/User settings support spaces and end in `Host *` to reset Include scope. Unsupported expansion/quoting requires manual setup.

The main `.ssh/config` remains unowned. The CLI previews the exact prepend `Include "ssh-keyflow/config"` before every Host/Match, explains precedence and offers the same manual line. Only exact `yes` authorizes this edit; decline leaves the file unchanged. An existing direct or recursively included managed file is not duplicated; scoped/ordered Includes require manual review. After approval, recheck owned digests, every selected alias and the previewed main-file bytes. Save a ConfigEdit intent before exclusively creating an exact unique backup of an existing main file, then atomically prepend while retaining its newline style. No backup is needed when creating a missing main file. Symlink/reparse/nonregular paths stop edits.

Interrupted Include edits reconcile the saved before/after digests and original backup before retry. Already-applied content completes the intent without rewriting; unchanged content requires fresh approval and reuses the original backup. A changed/missing required backup or unexpected main-file content stops for review. Completed ConfigEdit records do not establish ongoing ownership of the main config. The operation lock excludes other application instances; users must not concurrently edit involved config/Include files during the operation.

Unregister/local deletion reuse these digests and whole-operation exclusion, regenerate only owned dedicated content, and preserve the explicitly approved Include/backup evidence. They keep registration and connection results separate, reconcile unknown remote effects before retry and preserve imported/created provenance.

## Public-key revocation

Choice 4 selects a saved registration under the operation lock and validates local identity and owned configuration. A fresh password/keyboard-interactive connection or an explicitly selected alternate private key must authenticate independently of the target fingerprint. Native interactive `ssh-keygen -y` derives the alternate fingerprint, rejecting the target even at another path; its private-file digest is rechecked before each SSH command. Connections disable multiplexing, agents, certificates, default identities, GSSAPI, host-based authentication and PKCS#11. Key mode permits only the selected public-key identity; password mode disables public-key authentication. Secrets stay with native processes. The first connection requests host-key confirmation; subsequent connections require the known host key.

The alternate probe verifies the account, OS and target, followed by explicit effective-daemon-configuration confirmation. Every attempt reconciles read-only before mutation. Absent records `removed`/`complete` without a remote write. Present saves `unregister`/`planned`; shared administrator targets require separate impact approval. Final confirmation identifies host, user, port, target and fingerprint. Save `unknown`/`pending` and append history before sending removal. Completion requires a successful guarded command ending with its success marker. A failed/lost response remains unknown; retries start read-only. Unreachable hosts or unavailable alternate access print manual guidance and preserve local files. The current session is never proof of alternate access.

`ssh.StandardRevocation` returns `Target`, `Shared`, `Script` and `InspectScript`. Automatic execution and manual guidance use the same removal script; inspection and mutation share one platform template. Parsing removes exact type-plus-blob matches only in key fields, respects quoted/escaped authorized_keys options and rejects malformed quoting. Unrelated bytes, comments, BOM, line endings and missing final newlines survive successful removal. Already-absent removal is idempotent.

Linux/macOS scripts require Perl, inspect `/etc/ssh/sshd_config`, and refuse custom sources, Includes/Match and an existing secondary `authorized_keys2`. Symlinks, unsafe ownership/modes, nonregular files and hardlinks stop the operation. Mutation shares the registration lock and checks file identity. Before replacement it copies the complete original to a unique `authorized_keys.keyflow-backup-*`, preserves owner/modes/ACLs, synchronizes the backup and a complete staged replacement, and rechecks target identity, metadata and exact bytes. It renames the completed replacement and synchronizes the containing directory; interruption leaves the original or completed output recoverable alongside the backup. Inspection creates no lock/file. Leftover operation locks or staged files after interruption require manual review before retry; do not overwrite a changed live target with a backup automatically. Windows validates absolute paths, reparse points, hardlinks, UTF-8 configuration, account membership and existing owner/ACLs. Windows synchronizes a unique backup and staged output with the original owner/SDDL, rechecks target identity and exact contents, then uses `File.Replace` with a second replacement backup. Existing owner/ACLs must survive exactly. Backups remain for recovery; failed staging never truncates the live target, and changed targets are refused. Shared administrator mutation requires an already elevated session and script approval; it never elevates or resets ACLs. Unsupported effective configurations require manual review.

Successful removal clears only the matching host/key binding in owned dedicated configuration and regenerates other aliases using pending/current digests. Main Include, backup and `config_edits` evidence remain intact. Local key files require separate choice 6 approval.

Windows mutation holds an exclusive `ssh-keyflow-revocation.lock` opened with create-new and delete-on-close semantics, serializing atomic replacements; an existing lock is refused. Original-file handles deny concurrent writers, including registration. Inspection creates neither lock nor backup.

## Guarded local deletion and retained history

Choice 6 requires every recorded registration for the selected key to be `removed`/`complete` with operation `unregister`; the prompt explains that only recorded destinations are known. Pending/unknown registration evidence, changed/unsafe paths, owned-config drift and unfinished Include edits stop deletion. Public-only keys are supported.

Optional version-1 `deletions` records retain stable deletion/key IDs, timestamp, expected public/private digests, stage and per-file completion flags. Save intent before side effects, recheck bytes and paths immediately before removal, and checkpoint each file. Resume accepts absent files against saved intent, but refuses changed files or reappeared paths already marked removed. Other managed keys and owned configuration must still pass integrity checks. The application lock excludes other instances; users must not concurrently edit involved files.

Completion moves the entire key record to optional `deleted_keys` with deletion time/ID. Preserve key ID, fingerprint, paths, imported/created provenance, creation/onboarding references, registration IDs and all history events. Tombstones reserve fingerprints and paths against accidental reuse. Listings show deleted keys and unfinished intents. Repeating unregister for a tombstoned removed key performs no SSH. Deletion retains Include/backup history and other aliases.

## Validation evidence and limits

Automated tests exercise `app.Run` with temporary homes and replacement external processes only. They cover bootstrap input preservation, target-key argument isolation, changed-host-key stopping, unknown-result reconciliation, append-only application evidence, menu 3, alias/Include collisions, digest drift, explicit consent/backups and interrupted Include recovery. Existing manual onboarding, creation resume, provenance and unmanaged-file regression tests remain active. The version test builds a temporary CLI executable with VCS stamping disabled because the managed worktree requires command-local Git trust.

POSIX automatic scripts execute in isolated Git Bash fixtures with substituted uname/id/stat/sshd boundaries, including standard/nonstandard effective targets, OS mismatch, metadata refusal, byte preservation and read-only retry. This is script-fixture evidence, not native Linux/macOS server validation. Windows PowerShell 5.1 executes automatic standard-account registration and read-only reconciliation against an isolated NTFS home, including real ACL creation and BOM/CRLF preservation. Administrator scripts pass parsing and encoded-error fixtures; elevated shared administrator ACL execution remains untested. Native `ssh -G` checks the actual argument arrays, spaced isolated homes/identity paths and disabled control sockets without connecting. Mocked CLI host-key failures verify application behavior, not a live OpenSSH handshake.

Ticket 04 adds CLI tests for alternate-fingerprint exclusion, identity drift, argument isolation, confirmations, read-only retries, unknown outcomes, shared-target approval, owned-config integrity, Include journal preservation and guarded deletion/resume/tombstones. Isolated Linux/macOS Git Bash fixtures execute actual guarded application removal payloads. Native Windows PowerShell/NTFS fixtures execute standard-account inspection/removal/retry and compare original bytes and ACL SDDL; refusal fixtures cover unsafe ACLs, paths, hardlinks, configuration and administrator account/elevation requirements. The uncached full suite and vet pass on integration `a2c21d0` plus ticket04 changes.

Live SSH password/existing-key login, changed-host-key handshakes, target-only authentication against a real server, native Linux/macOS and elevated Windows administrator server behavior still require external validation. No real user .ssh files or remote servers are modified by these tests. Cross-builds, when reported, are compilation evidence only.
