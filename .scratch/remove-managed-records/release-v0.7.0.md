# SSH KeyFlow v0.7.0

## Changes since v0.6.0

- Remove a managed key with `ssh-keyflow keys remove --key ID_OR_NAME`.
- Remove a destination with `ssh-keyflow hosts remove --host ID_OR_ALIAS`.
- Disconnect one key/destination relationship with
  `ssh-keyflow hosts dissociate --host ID_OR_ALIAS --key ID_OR_NAME`.
- In the TUI, press `d` to remove the selected key or destination, or `u` to
  choose an existing association to disconnect. Every change requires explicit
  confirmation.
- Removing a key or destination clears its local associations. Disconnecting
  clears only the selected pair. Key files, SSH configuration, remote authorized
  keys, creation results, and legacy history are preserved. Removal does not
  revoke SSH access.

## Compatibility and migration

Existing supported state loads without a manual migration. Removed metadata is
retained in optional v2 `removed_keys` / `removed_hosts` arrays so historical
references remain valid. Reimporting a key or readding a destination creates a
new identity.

**Downgrade limitation:** older binaries reject state containing these archive
arrays. Before using removal, keep a separate copy of `state.json` if you may
need to return to an older version. `state.json.bak` is a rolling backup and is
not a permanent downgrade copy. No key files or SSH settings are deleted.

## Installation and update

Exit SSH KeyFlow and rerun the one-line installer documented in README.md, or
download the archive for your OS/CPU from this release. Check its SHA256 against
`SHA256SUMS`, extract it, and retain `THIRD_PARTY_NOTICES.txt` with the executable.
Run `ssh-keyflow --version` to confirm `SSH KeyFlow v0.7.0`.

## Verification limits

Local Go tests, vet, 85 Python regression tests, six-target packaging, checksum
and archive validation, and Windows amd64 version/help smoke checks passed.
GitHub Actions verifies Linux, macOS, and Windows and builds all six targets
before publication. Crossbuilds do not establish native execution on all targets.
Manual fullscreen TUI, OSC52 clipboard, and live SSH-server checks were not
performed for this release.
