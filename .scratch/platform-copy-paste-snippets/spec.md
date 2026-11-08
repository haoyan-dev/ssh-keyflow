# Platform-specific terminal snippets

## Objective

The guide must provide scripts users can copy directly into the corresponding
terminal, rather than requiring them to translate prose into commands.

## Scope and acceptance

- Generate a destination public-key placement script for Linux/macOS POSIX shells
  or Windows PowerShell, respecting standard versus administrator Windows accounts.
- For a managed private key, generate a local SSH config append script for the
  OS running SSH KeyFlow, followed by a separate connection command.
- Label every step with local/destination context, platform, account requirements
  and file effects. Server installation, services and firewall setup remain outside
  this feature.
- TUI guide opens at step one. Left/right or b/n selects a step; y copies only
  that step's executable script. CLI prints all labeled steps.
- Preserve existing authorized keys and SSH config bytes. Placement is repeatable
  without duplicate keys; config setup refuses an existing literal alias, including
  grouped aliases. Existing wildcard/Match/Include precedence remains user-reviewed.
- Do not execute generated scripts, modify app state, claim remote success, or
  disclose private material when displaying/copying.
- Preserve identity validation, public-only behavior, unsupported expandable-path
  handling, and stale-copy invalidation, including navigation through Help.
- Exercise native POSIX and PowerShell scripts only against temporary files.

## Decisions

The existing `Guide` API remains the CLI text entry point. `PrepareGuide` supplies
the same validated guide as structured steps for the TUI. Destination OS comes from
the saved host; local OS comes from the application build. No new dependencies,
flags, state schema, installer behavior, or release publication.

The user request authorizes users running generated placement/config scripts.
ADR 0002 clarifies that these are distinct from the server provisioning scripts
excluded by ADR 0001.

## Starting point

Fetched origin/main `e5e826c32cb79b5267cd5ee4f327ad516ffb659c`;
work branch `feat/05-platform-copy-paste-snippets`.
