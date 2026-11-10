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
- 2026-11-10: The user requested "tag and push". Preparing v0.8.0 as a
  backward-compatible feature release, including the guide platform changes
  already integrated since v0.7.0. Fresh full Go tests and vet passed on
  `a39fce80e0fe30d8621848716ec81cec2c6345ac`; integration and publication
  will follow the repository's PR and release checks.
- 2026-11-10: PR #18 merged after all three OS checks and six-target packaging
  passed. Candidate `b0b353e875a4d3d4396012602cb5dd67e3e7a646` exactly matches
  the verified PR tree. Annotated v0.8.0 was created and pushed on that commit;
  all 85 local Python release tests also passed. Publication audit follows.
- 2026-11-10: Published
  [v0.8.0](https://github.com/haoyan-dev/ssh-keyflow/releases/tag/v0.8.0)
  as latest stable.
  Release run 42836604663 succeeded. All seven downloaded assets, manifest,
  archive contents, dependency notices, embedded versions, and Windows amd64
  smoke checks passed. Production PowerShell install/update/uninstall passed
  in a disposable home, preserving SSH data and profile bytes. Publication
  and testing limits are recorded in `docs/verification.md`; the published
  tag and assets are unchanged.
