# 04 — Fleet root key and roster signing

Status: needs-triage
Blocked by: 02, 03

Criterion 3. Create the passphrase-protected Ed25519 fleet root key with native
ssh-keygen, store only its path and fingerprint, and sign rosters under
namespace `ssh-keyflow-roster`. The private key never appears in state or output.

## Comments
