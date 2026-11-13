Status: ready-for-human
Released: v0.9.0; integrated by PR #22 (`f9956c9`), record in PR #23 (ADR 0005)

# ssh-agent automatic startup for passphrase-protected keys

## Objective

A key created with a passphrase should not require typing the passphrase on
every connection. For keys detected as passphrase-protected, the local guide
adds a platform-specific step so the user's ssh-agent starts automatically and
OpenSSH loads the key on first use. The app keeps generating scripts for the
user to paste; it never runs them, starts services or receives passphrases
(ADR 0002).

## Passphrase detection

- The guide reads only the format header of the selected managed private key:
  the OpenSSH `cipher` field, PEM `Proc-Type: 4,ENCRYPTED`, or
  `ENCRYPTED PRIVATE KEY`. It never decrypts, prompts, starts ssh-keygen or
  shows key material, and records nothing. This replaces the earlier
  `ssh-keygen -y -P ""` plan: the outcome is the same, there is no child
  process, and a prompt is impossible.
- `none` / unencrypted PEM: the key has no passphrase, so the guide is unchanged.
- Any cipher or encrypted PEM: the key is passphrase-protected, so the agent
  step and config line are added.
- Undeterminable (unreadable or unrecognized key format, or
  a foreign-terminal IdentityFile override that the app must not inspect per
  ADR 0003): add the agent step and config line anyway, with a one-line
  notice that passphrase protection could not be checked and the step is
  only needed if the key has a passphrase.
- Public-only keys have no local steps, so they get no agent step.
- Detection is per guide generation and read-only; nothing is saved.

## Behavior for passphrase-protected or undeterminable keys

- The generated `Host` block adds `AddKeysToAgent yes`. Keys stay loaded
  until the agent stops; no lifetime is set.
- The guide gains a local "ssh-agent" step between the SSH config step and
  the connection step, for the selected local OS (ADR 0003). Scripts check
  the current state first and change nothing that already works, so they are
  safe to paste again.
- Per platform:
  - **Windows** (PowerShell, Windows OpenSSH): the `ssh-agent` service
    ("OpenSSH Authentication Agent") is disabled by default. The script
    checks `Get-Service ssh-agent`. If the startup type is not Automatic, it
    states that an elevated terminal is required and runs
    `Set-Service ssh-agent -StartupType Automatic` and
    `Start-Service ssh-agent`. It warns when `ssh` on PATH is not
    `%SystemRoot%\System32\OpenSSH\ssh.exe` (for example Git for Windows'
    ssh), because that client does not use the Windows service.
  - **macOS** (POSIX shell): launchd already starts the agent for each login
    session. The config snippet also emits `IgnoreUnknown UseKeychain` and
    `UseKeychain yes`, so the passphrase is stored in the login Keychain. The
    step offers `ssh-add --apple-use-keychain <key>` for an immediate load.
  - **Linux** (manual guide only): the script only checks whether
    `SSH_AUTH_SOCK` points to a reachable agent (`ssh-add -l` exit status
    0 or 1) and reports the result. It installs no units and edits no
    profiles. The step text explains the common options for the user to
    choose: a desktop keyring agent, a systemd user service, or starting
    `ssh-agent` from the shell profile.
- CLI `hosts guide` prints the new step with its paste target, like the
  existing steps. The TUI shows it as another switchable step with the same
  script-only copying.

## Acceptance criteria

- [x] Detection never prompts, reads only the key format header and records
      nothing; tests cover protected, unprotected and undeterminable keys.
- [x] Passphrase-protected and undeterminable keys get `AddKeysToAgent yes`
      and the agent step (undeterminable with the notice); keys without a
      passphrase and public-only keys keep the current guide.
- [x] Windows, macOS and Linux steps match the behavior above; macOS alone
      emits `IgnoreUnknown UseKeychain` / `UseKeychain yes`.
- [x] Windows script states the required elevation and warns about a
      non-Windows-OpenSSH client.
- [x] Linux step is check-and-instructions only, with no file or service
      changes.
- [x] Scripts are idempotent; existing agents and services are reported
      unchanged.
- [x] No state schema change; guide refresh re-runs detection.
- [x] Tests, an ADR updating ADR 0002/0003, glossary and
      docs/verification.md updated.

## Decisions

- 2026-11-11: The user decided: show the step only for keys detected as
  passphrase-protected; the Linux step is a manual guide; `AddKeysToAgent yes`
  keeps keys until the agent stops.
- 2026-11-11: The user decided that undeterminable keys still get the step,
  with a notice.
- 2026-11-11: Detection reads the key format header instead of running
  ssh-keygen; the result is the same and no prompt or child process is possible.
- 2026-11-11: In the TUI, F5 invalidates the shown guide and the user reopens
  it; reopening re-runs detection. No live in-place refresh was added.
- 2026-11-11: Windows keeps keys added to its ssh-agent service across
  restarts; the Windows step states this instead of "until the agent stops".
