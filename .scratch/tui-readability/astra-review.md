# Astra TUI investigation

Date: 2026-11-08
Reviewer: Astra (`auto/gpt-6-astra`, high effort), read-only delegated audit.
Baseline: `41fe1ea2359f0a7c23ab07209fc2df5a4b3b4d68`; captures in `.tmp/tui-readability/before/`.

Scope: lists, related selectors, entity/task overviews, results, Help, history,
discovery, shared chrome. Forms and review were investigated by the implementing
agent. Results findings used source inspection; baseline result captures were
not available. These are baseline findings, not final implementation approval.

## Findings and adopted decisions

1. **P0 — Detail panels consume the first viewport.** At80×24 the first
   relationship key appears around row21; at60×18 actions fall below the fold;
   at32×12 connection evidence is hidden. Replace mandatory panels with compact
   sections. Put pair-specific registration and last connection evidence before
   aggregate counts, configuration and long references. Color supplements labels.
2. **P0 — Tasks hide their actual reasons.** Show why attention is needed
   immediately, then continuation availability and target. Retain complete
   reasons/source IDs in Technical details. Unsupported recovery stays read-only.
3. **P0 — Selectors lose their purpose on small terminals.** Always identify the
   operation, the kind being chosen and the fixed counterpart. Lead key summaries
   with availability; keep full fingerprints and IDs searchable.
4. **P0 — Results delay instructions behind technical records.** Order outcome,
   persistence/uncertainty warning, next action, current-operation facts, prior
   evidence, technical references. Provide direct session instructions access.
   Preserve typed outcome classification; never infer success from diagnostics.
5. **P1 — Keys look identical.** Distinguish duplicate names with compact
   references; put public-only availability before fingerprint hints.
6. **P1 — Narrow registrations truncate connection failure.** Give registration
   and connection separate labelled lines where necessary.
7. **P1 — Help buries relevant actions.** Lead with current-page actions, followed
   by navigation/search and deeper explanations. Do not advertise local deletion
   as a destination operation.
8. **P1 — Discovery repeats “requires review” and foregrounds paths.** Separate
   saved-state review from discovery reports with short headings. Preserve
   original freeform diagnostics rather than inventing structured conclusions.
9. **P2 — Empty history lacks scope.** Name the actual included scope and say
   “No recorded events in this view.” Preserve inclusion/order; a category is
   not necessarily the scope of all included events.
10. **P2 — Documents inherit misleading category chrome.** Use document location,
    scroll position and contextual controls. Retain textual `>`, brackets,
    status labels and numeric selection position without color.

## Verification expectations

Inspect80×24,60×18,32×12 and NO_COLOR. Verify task reason, picker purpose/fixed
target, independent registration/connection labels, result control position,
empty history scope, complete reachable technical values, resize/origin
retention and absence of unavailable advertised actions. A saved private-key
path is not proof that the file is currently usable. Preserve fixed IDs,
eligibility, unregister precedence, stale restrictions and native approvals.
