# Troubleshooting

[简体中文](../../zh-CN/guides/troubleshooting.md) · English · [日本語](../../guides/troubleshooting.md)

[Documentation index](../README.md)

## Command not found

Open a new terminal and try again. If you declined adding to PATH, use the full path of the executable that the installer printed. Windows PowerShell and PowerShell 7 use different profiles. See [install locations and PATH](installation.md).

## The TUI does not open or a changing operation is refused

The TUI requires an interactive terminal for both input and output. Changing operations also require confirmation in a real terminal. Do not substitute a `yes` piped into the command for the confirmation; run it in an interactive terminal. Launching with no arguments outside a terminal shows help. Listing, public key and guide output can be redirected.

## Cannot copy the public key or guide

Copying in the TUI requires a terminal that supports OSC52. If yours does not, take the output of the following commands into an editor or the OS clipboard. Replace the names with your own managed name, alias or fixed ID.

```sh
ssh-keyflow keys public --key laptop
ssh-keyflow hosts guide --host my-server --key laptop
```

Keep the public key on one line, and do not paste display wrapping as line breaks. Run the guide's steps separately on the terminal each one is meant for.

## Key creation was interrupted

```sh
ssh-keyflow keys list
ssh-keyflow resume --id creation-example
```

Replace `creation-example` with the unfinished creation ID from the list. Resuming uses the same ID and saved paths. Do not delete partial files and start over, or overwrite the same location with a different key; if there is a fingerprint or file mismatch, review the contents.

## Key, state or lock errors

Check whether the target files were moved or replaced, their permissions, and whether another process is running. The app does not repair unknown locks or invalid state automatically. If another operation is in progress, wait for it to finish. Do not delete locks or state unconditionally; stop the app and read [Data and recovery](data-and-recovery.md). A key with an unfinished legacy key deletion requires manual review.

## SSH still fails after running the guide

Associating, showing and copying are not evidence of a successful connection. Check the OS, Windows account type, paste target and private key path the guide specifies. Also check the destination's SSH server, the network, the actual `AuthorizedKeysFile` / `Match`, and the precedence in your existing config. These are outside what the app verifies automatically.

If you are repeatedly asked for a passphrase, check the guide's ssh-agent step. When the agent exits, loaded keys are not kept. Follow the OS-specific steps for the macOS keychain, agent management on Linux, and the Windows service.

## "Host key verification failed" is shown

For a destination whose host keys are pinned, any key other than the pinned ones is rejected. If you reinstalled the destination or updated its host keys, check the new host public keys on the destination's own terminal, then run `ssh-keyflow hosts pin --host <ID> --replace` in a real terminal. If you do not know why the key changed, it may be an attack, so do not replace it. If you unpin, also delete the three lines `HostKeyAlias`, `UserKnownHostsFile` and `StrictHostKeyChecking` that you added to your SSH config. If `hosts list` shows `unknown`, check `~/.ssh/ssh-keyflow/known_hosts`. The app does not rewrite an invalid pin file.

If you cannot connect with a public-key-only key, check that the matching private key is loaded into the agent. On Windows, an `UNPROTECTED PRIVATE KEY FILE` warning that names the public key file (`.pub`) also indicates that the key is not in the agent: OpenSSH could not find the key in the agent and tried to read the public key file as a private key. Do not change the public key file's permissions; load the private key into the agent. If the warning names a private key file, restrict that file's permissions so other users cannot read it.

## The installer refuses to update or remove

Existing files without an ownership record, and files changed after installation, are not overwritten. If you replaced the binary manually, the ownership hash no longer matches either. Read [the old installer and manual maintenance](installation.md#users-of-the-v010-installer), and do not delete other files or whole profiles.

## What to include in a report

Follow the reporting steps in [Contributing](../../../CONTRIBUTING.en.md) and record the version, OS / CPU, terminal, command run, expected result, actual diagnostics and minimal steps to reproduce. Do not include private keys, passphrases or authentication tokens, and review host names, user names, paths and state contents before publishing them.
