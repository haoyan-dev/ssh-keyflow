# 01: One-line installers for Portable releases

Status: ready-for-agent
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
