# One-line installation

## Objective and scope

Provide the terminal installation experience requested on 2026-11-07, using
Herdr's download page as the reference: paste one command on macOS/Linux or
Windows, then launch SSH KeyFlow.

This request supersedes the v0.2.0 decision to provide only manual Portable
installation. Keep the existing six archives, notices, and six-entry SHA256SUMS
unchanged. Add source-hosted installers compatible with the published v0.2.0
archives. No new application version, domain, website, package manager, background
updater, SSH data migration, or remote publication is required for this change.

## Behavior

- POSIX: `curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh`.
- PowerShell: `irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex`.
- Default repository: `haoyan-dev/ssh-keyflow`; resolve the latest stable GitHub
  release or accept an explicit version when invoking a saved script.
- Detect OS and amd64/arm64, download the appropriate archive and SHA256SUMS,
  require one matching checksum, and allow only the two regular archive members.
- Install executable and notices to the v0.1.0 user-local default locations,
  retain ownership receipts, and preserve the previous installation on failure.
- Offer persistent PATH changes only with terminal consent. Explicit
  `--add-to-path` / `-AddToPath` is consent for unattended use. Piped scripts
  must not consume their own source as confirmation input.
- Preserve exact owned PATH stanzas and remove only those on uninstall. Refuse
  modified/unowned executable or notices. Upgrade intact v0.1.0 receipts.
- Never modify SSH keys, state, SSH configuration, or remote registrations.
- Document that source URLs work after the scripts are merged/pushed; do not
  present unshipped commands as already available.

## Acceptance and verification

Verify install, update, removal, latest/version/architecture selection,
checksums, malformed archives, ownership and rollback with isolated loopback
fixtures. Exercise `sh` pipeline and PowerShell `Invoke-Expression` entry points.
Check native Windows PowerShell 5.1 and PowerShell 7, plus shell paths via Git
Bash locally. Run installer tests in the existing three-OS CI matrix.
Document native Linux/macOS and ARM64 gaps separately.

## Release phase (2026-11-07)

The user's follow-up, "tagをつけて、リリース", authorizes committing and pushing
this implementation to main, creating tag v0.2.1, and publishing the release
through the existing tag-triggered workflow. This supersedes the initial
exclusion of a new version and remote publication. Verify hosted CI, all seven
published assets, source-hosted URLs and an isolated Windows install/update/
uninstall against the published release, then record results without moving
the released tag.
