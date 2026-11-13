# Your first connection setup

[简体中文](../../zh-CN/tutorials/first-connection.md) · English · [日本語](../../tutorials/first-connection.md)

[Documentation index](../README.md)

You will create one key, associate it with a destination, and display the manual setup guide. You run the final SSH connection yourself to confirm it works. An association saved in the app does not, on its own, change anything on the destination.

## What you need

- [SSH KeyFlow installed](../guides/installation.md) and an interactive terminal.
- `ssh` and `ssh-keygen` on the local terminal.
- A destination with an available SSH server that you are allowed to configure. If your public key is not registered there yet, you must still be able to log in to that machine by some other means.
- The destination's host name, user name and OS. Replace `server.example` and `alice` below with your own destination's values.

If you have been using an older version, read [Backing up and migrating state](../guides/data-and-recovery.md) before any changing operation.

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

To use an existing key instead, run the following. Replace the two paths with the absolute paths of your own key pair. On Windows, use a path such as `C:\Users\alice\.ssh\id_ed25519`.

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

Importing does not move or rewrite the original key files. If you omit `--private`, the key is public-key-only. In that case the guide uses the public key file as `IdentityFile`, and the matching private key is used from an SSH agent (for example, a FIDO key).

```sh
ssh-keyflow keys list
ssh-keyflow keys public --key laptop
```

Check the managed name `laptop` and the public key. If more than one key has the same managed name, use the fixed ID shown in the list. If creation was interrupted, go to the [recovery guide](../guides/troubleshooting.md#key-creation-was-interrupted).

## 3. Save the destination

The following is an example for a Linux destination. `my-server` is an alias you choose. Pick a name that does not clash with your existing SSH config.

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts list
```

After you confirm, the destination and its association with the key are saved locally. The default SSH port is 22. Use `--port` for a different port and `--os macos` for a macOS destination. A Windows destination requires `--os windows --windows-account standard`, or `--windows-account administrator` for an administrator account.

## 4. Open the manual setup guide

```sh
ssh-keyflow hosts guide --host my-server --key laptop
```

The output shows, for each step, which terminal to run it on and the script to run. Displaying the guide does not change any settings.

1. On the **destination terminal**, review and run the public key deployment step.
2. To skip the host key confirmation on first connection, do the next two steps. On the **destination terminal**, display the host public keys (on Windows, in an administrator PowerShell), and on the **local terminal**, paste the output into `ssh-keyflow hosts pin --host <ID>`. After checking the fingerprints and typing `yes`, open the guide again. The config append step now contains three extra lines for pinning, and the display and pinning steps disappear.
3. On the **local terminal**, check existing Host / Include / Match / wildcard entries and their precedence, then run the config append step. The append stops if the same explicit alias already exists.
4. For a key with a passphrase, or one whose passphrase status cannot be determined, read the ssh-agent step for the **local terminal** and make the necessary settings. Configuring the Windows service requires an administrator PowerShell. On Linux, choose and configure the agent yourself.
5. On the **local terminal**, run the final connection step. In this example it is `ssh my-server`. If you have not pinned the host key, verify the destination's host key through a trusted method on first connection.

If you will paste into an OS different from the one running the app, first read [specifying OSes for the guide in the CLI](../guides/cli.md). The app does not transfer keys to another machine and does not guess how paths correspond.

## 5. Result and next steps

Confirm that you can connect from your own SSH terminal. SSH KeyFlow does not verify the connection and has no operation for recording success. If you cannot connect, see [Troubleshooting](../guides/troubleshooting.md).

To do the same management in the TUI, launch `ssh-keyflow` and see [Using the TUI](../guides/terminal-ui.md). In the guide, `y` copies only the step currently displayed.

For dissociation and removal from management, see the [CLI guide](../guides/cli.md). Neither revokes SSH access. To remove the public key from the destination, read the removal instructions in the manual setup guide, and make sure you have another way to log in before you start.
