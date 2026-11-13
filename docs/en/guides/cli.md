# Using the CLI

[简体中文](../../zh-CN/guides/cli.md) · English · [日本語](../../guides/cli.md)

[Documentation index](../README.md)

## Command examples

The CLI lets you specify the same operations directly. See `--help` for details of each command.

Replace the names, host names and paths in the examples with your own. The following are examples grouped by purpose, not a procedure that runs every block in sequence. For a first run, use the [tutorial](../tutorials/first-connection.md).

### Help, version and lists

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list --json
ssh-keyflow hosts list --json
ssh-keyflow tui
```

### Creating and importing keys, showing public keys

Choose either creation or import as needed. `existing` is the managed name of the public key to import.

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub
ssh-keyflow keys public --key laptop
```

### Destinations and associations

`laptop` and `existing` refer to keys that have already been created or imported.

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts associate --host my-server --key existing
```

### Manual setup guide

```sh
ssh-keyflow hosts guide --host my-server --key laptop
ssh-keyflow hosts guide --host my-server --key laptop --local-os linux --identity-file /home/alice/.ssh/id_ed25519 --remote-os windows --windows-account standard
```

### Host key pinning

```sh
ssh-keyflow hosts pin --host my-server
ssh-keyflow hosts pin --host my-server --file host-keys.txt --yes
ssh-keyflow hosts pin --host my-server --replace
ssh-keyflow hosts unpin --host my-server
```

`hosts pin` shows the command to run on the destination. Paste the host public keys printed on the destination's own terminal, then press Enter on an empty line (on Windows, display them in a PowerShell opened as administrator). Check the type and SHA256 fingerprint of each key, and type `yes` to save them to `~/.ssh/ssh-keyflow/known_hosts`. `--file` reads a file where you saved the output, and input from a pipe or file requires `--yes`. `--yes` does not replace keys when different keys are already pinned. Specify `--replace` in a real terminal only when you have confirmed on the destination itself that its host keys changed. `hosts unpin` removes the pin after confirmation in a real terminal. Remove the three pinning lines that were added to your SSH config yourself.

If you run `hosts guide` again after pinning, the Host snippet gains `HostKeyAlias`, `UserKnownHostsFile` and `StrictHostKeyChecking yes`. For a Host already in your SSH config, add these three lines yourself. `hosts list` shows the pin status in the `HOST KEYS` column. Pins remain even after you remove the destination from management, so run `hosts unpin` before removal if you no longer need them.

### Resuming an interrupted key creation

```sh
ssh-keyflow resume --id creation-example
```

Adding `--private /absolute/path/id_ed25519` to `keys import` verifies the key pair and imports it. Without it, the key is public-key-only. Replace paths with absolute paths in your environment. For `hosts add`, `--port` defaults to 22 and `--os` is `linux` / `macos` / `windows`. On Windows, also specify `--windows-account standard` or `--windows-account administrator`. Replace the `resume` ID with an unfinished local key creation ID shown by `keys list`.

## Removal from management and dissociation

Choose according to what you want to clean up. These are also independent examples.

```sh
ssh-keyflow hosts dissociate --host my-server --key existing
ssh-keyflow hosts remove --host my-server
ssh-keyflow keys remove --key existing
```

`keys remove` / `hosts remove` remove the managed item and all its associations after confirmation. `hosts dissociate` removes only the association for the specified pair and keeps the key and the destination. In the TUI, press `d` in a list or detail view, and for dissociation press `u` and choose an existing pair. None of these change key files, SSH configuration or public key registrations on destinations, and none revoke SSH access. Keys whose files cannot be found can also be removed from management. A key with an unfinished legacy key deletion requires manual review. To preserve creation results and legacy history, removed metadata is kept in `removed_keys` / `removed_hosts` in the state, and re-importing a key or adding a destination with the same alias assigns a new ID. Older binaries cannot read state that contains these arrays, so to go back to an older version, use state you backed up separately beforehand.

Specify a destination by fixed ID or alias, and a key by fixed ID or unique managed name. Ambiguous key names are never selected automatically. `hosts guide` is a read-only operation that names the destination and key explicitly and does not save an association. Changing operations require a real terminal for confirmation. You enter passphrases directly into the OS `ssh-keygen`. `list`, `keys public` and `hosts guide` can be redirected, and reading alone does not create or migrate the storage area.

## OSes and private key paths for the guide

`--local-os` and `--remote-os` for `hosts guide` accept `linux` / `macos` / `windows`. `--windows-account` is for Windows destinations. When omitted, the OS running the app and the saved destination OS and account are used. `--identity-file` is the private key path as seen from the local terminal, and it is required for a key pair when you choose a local OS different from the one running the app. Use an absolute path for the specified OS or `~/path`. For a public-key-only key, the saved public key path is used as `IdentityFile`, so with a local OS different from the one running the app, pass the public key path as seen from that terminal to `--identity-file`. Host key pinning steps and lines are shown only when the local OS is the same as the one running the app. When pasting from the Windows build into WSL, choose `--local-os linux` and specify the path of the matching key as seen from WSL. The app does not verify or transfer the key at that path and does not guess drive mappings. These options apply only to the guide and do not change the saved destination or the paths of managed keys.

## JSON and exit codes

JSON goes to standard output and diagnostics to standard error. `keys list --json` returns `keys` and `creations`; `hosts list --json` returns `hosts`, `associations` and `host_key_pins` (type and fingerprint per destination ID). If the pin file cannot be read, `host_key_pins` is omitted and the reason is written to standard error. Exit codes are 0 for success, 1 for execution failure or cancellation, and 2 for argument or usage errors. Launching with no arguments outside a terminal shows help and exits with 0.

To try this step by step, see [Your first connection setup](../tutorials/first-connection.md); for the exact input and output specification, see the [application contract](../reference/application-contract.md#cli) (overview) or its [authoritative CLI section](../../application-contract.md#cli) (Japanese).
