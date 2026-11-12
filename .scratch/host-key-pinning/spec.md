Status: implemented and verified locally; awaiting human integration review

# Host key pinning

## Objective

The first `ssh <alias>` to a destination should not show a trust-on-first-use
(TOFU) prompt, and later connections should refuse any host key except the one
the user saw on the remote itself. The user pastes the remote's host public
keys into SSH KeyFlow, which keeps them in an app-owned `known_hosts` file.
The generated Host snippet for a pinned destination uses only that file.

Installing the user's public key does not change. The ADR 0002 placement
scripts still do it. Nothing is installed on the remote. The app never opens
network connections and never runs `ssh` or scripts (ADR 0001, ADR 0002).
Decision record: `docs/adr/0006-host-key-pinning.md` (accepted).
OpenSSH behaviour checks: `openssh-spike.md`. Work items: `issues/`.

## Positioning

Password managers with SSH agents, such as 1Password, manage the client-side
key but don't install keys on servers or handle host keys. SSH KeyFlow covers
the remote side: placing the key, pinning the host keys, and the Host config
that ties them together. Public-only keys whose private half lives in an agent
or on a FIDO token are supported throughout.

## Guide flow

Pinned status is read from the pin file when the guide is generated. Until the
destination is pinned, `hosts guide` and the TUI guide add two steps between
placement and the SSH config step:

| # | Where | Step | Status |
| --- | --- | --- | --- |
| 1 | Remote | Place public key | existing |
| 2 | Remote | Print host keys | new, read-only, only while unpinned |
| 3 | Local | Pin host keys (`ssh-keyflow hosts pin --host <id>`) | new, only while unpinned |
| 4 | Local | Append SSH config | existing; pinning lines once pinned |
| 5 | Local | ssh-agent autostart | existing, protected private keys only (ADR 0005) |
| 6 | Local | Connect | existing |

- After pinning, the user generates the guide again. Steps 2–3 disappear and
  step 4 gains the pinning lines. The step 4 instructions say which state
  applies.
- Steps 2–3 and the pinning lines appear only when the guide's local OS is the
  computer running the app. The pin file path and the `hosts pin` command
  belong to this computer. Other local terminals get today's guide.
- If the pin file can't be read, the guide says `Host keys: unknown (<reason>)`
  and generates no pinning lines and no pin steps. It never fails because of
  the pin file.

Step 2 commands, chosen by the destination OS:

- POSIX (Linux, macOS): `cat /etc/ssh/ssh_host_*_key.pub`
- Windows: `Get-Content "$env:ProgramData\ssh\ssh_host_*_key.pub"`, in
  PowerShell opened with **Run as Administrator** for both account types.
  Standard users can't read these files (spike check 4).

The step also gives an optional remote check (`ssh-keygen -lf <file>`) so the
user can compare fingerprints. Pinning doesn't depend on placement, so it works
for keys installed by cloud-init or already present.

## CLI

| Command | Behavior |
| --- | --- |
| `hosts pin --host <ref>` | In a terminal, prints the step 2 command for the destination OS and reads the pasted output until an empty line. Otherwise reads stdin, or `--file <path>`. Shows ignored lines and the type and SHA256 fingerprint of each key, then asks for a typed `yes`. Re-pinning an identical set succeeds without writing. |
| `hosts pin ... --yes` | Skips the typed confirmation only when no different keys are pinned. Required for non-terminal input. `--yes` never replaces. |
| `hosts pin ... --replace` | Required when different keys are already pinned. Shows the old fingerprints and a warning, then needs a typed `yes` in a real terminal. Non-terminal input with `--replace` is refused. |
| `hosts unpin --host <ref>` | Real terminal and typed `yes`, like `hosts remove`. Shows the pinned fingerprints and reminds the user to remove the pinning lines from their SSH config. Later guides use the unpinned form. |
| `hosts list` | Text adds a `HOST KEYS` column: `pinned (N)`, `not pinned` or `unknown`. JSON adds `host_key_pins`: destination ID → `[{type, fingerprint}]`. The field is omitted and stderr reports the reason when the pin file can't be read. |

Only managed (not removed) destinations can be pinned or unpinned. Input is
limited to 64 KiB. Exit codes follow the existing contract.

## TUI

- Destination list and details: `t` pins and `T` unpins the selected
  destination. On a key, the status line asks for a destination.
- Both hand off to the native terminal, like other changes. `t` uses the
  terminal paste flow above with `Replace` set, so different pinned keys show
  the warning with both fingerprint sets and still need a typed `yes`. There is
  no text area: paste and confirmation stay in the native terminal, where
  cancellation works as for other handoffs.
- The details view shows `Host keys: pinned (N)`, `not pinned` or
  `unknown (review the pin file)`. Help and footers list `t`/`T`.

## Storage

- File: `<Home>/.ssh/ssh-keyflow/known_hosts`. One line per key:
  `ssh-keyflow-<host-id> <type> <base64>`, LF only.
- The entry name uses the stable destination ID, so renaming the alias or
  changing HostName or port keeps the match. The `ssh-keyflow-` prefix can't
  collide with entries the user writes.
- Every write holds the existing state lock and uses the same atomic replace,
  symlink refusal and permission checks as `state.json`. Entries for other
  destinations keep their order and content. A destination's new keys take the
  position of its first old entry. The file is created only on the first pin.
- The state schema doesn't change. Pinned status comes from the file only, so
  older binaries ignore it and downgrading is safe.
