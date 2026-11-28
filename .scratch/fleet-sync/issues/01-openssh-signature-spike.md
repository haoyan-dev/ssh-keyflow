# 01 — Spike `ssh-keygen -Y` signing across platforms

Status: ready-for-human
Blocked by: none

Criterion 1. Research only: throwaway scripts, no product code. Check
`ssh-keygen -Y sign`, `-Y verify` and `allowed_signers` with namespace
`ssh-keyflow-roster` on Windows 11, macOS and Linux (local plus CI runners).
Cover a passphrase-protected Ed25519 key, signing through ssh-agent, the exit
codes for a bad signature, a wrong namespace and an unknown signer, and how
passphrase prompts behave without a TTY. Record the minimum OpenSSH version and
findings in `.scratch/fleet-sync/openssh-spike.md`.

## Comments

- 2026-11-28: Windows 11 (OpenSSH 9.5p2) and Ubuntu 24.04 via WSL (9.6p1) pass
  all cases; see `openssh-spike.md`. macOS and the CI runners are still pending,
  as are agent signing on Windows and a real-terminal prompt.
- 2026-11-28: CI run 46111885878 passed on ubuntu (10.2p1), macOS (10.5p1,
  including agent signing) and windows-latest (10.6p1, Git for Windows' build).
  Remaining gaps are listed in `openssh-spike.md`; ready for review.
- 2026-11-28: Standards and spec reviews found no hard violations. Fixed the
  case counts (18 on Unix, 15 on Windows), marked the -1 exit code as
  local-only evidence, and labelled the minimum version (8.4) and design
  consequences as proposals for ticket 02.
