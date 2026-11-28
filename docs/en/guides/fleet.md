# Creating and accepting a fleet roster

[简体中文](../../zh-CN/guides/fleet.md) · English · [日本語](../../guides/fleet.md)

[Documentation index](../README.md)

> This page describes an unreleased feature on `main` (fleet sync phase 1). It is not in any published version.

A fleet is a set of your machines (nodes) that pin the same fleet root key as their trust anchor. The admin node signs the roster (the list of nodes and public keys), and the other nodes receive the roster as files and accept it after verifying it with the pinned root public key. Phase 1 stops at creating and distributing the roster; it does not write `authorized_keys`, known_hosts or SSH config. For terms, see the [glossary](../reference/glossary.md).

## Prerequisites

- Signing and verification require OpenSSH 8.4 or later. The first `ssh-keygen` on PATH is used, and its version is checked with `ssh -V` in the same directory. `--dry-run`, for example `fleet init --dry-run`, shows the path and version of the `ssh-keygen` that will be used.
- Commands that change the state show a plan, then ask for `yes` on a real terminal. With `--dry-run`, only the plan is shown and nothing is changed.
- The first time anything on `main` changes the state (not only fleet commands, but also key creation and so on), the v0.11.0 `state.json` (schema v3) is kept as `state.v3-<UTC time>.json` and the state is then saved as v4. v0.11.0 and earlier cannot read v4. Read [Data, migration and recovery](data-and-recovery.md) first.

## 1. Set the node description on each machine

A client is the side that connects over SSH; a remote is the side that is connected to. A client needs a managed key (`keys create` or `keys import`).

```sh
# A node that is only a client
ssh-keyflow node set --name laptop --roles client --key laptop

# A node that is both a client and a remote
ssh-keyflow node set --name desk --roles client,remote --key desk --addresses desk.lan,192.0.2.10
ssh-keyflow node show
```

A remote's host public keys are read from the `HostKey` lines of `sshd_config` (default `/etc/ssh/sshd_config` on Unix, `%ProgramData%\ssh\sshd_config` on Windows). If it is elsewhere, specify `--sshd-config`. Specify `--port` if the port is not 22, and `--login-account` if it differs from the shared login account. Give list values as one comma-separated value.

## 2. Create the fleet on the admin node

Choose one machine to sign the roster.

```sh
ssh-keyflow fleet init --login-account user --dry-run
ssh-keyflow fleet init --login-account user
```

`--login-account` is the shared login account used on all remotes. The fleet root key is created at `~/.ssh/ssh-keyflow/fleet/root_ed25519`, and `ssh-keygen` asks for its passphrase directly. An empty passphrase is not allowed. This key cannot be recreated, so back up the whole `~/.ssh/ssh-keyflow/` to an access-restricted location.

If only signing failed, running the same command again signs version 1 with the key already created. After changing the admin node's own node description, sign the next version with `fleet sign`.

## 3. Add other nodes to the roster

On the machine to add, write the node description file and copy it to the admin node.

```sh
ssh-keyflow node export --out ./desk-node
```

On the admin node, add it and sign the next version.

```sh
ssh-keyflow fleet add ./desk-node
```

Compare the name, roles and fingerprints shown with `node show` on the machine being added, then enter `yes`. Adding the same node again is a replacement that shows the old and new values side by side.

## 4. Distribute and accept the roster

On the admin node, write the roster and copy it to the machine that will accept it.

```sh
ssh-keyflow fleet export --out ./roster
```

This creates three files: `roster.json`, `roster.json.sig` and `fleet-root.pub`. `--out` must be a missing or empty directory. On the accepting machine, run the following.

```sh
ssh-keyflow fleet accept ./roster --dry-run
ssh-keyflow fleet accept ./roster
ssh-keyflow fleet show
```

The first time, the root public key fingerprint and fleet ID are shown. Check that they match `fleet show` on the admin node, then enter `yes`. This pins the root, and after that only rosters signed by the pinned root are accepted.

`roster.json` is handled as the signed bytes. If Git line-ending conversion or an editor changes it, it is rejected as a non-canonical roster.

## Rejections and warnings

In the following cases, `fleet accept` changes nothing and exits with code 1.

- A root different from the pinned one, a signature mismatch, or a modified `roster.json`
- A version lower than the accepted one, the same version with different content, or a roster from another fleet
- A directory with files other than the three, or a non-canonical roster

Accepting the same roster again changes nothing. If the roster is older than `--max-age` (default 30 days) or its creation time is more than 5 minutes in the future, a warning goes to standard error and processing continues. Running `fleet sign` on the admin node updates the creation time.

For the exact rules, JSON and state format, see the [application contract](../reference/application-contract.md#fleet-sync-phase-1-unreleased) (overview) or its [authoritative section](../../application-contract.md#フリート同期フェーズ1未リリース) (Japanese).
