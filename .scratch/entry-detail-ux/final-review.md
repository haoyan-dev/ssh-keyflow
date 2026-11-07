# Entry detail redesign: final review

Reviewed on 2026-11-08 against refreshed base
`76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.
These are separate local review passes by the implementing agent.
The exact tested source commit and execution evidence are recorded in
[verification.md](verification.md).

## Standards compliance

- Domain vocabulary matches `GLOSSARY.md`: destination, registration,
  unregister and local key deletion remain distinct.
- Production changes stay in the TUI layer. State schema, CLI, authentication,
  locked backend checks, dependencies and operation execution are unchanged.
  The unregister-choice and deletion-blocker extractions preserve the original
  predicates and error messages; both hints and actions use those helpers.
- Overview composition explicitly uses detail context before the page changes.
  It cannot inherit category-list continuation fallbacks.
- Technical disclosure uses existing document origins, soft wrapping and
  scrolling. The visible-target and action page guards reject technical-page
  actions, retaining the existing confirmation boundary.
- Public `NewProgram` journeys exercise observable behavior. Their rejecting
  Runner asserts navigation performs no dependency probes or commands.
  Navigation tests compare saved state before and after.
- Go formatting and diff whitespace checks pass. Formatting-only changes to
  unrelated TUI files are excluded from the final Git diff.

No unresolved standards findings.

## Specification compliance

| Criteria | Evidence |
| --- | --- |
| AC01 | All four entry kinds have labelled identity, Saved status and Next actions; tests assert their order and exclude raw diagnostic fields. |
| AC02 | Pair-specific verified/failed/not-checked/unknown cases, public-only limits, empty states and alias bound to a different key are tested. |
| AC03 | Relationship rows appear once per registration, with fingerprint hints or destination login; full recursive dumps remain in technical disclosure only. |
| AC04 | Hints use existing eligibility and continuation sources. Missing related entities, blocked local deletion and review-only Tasks have explicit guidance. |
| AC05 | Full values remain accessible through `v`. Technical action rejection, overview scroll restoration, Help origin restoration and result→overview→technical navigation are covered. |
| AC06 | All four overview/technical kinds fit 80x24, 60x18 and 32x12. Existing Unicode, long-value vertical scrolling, search, selection, resize, refresh and small-window guard journeys remain covered. |
| AC07 | README/application contract updated; full tests, vet, native build and console inspection recorded against the source commit in verification. |

Finding P01: standalone configuration Tasks inherited a full file path in
their title, undermining the overview's readable identity. Resolved by using
“Requires review: configuration file” in the overview while retaining the
complete original title, path and recovery guidance in technical details.
A public-event regression test covers both views.

Finding P02: the technical-value assertion assumed an unwrapped path. Resolved
by normalizing rendered whitespace when checking complete saved values; the
existing 32-column scrolling test still verifies full-value reachability.

Finding P03: expanding the layout matrix to every entry kind exposed an
existing 33-column Registrations header at minimum width. Native 32x12
inspection also showed clipping. Resolved by shortening the compact category
hint to “(tab: switch)”; all four overview and technical pages are included
in the layout regression check. Native inspection also exposed the resumable
Task footer splitting “scroll” across lines at 32 columns. Its compact control
groups now use explicit line breaks, covered by the same public-event test.

No unresolved specification findings. Native/live-platform limits are stated
in the verification record and are not treated as evidence from fake Runners.
