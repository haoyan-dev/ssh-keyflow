# SSH KeyFlow glossary

[简体中文](../../zh-CN/reference/glossary.md) · English · [日本語](../../../GLOSSARY.md)

> This page is an English overview. The authoritative glossary is the Japanese [GLOSSARY.md](../../../GLOSSARY.md); if they differ, the Japanese page wins.

[Documentation index](../README.md)

SSH KeyFlow manages local SSH keys and destinations, and supports deploying public keys and setting up connections through copy and paste. The English documentation uses the terms below consistently.

| English term | Canonical term (GLOSSARY.md) | Meaning | Terms to avoid |
| --- | --- | --- | --- |
| Destination | 接続先（Destination） | The combination of a machine to connect to over SSH with its login user, port and OS. In storage and the CLI it is called Host / `hosts`. It does not mean the key is registered or that a connection is possible. | |
| Alias | 別名 | The name used to specify a destination. It is not applied to the SSH config automatically; before pasting, the user checks for clashes with existing Host entries. | |
| Managed key | 管理対象の鍵（Key） | A key created by the tool or explicitly imported by the user, including public-key-only keys. Its fixed ID, fingerprint, file paths and origin (created or imported) are saved. | |
| Association | 関連付け（Association） | Locally saved metadata pairing a key with a destination, stored in v2 `associations`. It is independent of registration history and connection success, and is not converted automatically from legacy registration history. | |
| Removal from management | 管理からの削除（Removal） | Removing a key or destination from the managed list together with all its associations. The removed metadata is kept in `removed_keys` / `removed_hosts` so that references from creation results and legacy history stay valid; key files, SSH configuration and public key registrations on the destination are not changed. | |
| Dissociation | 関連付けの解除（Dissociation） | Removing only the specified key–destination pair from `associations`. The key and destination stay managed, and SSH access is not revoked. | |
| Manual setup guide | 手動設定ガイド（Guide） | Shows a public key deployment script and manual removal instructions for the selected destination OS and account. If the key has a private key or is public-key-only (used through an agent), it also includes an SSH config append script for the selected local OS and a separate `ssh 別名` connection step. For an unpinned destination it includes steps to display and pin the host keys; for a pinned one, the SSH config includes the pinning lines. For a key with a passphrase, or one whose status cannot be determined, it also includes `AddKeysToAgent yes` and an ssh-agent autostart step. The defaults are the OS running the app and the saved destination OS and account, and these can be changed for the guide only. If the local OS differs, the private key path as seen from that terminal is specified. Each step names the terminal to paste into, and the TUI copies only the runnable script of the selected step. Showing and copying involve no configuration change, execution or success record; files change only when the user runs the scripts on a terminal. | |
| Host key pinning | ホスト鍵の固定（Host key pinning） | Saving the host public keys displayed on the destination's own terminal into the pin file per destination ID. The pinned destination's Host snippet refers only to the pin file, skips the first-connection confirmation and rejects other keys; replacing with a different key needs an explicit option and confirmation. It does not mean a connection succeeded or a public key was deployed. | Host key registration (ホスト鍵の登録), adding to known_hosts (known_hostsへの追加) |
| Pin file | 固定ファイル（Pin file） | The app-owned `~/.ssh/ssh-keyflow/known_hosts`, containing only lines named `ssh-keyflow-<host-id>`. It is separate from the user's `~/.ssh/known_hosts`, and invalid content is not repaired automatically. | |
| Unverified | 未確認（Unverified） | The state in which the app has not verified whether registration or connection currently succeeds. New keys, associations and guides are treated this way, and there is no operation for recording manual success. | |
| Local key creation | ローカル鍵作成（Creation） | Key creation with the native ssh-keygen, together with the saved progress used to resume with the same ID and paths after an interruption. It is the only operation the current `resume` handles. | |
| Legacy registration history | 旧登録履歴 | Records of registration, unregistration, connection checks and progress saved by earlier versions. Read-only past information that keeps its fixed IDs and values; it does not guarantee the current registration or connection state. | |
| Manual unregistration | 手動の登録解除 | The user removing the line of the specified public key from the destination's key file. The app neither performs nor verifies it. | Local key deletion (ローカル鍵の削除) |
| Local key deletion | ローカル鍵の削除 | Removing public and private key files from the machine running the app. Not part of the current features; records of legacy deletions in progress are kept, and only the affected key becomes unusable because it needs manual review. | Unregistration (登録解除) |
| Previous-state backup | 直前状態のバックアップ | `state.json.bak` written when the state is saved: a rolling copy of the bytes of the last valid state. It is not a migration archive that keeps v1 permanently. | |

For the exact wording of each term, see the authoritative [GLOSSARY.md](../../../GLOSSARY.md#language) (Japanese).
