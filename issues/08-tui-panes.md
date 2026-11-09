Status: ready-for-human

# 08: TUI color, spacing, and panes

Specification: [.scratch/tui-panes/spec.md](../.scratch/tui-panes/spec.md)

Blocked by: none

Add color, space between entries, and separate panes for messages, inputs,
content, and menus. Preserve terminal size support and keyboard workflows.

## Comments

- 2026-11-10: Started on `feat/08-tui-panes` from refreshed `origin/main`
  `58d15de1bed2a2a23654b5b02accbcdbc7e2deed`. Working tree was clean.
- Shared pane borders conserve rows at the documented 32x12 minimum.
  Form buttons remain in the menu while inputs scroll; colors use the
  terminal palette with text cues and NO_COLOR support.
- 2026-11-10: Implemented all three requested improvements. Full Go tests,
  final affected TUI tests, vet, formatting, whitespace checks, and a Windows
  development build/version/help smoke check passed. Fixture renders were
  inspected at 80x24 and 32x12; tests cover sizes through 120x40, paging,
  long/CJK input, persistent form actions, stale messages, color opt-out,
  and terminal-control sanitization.
- Separate standards and specification reviews have no unresolved findings.
  See the TUI panes entry in `docs/verification.md` for commit evidence
  and limits. Changes are local; no push, merge, or release.
