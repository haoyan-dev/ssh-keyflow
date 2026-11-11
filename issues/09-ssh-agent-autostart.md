Status: ready-for-agent

# 09: ssh-agent automatic startup for passphrase-protected keys

Specification: [.scratch/ssh-agent-autostart/spec.md](../.scratch/ssh-agent-autostart/spec.md)

Blocked by: none

For keys detected as passphrase-protected (non-prompting empty-passphrase
check), add `AddKeysToAgent yes` to the generated SSH config. Also add a local
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
