Status: ready-for-agent
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

- [ ] AC01–AC03: readable identity, independently labelled saved facts,
  compact relationships and honest empty/public-only/configuration states.
- [ ] AC04: contextual actions and blockers, exact saved continuation.
- [ ] AC05: complete read-only technical document; correct origin restoration.
- [ ] AC06: all supported sizes, long values, refresh and navigation.
- [ ] AC07: focused/full checks, native build, rendered inspection,
  documentation, separate standards/spec reviews and verification record.

## Comments

- 2026-11-08: User authorized review -> to-spec -> to-ticket -> implement
  autonomously. Review and specification completed before implementation.
  No remote publication is included.