- Reading the file is strict. Every line must be exactly three single-space
  fields with the prefix, a clean ID, a valid key and no duplicate, ending in
  LF. Comments, markers, CRLF and tabs are refused. A malformed file stops pin
  and unpin with an error that asks for manual review. Guide, list and details
  show `unknown`. The file is never rewritten silently.
- Removing a destination keeps its entries, matching the contract that removal
  changes nothing outside managed state and doesn't revoke access. If pins
  exist, the removal confirmation says to cancel and run `hosts unpin` first,
  because a removed destination can't be unpinned. Archived destination IDs
  are never reused, so orphaned entries are harmless.

## Parsing pasted output

- Accept only lines of the form `<type> <base64> [comment]` with type
  `ssh-ed25519`, `ecdsa-sha2-nistp256`, `ecdsa-sha2-nistp384`,
  `ecdsa-sha2-nistp521` or `ssh-rsa`.
- The base64 must be canonical standard encoding, and the algorithm inside the
  blob must match the type field.
- Drop comments and a leading BOM. Ignore blank lines. Report other lines
  (prompts, certificate types, `@cert-authority`, `@revoked`) as ignored, with
  control characters replaced and the text shortened. They never block valid
  keys.
- Limit input to 64 KiB and at most 16 keys. Require at least one valid key.
  Remove duplicate keys.
- The app writes lines from parsed fields only and never copies pasted text.

## Snippet changes

Pinned destinations get these lines after `IdentitiesOnly yes`:

```
    HostKeyAlias ssh-keyflow-<host-id>
    UserKnownHostsFile "<absolute home>/.ssh/ssh-keyflow/known_hosts"
    StrictHostKeyChecking yes
```

- Only the app file is listed. The alias never matches `~/.ssh/known_hosts`,
  and with `StrictHostKeyChecking yes` ssh never writes to it.
- The path is absolute with forward slashes, with the same quoting, `%`
  escaping and `${...}` refusal the guide applies to `IdentityFile`. A path
  with control characters or `${` gets a notice and no pinning lines.
- Unpinned destinations get exactly today's snippet. If the Host is already in
  the user's SSH config, the guide tells them to add the three lines
  themselves; the append script still stops on a duplicate alias.

### Public-only keys (separate contract change)

Before this change the guide generated no Host snippet or connection command
for public-only keys. They now get `IdentityFile "<key.public_path>"` with
`IdentitiesOnly yes`, so OpenSSH asks the agent (1Password, ssh-agent with a
FIDO key) for that exact key. The connection command is generated too.

- `AddKeysToAgent` and the ssh-agent autostart step (ADR 0005) aren't added,
  because the agent belongs to the user's key holder.
- The guide says the matching private key must be loaded in the agent. On
  Windows, an `UNPROTECTED PRIVATE KEY FILE` / `bad permissions` warning means
  the key wasn't in the agent (spike check 3b).
- `--identity-file` still overrides the path. For another local OS without it,
  the guide keeps placement only and asks for `--identity-file`, because the
  saved public path belongs to this computer.

`public_path` is already absolute for every key and is re-checked against the
stored fingerprint before use.

## Security model

| Threat | Handling |
| --- | --- |
| Fake host key at first connection (TOFU) | Keys come from the remote's own files over a console or session the user already trusts. Afterwards ssh refuses anything else. |
| Clipboard tampering or pasting the wrong output | Fingerprints are shown before saving. An optional remote `ssh-keygen -lf` check lets the user compare. |
| Silent re-pin after a compromise | Replacing needs `--replace` (or the TUI handoff) with both fingerprint sets shown and a typed `yes` in a real terminal. `--yes` never replaces. |
| Host key rotation | ssh fails hard. The guide and docs point to `hosts pin --replace`. |
| Injection into the pin file | Lines are written from parsed fields only, after strict validation and size limits. |

## Out of scope

- Remote-side `pair accept` / `pair send` exchange (backlog).
- Network transport, PAKE, SSH certificates and CAs.
- Editing `~/.ssh/known_hosts` or `/etc/ssh/ssh_known_hosts`.
- Editing `sshd_config`, or detecting `AuthorizedKeysFile`, `AllowUsers` or
  `Match`.
- Editing an existing Host block in the user's SSH config.
- Unpinning automatically when a destination is removed.
- Recording placement or connection success.

## Compatibility and release

- No state schema change.
- New commands `hosts pin` and `hosts unpin`, new guide steps, the
  `host_key_pins` field and `HOST KEYS` column, TUI `t`/`T`.
- Changed output: pinned snippets, and snippets and connection commands for
  public-only keys.
- Release as v0.10.0 (MINOR), after integration and with explicit release
  authorization (AGENTS.md).

## Verification

1. OpenSSH behaviour: `openssh-spike.md` (Linux 9.6p1 and 9.2p1, Win32-OpenSSH
   9.5p2, Git for Windows 10.2p1). macOS and the 1Password agent are
   unverified.
2. Unit tests cover parser cases, atomic writes keeping other destinations,
   identical re-pin without writing, refusal without `--replace`, `--yes`
   never replacing, malformed files stopping pin and unpin, unpinned snippets
   equal to the pinned ones minus the three lines, public-only snippets, native
   `ssh -G` parsing of the pinned snippet, the TUI handoff and removal notices.
3. Results are recorded in `docs/verification.md`.
