# Ticket03 selectors and staged forms handoff

Status: ready-for-human
Date: 2026-11-07
Branch: feat/03-selectors-staged-forms

Clean starting base: `dce6c7c6b9c8d71c92e7ed71fdb77b2445cce5c3`. Product implementation: `d032cd34f7a15f8b391dd32af571e9534f4db3f1`. Required integration merge includes `93c27798ce9aa077ccc704bb19707938892fe281`. Exact final verified product source: `33654a7240e8da7c0e6ddb0f60dc7e974d584a43`. The subsequent completion commit adds only this handoff and ticket evidence; its full SHA is supplied in the completion response.

## Completion

Ticket03 is complete. Add Destination now uses Destination -> Key -> Access method/options. OS is explicit; Windows alone exposes account type. Existing managed key versus new Ed25519 name is exclusive. Authentication identity appears only for key authentication. Switching controlling choices clears hidden account/identity/key/new-name values. Manual and configure=no defaults remain. Create/import remain separate focused journeys; inline onboarding import is deferred as designed. No secret field, dependency, framework, persisted schema or SSH capability was added.

Current managed selectors show human labels with fixed IDs as secondary identifiers and support name/login/ID/fingerprint search. Normal target selection never asks the user to recall an ID. Fixed form fields remain focusable and display destination alias/login, key name or registration/saved-operation label even though the user cannot edit the ID. Review includes resolved human labels and full fingerprints. Public-only limits, bootstrap purpose, config eligibility and unregister alternate-credential/local-file guidance are available in forms and form Help.

## Draft and event contract for04

- `model.draft map[string]string` retains applicable editable values. `step` is0 Destination,1 Key,2 Access for add; other operations use one step. `fields` contains only the current step's applicable controls, with `focus` pointing into that slice. `formError`/`errorField` retain the named error. Textinput cursor models survive ordinary input edits; field rebuilds occur on controlling changes, step changes, selection or validation.
- `formTarget app.Request` is the operation seed with fixed Host/Key/Registration/Resume IDs. Display labels never resolve an execution target. `request app.Request` is the validated prepared intent set by `reviewForm`; there is no exported PreparedRequest or other test-only production API.04 can consume this same request at native handoff and collect01's reports without changing the form seam.
- `saveFields` saves applicable inputs; `clearHidden` clears stale conditional values; `buildFields` constructs visible controls from the retained draft. `validateInputs` checks destination fields through the backend Host validator with valid surrounding fields, checks name/auth through Request.Validate, and adds the explicit TUI port/OS/account/choice requirements. `draftRequest` copies only applicable inputs; final Request.Validate remains mandatory.
- Continue validates the current step. Ctrl+S validates all steps and returns to the first invalid field, preserving valid data. Tab/Shift+Tab traverse fields, Continue and Help. q/?/action letters remain text in editable fields. The labelled Help control opens a scrollable document through the existing origin stack.
- Esc from add steps moves one step back; e/Esc from Review restores the retained form and current field. First-step Esc/Ctrl+C still use the pre04 immediate cancel behavior.04 owns dirty detection, Keep editing/Discard, explicit review controls and the complete cancel contract. Do not mistake03's draft retention for completed04 cancellation behavior.
- `content` is the existing viewport. Forms render the focused field first, include full wrapped long values and errors, and support PgUp/PgDn vertical access. Focus/input/step changes reset the form offset. Help/picker origins retain viewport and leave draft/fields/focus intact. Review currently reuses this viewport;04 may refine exact review/edit scroll restoration.
- Existing02 navigation/query/visibleTarget/size guards remain. Mutation entry cannot borrow an underlying row from overlays. Below32x12 the model retains draft/focus and accepts only resize/exit behavior. Native execution.go is unchanged;04 owns duplicate handoff/input isolation and report capture.

## Selector and re-resolution contract

