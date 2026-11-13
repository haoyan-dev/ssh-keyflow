# SSH KeyFlow

[简体中文](README.zh-CN.md) · English · [日本語](README.md)

SSH KeyFlow is a TUI / CLI written in Go that manages local SSH keys and destinations, and helps you deploy public keys and set up connections through copy and paste.

[Get started](docs/en/tutorials/first-connection.md) · [Documentation](docs/en/README.md) · [Contributing](CONTRIBUTING.en.md) · [Releases](https://github.com/haoyan-dev/ssh-keyflow/releases)

## What it does

- Creates Ed25519 keys, imports existing key pairs or public keys, and resumes interrupted key creation.
- Manages keys, destinations and associations through the Keys / Destinations TUI and the CLI.
- Shows and copies public key deployment, SSH config append and connection steps for Linux / macOS / Windows.
- Pins host keys by pasting the host public keys displayed on the destination. This skips the first-connection confirmation and rejects other host keys.
- Configures keys whose private key lives in an SSH agent, such as FIDO keys, by selecting them with the public key file.

The app does not open SSH connections or perform remote registration. An association is local management data and does not mean a connection succeeded. You configure the destination by running the generated scripts on the target terminal yourself. Removal from management is a separate operation from deleting key files or revoking SSH access.

## One-line install

Supports amd64 (x64) / arm64 on Windows / Linux / macOS. Go and administrator rights are not required; the OS `ssh` and `ssh-keygen` are required.

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

The script is fetched from `main`; it verifies the latest stable release and its checksum and then installs it. If you want to review the script first, save it, read it and then run it. PATH is changed only if you approve the prompt. See [Install, update and uninstall](docs/en/guides/installation.md) for details.

## Launch and use

Run the following in a new terminal. If you did not add SSH KeyFlow to PATH, use the full path that the installer printed.

```sh
ssh-keyflow --version
ssh-keyflow
```

Use the TUI in an interactive terminal. To prepare your first key and destination, see [Your first connection setup](docs/en/tutorials/first-connection.md); for keys see [Using the TUI](docs/en/guides/terminal-ui.md); for command examples see [Using the CLI](docs/en/guides/cli.md).

## Documentation

| Goal | Page |
| --- | --- |
| Prepare your first key and destination | [Tutorial](docs/en/tutorials/first-connection.md) |
| Find procedures and fixes for problems | [Documentation index](docs/en/README.md) |
| Check the promises for the CLI, JSON and saved state | [Application contract (overview)](docs/en/reference/application-contract.md) |
| Contribute fixes or documentation improvements | [Contributing](CONTRIBUTING.en.md) |

## Safety and data

Management data is stored in `~/.ssh/ssh-keyflow/state.json`, generated keys in `keys/` in the same directory, and pinned host public keys in `known_hosts`. Before migrating from an older version or downgrading, read [Data, migration and recovery](docs/en/guides/data-and-recovery.md). `state.json.bak` is a rolling backup of the previous state, not a permanent archive for returning to an older version.

## Getting and running the Portable release

You can use the per-OS / CPU archives and `SHA256SUMS` from [Releases](https://github.com/haoyan-dev/ssh-keyflow/releases). See the [verification and extraction steps](docs/en/guides/installation.md#getting-and-running-the-portable-release).

## Manual Portable updates, PATH and removal

See [Manual updates, PATH setup and removal](docs/en/guides/installation.md#manual-portable-updates-path-and-removal). Moving the executable does not move the storage location; it stays under your home directory.

## Users of the v0.1.0 installer

See [Migrating from the old installer](docs/en/guides/installation.md#users-of-the-v010-installer). It explains how ownership records, PATH and manual replacement are handled.

## Development and release contract

For the development environment and checks, see [Contributing](CONTRIBUTING.en.md); for distribution and publishing, see the [release guide](docs/en/maintaining/releases.md). Release approval scope and completion criteria follow [AGENTS.md](AGENTS.md#tags-and-releases).

Licenses and copyright notices for dependencies are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt).
