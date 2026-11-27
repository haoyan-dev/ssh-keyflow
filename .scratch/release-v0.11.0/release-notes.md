# SSH KeyFlow v0.11.0

This release narrows SSH KeyFlow to one job: managing SSH key pairs on this
computer, with only the public key leaving it. It is a breaking release. The
destination, copy-paste guide, host key pinning and TUI features are retired,
and the state file moves to schema v3 the first time you change something.

## Changes since v0.10.0

- The CLI keeps `keys list [--json]`, `keys public --key <key>`, `keys create`,
  `keys import`, `keys remove` and `resume`.
- `keys import` now always needs `--name`, `--public` and `--private`.
  Importing only a public key is retired.
- Retired commands:
  - every `hosts` command: `add`, `list`, `remove`, `associate`, `dissociate`,
    `guide`, `pin` and `unpin`;
  - the copy-paste setup guide, including the ssh-agent autostart step;
  - the terminal UI and its `tui` command.

  Running SSH KeyFlow with no arguments now prints help. Retired commands and
  flags exit with code 2 and change nothing.
- `keys list --json` still returns `keys` and `creations`, but each key object
  has changed:
  - `connection_check` is gone;
  - `imported_at` is omitted for created keys;
  - `private_path` and `private_sha256` are always present.
- Third-party notices list only Go and the five modules the binaries ship. The
  Bubble Tea, Bubbles and Lip Gloss TUI libraries are gone. The small
  `charmbracelet/x/term` module stays; it only checks whether input is a real
  terminal.
- The documentation is available in Japanese, English and Simplified Chinese,
  and describes only the retained functions and the migration.

## Breaking changes and migration

- **Retired commands:** scripts that call `hosts …` or `tui` must stop doing
  so. Place public keys yourself (the tutorial covers both Linux/macOS and
  Windows destinations):
  1. Print the key with `ssh-keyflow keys public --key <key>`.
  2. Append it to `~/.ssh/authorized_keys` on the destination. On Windows use
     `%USERPROFILE%\.ssh\authorized_keys`, or
     `%ProgramData%\ssh\administrators_authorized_keys` for an administrator
     account.
  3. Connect with `ssh -i <private key path> -o IdentitiesOnly=yes user@host`.
- **Before updating:** exit SSH KeyFlow and back up `~/.ssh/ssh-keyflow/` to a
  separate, access-restricted location.
- **State migration:**
  - v0.11.0 reads a `state.json` from v0.10.0 or earlier (schema v1/v2)
    without changing it. `keys list` and every changing command print what will
    not carry over; changing commands print it before you confirm.
  - On the first change, the original file is copied unchanged to a new
    `~/.ssh/ssh-keyflow/state.v<N>-<UTC time>.json`. An existing archive is
    never overwritten. A retry after a failure reuses an archive with identical
    content. If the copy fails, nothing changes.
  - Only key pairs and unfinished key creations are then saved as schema v3.
    Everything else stays in the archive: destinations, associations, history,
    public-only keys, keys with an unfinished legacy deletion, and SSH config
    edit records.
  - A legacy file with unknown fields or broken values is refused, not
    repaired.
- **Public-only keys:** to keep managing one, import it again together with its
  private key using `keys import`.
- **Files that are not touched:** your SSH config, any Includes, Host blocks
  added from the old guide, and the old `~/.ssh/ssh-keyflow/known_hosts` and
  `~/.ssh/ssh-keyflow/config`. Pinning lines keep working while those files
  remain. Remove them yourself if you no longer want them.
- **Downgrading:** v0.10.0 and older cannot read schema v3.
  1. Exit SSH KeyFlow.
  2. Back up the current `~/.ssh/ssh-keyflow/`.
  3. Restore the archive as `state.json`.

  `state.json.bak` is v3 after the migration, so it cannot be used to
  downgrade. Keys created or imported after the migration are not in the
  restored file; import them again with the older version if needed.
## Installation and update

Exit SSH KeyFlow before updating. Use the source-hosted installers from `main`,
or download the archive for your OS/CPU and `SHA256SUMS` from this release.
Verify the checksum before extraction and keep `THIRD_PARTY_NOTICES.txt`.
OpenSSH (`ssh` and `ssh-keygen`) is required; end users do not need Go.
The installers are unchanged since v0.10.0.

Archives support Windows, Linux and macOS on amd64 and arm64. The release
consists of six archives and one checksum manifest. The installers remain
source-hosted.

## Verification and limits

Prepared against `f718b817527bec8ef4b6a6d27fe56aee33cc931e`. The final
tagged source and the evidence comparison are added when publishing. Passing
evidence and execution conditions are in `docs/verification.md`.

Confirmation prompts were covered by tests with scripted input, not a live
terminal session. Local runs were on Windows amd64. CI ran the tests on Linux,
macOS and Windows. Cross-builds are not native runs on each target.
Published-asset verification is recorded after publication.
