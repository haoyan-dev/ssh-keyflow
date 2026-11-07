# Readable terminal pages

Date: 2026-11-08 · Mode: Operate, with Read documents.
Implementation authority: this design plus [spec.md](spec.md). Astra's independent findings are incorporated in [astra-review.md](astra-review.md).

## Direction

Use a quiet terminal workspace with strong textual hierarchy. The terminal owns the background and normal foreground. Cyan marks focus/actions/headings; green/yellow/red supplement explicit evidence labels. No fixed RGB background, decoration, icons or color-only distinctions. Keep `>` for selection, brackets for current category/control, and labelled sections for reading.

One page has three stable regions: two-line location, bounded content, contextual controls. List/entity pages retain category navigation. A draft, picker or document identifies its actual location instead of implying Tab can switch categories. Footer controls name what Enter does here. Reading pages expose a position indicator so users know when more text exists.

Use compact section headings throughout. Astra found that the earlier two-panel detail layout delayed the evidence and actions at every target size. Remove obligatory borders and use color for emphasis, explicit labels for meaning, and whitespace only when it improves scanning.

## Lists and search

One compact category heading includes selected/total count. Each row has a primary name and one descriptive line. Selection has `>` plus emphasis. A key row leads with availability and a fingerprint hint, and retains a compact disambiguating ID. A registration row separates destination/key identity from saved registration and connection labels. A task row leads with the human reason, not source IDs; source IDs remain searchable/inspectable.

```text
SSH KeyFlow
[Destinations] | Keys | Registrations | Tasks
Managed destinations  1/9

> build-1
  alice@server-1.example:22 · linux

  build-2
  alice@server-2.example:22 · linux

...

a add · r register · Enter details · / search
Tab category · ? help · q quit
```

At32 columns, retain current category with `tab: switch`, one/two rows of selected records, and concise primary shortcuts. Search has an explicit Search prompt/query, result count, and Clear search. Empty pages teach the relevant next step; empty Tasks never claims current SSH health.

## Forms

Show all applicable fields as one scrollable document, retaining preceding fields until focus requires scrolling. Inactive fields are compact label/value rows; the active input is marked `>` with its label above. Controls remain keyboard-focusable in their existing sequence. Put errors immediately with the relevant field. Never include prompt/cursor overhead inside the input's content width.

```text
SSH KeyFlow
Prepare · Add destination
Destination · step 1/3

Destination alias: build-box
Host / IP: build.example.test
Login user: alice
Port: 22
> OS: linux/macos/windows
  linux
  ←/→ choose

  [Continue]   [Help]   [Cancel]

tab move · enter next · ctrl+s review · esc back
PgUp/PgDn scroll · Help control
```

At60×18 keep the same hierarchy with wrapped lines. At32×12 reserve two footer lines and scroll the body to the focused field, value and any guidance/error. PgUp/PgDn exposes full values and earlier context. Resizing, Help, review Edit and Keep editing preserve draft/caret; the viewport changes only as necessary to keep the focused field visible.

## Review and discard

Review starts with the human operation and target; then options, expected effects and native handoff. Full IDs/fingerprints/paths remain in References below the decision summary, except a compact selected-key reference where useful to distinguish names. No duplicate raw-ID field list above effects.

```text
SSH KeyFlow
Review · Register public key
Review register
No changes yet.

Target
build-1 · alice@server-1.example:22 [host-1]
Laptop [key-1] · public only

Options
Auth: manual
SSH configuration: no

Expected changes
Prepare manual registration instructions.
Remote registration remains unconfirmed.

Continue in the terminal for exact checks and approvals.
...
[Continue to terminal] · Edit · Cancel · Help
Tab choose · Enter select · e/Esc edit · ? Help
```

At32×12 use `[Continue] Edit Cancel Help` plus two short key rows. The first body rows identify the action and target before a fingerprint. Discard is an interruption page with the affected draft, no-changes explanation, and explicit `> Keep editing` default / Discard draft alternative. Esc returns safely.

## Details and Tasks

Entity overviews use identity → pair-specific Saved status → Next actions → configuration → History. Related key/destination identity immediately precedes its independent registration and connection labels. Multiple relationships remain separately readable; no aggregate replaces pair evidence. Technical details contains full references.

Task overview adds **Why this needs attention** with the saved human reasons ahead of technical recovery data. Show whether continuation exists; `s Continue` opens a retained-ID draft and never executes directly. Missing references and unsupported config-only recovery remain read-only.

## Results

Order: outcome → readable target → next supported action → this-operation facts → checkpoint details → saved evidence → diagnostics/instructions. The first viewport answers what happened and what to do next. Result controls remain target-bound. Add a labelled Instructions / diagnostics control and `o` shortcut opening a read-only session document; Esc restores the result. This is a navigation affordance, with no new persistence or execution.

Retain typed outcome classification unchanged. Green never follows from generic success output. Manual work, saved alias versus Include, failed connection, unknown effects, no durable task and stale state remain explicit. Full diagnostic text is available without scrolling past every saved technical record.

## Read-only pages and edge states

Help starts with the current page's relevant actions, then Move/find and Read/refresh before deeper notes. History names its actual scope and explicitly identifies an empty view. Discovery separates Review from the original discovery report without parsing diagnostic prose into invented facts. Technical details is explicitly labelled and preserves all values. All have scroll feedback and Esc back.

Stale/reload errors show the failed read, retained evidence and F5 Refresh / Quit; mutations stay disabled. Long notices are summarized in chrome and fully available in Help. Undersized terminals retain inputs, show32×12 minimum and Exit, and suppress mutation controls until restored.

## Page coverage

| Family | Variants |
| --- | --- |
| Lists | Four categories, populated/empty, filtered/no matches, retained cursor |
| Selectors | Key/destination/registration/recovery, duplicate names, cancelled search |
| Overviews | Destination/key/registration/task, blocked/read-only/missing reference |
| Drafts | Create/import/add3 steps/register/unregister/delete/resume, conditional/error |
| Decisions | Review/edit/help/discard default and alternative |
| Execution | Existing native OpenSSH and exact approvals; restored terminal |
| Results | Complete/manual/partial/unknown/cancelled/stale, task/entity/instructions links |
| Reading | Technical/history/discovery/help/instructions, top/scrolled/end/resize |
| Global | NO_COLOR, Unicode/long values, refresh,32×12 and undersized |
