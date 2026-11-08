# Whole-TUI concept selection

Status: ready-for-human
Date: 2026-11-08
Decision: pending user choice

## Objective and scope

Make three materially different readable TUI directions concrete enough for
the user to choose. The user's supplied destination overview is the main
comparison case. Cover the complete existing page-family inventory with
placeholder data. Deliver a local browser viewer of terminal text layouts.
Production rendering and backend behavior are outside this iteration.

## Acceptance

- [x] Three structural directions: grouped panels, guided pages, compact tables.
- [x] The same selected page can be compared in all three directions.
- [x] Lists, overviews, selectors, forms, decisions, outcomes, documents and
  important edge states have placeholders in each direction.
- [x] Viewer supports 80×24, 60×18 and 32×12, scroll, and color on/off.
- [x] Registration evidence, connection evidence, remote/local effects and
  unknown outcomes remain distinct in the proposed wording.
- [x] Static example fields and simulated operation screens are clearly disclosed.
- [x] Prototype runs locally without dependencies and without real operations.
- [ ] User chooses a direction or a combination for production implementation.

## Open questions

Which structure is easiest for the user to understand? Should lists, overviews
and forms share one direction or combine strengths from different directions?
These questions are deliberately left to the user's review of the
[prototype](prototype.html), not inferred from a reviewer preference.

## Dependencies

User rejection and prior audit → three prototype directions → user selection →
revised production specification and tickets → implementation.

No durable visual system, production code, or domain decision is selected by
creating these concepts. See [README](README.md) for tradeoffs and coverage.
