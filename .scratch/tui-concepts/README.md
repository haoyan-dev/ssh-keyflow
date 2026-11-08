# Three TUI design prototypes

Status: ready-for-human
Date: 2026-11-08
Decision: awaiting the user's choice; no production direction selected.
Source base: `e99e000` on `feat/entry-detail-ux`.
Prototype branch: `docs/tui-design-prototypes`.

The user rejected the previous destination overview because its stacked labels,
status qualifiers, and competing shortcuts remained difficult to understand.
This prototype asks which information structure is easiest to read across the
whole application. It deliberately revisits the previous recommendation to
remove panels. Prior review approval did not establish that the user found the
result understandable.

From the repository root:

```powershell
node .scratch/tui-concepts/serve.cjs
```

Then open `http://127.0.0.1:4318/prototype.html`. The server listens only on
loopback. Stop it with Ctrl+C. To use another port, set the process environment
variable `KEYFLOW_PROTOTYPE_PORT` before starting.
The localhost viewer is the verified launch path. Direct-file opening was not
verified because the test browser blocks the `file:` protocol.

## Three different structures

| Direction | Information order | Character | Tradeoff |
| --- | --- | --- | --- |
| A — Grouped panels | Identity → status beside next action → explanation | Bordered groups and a two-column overview at 80 columns | More border overhead; stacks and compacts at narrow sizes |
| B — Guided pages | Plain-language answer → one next step → supporting facts | One question at a time, especially in forms | More scrolling when comparing many facts |
| C — Compact tables | Aligned facts → explanation → commands | Dense reference view and tabular lists | Familiarity with short labels helps; narrow layouts need stacked rows |

Each direction has its own structure rather than a different color palette.
All use actual monospace text, terminal box drawing, one-cell line heights,
fixed terminal dimensions, and ordinary keyboard affordances. Browser controls
outside the terminal are evaluation tools and are not proposed TUI controls.

## How to compare

1. Start on Destination overview, which mirrors the user's problem with fictional
   values. Use the direction controls above the previews or Left/Right arrows
   outside the terminal to compare A, B, and C.
2. Enable Compare all three to see the same page in all directions.
3. Choose any page from the sidebar or the Page dropdown.
4. Try 80×24, 60×18, and 32×12. Content scrolls within the fixed terminal.
5. Disable Color. Status words, selection markers, headings, and structure remain.
6. Click a terminal to try keyboard navigation. Scroll buttons and the mouse
   wheel expose the entire document. Shift+Tab returns keyboard focus to Page.

The query string preserves page, direction, terminal size, comparison mode, and
color choice across reloads; there is no stored preference or product data.
The form values are fixed examples, not editable inputs. Handoff and result
screens are illustrative. No authentication, SSH, file mutation, import,
registration, or key deletion takes place.
The first selected row follows the main example walkthrough. Other rows
illustrate list density; pressing Enter on them shows a prototype notice rather
than silently opening a different target.

## Coverage

50 screens in each of 3 directions:

- Four category lists: destinations, keys, registrations, saved tasks.
- Seven overview scenarios: destination, key, registration, saved task,
  several keys with mixed outcomes, duplicate key names, configuration-only work.
- Four pickers: key, destination, registration, saved work.
- Ten form states: create, import, destination identity/access/options,
  register, unregister, local deletion, continuation, validation.
- Three decisions: review, discard, terminal handoff.
- Seven outcomes: complete, manual, partial, unknown, failed connection,
  cancelled, and an unfinished SSH Include.
- Nine reading pages: help, scoped history, empty scoped history, discovery,
  technical references, long technical values, registration instructions,
  SSH Include instructions, and session error output.
- Six edge states: empty destinations, empty tasks, search with no results,
  failed refresh, missing reference, undersized terminal.

These are page-family prototypes. They cover every current page family and key
outcome, rather than every backend option combination or native OpenSSH prompt.
The below-minimum example explains the real TUI's pause behavior within a
readable preview frame; the gallery does not emulate a terminal smaller than
32×12.

## Product facts that stay true in every direction

- Unknown registration means the result was not captured. The public key may
  already be installed; it is not relabelled as absent.
- A connection check is distinct from a registration record. Not checked means
  that evidence is missing, not that access failed.
- Saved file paths do not establish current file availability. No saved alias
  binding does not establish the absence of an unmanaged SSH alias.
- Saved records do not prove present server health.
- Unregister removes the selected remote public key; it keeps local key files.
- Local deletion removes local files; it does not unregister remote public keys.
- Review precedes terminal handoff. OpenSSH and the real application's terminal
  checks own secrets and final approval.
- Continuing work reviews the saved target and checkpoint. It is not an
  automatic retry.
- Read-only discovery does not import files. Failed refresh and missing
  references expose recovery without enabling unsafe actions.

## Decision handoff

Tell us A, B, or C, or specify a mix such as “A details, C lists, B forms.”
That choice will define the next production specification and tickets.
This prototype changes no product rendering or backend behavior.

The sample registration walkthrough preserves lab-box/work-laptop. Screens for
other operations stay standalone rather than opening a registration review.
Unsupported example shortcuts show a notice. Configuration-only work and failed
refresh never lead to a continuation draft. Left/Right changes direction only
outside the terminal so it cannot masquerade as a field-choice shortcut.

Related: [design brief](spec.md), [prototype ticket](issues/01-three-directions.md),
[verification](verification.md).
