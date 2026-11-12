# OpenSSH behaviour spike (2026-11-12)

Checks the unverified items in `spec.md` before implementation. Throwaway keys
and an unprivileged sshd only. No user SSH files were changed. All temp files,
processes and the agent key were removed afterwards.

## Setup

- Linux servers: an unprivileged `/usr/sbin/sshd -D -e -f <temp config>`
  listening on `127.0.0.1:22471`, with a temp host key and a temp
  `AuthorizedKeysFile`:
  - ts-dev-3060: Ubuntu, OpenSSH 9.6p1, x86_64.
  - sakura: Debian 12, OpenSSH 9.2p1, x86_64.
- Linux clients: the same machine, `ssh -F <temp config>` with `BatchMode yes`.
- Windows 11 clients: Win32-OpenSSH 9.5p2 (`C:\Windows\System32\OpenSSH\ssh.exe`)
  and Git for Windows OpenSSH 10.2p1. They reached the ts-dev-3060 sshd
  through a loopback-only `ssh -L 127.0.0.1:22471:127.0.0.1:22471` forward.
- Pinned config form under test:
  `HostKeyAlias ssh-keyflow-h1`, `UserKnownHostsFile "<abs>"`,
  `StrictHostKeyChecking yes`. The pin file has lines of the form
  `ssh-keyflow-h1 <type> <base64>`.
- Not tested: macOS (no machine available) and 1Password (not installed). The
  Linux arm64 machine wasn't used because the client code paths are the same.

## Results

| # | Check | Linux 9.6 | Linux 9.2 | Win32 9.5 | Git 10.2 |
| --- | --- | --- | --- | --- | --- |
| 1a | Correct pin, port 22471 connects with no prompt | pass | pass | pass | pass |
| 1b | Wrong pin is refused (`Host key verification failed.`) | pass | pass | pass | pass |
| 1c | `[ssh-keyflow-h1]:22471` entry is refused, so the lookup name is the bare alias and the port is ignored | pass | pass | — | — |
| 2a | `UserKnownHostsFile "C:/Users/.../hkp spike/known_hosts"` (forward slashes, space) | — | — | pass | pass |
| 2b | `UserKnownHostsFile "C:\\Users\\...\\hkp spike\\known_hosts"` (escaped backslashes, space) | — | — | pass | pass |
| 3a | `IdentityFile "<x>.pub"` with no private file, `IdentitiesOnly yes`, key in agent: authenticates | pass | pass | pass (Windows agent service) | — |
| 3b | Same with the key not in the agent: `Permission denied (publickey)` | pass | pass | pass, with warning* | — |
| 4 | Host key `.pub` files readable by a non-elevated user | yes (`root:root 0644`) | yes | **no, access denied** | — |

\* When no agent key matches, Win32-OpenSSH tries to load the `.pub` file as a
private key and prints an `UNPROTECTED PRIVATE KEY FILE` / `bad permissions`
warning before it fails. This only happens on the failure path.

Every Linux host key line printed by `cat /etc/ssh/ssh_host_*_key.pub` had
three fields: `<type> <base64> <comment>`. The types seen were `ssh-ed25519`,
`ecdsa-sha2-nistp256` and `ssh-rsa`.

## Design consequences

- Pinned snippet form confirmed. Keep the forward-slash absolute path with the
  existing `IdentityFile` escaping. The port doesn't affect the lookup, so a
  port change keeps the pin.
- `.pub` `IdentityFile` for public-only keys is confirmed. The guide text must
  say the matching key has to be loaded in the agent. On Windows, the
  permission warning means the key wasn't in the agent.
- On Windows the "Print host keys" step needs an **elevated PowerShell**
  (Run as Administrator) for both account types, because standard users can't
  read `%ProgramData%\ssh\ssh_host_*_key.pub`.
- Still unverified: macOS and the 1Password agent. Record these as limits in
  `docs/verification.md`.
