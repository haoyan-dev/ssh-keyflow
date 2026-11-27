Status: ready-for-agent
Blocked by: none

# 01: Prepare and integrate v0.11.0

Complete spec criterion 1 and separate standards/specification reviews.
Use the tested source SHA, not a self-referential preparation commit SHA.
Reuse evidence only when the inputs and conditions support it.

## Comments

- 2026-11-28: Local README checks passed on a clean `f718b81` checkout.
  test_installers had one intermittent error in the first run; reruns passed
  (see the record in `docs/verification.md`). The preparation PR is pending.
- 2026-11-28: The separate standards and spec reviews of the preparation
  found no hard violations. Their findings (wrong `keys public` syntax, the
  JSON change description, archive and downgrade wording, precedent details)
  are fixed and recorded. The user authorized merging after CI, then the tag
  and release.
