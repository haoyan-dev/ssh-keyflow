# TUI color, spacing, and panes

Objective: make SSH KeyFlow's terminal interface easier to scan and operate.
Scope: presentation, viewport sizing, form focus/caret placement, and list paging.

Acceptance criteria:

- Use the terminal palette for active navigation, focused fields, information,
  warnings, and validation errors. Preserve text/shape cues and honor NO_COLOR.
- Separate list entries with a blank row when space permits. Keep selection
  visible when navigating and paging.
- Use labeled panes for navigation, records/documents, search, form input,
  messages, and menus. Keep form actions in the menu while inputs scroll.
- Preserve the documented minimum of 32 columns by 12 rows, including long
  CJK labels, filtered lists, messages, and validation errors.
- Preserve existing keyboard actions, draft/back navigation, raw clipboard
  content, terminal-control sanitization, and local/remote operation semantics.

Dependencies: shared pane geometry precedes cursor, wrapping, and paging
changes. Verification follows final rendering changes.

Design decision: stack panes with shared borders to conserve terminal rows.
Use the terminal background and ANSI palette rather than forcing a theme.
At short heights, hide idle messages and reduce visible list density; show
messages whenever there is feedback. No backend, storage, CLI, or release work.

Open questions: none needed to implement the requested refinement.
