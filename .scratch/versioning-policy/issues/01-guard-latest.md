Status: ready-for-human
Resolution: completed

# Guard latest against older maintenance releases

## Problem

The release workflow passes `--latest` for all normal releases. Publishing an
older maintenance series can replace the newer stable latest release.

## Acceptance criteria

- Resolve the repository's current latest release before publication and compare
  versions by the declared component scheme, including numeric ordering.
- Publish older supported maintenance releases without moving latest backwards.
- Keep prereleases excluded; allow a first normal release when latest is absent.
- Treat lookup failures separately from a confirmed absence of releases.
- Cover 1.9.1 versus 1.10.0, first release, prerelease, lookup failure, and
  concurrent publication of different tags.
- Preserve immutable assets, target repository, and tag-to-source verification.

## Comments

2026-11-11: Found during the versioning policy audit. Implementation is outside
the requested documentation and personal-skill changes.

2026-11-11: User authorized the follow-up. Implementation is isolated on
`fix/01-release-latest` in `.worktrees/release-latest`; its specification and
verification record are `.scratch/release-latest/spec.md` and
`docs/verification.md`. Numeric stable comparison, confirmed first-release
handling, fail-closed lookup errors, preview exclusion, and a shared publication
queue are implemented. Seventeen mocked publication tests and a read-only live
latest lookup pass. Go tests/vet, 5 portable tests, 50 installer tests, and 30
profile tests passed. Workflow YAML/wiring and standards/specification reviews
passed. Hosted scheduling and live publication remain untested; no release was
published. Integration is the remaining delivery step.

2026-11-11: Integrated through SSH KeyFlow PR #26 at
`7641bc0b44dedf3269963271f453077763399bd8`. Push run 42969232585 and PR run
42969239502 passed Linux, macOS, Windows, and six-target cross-build checks on
reviewed head `b4cefe0d91c37f106685661f739e43a7dc7b2151`. Independent standards
and specification reviews reported zero findings. Hosted tests verify the code;
live release publication and the publication queue itself remain unexercised.

2026-11-13: Released in v0.9.1, which contains `7641bc0`.
