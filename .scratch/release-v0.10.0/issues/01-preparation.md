Status: ready-for-human
Blocked by: none

# 01: Prepare and integrate v0.10.0

Complete spec criterion 1 and separate standards/specification reviews.
Use the tested source SHA, not a self-referential preparation commit SHA.
Reuse installer/matrix evidence only when the inputs and conditions support it.

## Comments

- 2026-11-12: Local checks passed on `aea242f`; preparation PR pending.
- 2026-11-12: CI run `43213790107` passed on `aea242f`; v0.10.0 still unused
  locally and remotely.
- 2026-11-12: Reviews fixed in `0ff418f`; all local checks reran on that clean
  tree and it is the new tested source.
- 2026-11-12: Preparation integrated by PR #34 as `0af2631`; required CI passed.
