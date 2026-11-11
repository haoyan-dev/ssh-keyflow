Status: ready-for-human

# 09: ssh-agent automatic startup for passphrase-protected keys

Specification: [.scratch/ssh-agent-autostart/spec.md](../.scratch/ssh-agent-autostart/spec.md)

Blocked by: none

For keys detected as passphrase-protected from the private key's format
header, or whose protection cannot be checked, add `AddKeysToAgent yes` to the generated SSH config. Also add a local
guide step that sets up an automatically started ssh-agent: the Windows
`ssh-agent` service, macOS launchd plus Keychain, or a Linux check with
manual instructions. Scripts remain copy-paste only; the app never handles
passphrases or starts services.

## Comments

- 2026-11-11: Filed at the user's request.
- 2026-11-11: The user decided the open questions. Show the step only for
  detected passphrase-protected keys. The Linux step is a manual guide.
  `AddKeysToAgent yes` keeps keys until the agent stops. Keys that cannot be
  checked get no step, only a notice. Ready for agent.
- 2026-11-11: The user revised the earlier rule. Keys whose passphrase
  protection cannot be checked still get the step and config line, with a
  notice that they apply only if the key has a passphrase.
- 2026-11-11: Implemented on `feat/09-ssh-agent-autostart` from
  `origin/main` `a08bf976994e3ae91955ee65773a0070388f9251`. Detection reads
  only the key format header instead of running `ssh-keygen -y -P ""`; same
  outcome, no child process, no possible prompt (spec and ADR 0005).
- 2026-11-11: Separate standards and specification reviews found ticket,
  wording, notice, `~` quoting and test-cleanup issues; all fixed in
  `d5a8456e108f8f477ca6fd673403a7cde397e709`. Full tests and vet passed.
  Evidence and limits are in `docs/verification.md`. Ready for PR review.
- 2026-11-11: The user requested merge. PR #22 merged as
  `f9956c92c26027bfeae642361e9e56b3eab4adcf` after all three OS tests and
  crossbuild passed on head `880680609d755c1a4a9e6b6737235dd1b982b425`; the
  merged tree matches that head. Main Verify run 42951426479 passed. #20
  closed by the merge; ticket PR #21 closed as superseded. Task branches
  deleted locally and remotely. No release requested.
