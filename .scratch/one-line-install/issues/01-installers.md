# 01: One-line installers for Portable releases

Status: ready-for-human
Blocked by: None
Resolution: completed

## Acceptance criteria

- [x] Source-hosted POSIX and PowerShell one-line installation.
- [x] Published v0.2.0 two-member archive compatibility and stable default source.
- [x] Ownership, rollback, legacy receipt upgrades, and explicit PATH consent.
- [x] Installer regression tests and three-OS CI integration.
- [x] README and application contract describe install, update, and uninstall.
- [x] Verification and independent standards/specification review recorded.

## Comments

2026-11-07: User requested installation like https://herdr.dev/download/.
The scope and behavior are defined in ../spec.md.

2026-11-07: Completed over main `3e71b0e`. All 50 installer regression cases and
30 profile/caller cases passed across the final platform-specific runs; all five
Portable tests passed. Actual released v0.2.0 Windows amd64 installation, latest
update and removal succeeded through PowerShell 5.1/7 with an isolated profile.
Separate Standards and Specification re-reviews reported no remaining findings.
Script hashes, commands, execution conditions and native Linux/macOS/ARM64 gaps
are recorded in docs/verification.md. CI runs both suites on three OSes.
No commit, push, tag or remote publication was performed; one-line URLs become
available when the scripts are published to main.

2026-11-07: User authorized tagging and release. Publish the completed installers
to main and release v0.2.1 through the existing workflow. The release phase in
../spec.md supersedes the earlier no-publication scope; published verification
will be recorded after the workflow completes.

2026-11-07: Released v0.2.1 from `3e5674430cf7d0a2e031aa1c214a70af5c666982`
and confirmed it is the latest stable release. Release workflow 42351144039 and
main workflow 42351143967 passed on this commit. Downloaded and verified all
seven published assets, the tagged notices, six archive checksums and both raw
installer URLs. Actual published v0.2.1 Windows amd64 install (PowerShell 5.1),
latest update/PATH setup (PowerShell 7) and uninstall (PowerShell 5.1) passed
using isolated paths; the exact original profile bytes were restored. Public
release notes include installation and usage. Details and remaining validation
limits are in docs/verification.md. The released tag remains on the tested
implementation commit; the final documentation update is a separate commit.

2026-11-13: Released in v0.2.1 (`3e56744`); the installers are still the supported install path.
