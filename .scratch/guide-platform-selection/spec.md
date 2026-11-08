# Explicit guide platforms

## Objective and scope

Both terminals must have explicit platform choices: linux, macos, windows.
Keep generating scripts for the user to paste; never execute them or modify
saved destinations/keys while preparing a guide.

## Behavior and acceptance

- After selecting a destination/key, the TUI opens a form showing local OS
  (default: running app OS), destination OS/account (default: saved values),
  and an optional local IdentityFile path. Ctrl+S generates the guide;
  validation errors remain on the form; Esc cancels.
- CLI hosts guide adds --local-os, --remote-os, --windows-account and
  --identity-file. Omitted options preserve current defaults.
- Overrides apply only to this guide. A changed destination OS never inherits
  a Windows account from a different platform; Windows requires a confirmed
  standard/administrator choice. Reject account input for a non-Windows target.
- A key pair with a different local OS requires an explicit path as seen by
  that local terminal. Do not guess WSL drive mappings or copy private keys.
  Accept an absolute path for that OS or ~/path; reject relative paths,
  control characters and paths from the other platform. Do not inspect a
  foreign terminal's filesystem or claim that its key matches.
- Public-only keys still get only the destination step, regardless of platform
  choices; an IdentityFile override cannot turn them into a managed key pair.
- Each step identifies the selected platform and terminal; the connection step
  uses the same local platform as local configuration.
- Preserve fingerprint/digest checks, unsupported environment-expansion
  handling, existing data guards, script-only clipboard copying and F5
  invalidation including help/back history.

## Decisions and dependencies

Use the existing TUI form and shared guide generator; no new dependencies or
state schema. Update ADR 0002's runtime-only local OS rule through ADR 0003.
Default detection is a starting value users can change, not an assumption that
all clients use Windows. No open questions block implementation.
