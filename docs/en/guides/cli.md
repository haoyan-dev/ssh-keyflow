# Using the CLI

[简体中文](../../zh-CN/guides/cli.md) · English · [日本語](../../guides/cli.md)

[Documentation index](../README.md)

## Command examples

SSH KeyFlow is operated through the CLI only. See `--help` for details of each command. Launching with no arguments shows help.

Replace the names and paths in the examples with your own. The following are examples grouped by purpose, not a procedure that runs every block in sequence. For a first run, use the [tutorial](../tutorials/first-connection.md).

### Help, version and lists

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list
ssh-keyflow keys list --json
```

### Creating and importing keys, showing public keys

Choose either creation or import as needed. Importing requires the paths of both the public key and the private key.

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
ssh-keyflow keys public --key laptop
```

`keys public` writes the public key as one line to standard output. The comment in the file is not included. You can use it with redirection or pipe it to the clipboard.

```sh
ssh-keyflow keys public --key laptop > laptop.pub
ssh-keyflow keys public --key laptop | pbcopy            # macOS
ssh-keyflow keys public --key laptop | Set-Clipboard     # Windows PowerShell
```

### Resuming an interrupted key creation

```sh
ssh-keyflow resume --id creation-example
```

Replace the ID with an unfinished local key creation ID shown by `keys list`.

## Removal from management

```sh
ssh-keyflow keys remove --key existing
```

After confirmation, the key is removed from management. Key files, SSH configuration and public key registrations on destinations are not changed, and SSH access is not revoked. Keys whose files cannot be found can also be removed from management. You can manage the same key pair again with `keys import` (it gets a new ID).

Specify a key by fixed ID or unique managed name. Ambiguous names are never selected automatically. Changing operations (create / import / remove / resume) require a real terminal for confirmation. You enter passphrases directly into the OS `ssh-keygen`. `keys list` and `keys public` can be redirected, and reading alone does not create or migrate the storage area.

## Retired commands

v0.11.0 retired `hosts` (add / list / remove / associate / dissociate / guide / pin / unpin), `tui` and `keys import --public-only`. Specifying them results in a usage error (exit code 2). Deploy public keys and set up your SSH config yourself. Example steps are in the [tutorial](../tutorials/first-connection.md).

## JSON and exit codes

JSON goes to standard output and diagnostics to standard error. `keys list --json` returns only `keys` and `creations`. Exit codes are 0 for success, 1 for execution failure or cancellation, and 2 for argument or usage errors.

For the exact input and output specification, see the [application contract](../reference/application-contract.md#cli) (overview) or its [authoritative CLI section](../../application-contract.md#cli) (Japanese).
