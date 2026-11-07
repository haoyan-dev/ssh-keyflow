# Ticket02 navigation handoff

Status: ready-for-human
Date: 2026-11-07
Branch: feat/02-navigation-details
Integration base: 262c0bc01116dd33848f038c967c15fd8daa9802

## Completion and scope

Navigation is ready for ticket03. This handoff, the checked ticket02, and its six TUI source/test files belong to the same implementation commit; the completion report supplies that commit's full SHA. The original clean worktree base was bb20d99b997bed2b42e6afbb1e2649f79d560a79. Integration documentation through262c0bc was merged before final verification. No remote refresh, push, main merge, or release was performed under this local coordination task.

Destinations, Keys, Registrations, and Tasks have separate list/query state. Search is case-insensitive substring matching of displayed identity plus fixed IDs and related fingerprints. Enter accepts the search; Esc ends editing and retains the query; x clears it. Tab/Shift+Tab changes category from lists/entities. Detail/back/help preserves the category cursor and pagination. F5 reloads Snapshot only, retains selected fixed IDs across reorder, and announces a nearest-row fallback when a target disappears. Repeated failed refreshes preserve one origin; stale views cannot launch mutations until refresh succeeds.

Details join destination alias/login, managed key identity and full fingerprint/paths, registration status, per-destination recorded connection evidence, configuration binding/file evidence, and history. Key-wide verification never becomes proof for a destination. Public-only verification is unavailable. A saved binding does not prove main-config Include activation. Invalid saved relationships fail the read-only load boundary instead of becoming invented joins or empty data.

History (h) and discovery (g) are read-only documents. Deleted-key tombstones and completed local creation/deletion records are inspectable in Keys history, outside current Keys. Unmanaged discovery never becomes a destructive target. Home/End and normal viewport scrolling make full values accessible without horizontal-only access.

Tasks intentionally has no projection or executable rows yet. Fresh state says No recorded follow-up / Remote access is not verified / Tasks projection pending. Existing operation/config evidence says Saved work may need attention. Ticket05 must replace this interim content; no recovery completion is claimed.

## Stable extension contracts

- navigation.go owns categoryEntries, refreshList, entityDetails, historyDetails, helpText, refreshSnapshot, visibleTarget, openDocument, and back. tui.go owns event precedence, sizing, and category shell rendering. actions.go owns contextual action resolution and the minimal related picker. forms.go has only an adapter to consume the chosen IDs; its existing form design remains for03/04.
- visibleTarget returns the selected current list entry or the explicit details target. It returns no entity from result/history/discovery/help/picker/error pages. Never infer an action target from the underlying list on those pages. A future result/task link must carry its own explicit target.
- entry identity is kind + fixed id. Display labels, aliases, and names are presentation. readOnly blocks mutation for historical references. Lists store cursor and pagination per category; queries are independent. refreshList returns whether a previously selected ID disappeared.
- openDocument(page, target, text) pushes returnContext{page, target, viewport}; back restores the exact document origin. Category lists retain their own state. Help uses this path and Ctrl+C closes help before exit. This small stack deliberately does not define retained draft/step state;03/04 must add that state.
- beginAction resolves only the visible target. Register from a destination chooses a key, from a key chooses a destination, and from a registration uses its recorded pair. Unregister resolves a current eligible registration. Delete accepts only a managed key and explains recorded registration/config blockers. Backend locked checks remain authoritative.
- formTarget is an app.Request seed containing fixed Host/Key, Registration, or Resume IDs and Operation. openPicker/chooseTarget fill it from labelled searchable choices. Legacy target-ID inputs are fixed, so editing cannot make review labels disagree with the request. Escape returns to the origin to choose again. No normal contextual action opens an unseeded raw-ID form.
- Related pickers search names/login/fingerprints/fixed IDs. Empty eligible sets explain create/import/add or registration prerequisites. Full draft picker restoration, conditional fields, validation focus, and re-resolution before handoff remain03/04 work. Replace the legacy form adapter; do not resurrect list.SelectedItem as a form target source.
- Completion uses a distinct result page with no implicit target. Existing native execution and result wording are otherwise unchanged. Typed reports, authoritative outcome classification, diagnostic retention behavior, and native handoff isolation belong to04/05.
- Minimum supported size is32x12. Below it, only resize guidance and exit are available; input state survives resize and mutation keys are ignored. Event fixtures cover80x24,60x18,32x12.03/04 must retain this guard and compact control visibility when changing forms.

## Verification

Public seam: tui.NewProgram with key/window events and View output, as already agreed in spec.md. Fixtures enter through filesystem state or public app.Run. A fake external Runner rejects dependency/process access; navigation journeys assert it is never used. Refresh fixtures change external saved state immediately before a public F5 event through tea.WithFilter. No model internals are inspected by tests.

Vertical red/green slices cover search/category restoration; readable details and help origin; visible-target actions/related choices; fixed-ID reorder/removal; compact layouts and small-size mutation guard; tombstone history and unchanged read-only storage; related fingerprint searches; selected-ID edit rejection; and repeated refresh-failure recovery. Long Unicode paths and full fingerprints are recovered from vertically scrolled document bodies, excluding repeated screen chrome. Missing saved joins fail without changing the document.

Windows amd64, Go1.26 toolchain, November7,2026. Final source checks:

- go test ./internal/tui -count=1: passed after the final related-picker fix (2.152s).
- powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1': passed on the reviewed implementation before the final related-picker search-only fix (app36.115s; ssh22.787s; TUI2.141s). Full suite is rerun at the committed source below in the completion report.
- go vet ./...: passed after the final fix.
- go build -buildvcs=false -o <temporary ticket02 executable> ./cmd/ssh-keyflow: passed after the final fix.
- git diff --check: passed. Integration merge reports Already up to date at262c0bc.

ExecutionPolicy Bypass applies only to the spawned test process for existing native PowerShell fixtures; no persistent policy change. No real keys, SSH servers, or user SSH state were used. Event layout checks do not prove native terminal focus/handoff or actual server behavior. Integrated native terminal and cross-platform acceptance, shipped README/application-contract updates, and release packaging remain06.

## Standards review

Reviewed the final TUI call paths separately from acceptance review. Existing Bubbles list/viewport/textinput components are reused; no dependency, persisted schema, backend guard, SSH operation, or generic router was added. Action and form entry no longer consult hidden rows. Selected IDs cannot drift through editable legacy fields. Error recovery retains stale-state mutation guards and only retries Snapshot. Only task-owned files/artifacts are changed; execution.go is unchanged. No unresolved standards finding remains in02's scope.

## Specification review

AC01 and02 and the non-draft portions ofAC05/11 are supported by the public event tests and source review. Result-page action isolation is verified by the explicit result page, visibleTarget and beginAction gates; history/discovery isolation also has event coverage. Invalid links are rejected by the existing strict Snapshot load rather than fabricated into a usable entity. Advisory eligibility never bypasses runtime checks. The Tasks category is a clearly labelled interim shell. Full AC03/04 selectors/staged forms are03, draft-specificAC05 and native outcome work are04, full Tasks/resultsAC08/09 are05, and integrated native acceptance/docs are06. These dependencies do not block the completed02 navigation contract.
