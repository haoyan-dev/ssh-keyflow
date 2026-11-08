# Whole-TUI readability review

Date: 2026-11-08
Baseline: `41fe1ea` on `feat/entry-detail-ux`; origin/main refreshed before editing.
The branch retains the six local entry-detail changes because this task expands the same UX objective.

## Objective and scope

Make every TUI page understandable at a glance: where am I, what record or draft am I looking at, what does the evidence mean, and what can I do next?

The user requested Astra, full-page investigation, readable local designs, and autonomous review → spec → ticket → implementation. Scope includes category lists, search, selectors, entity/task overviews, forms, review, discard, results, technical documents, history, discovery, help, empty/error/stale and undersized states. Preserve CLI/backend semantics, exact fixed-ID targeting, locked checks, native authentication/approvals and saved-state schema.

## Evidence and open questions

The temporary public-event capture harness uses the same `tui.NewProgram` entry as the application. Baseline text renders for 16 page states at 80×24, 60×18 and 32×12 are in `.tmp/tui-readability/before/`. They use synthetic data only, a runner that rejects dependency probes/commands, and a read-only filesystem fixture. These are rendered text evidence, not screenshots of every terminal/theme.

Astra audits lists, pickers, details/tasks, results and read-only documents independently. The main agent inspects forms/review/discard/native handoff and the shared layout budget. No user input is required to preserve current product truth; optional aesthetic choices are resolved through restrained terminal-native design.

## Main-agent findings

1. **P1 — field layout wastes most of the screen.** `forms.go:buildFields` sizes input content to width−4, but prompt/cursor overhead exceeds the width−2 viewport. Wrapping produces blank rows. `formBody` guesses each field occupies three rows instead of measuring it. Captured Add starts with the OS value below the first viewport.
2. **P1 — moving focus removes context.** `formBody` starts rendering at `m.focus`. Earlier destination fields disappear even when there is room. Register shows repeated fixed targets as fields and gives no visual marker distinguishing editable focus from ordinary input prompts.
3. **P1 — narrow review shows identifiers instead of consequences.** `reviewForm` places a complete fingerprint, IDs, then duplicate IDs above effects. `reviewControls` hard-wraps into five rows at 32 columns. The first viewport cannot explain the action.
4. **P2 — navigation chrome describes categories during a draft.** The active destination tab remains visible during create/import/register/review, implying category navigation that is unavailable. Global integrity messages recur on every page and compete with draft guidance.
5. **P2 — list rows truncate their meaning.** Stock list titles/padding/pagination spend rows without giving a readable selected-position count. Picker rows named “Laptop” lose availability and the fixed-ID disambiguator after a long fingerprint.
6. **P2 — reading pages lack visible position.** Scrollable content does not say whether more content exists. Technical/history/discovery/help are structurally unrelated to forms/details.

Independent Astra findings and the final disposition will be recorded alongside the concrete design.
