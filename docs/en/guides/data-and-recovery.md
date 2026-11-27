# Data, migration and recovery

[简体中文](../../zh-CN/guides/data-and-recovery.md) · English · [日本語](../../guides/data-and-recovery.md)

[Documentation index](../README.md)

## Storage location and backups

Management data is stored in `~/.ssh/ssh-keyflow/state.json` and generated keys in `keys/` in the same directory. Importing references the files you specify and does not move or rewrite the originals. The app verifies the public key fingerprint and the private key file digest, and stops for review on a mismatch or if a file cannot be read. Private key contents and passphrases are never saved in the state JSON or in arguments.

Before updating or migrating, exit the app and back up `~/.ssh/ssh-keyflow/` to a separate, access-restricted location. The state JSON alone cannot restore private keys. Also back up the original files of imported keys and any SSH config you set up yourself, as needed. Do not put backups that contain private keys in public repositories or shared logs.

Before restoring, also move the current data aside, and check that the version you will use can read the backed-up state. Confirm that the restored paths and key fingerprints match. The app has no automatic restore and no state merging.

## Interruptions and the previous state

Key creation uses the native `ssh-keygen` and never overwrites existing paths. If creation is interrupted, `resume` it with the creation ID from `keys list`. The saved paths, fingerprint and digest are checked against the files, and creation resumes without automatically deleting partial files. While an unfinished creation exists, no other key pair is newly created. State changes are protected by an exclusive lock; the verified previous state is saved to `state.json.bak` before the state is replaced atomically. Unknown locks and invalid state are not repaired automatically.

## Migrating from v0.10.0 and earlier

v0.11.0 is an intentional incompatible change that narrows the app to managing key pairs only. Destinations, associations, the manual setup guide, host key pinning, the TUI and public-only import are retired.

`state.json` from v0.10.0 and earlier (schema v1 / v2) is not changed when read. `keys list` prints to standard error what the next change will not carry over. Changing operations show the same notice before asking for confirmation. An old file with unknown fields or broken values is not loaded, and nothing is changed. On the first changing operation (create, import, remove or resume), the app does the following in order.

1. It writes the original `state.json` as-is to `~/.ssh/ssh-keyflow/state.v<N>-<UTC time>.json`. An existing file is never overwritten. If writing fails, nothing is changed. If you retry an operation that failed partway, the archive with the same content is reused.
2. It carries over only keys that have a private key path and unfinished key creations, and saves them with schema v3.

What is not carried over: destinations and associations, removed records, the records and history of legacy registrations, config, SSH config edits, onboardings and deletions, public-only keys, and keys with an unfinished legacy key deletion. These remain in the archive. If an unfinished SSH config edit is listed, check that file manually. If you need a public-only key, pair it with its private key and import it again with `keys import`.

`~/.ssh/ssh-keyflow/known_hosts` (the host key pinning file) created by older versions, `~/.ssh/ssh-keyflow/config`, your normal `~/.ssh/config`, Include directives and Host blocks appended from the guide are not changed. The pinning lines `HostKeyAlias`, `UserKnownHostsFile` and `StrictHostKeyChecking` keep working as long as you keep the files. Remove them yourself if you no longer need them.

## Returning to an older version

Older versions cannot read schema v3. To return to an older version, exit the app and then do the following.

1. Back up the current `~/.ssh/ssh-keyflow/` to a separate location.
2. Restore `state.v<N>-<UTC time>.json` under the name `state.json`.

Keys created or imported after the migration are not included in the restored state. If needed, import them again with the older version. `state.json.bak` is a rolling backup of the previous state; after the migration it is v3, so it cannot be used to return to an older version.

For past verification results, see the [verification records](../../verification.md) (Japanese); for the current specification, see the [application contract](../reference/application-contract.md) (overview) or the [authoritative contract](../../application-contract.md) (Japanese).
