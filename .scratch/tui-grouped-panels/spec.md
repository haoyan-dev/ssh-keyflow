# Readable TUI — A: grouped panels

Status: ready-for-human
Date: 2026-11-08
Decision: user selected A from `.scratch/tui-concepts`.
Base: 6655720; origin/main refreshed to 76d392826f757e8aa4cfdb72d2d7b5216660f0fc.

## Objective

Replace the difficult-to-scan flat TUI with the selected grouped-panel design
across the whole application. Use the existing saved data and guarded workflows.
The first viewport should explain the item, its recorded state or reason for
attention, and a useful next action.

## Scope and dependencies

Review and selected prototype → shared panel geometry → page families →
event-driven verification → separate standards and specification reviews.
Local implementation, native development build and test instructions are in
scope. Publishing, releases and backend/domain changes are outside this work.

## Design

- Use terminal background, thin neutral borders and cyan titled groups.
  Green, yellow and red reinforce explicit state words. Meaning and keyboard
  focus remain visible without color.
- Detail pages: identity, saved status, next action, other actions, configuration
  and further reading. At 70 columns and wider, status and next action share a
  row. Narrow screens combine identity, concise evidence and the next action
  at the top, then stack the complete supporting groups below.
- Give each related destination/key pair its own evidence block. Never infer
  an aggregate connection result. Separate saved registration, past connection
  checks, configuration binding and live usability.
- Lists and pickers: one framed selection area, a visible position/count and
  an unambiguous selected row. Preserve duplicate-name references. Pagination
  must match the actual number of visible rows, including border overhead.
- Forms: framed fields, explicit focus, nearby guidance and inline errors.
  Keep the active field visible after focus moves or resize; preserve the full
  value, caret and draft through overlays.
- Review/discard: framed target/effects/approval or decision groups. The native
  terminal retains exact checks, secrets and approvals.
- Results: outcome and next step before current-operation facts, checkpoints,
  saved evidence and diagnostics. Typed reports remain authoritative.
- Help, history, discovery, technical documents, session output and edge states
  use titled readable groups. Long IDs, fingerprints, paths and reasons remain
  fully scrollable. Reflow from source content on resize.

## Acceptance

- All page families use A's hierarchy, with no border or content overflow at
  80×24, 60×18 or the supported minimum 32×12.
- First detail viewport contains identity, relevant concise evidence and the
  next action; task reason is near the top. Long names remain fully available.
- Search, selection, PageUp/PageDown, fixed target IDs, stale/read-only guards,
  typed outcomes, edit/cancel and native handoff continue to work.
- Panel geometry uses terminal cell widths for ANSI and CJK; color is optional.
- No new dependency, SSH probe or saved-data mutation is introduced by rendering.
- Go tests and vet pass; production captures receive visual review; local
  build location and execution instructions are documented.

## Open questions

None blocking. A is selected; compact layout details follow its structural
rules while accommodating real fields and supported terminal dimensions.

## Completion

Implemented in `b27b6ac` with the final reference correction in `fbb2c37`.
All three tickets are complete for local human acceptance. Separate Astra
standards and specification/visual reviews passed with no unresolved blockers.
See [verification](verification.md) for the exact tested source, results and
limits, and [local test instructions](README.md).
