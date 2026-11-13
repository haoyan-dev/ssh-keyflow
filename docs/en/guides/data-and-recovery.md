# Data, migration and recovery

[简体中文](../../zh-CN/guides/data-and-recovery.md) · English · [日本語](../../guides/data-and-recovery.md)

[Documentation index](../README.md)

## Storage location and backups

Management data is stored in `~/.ssh/ssh-keyflow/state.json`, generated keys in `keys/` in the same directory, and pinned host public keys in `known_hosts`. `known_hosts` is a separate file from the state schema, and older versions do not read it. Importing references the files you specify and does not move or rewrite the originals. The app verifies the public key fingerprint and the private key file digest, and stops for review on a mismatch or if a file cannot be read. Private key contents and passphrases are never saved in the state JSON or in arguments.

Before updating or migrating, exit the app and back up `~/.ssh/ssh-keyflow/` to a separate, access-restricted location. The state JSON alone cannot restore private keys. Also back up the original files of imported keys and any SSH config you set up yourself, as needed. Do not put backups that contain private keys in public repositories or shared logs.

Before restoring, also move the current data aside, and check that the version you will use can read the backed-up state. Confirm that the restored paths and key fingerprints match, and reconcile keys, destinations and associations added after the migration. The app has no automatic restore, no state merging and no reverse conversion to the old schema.

## Interruptions and the previous state

Key creation uses the native `ssh-keygen` and never overwrites existing paths. If creation is interrupted, `resume` it with the creation ID from `keys list`. The saved paths, fingerprint and digest are checked against the files, and creation resumes without automatically deleting partial files. While an unfinished creation exists, no other key pair is newly created. State changes are protected by an exclusive lock; the verified previous state is saved to `state.json.bak` before the state is replaced atomically. Unknown locks and invalid state are not repaired automatically.

## Migrating from older versions and downgrading

The v0.5.0 simplification is an intentional incompatible change. The former `register`, `unregister`, `registrations list` and `keys delete`, as well as `--new-key`, `--auth`, `--identity` and `--configure`, are no longer available. The old numbered menu, Tasks and recursive SSH config discovery are also removed. Before pasting a snippet, check for duplicates and precedence against existing `Host` / Include / wildcard settings yourself.

Existing v1 state is not changed when read, and becomes v2 on the first state save. The new `associations` are independent of the legacy registration history. History and interruption records of past registrations, connection checks, config edits and deletions are kept, and are not rewritten as current success or completion. Legacy remote operations are read-only and are not carried over by `resume`. A key with a legacy deletion in progress is refused because it requires manual review, but this does not block operations on other keys.

**If you might go back to an older version, exit the app and save the v1 `state.json` in a separate location before the first changing operation on v0.5.0 or later.** `state.json.bak` is a rolling backup of the previous state and does not keep v1 permanently. Even creating your first key saves the state several times, so the backup can be updated to v2 within the same operation. Older versions cannot read v2, so a downgrade requires manually restoring the saved v1 state and reviewing keys, associations and other data added after the migration. Reading alone never creates a migration copy automatically.

Previously generated `~/.ssh/ssh-keyflow/config`, your normal `~/.ssh/config`, Include directives and existing backups are not changed. A remaining Include may continue to load the former settings. Review the contents yourself if needed. For past verification results, see the [verification records](../../verification.md) (Japanese); for the current specification, see the [application contract](../reference/application-contract.md).
