# Entry detail redesign

Status: ready-for-human
Review: [review.md](review.md)

## Objective and scope

Make Enter-opened TUI entries readable without knowledge of the saved schema.
Cover destinations, keys, registrations and tasks and entry links from results.
Preserve existing state, CLI, authentication, confirmation, recovery and
destructive-operation contracts. No backend feature, schema change, live SSH
probe, new dependency, remote push, release or merge is required.

No blocking open questions: interpret "entry detail page" as all four TUI
entry types. Use existing English UI vocabulary and the domain glossary.

## Acceptance criteria

- AC01 Identity: the overview starts with a labelled entity name and readable
  login/OS, key availability, destination/key pair or task context. Bold
  headings and spacing distinguish identity, saved status, actions and related
  entries. Default pages exclude raw IDs, paths and shared digests.
- AC02 Meaning: registration and destination-specific connection checks are
  independent saved facts, never a live-access claim. Explain public-only
  verification limits, unknown/not-checked outcomes and empty relationships.
  A saved configuration binding does not prove Include activation.
- AC03 Relationships: show each related registration once, with a readable
  related name and fingerprint hint (keys) or login (destinations). Avoid
  recursive full-detail dumps. Keep configuration for a key tied to each
  destination rather than inventing a global key verification status.
- AC04 Actions: overview and footer offer applicable existing shortcuts;
  blockers explain missing related entities, outstanding registrations, stale
  state and read-only targets. Local deletion belongs only to managed keys.
  Saved continuation derives from existing Tasks and exact saved IDs. Reuse
  existing eligibility logic where practical; backend checks still decide.
- AC05 Technical details: `v` opens a read-only document for the visible
  entry, with complete IDs, fingerprints, paths, provenance, raw saved stages,
  configuration evidence and task source IDs as applicable. Shared file
  evidence is labelled global, not proof of alias activation. Esc and Help
  restore the previous viewport; action keys cannot borrow a hidden target.
  F5 is available on the overview and reloads saved state only.
- AC06 Layout/navigation: usable at 80x24, 60x18 and 32x12, with long Unicode
  names, many relationships and long technical values accessible by vertical
  scrolling. Preserve selection, filters, resize, Home/End, refresh and
  result-entry navigation. Existing below-minimum mutation guard remains.
- AC07 Verification: public NewProgram journeys verify hierarchy, status
  semantics, contextual actions, technical/back/help, refresh and full-value
  accessibility. Assert no navigation writes or external commands. Run
  `go test -buildvcs=false ./...` and `go vet -buildvcs=false ./...`, build
  the native executable, inspect rendered pages, and record the exact tested
  commit, environment and native/other-platform limits. Update README and
  the application contract.

## Traceability

F01 -> AC01/AC05; F02 -> AC03; F03 -> AC02; F04 -> AC02/AC05;
F05 -> AC04/AC06. [Ticket 01](issues/01-entry-detail-redesign.md) implements
AC01–AC07 in order, with standards and specification reviews recorded
separately.

## Local color and panel follow-up

User approved the color/panel direction on 2026-11-08. Scope is presentation
of the existing four overview kinds and their resize behavior, on the same
work branch. No blocking open questions.

- VP01: Bold identity, cyan section headings and action hints. Green means
  successful saved registration/verification; yellow means unknown, unchecked
  or unfinished evidence; red means failed checks or blocked actions. Text
  labels always carry the meaning. Removed registrations are neutral.
- VP02: Saved status and Next actions use thin bordered panels with horizontal
  padding and terminal-default backgrounds at widths of 60 columns or more.
  Below 60, use plain headings. Related registrations remain compact rows.
  Supporting copy uses terminal-default text to respect terminal themes.
- VP03: Recompose panels on width changes, including returning from Help or
  technical disclosure after resizing. Preserve scroll offsets where possible,
  complete values and navigation safety. Long Unicode text remains reachable.
- VP04: Verify all four kinds at 80x24, 60x18 and 32x12, resize in both
  directions, color semantics and label-only legibility, no navigation writes
  or external commands. Run full tests/vet, inspect the native local build,
  and record separate standards/specification reviews and exact source SHA.

[Ticket 02](issues/02-detail-color-panels.md) implements VP01–VP04.