- `openPicker` uses the existing Bubbles list and explicit search input; `refreshChoices` retains selected fixed identity when possible. Search Enter accepts, Esc ends search retaining query, and another Esc cancels the picker. Cancelling a picker returns to the caller with draft/focus unchanged. Accepting an add key stores its fixed ID and focuses Continue.
- Contextual Register/Unregister/Delete preserve02's eligible visible-target behavior. Empty choices explain create/import/add or recorded blockers rather than opening editable ID fields. Native locked validators remain authoritative.
- `resolveFormTargets` takes a fresh read-only Snapshot before Review and again immediately before native handoff. It matches exact saved IDs. Host alias and key name/aggregate ConnectionCheck may change without changing target identity; reordered collections are harmless. Changed host login/OS/port/account, key material/provenance/path/private availability, or saved registration/creation/deletion/onboarding record returns a retained-draft error. Missing targets or failed reload do the same. Snapshot updates the view only after successful comparison. Backend locked re-resolution still covers races and all original safety checks.
- `resumeChoices` exposes unfinished local creations/deletions from category lists (or a matching key detail), and supported remote continuations for the displayed host/key/registration. Local forms omit auth/identity/config entirely. Complete local operations are excluded. Public-only complete registrations do not create verification-only continuations. Current registration intent suppresses stale onboarding resume for that pair. This is an eligible selector, not05's grouped Tasks projection.
- Unregister and unregister-resume omit configuration, ask for fresh alternate authentication or manual instructions, and state local files remain. They never select the target key as an alternate credential. The backend still rejects a supplied alternate key matching the target fingerprint. General integrity warnings do not blanket-disable dedicated local recovery.

## Verification and red/green evidence

Public test seam: `tui.NewProgram` with input/window events and rendered View; real `app.Run` fixtures; fake Runner only at the external command boundary. No model internals or test-only accessor. Fixtures are disposable, contain no credentials, and never contact a server.

Red/green slices recorded in task-owned Herdr terminals: staged step validation (legacy11-field red -> focused steps green); exact-ID rename/reorder/path/missing comparison (accepted drift red -> retained error green); local creation/deletion resume (registration-only action red -> reachable type-specific forms green); ordinary caret editing (acb red -> abc green); compact port guidance and unregister labels (missing/awkward error and ID-only fields red -> labelled field guidance green); long paths (beginning required horizontal scrolling red -> PgUp/PgDn full wrapped value green). Tests also verify exclusive choices, auth/account clearing, Unicode, help/picker draft retention, validation across earlier steps and refusal after a target changes between Review and Enter.

The actual handoff test drives NewProgram through native confirmation into real app.Run and captures its first fake SSH probe. It asserts destination, port, interactive key/password authentication arguments, and absence of stale -i after key->password. Before Enter, saved bytes are identical and Runner has seen no dependency or command call. The fixture provides only its explicit disposable registration approval and refuses the first probe; it proves intent, not SSH server behavior.

Final checks at `33654a7240e8da7c0e6ddb0f60dc7e974d584a43`, Windows amd64, Go1.26.2:

- `go test ./internal/tui -count=1`: PASS,3.356s (all02/03 public journeys).
- `go vet ./...`: PASS.
- `go build -buildvcs=false -o <task-owned temporary executable> ./cmd/ssh-keyflow`: PASS.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'`: PASS, app41.852s/ssh23.174s/TUI2.834s. This uses only process execution policy for the repository's native PowerShell fixtures, as documented in implementation.md.
- `git diff --check`: PASS. The build artifact and all task-owned Herdr check terminals were removed. No other worker terminals or files were removed.

The full suite ran once after integration. Subsequent changes are completion documentation only. No native terminal03 smoke, actual server test, Linux/macOS native run, packaging rerun or remote transfer was performed. Coordinator navigation smoke in implementation.md covers02 only.06 owns native form/help/dynamic resize/handoff/interruption acceptance and final runtime documentation. No push, main merge or release.

## Standards self-check

Reviewed separately against AGENTS.md, domain/tracker rules, TDD and ponytail. Existing textinput/list/viewport and authoritative validators are reused. Tests enter only at agreed public seams, with external fake commands and literal intent expectations. No secret, backend guard, schema, CLI/report implementation, dependency or framework changes. Only03 TUI files and03 completion artifacts are owned;01 app files arrived through the authorized merge. No unresolved standards finding remains.

## Specification self-check

Reviewed ticket03 and AC03/04 separately. All ticket boxes have public journey evidence: labelled searchable fixed targets; duplicate/fingerprint/ID search; empty guidance; picker cancel; exact-ID retention/re-resolution; conditional fields and request values; manual/config defaults; public-only limitations; first-invalid validation; type-specific local resumes; unregister alternate authentication and local preservation.02 navigation tests still pass.03 deliberately establishes the retained draft/step seam;04's full dirty-cancel/review/native outcome behavior,05's Tasks/results projection and06's native/integrated checks remain with those tickets. No unresolved03 spec finding remains.
