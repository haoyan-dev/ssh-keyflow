# Release v0.9.2

Status: ready-for-agent
Previous release: v0.9.1
Tested source: `aaa8defaa11fdada24b9c0e37175b8c3681cc679`

## Objective and scope

Publish the next supported stable SSH KeyFlow release from integrated `main`,
following all preparation, candidate verification, publication, and audit gates.
The user's explicit release request authorizes the preparation/audit PRs,
their integration after checks/reviews, the annotated tag, and automatic release.

Select v0.9.2: all changes since v0.9.1 are compatible documentation and
documentation-validation improvements. No runtime, dependency, state, installer,
packaging, or publication-script input changed. No feature increment or 1.0.0
graduation is warranted. Local and remote tags/releases were refreshed and v0.9.2
was unused when selected; recheck immediately before tagging.

## Acceptance criteria

1. Merge a preparation PR containing tracked notes and a `docs/verification.md`
   record with version rationale, previous release, full tested source SHA,
   environment, results, reuse justification, and limitations.
2. Select refreshed `origin/main` after preparation merges; account for every
   difference from the tested source. Verify clean state, required CI/reviews,
   and latest-channel safeguards.
3. Report the version and final source SHA, then create an annotated v0.9.2 tag
   and push only that tag. Release workflow must pass on that exact source.
4. Replace generated notes with the prepared notes plus final source/comparison,
   preserving asset IDs/digests. Confirm stable and latest status.
5. Download all seven assets and verify exact filenames, hashes, archive members,
   notices, Unix permissions, binary build metadata/version, and native Windows
   amd64 version/read-only smoke. Check source-hosted installer behavior against
   the specific main commit in isolated paths; record unavailable native targets.
6. Merge the publication-audit PR, preserve immutable release assets/tags, clean
   task-owned branches/terminals/files, and report publication/audit status.

## Dependencies and limits

01 preparation → 02 candidate/tag/publication → 03 downloaded-asset audit.
No open scope questions. Translations and locale validation remain pending
documentation tickets 04–05; this release must not advertise completed
Chinese/English/Japanese translations.
