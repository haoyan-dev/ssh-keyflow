# 05 — File export, acceptance and verification

Status: needs-triage
Blocked by: 04

Criteria 4, 7 and 8. Export the signed roster and a node's public description as
files. Acceptance verifies against the pinned root keys, pins the root on first
acceptance after fingerprint confirmation, rejects bad or unknown signatures,
lower versions and same-version different content, and warns on age. Supports
`--dry-run`. Tests on all three CI OSes; contract and ja/en/zh-CN docs at
release.

## Comments
