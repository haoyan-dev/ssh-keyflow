# Release v0.10.0

Status: ready-for-human
Previous release: v0.9.2
Tested source: `aea242f57153e7819437e29d1bf15e5b1302fea0`

## Objective and scope

Publish the next supported stable SSH KeyFlow release from integrated `main`,
following all preparation, candidate verification, publication, and audit gates.
The user's explicit request ("do the cleanup then release v0.10.0") authorizes
the preparation and audit PRs, their integration after checks and reviews, the
annotated tag, and the automatic release.

Select v0.10.0: since v0.9.2 the release adds compatible features (host key
pinning commands, guide steps, list output, TUI keys) and new guide output for
public-only keys. 0.x development increments MINOR for compatible features and
resets PATCH. No existing command, flag, exit code, JSON field or state schema
changes incompatibly, and no 1.0.0 graduation is requested. Local and remote
tags and releases were refreshed and v0.10.0 was unused when selected; recheck
immediately before tagging.

The same preparation PR removes the remaining 1Password mentions from the
guides, ADR 0006, the host key pinning spec and the guide text, at the user's
request. Dated verification and spike records stay as written.

## Acceptance criteria

1. Merge a preparation PR containing tracked notes and a `docs/verification.md`
   record with version rationale, previous release, full tested source SHA,
   environment, results, reuse justification, and limitations.
2. Select refreshed `origin/main` after preparation merges; account for every
   difference from the tested source. Verify clean state, required CI/reviews,
   and latest-channel safeguards.
3. Report the version and final source SHA, then create an annotated v0.10.0
   tag and push only that tag. The release workflow must pass on that exact
   source.
4. Replace generated notes with the prepared notes plus final source and
   comparison, preserving asset IDs and digests. Confirm stable and latest
   status.
5. Download all seven assets and verify exact filenames, hashes, archive
   members, notices, Unix permissions, binary build metadata and version, and a
   native Windows amd64 version and read-only smoke test. Record unavailable
   native targets.
6. Merge the publication-audit PR, preserve immutable release assets and tags,
   clean task-owned branches and files, and report publication and audit
   status.

## Dependencies and limits

01 preparation → 02 candidate/tag/publication → 03 downloaded-asset audit.
No open scope questions. macOS OpenSSH behaviour and a live interactive paste
or TUI session remain unverified and must be stated as limits.
