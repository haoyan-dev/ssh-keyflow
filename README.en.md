# SSH KeyFlow

[简体中文](README.zh-CN.md) · English · [日本語](README.md)

SSH KeyFlow is a CLI written in Go that creates, imports and manages the SSH key pairs on this machine, and shows public keys in a form you can copy.

[Get started](docs/en/tutorials/first-connection.md) · [Documentation](docs/en/README.md) · [Contributing](CONTRIBUTING.en.md) · [Releases](https://github.com/haoyan-dev/ssh-keyflow/releases)

## What it does

- Creates Ed25519 keys, imports existing key pairs, and resumes interrupted key creation.
- Lists keys (as a table or JSON) and shows a public key as a single line for copying.
- Removes keys from management while keeping the key files.

The app does not open SSH connections, perform remote registration or edit SSH config. You deploy public keys yourself. Removal from management is a separate operation from deleting key files or revoking SSH access. v0.11.0 retired destinations, the manual setup guide, host key pinning, the TUI and public-only import ([migration](docs/en/guides/data-and-recovery.md#migrating-from-v0100-and-earlier)).

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
ssh-keyflow keys list
```

Use an interactive terminal to confirm changing operations. To prepare your first key and connect, see [Your first connection setup](docs/en/tutorials/first-connection.md); for command examples see [Using the CLI](docs/en/guides/cli.md).

## Documentation

| Goal | Page |
| --- | --- |
| Prepare your first key and connect | [Tutorial](docs/en/tutorials/first-connection.md) |
| Find procedures and fixes for problems | [Documentation index](docs/en/README.md) |
| Check the promises for the CLI, JSON and saved state | [Application contract (overview)](docs/en/reference/application-contract.md) |
| Contribute fixes or documentation improvements | [Contributing](CONTRIBUTING.en.md) |

## Safety and data

Management data is stored in `~/.ssh/ssh-keyflow/state.json`, and generated keys in `keys/` in the same directory. State from v0.10.0 and earlier is written out to `state.v<N>-<UTC time>.json` on the first change before it is migrated. Before migrating from an older version or downgrading, read [Data, migration and recovery](docs/en/guides/data-and-recovery.md). `state.json.bak` is a rolling backup of the previous state, not a permanent archive for returning to an older version.

## Getting and running the Portable release

You can use the per-OS / CPU archives and `SHA256SUMS` from [Releases](https://github.com/haoyan-dev/ssh-keyflow/releases). See the [verification and extraction steps](docs/en/guides/installation.md#getting-and-running-the-portable-release).

## Manual Portable updates, PATH and removal

See [Manual updates, PATH setup and removal](docs/en/guides/installation.md#manual-portable-updates-path-and-removal). Moving the executable does not move the storage location; it stays under your home directory.

## Users of the v0.1.0 installer

See [Migrating from the old installer](docs/en/guides/installation.md#users-of-the-v010-installer). It explains how ownership records, PATH and manual replacement are handled.

## Development and release contract

For the development environment and checks, see [Contributing](CONTRIBUTING.en.md); for distribution and publishing, see the [release guide](docs/en/maintaining/releases.md). Release approval scope and completion criteria follow [AGENTS.md](AGENTS.md#tags-and-releases).

Licenses and copyright notices for dependencies are in [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt).
