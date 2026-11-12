# SSH KeyFlow v0.10.0

This minor release adds host key pinning: you paste a destination's host public
keys once, and the generated Host snippet then accepts only those keys, with no
first-connection prompt. Keys whose private half lives in an SSH agent now get
a Host snippet and connection command too.

## Changes since v0.9.2

- New `hosts pin --host <ref>` command. In a terminal it prints the command to
  run on the destination (`cat /etc/ssh/ssh_host_*_key.pub`, or an
  Administrator PowerShell `Get-Content` on Windows), reads the pasted output,
  shows each key's type and SHA256 fingerprint, and asks for a typed `yes`.
  `--file` and stdin with `--yes` are also accepted.
- Pinned keys are kept in the app-owned `~/.ssh/ssh-keyflow/known_hosts`,
  named by the stable destination ID. The file is read strictly and never
  repaired; `~/.ssh/known_hosts` is never edited.
- Replacing different pinned keys needs `--replace`, shows both fingerprint
  sets, and needs a typed `yes` in a real terminal. `--yes` never replaces.
- New `hosts unpin --host <ref>` command (real terminal and typed `yes`).
- Until a destination is pinned, `hosts guide` and the TUI guide add two steps
  before the SSH config step: print host keys on the destination, then pin them
  locally. Once pinned, the Host snippet gains `HostKeyAlias`,
  `UserKnownHostsFile` and `StrictHostKeyChecking yes`.
- `hosts list` shows a `HOST KEYS` column; `--json` adds `host_key_pins` for
  managed destinations.
- TUI: `t` pins and `T` unpins the selected destination through the native
  terminal; details show `Host keys: pinned (N)`, `not pinned` or `unknown`.
- Removing a pinned destination keeps its pins and says to run `hosts unpin`
  first.
- Public-only keys now get `IdentityFile "<public key path>"` with
  `IdentitiesOnly yes` and a connection command, so ssh uses the matching key
  from your SSH agent (for example ssh-agent with a FIDO key). No
  `AddKeysToAgent` or agent autostart step is added for them.
- The documentation no longer names a third-party key manager.

## Compatibility and migration

No breaking changes to existing commands, flags, exit codes or JSON fields;
`host_key_pins` and the `HOST KEYS` column are additions. The state schema is
unchanged. Pins live in a separate file that older versions ignore, so
downgrading to v0.9.2 is safe; snippets with the pinning lines keep working
with plain OpenSSH because the pin file is a standard `known_hosts` file.

Guide output changes:

- Unpinned destinations on this computer get two new steps. Their Host snippet
  is unchanged until they are pinned.
- Public-only keys get a Host snippet and connection command where v0.9.2 gave
  only the placement step.

Hosts already in your SSH config are never edited. To pin one, run
`hosts pin`, then add the three pinning lines from the guide yourself. If a
destination's host keys change, ssh refuses to connect until you confirm the
new keys on the destination and run `hosts pin --host <ref> --replace`.

## Installation and update

Exit SSH KeyFlow before updating. Use the source-hosted installers from `main`,
or download the archive for your OS/CPU and `SHA256SUMS` from this release.
Verify the checksum before extraction and retain `THIRD_PARTY_NOTICES.txt`.
OpenSSH (`ssh` and `ssh-keygen`) is required; end users do not need Go.

Archives support Windows/Linux/macOS amd64 and arm64. The release consists of
six archives and one checksum manifest; installers remain source-hosted.

## Verification and limits

Prepared against `aea242f57153e7819437e29d1bf15e5b1302fea0`.
The final tagged source and evidence comparison are added when publishing.
Passing evidence and execution conditions are in `docs/verification.md`.

OpenSSH behaviour was checked on Linux OpenSSH 9.6p1 and 9.2p1, Win32-OpenSSH
9.5p2 and Git for Windows OpenSSH 10.2p1. macOS clients and servers were not
tested. Interactive paste and the TUI were covered by tests with scripted
input, not a live terminal session. Crossbuild evidence is distinct from native
execution. Published-asset verification is recorded after publication.
