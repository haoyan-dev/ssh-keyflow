# Prototype verification

Date: 2026-11-08

Tested prototype source: `2bc5b57a8d58c0a0ac0e22576ddf67b44d059fd8`

Branch: `docs/tui-design-prototypes`
Production base: `e99e000`

## Environment and scope

Windows, Node v22.14.0, Playwright-controlled Chromium reporting Chrome
156.0.0.0. The dependency-free Node server served the gallery at
`http://127.0.0.1:4318/prototype.html`, bound only to loopback.

This verifies a throwaway gallery of terminal text, fictional records, static
forms, and simulated outcomes. It does not verify native terminal rendering,
OpenSSH prompts, or production workflows. No production source changed; Go
tests and native builds were not repeated for this prototype-only change.

Only localhost launch was verified. The test browser rejected the `file:`
protocol; direct-file opening is unverified. README makes localhost the primary
launch path.

## Checks

- JavaScript syntax passed for `prototype.js` and `serve.cjs`.
- All 450 page/direction/size configurations rendered successfully: 50 screens,
  three directions, and 80×24, 60×18, and 32×12 sizes.
- Top and bottom frames were checked for each configuration: 900 frames passed
  exact row and column lengths with no over-width content lines.
- Before the final correction, 29 browser interaction checks passed against
  `d49609fb31deaa26f22ff0304fc32910a569aea6`. These covered target preservation,
  blocked actions, the sample registration walkthrough, output origin,
  monochrome, refresh disclosure, and narrow-page next actions.
- After correction, all 900 frame checks and 11 affected browser checks passed:
  four result transitions and configuration-only evidence stay explicitly
  standalone; Shift+Tab moves actual browser focus to Page; that exit is
  disclosed; the switcher stays in normal flow above previews; B/C direction
  controls scroll to the selected comparison.
- C reaches full visibility at the document end; top alignment is limited by
  the browser's maximum scroll position. The initial stricter top-alignment
  assertion was corrected to test visibility.
- Mobile at 390×844 has no page-level horizontal overflow and keeps the
  direction switcher above all previews.
- HTTP assets returned 200; a deliberate out-of-allowlist request returned 404.
  No external requests are needed.
- Local Markdown links and `git diff --check` passed. A production-source diff
  against `e99e000` was empty for `internal`, `cmd`, root README, and Go modules.

Local execution evidence is in ignored `.tmp/tui-concepts/`: `final-checks.json`
contains the earlier suite; `correction-checks.json` contains the final 900
frames and 11 affected checks.

## Visual captures

All four captures were regenerated after the switcher fix and opened for
visual inspection. No direction selector obscures captions or terminal text.

| Capture | Browser viewport | Terminal / example |
| --- | --- | --- |
| `desktop.png` | 1440×1100 | 80×24, destination, A/B/C, color |
| `multiple.png` | 1440×1100 | 80×24, mixed key outcomes, A/B/C, color |
| `forms.png` | 1440×1100 | 60×18, registration form, A/B/C, color |
| `mobile.png` | 390×844 | 32×12, destination, A/B/C, monochrome |

Captures are local evidence under `.tmp/tui-concepts/`; the live gallery is the
review deliverable.

## Review

Astra investigated the page inventory and reviewed the prototypes. The initial
review prompted explicit simulation disclosure, consistent target handling,
read-only recovery examples, and expanded coverage.

A separate fresh Astra finish review on `d49609f` passed standards/safety and
coverage/distinctness, and requested three bounded corrections: the overlapping
direction switcher, contradictory fixture links, and trapped keyboard focus.
All three were corrected in the tested source above. Astra's resolution review
returned **SHIP at prototype scope** for `2bc5b57`: all four recaptures were
inspected, navigation/focus handlers were independently checked, and no
resolution requirements remained. Browser-specific confirmation relies on the
recorded browser checks; the reviewer did not rerun browser automation.

The design detector ran once and flagged the gallery's cream background. The
wrapper was changed to neutral gray (`#f4f5f5`) and buttons to white before final
captures. The detector was not rerun; visual review inspected the correction.

No winning direction has been selected. User choice remains the next gate for
the production specification and tickets.
