Status: ready-for-agent

# ssh-agent automatic startup for passphrase-protected keys

## Objective

A key created with a passphrase should not require typing the passphrase on
every connection. For keys detected as passphrase-protected, the local guide
adds a platform-specific step so the user's ssh-agent starts automatically and
OpenSSH loads the key on first use. The app keeps generating scripts for the
user to paste; it never runs them, starts services or receives passphrases
(ADR 0002).

## Passphrase detection

- The guide checks the selected managed private key with
  `ssh-keygen -y -P "" -f <private key>`, with no terminal input attached, so
  it can never prompt. Only the empty passphrase is tried; no passphrase is
  requested, stored or logged, and the derived public key is discarded.
- Success: the key has no passphrase, so the guide is unchanged.
- Failure reported as an incorrect passphrase: the key is
  passphrase-protected, so the agent step and config line are added.
- Undeterminable (ssh-keygen unavailable, unreadable or unsupported key, or
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

- [ ] Detection never prompts, tries only the empty passphrase and records
      nothing; tests cover protected, unprotected and undeterminable keys.
- [ ] Passphrase-protected and undeterminable keys get `AddKeysToAgent yes`
      and the agent step (undeterminable with the notice); keys without a
      passphrase and public-only keys keep the current guide.
- [ ] Windows, macOS and Linux steps match the behavior above; macOS alone
      emits `IgnoreUnknown UseKeychain` / `UseKeychain yes`.
- [ ] Windows script states the required elevation and warns about a
      non-Windows-OpenSSH client.
- [ ] Linux step is check-and-instructions only, with no file or service
      changes.
- [ ] Scripts are idempotent; existing agents and services are reported
      unchanged.
- [ ] No state schema change; F5 refresh re-runs detection.
- [ ] Tests, an ADR updating ADR 0002/0003, glossary and
      docs/verification.md updated.

## Decisions

- 2026-11-11: The user decided: show the step only for keys detected as
  passphrase-protected; the Linux step is a manual guide; `AddKeysToAgent yes`
  keeps keys until the agent stops.
- 2026-11-11: The user decided that undeterminable keys still get the step,
  with a notice.
