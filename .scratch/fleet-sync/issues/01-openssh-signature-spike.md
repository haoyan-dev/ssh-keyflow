# 01 — Spike `ssh-keygen -Y` signing across platforms

Status: ready-for-agent
Blocked by: none

Criterion 1. Research only: throwaway scripts, no product code. Check
`ssh-keygen -Y sign`, `-Y verify` and `allowed_signers` with namespace
`ssh-keyflow-roster` on Windows 11, macOS and Linux (local plus CI runners).
Cover a passphrase-protected Ed25519 key, signing through ssh-agent, the exit
codes for a bad signature, a wrong namespace and an unknown signer, and how
passphrase prompts behave without a TTY. Record the minimum OpenSSH version and
findings in `.scratch/fleet-sync/openssh-spike.md`.

## Comments
