# 03 — State schema v4 and node identity

Status: needs-triage
Blocked by: 02

Criteria 5 and 6. Add the node record (id, name, roles, login account override,
addresses) and fleet data to state v4, with an archived migration from v3
following the v3 rules. A client lists a managed key. A remote reads its sshd
host public keys read-only, from the `HostKey` entries in `sshd_config`, falling
back to the platform defaults (`/etc/ssh`, Windows `%ProgramData%\ssh`).

## Comments
