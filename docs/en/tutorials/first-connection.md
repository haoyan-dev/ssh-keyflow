# Your first connection setup

[简体中文](../../zh-CN/tutorials/first-connection.md) · English · [日本語](../../tutorials/first-connection.md)

[Documentation index](../README.md)

You will prepare one key, deploy its public key to the destination yourself, and confirm the SSH connection. SSH KeyFlow only creates or imports the key and displays the public key; it does not configure the destination or open SSH connections.

## What you need

- [SSH KeyFlow installed](../guides/installation.md) and an interactive terminal.
- `ssh` and `ssh-keygen` on the local terminal.
- A destination with an available SSH server that you are allowed to configure. If your public key is not registered there yet, you must still be able to log in to that machine by some other means.
- The destination's host name and user name. Replace `server.example` and `alice` below with your own destination's values.

If you have been using an older version, read [State migration](../guides/data-and-recovery.md#migrating-from-v0100-and-earlier) before any changing operation.

## 1. Check that it launches

Run this on the local terminal. The same commands work on Windows.

```sh
ssh-keyflow --version
ssh-keyflow keys list
```

If the version is shown and the key list can be read, continue. It is fine if there are no keys. Development builds show the version `dev`. Reading the list alone does not create or migrate the storage area.

## 2. Prepare a key

If you need a new key:

```sh
ssh-keyflow keys create --name laptop
```

Review what is shown and type `yes` at the confirmation prompt. You enter the passphrase directly into the OS `ssh-keygen`. Existing paths are never overwritten.

To use an existing key pair instead, run the following. Replace the two paths with the absolute paths of your own key pair. On Windows, use a path such as `C:\Users\alice\.ssh\id_ed25519`.

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

Importing does not move or rewrite the original key files.

```sh
ssh-keyflow keys list
```

Check the managed name `laptop`, the fingerprint and the public key path. You can find the private key path in `private_path` of `keys list --json`. If creation was interrupted, go to the [recovery guide](../guides/troubleshooting.md#key-creation-was-interrupted).

## 3. Deploy the public key to the destination

On the local terminal, display the public key and copy it as a single line.

```sh
ssh-keyflow keys public --key laptop
```

On the **destination terminal** (the one you logged in to by some other means), append it to the login user's `authorized_keys`. For a Linux / macOS destination, do the following. Replace `<public key>` with the line you copied.

```sh
mkdir -p ~/.ssh && chmod 700 ~/.ssh
printf '%s\n' '<public key>' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

For a Windows destination, append it to `%USERPROFILE%\.ssh\authorized_keys` for a standard account, or to `%ProgramData%\ssh\administrators_authorized_keys` for an administrator account. Write permission on the administrator file must be limited to SYSTEM and Administrators. Do not delete existing lines.

## 4. Connect

On the local terminal, connect by specifying the private key. `<private key path>` is the path you checked in step 2.

```sh
ssh -i <private key path> -o IdentitiesOnly=yes alice@server.example
```

On first connection, verify the destination's host key fingerprint through a trusted method before typing `yes`. To avoid specifying this every time, add a Host block like the following to `~/.ssh/config` yourself. Check that it does not overlap with existing Host / Include / Match entries.

```text
Host my-server
  HostName server.example
  User alice
  IdentityFile <private key path>
  IdentitiesOnly yes
```

## 5. Result and next steps

Confirm that you can connect from your own SSH terminal. SSH KeyFlow does not verify the connection and has no operation for recording success. If you cannot connect, see [Troubleshooting](../guides/troubleshooting.md).

For removal from management, see the [CLI guide](../guides/cli.md#removal-from-management). It does not revoke SSH access. To remove the public key from the destination, make sure you have another way to log in, then delete the matching line from `authorized_keys` yourself.
