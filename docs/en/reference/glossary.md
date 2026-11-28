# SSH KeyFlow glossary

[简体中文](../../zh-CN/reference/glossary.md) · English · [日本語](../../../GLOSSARY.md)

> This page is an English overview. The authoritative glossary is the Japanese [GLOSSARY.md](../../../GLOSSARY.md); if they differ, the Japanese page wins.

[Documentation index](../README.md)

SSH KeyFlow is a CLI that manages the SSH key pairs on this machine and shows public keys in a form you can copy. The English documentation uses the terms below consistently.

| English term | Canonical term (GLOSSARY.md) | Meaning | Terms to avoid |
| --- | --- | --- | --- |
| Managed key | 管理対象の鍵（Key） | A key pair on this machine, created by the tool or explicitly imported by the user. Its fixed ID, fingerprint, public and private key paths, private key digest and origin (created or imported) are saved. Private key contents are not saved. | Public-only key (公開鍵のみの鍵, retired in v0.11.0) |
| Import | 取り込み（Import） | Verifying an existing public and private key pair and adding it to management without moving the files. Importing a public key alone is not provided. | |
| Local key creation | ローカル鍵作成（Creation） | Key creation with the native ssh-keygen, together with the saved progress used to resume with the same ID and paths after an interruption. It is the only operation `resume` handles, and it is removed from the state when completed. | |
| Removal from management | 管理からの削除（Removal） | Removing a key from the managed list. Key files, SSH configuration and public keys already placed on other machines are not changed, and SSH access is not revoked. | Local key deletion (ローカル鍵の削除) |
| Local key deletion | ローカル鍵の削除 | Removing public and private key files from the machine running the app. Not part of the current features. | Removal from management (管理からの削除) |
| Manual public key deployment | 手動の公開鍵配置 | The user appending the output of `keys public` to the destination's `authorized_keys` or similar. The app neither performs nor verifies it. | |
| Previous-state backup | 直前状態のバックアップ | `state.json.bak` written when the state is saved: a rolling copy of the bytes of the last valid state. | |
| Migration archive | 移行アーカイブ（Migration archive） | `state.v<N>-<UTC time>.json`, the v1 / v2 `state.json` written out before the first save. It is never overwritten or deleted automatically, and keeps the original bytes, including destinations and history not carried over to v3. Restore it manually when downgrading. | Backup (バックアップ; distinguish it from `state.json.bak`) |

For the exact wording of each term, see the authoritative [GLOSSARY.md](../../../GLOSSARY.md#language) (Japanese).

## Planned terms (ADR 0008, not implemented)

These terms come from [ADR 0008](../../adr/0008-signed-fleet-roster.md) (Japanese, accepted) and are not part of the current features yet.

| English term | Canonical term (GLOSSARY.md) | Meaning | Terms to avoid |
| --- | --- | --- | --- |
| Fleet | フリート（Fleet） | The user's nodes that pin the same fleet root key as their trust anchor. | |
| Node | ノード（Node） | A machine enrolled in the fleet. Its roles are client, remote, or both. | |
| Admin node | 管理ノード（Admin node） | A node that holds the fleet root key and signs the roster. | |
| Client | 接続元（Client） | The role of a machine you SSH from. The public key of a key pair created by the tool is listed in the roster. | Host (in OpenSSH, `Host`, host keys and `known_hosts` refer to the destination) |
| Remote | 接続先（Remote） | The role of a machine that accepts SSH through sshd. Its existing host public keys are listed in the roster, and client public keys are accepted through the managed block in `authorized_keys`. | Confusing it with the destination records retired in v0.11.0 |
| Fleet root key | フリートルート鍵（Fleet root key） | The passphrase-protected key on an admin node that signs the roster. Every node pins only this public key and any registered backup root public keys as trust anchors. | |
| Roster | 名簿（Roster） | A versioned document, signed by the fleet root key, listing the nodes with their public keys, host public keys, roles and login accounts. | |
| Managed block | 管理ブロック（Managed block） | A delimited range in `authorized_keys` owned by the tool. Lines outside the block are never changed. | |
| Relay | 中継サーバー（Relay） | An untrusted server that passes signed rosters and the enrollment code exchange between nodes. It can withhold updates but cannot forge keys. | |
