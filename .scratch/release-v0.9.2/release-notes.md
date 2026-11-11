# SSH KeyFlow v0.9.2

This patch release improves project documentation and its validation. Application
behavior and distribution formats are unchanged from v0.9.1.

## Changes since v0.9.1

- Reorganized the README into a shorter introduction with preserved section
  anchors and links to detailed documentation.
- Added a task-oriented documentation index, first-connection tutorial,
  installation/TUI/CLI/data-recovery/troubleshooting guides, contributor
  instructions, and documentation/release maintenance guidance.
- Added offline documentation link/heading validation and ten regression tests
  to CI.
- Clarified release preparation, final-candidate verification, publication,
  and post-publication audit gates.
- Recorded requirements for Simplified Chinese (`zh-CN`), English (`en`), and
  Japanese (`ja`). Translation implementation and locale validation are still
  pending; this release does not provide complete trilingual documentation.

## Compatibility and migration

No breaking application-contract changes or new state migration are introduced
relative to v0.9.1. Commands, flags, JSON, exit codes, runtime requirements,
key handling, and installation/distribution behavior are unchanged.

Existing state upgrade/downgrade limits still apply. Keep a separate compatible
state/key backup before downgrading; `state.json.bak` is a rolling backup.
Managing or removing local associations does not revoke SSH access.

## Installation and update

Exit SSH KeyFlow before updating. Use the source-hosted installers from `main`,
or download the archive for your OS/CPU and `SHA256SUMS` from this release.
Verify the checksum before extraction and retain `THIRD_PARTY_NOTICES.txt`.
OpenSSH (`ssh` and `ssh-keygen`) is required; end users do not need Go.

Archives support Windows/Linux/macOS amd64 and arm64. The release consists of
six archives and one checksum manifest; installers remain source-hosted.

## Verification and limits

Prepared against `aaa8defaa11fdada24b9c0e37175b8c3681cc679`.
The final tagged source and evidence comparison are added when publishing.
Passing evidence and execution conditions are in `docs/verification.md`.

No live SSH server, elevated Windows service setup, OSC52 delivery, or native
macOS/Linux interactive tutorial is claimed. Crossbuild evidence is distinct
from native execution. Published-asset verification is recorded after publication.
