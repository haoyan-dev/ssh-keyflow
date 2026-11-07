# TUI readability specification

Status: ready-for-agent
Date: 2026-11-08
Sources: [review](review.md), user authorization for autonomous local redesign.

## Acceptance

- AC01: Every page has a recognizable location, readable body and contextual keyboard footer. Category navigation appears on category/entity pages; draft and document pages identify their actual location. Current category, keyboard selection and field focus have textual markers even with color disabled.
- AC02: Lists and pickers show a readable title, target identity, useful status/availability and selected-position/total count. Fingerprints and IDs remain searchable; duplicate names remain distinguishable. Normal navigation uses no raw-ID entry and never changes its fixed target.
- AC03: Forms keep preceding fields visible while they fit, expose focused input without overflow-created blank rows, and scroll to focused fields/errors. Long values and guidance are accessible by vertical scrolling. Conditional fields, validation, caret, draft preservation and navigation contracts remain unchanged.
- AC04: Review presents operation, readable target, options, expected effects and native handoff explanation in distinct groups. Full references remain accessible below orientation. At 32×12 the focused Continue/Edit/Cancel/Help controls fit within three rows; the action and relevant identity appear before long fingerprints.
- AC05: Details/tasks use the same visual vocabulary; saved registration, destination connection and configuration evidence remain independent. Next actions stay target-bound. Recovery reasons and continuation are understandable before technical checkpoint data.
- AC06: Results separate overall outcome, next supported action, this-operation facts, saved evidence and diagnostics. Offer a direct route to session instructions. Unknown/manual/partial outcomes never become success through presentation or color.
- AC07: Help, history, discovery and technical documents have readable section headings and visible scroll position. History/discovery remain read-only. Stale errors expose recovery without hidden mutation paths.
- AC08: Layouts fit 80×24, 60×18 and 32×12, including Unicode, resize and long content. Below32×12 retain draft and block interactive mutations. Use terminal background/default foreground plus optional ANSI accents; NO_COLOR retains all meaning.
- AC09: Run existing TUI journeys and full Go checks, inspect representative native pages, record exact tested source and conditions. Review repository standards and this specification separately. Preserve all prior entry-detail and backend safety tests.

## Dependencies and delivery

Review and Astra recommendations → concrete [page design](design.md) → ticket01 common layout/lists → ticket02 forms/review → ticket03 results/documents → ticket04 verification/review. Work remains local; no remote push, merge, release, schema or SSH-capability changes.
