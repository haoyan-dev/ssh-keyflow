# Spike: `ssh-keygen -Y` roster signing (ticket 01)

Date: 2026-11-28. Base: `main` 18ebb9d. Script:
`spike/openssh_sign_spike.py` (throwaway, stdlib Python; temporary keys only,
never the user's keys or agent). Namespace `ssh-keyflow-roster`.

## Environments run

| Platform | OpenSSH | Result |
| --- | --- | --- |
| Windows 11 amd64 (local) | OpenSSH_for_Windows_9.5p2, LibreSSL 3.8.2 | 15/15 cases as expected; agent cases not run (see below) |
| Ubuntu 24.04 (local WSL 2) | OpenSSH_9.6p1 Ubuntu-3ubuntu13.20 | 19/19 cases as expected |
| CI ubuntu-latest | OpenSSH_10.2p1 Ubuntu-2ubuntu3.8, OpenSSL 3.5.5 | 19/19 as expected |
| CI macos-latest | OpenSSH_10.5p1, LibreSSL 3.3.6 | 19/19 as expected, including agent signing |
| CI windows-latest | OpenSSH_10.6p1, OpenSSL 3.5.9 | 15/15 as expected; agent cases not run |

CI: run 46111885878 on `8e3feff` (temporary workflow `spike-openssh.yml`,
removed afterwards). The script runs under bash there, so on windows-latest the
`ssh`/`ssh-keygen` found first on PATH is evidently Git for Windows' build
(OpenSSL, exit 255, Unix-style socket error from `ssh-agent`), not the built-in
Win32-OpenSSH tested locally. The LibreSSL 3.3.6 build on macOS suggests
Apple's system OpenSSH. The script reports `ssh -V`, not the resolved
`ssh-keygen` path, so these identifications are inferred.

## Results

Same behaviour on Windows and Linux unless noted.

- **Sign** (`-Y sign -f <key> -n ssh-keyflow-roster <file>`) writes
  `<file>.sig`, an armored `-----BEGIN SSH SIGNATURE-----` block; progress goes
  to stderr.
- **Verify** (`-Y verify -f allowed_signers -I <principal> -n <ns> -s <sig>`,
  data on stdin) exits 0 and prints `Good "ssh-keyflow-roster" signature for
  <principal> with ED25519 key SHA256:…` on stdout.
- **Rejected, non-zero exit:** tampered data, the same data with LF→CRLF, wrong
  namespace, unknown principal, a principal bound to another key, a key not in
  `allowed_signers`, and an `allowed_signers` line whose `namespaces=` excludes
  the namespace. Only some failures give a reason on stderr (`incorrect
  signature`, `namespace does not match`, `key is not permitted…`); principal
  and unlisted-key failures print only `Could not verify signature.` on stdout.
- **Exit codes differ:** Linux, macOS and Git for Windows' build return 255;
  the built-in Windows OpenSSH returns -1 (4294967295 as unsigned). Treat any
  non-zero exit as failure; do not match 255.
- **Which ssh-keygen runs matters on Windows:** two different builds can be on
  PATH (built-in Win32-OpenSSH and Git for Windows). The tool should resolve
  and report the ssh-keygen it uses, and check its version.
- `-Y find-principals` returns the principal for a signature;
  `-Y check-novalidate` checks a signature without a signer list.
- **Passphrase key without a terminal** (stdin closed; Linux new session with no
  controlling TTY, Windows detached process) fails fast with exit 255 and
  `incorrect passphrase supplied to decrypt private key`. It did not hang. Linux
  printed `Enter passphrase:` on stderr first.
- **`SSH_ASKPASS` + `SSH_ASKPASS_REQUIRE=force`** makes ssh-keygen read the
  passphrase from the helper with no terminal, on both platforms (a `.cmd`
  helper on Windows). The signature verifies.
- **ssh-agent (Linux, macOS):** a private `ssh-agent -D -a <socket>`, `ssh-add` via
  askpass, then `-Y sign -f <key>.pub` signs with the agent-held key; the
  signature verifies.
- **ssh-agent (Windows):** a private `ssh-agent -d -a <pipe>` cannot start
  (`cannot create agent root reg key, ERROR:5`). Windows' agent runs as a
  service and keeps added keys in the registry across reboots, so the spike did
  not touch it. Agent signing on Windows is untested.

## Minimum OpenSSH version

From the official release notes (openssh.org/txt, read 2026-11-28):
- 8.1 (2019-10-09): "add an experimental lightweight signature and verification
  ability" (`-Y sign` / `-Y verify`).
- 8.2 (2020-02-14): `find-principals`.
- 8.4 (2020-09-27): `$SSH_ASKPASS_REQUIRE` (notes name ssh and ssh-agent; the
  spike shows ssh-keygen honours it too on 9.5/9.6).

Proposal for ticket 02: require OpenSSH 8.4 or later if the tool relies on
`SSH_ASKPASS_REQUIRE`, otherwise 8.2. Every build tested (9.5p2 to 10.6p1) is far above it.
Check `ssh -V` at runtime and fail with a clear message below the minimum.

## Consequences for the design

1. The roster's canonical bytes must be fixed exactly (LF, no BOM). A CRLF
   change breaks the signature, so a git-based static backend must store
   rosters as binary (`.gitattributes -text`) or the tool must sign a canonical
   form it re-derives, not the file bytes as checked out.
2. Detect failure by exit code only; surface ssh-keygen's stderr as detail, and
   use `find-principals` to explain "signed by a key that is not your root".
3. Signing needs either an inherited terminal (interactive prompt), an agent,
   or an askpass helper. The phase-1 CLI is interactive, so the default can be
   to let ssh-keygen prompt on the inherited console; still to verify with a
   real terminal.
4. Pin roots by writing our own `allowed_signers` with `namespaces="ssh-keyflow-roster"`,
   so a root key cannot be reused to accept other namespaces.

## Not verified

- The built-in Win32-OpenSSH on a CI runner (only Git for Windows' build ran
  there); the exact ssh-keygen paths on the runners.
- Interactive passphrase prompt on a real terminal (the tool sessions have none).
- Agent signing on Windows; FIDO (`ed25519-sk`) keys.
- OpenSSH versions between 8.4 and 9.5.
