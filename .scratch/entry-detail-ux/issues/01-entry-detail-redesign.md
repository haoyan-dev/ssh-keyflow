Status: ready-for-human
Blocked by: none

# 01: Redesign entry details around readable saved evidence

Specification: [spec.md](../spec.md)
Review: [review.md](../review.md)

## Scope

Replace the default diagnostic dump with structured overviews for all four
entry kinds. Retain complete technical evidence behind `v`, reuse existing
action eligibility and navigation, and update focused public-event tests and
shipped documentation. Work on `feat/entry-detail-ux`, based on refreshed
`origin/main` at `76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.

## Acceptance

- [x] AC01–AC03: readable identity, independently labelled saved facts,
  compact relationships and honest empty/public-only/configuration states.
- [x] AC04: contextual actions and blockers, exact saved continuation.
- [x] AC05: complete read-only technical document; correct origin restoration.
- [x] AC06: all supported sizes, long values, refresh and navigation.
- [x] AC07: focused/full checks, native build, rendered inspection,
  documentation, separate standards/spec reviews and verification record.

## Comments

- 2026-11-08: User authorized review -> to-spec -> to-ticket -> implement
  autonomously. Review and specification completed before implementation.
  No remote publication is included.
- 2026-11-08: Implemented and verified on source
  `73fb603a2760207845bac607a8d2dd7e8b86423e`. Full uncached Go suite,
  vet, native build and Windows console inspection pass. Separate local
  [standards/specification reviews](../final-review.md) have no unresolved
  findings. [Verification](../verification.md) records exact commands,
  hashes, native source versions and platform limits. Ready for human review;
  changes are committed locally on `feat/entry-detail-ux`.
