# Troubleshooting

[简体中文](../../zh-CN/guides/troubleshooting.md) · English · [日本語](../../guides/troubleshooting.md)

[Documentation index](../README.md)

## Command not found

Open a new terminal and try again. If you declined adding to PATH, use the full path of the executable that the installer printed. Windows PowerShell and PowerShell 7 use different profiles. See [install locations and PATH](installation.md).

## A changing operation is refused

Creating, importing, removing and resuming require confirmation in a real terminal. Do not substitute a `yes` piped into the command for the confirmation; run it in an interactive terminal. Listing and public key output can be redirected.

## `unknown command "hosts"` or a similar message is shown

`hosts`, `tui` and `--public-only` were retired in v0.11.0. See [Using the CLI](cli.md#retired-commands) and the [migration notes](data-and-recovery.md#migrating-from-v0100-and-earlier).

## Copying the public key

Take the output of the following command into the clipboard or an editor. Replace the name with your own managed name or fixed ID.

```sh
ssh-keyflow keys public --key laptop
```

Keep the public key on one line, and do not paste display wrapping as line breaks.

## Key creation was interrupted

```sh
ssh-keyflow keys list
ssh-keyflow resume --id creation-example
```

Replace `creation-example` with the unfinished creation ID from the list. Resuming uses the same ID and saved paths. Do not delete partial files and start over, or overwrite the same location with a different key; if there is a fingerprint or file mismatch, review the contents.

## Key, state or lock errors

Check whether the target files were moved or replaced, their permissions, and whether another process is running. The app does not repair unknown locks or invalid state automatically. If another operation is in progress, wait for it to finish. Do not delete locks or state unconditionally; stop the app and read [Data and recovery](data-and-recovery.md).

## `state.json uses schema v1` / `v2` is shown

This is state from an older version. The next changing operation archives the original file and carries over only key pairs. Review what is listed under "Not carried over" before you proceed. Details are in the [migration notes](data-and-recovery.md#migrating-from-v0100-and-earlier).

## SSH still fails after deploying the public key

SSH KeyFlow does not verify connections. Check the destination's SSH server, the network, the login user, the actual `AuthorizedKeysFile` / `Match`, the permissions of `authorized_keys` and its parent directory, and the `IdentityFile` in your local SSH config. For a Windows administrator account, `%ProgramData%\ssh\administrators_authorized_keys` is used.

If you are repeatedly asked for a passphrase, load the key into the OS ssh-agent (`ssh-add <private key path>`). When the agent exits, loaded keys are not kept.

## The installer refuses to update or remove

Existing files without an ownership record, and files changed after installation, are not overwritten. If you replaced the binary manually, the ownership hash no longer matches either. Read [the old installer and manual maintenance](installation.md#users-of-the-v010-installer), and do not delete other files or whole profiles.

## What to include in a report

Follow the reporting steps in [Contributing](../../../CONTRIBUTING.en.md) and record the version, OS / CPU, terminal, command run, expected result, actual diagnostics and minimal steps to reproduce. Do not include private keys, passphrases or authentication tokens, and review host names, user names, paths and state contents before publishing them.
