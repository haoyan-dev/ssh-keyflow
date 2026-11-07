Status: ready-for-human
Blocked by: 02

# 03: Replace recalled IDs and mixed inputs with selectors and conditional steps

## Problem and intended behavior

Destination setup traverses eleven mixed fields; register/unregister/resume ask for IDs and irrelevant options. Users should choose labelled entities and see only applicable inputs, with errors located at the field they can fix.

Specification: [AC03 and AC04](../spec.md#ac03-eligible-selectors), relevant AC02/AC10/AC11. Design: [setup journey](../design.md#add-a-destination-and-set-up-key-access).

## Scope and seams

- Extend existing TUI forms/components with eligible searchable destination/key/registration selection, keeping fixed IDs in requests. No dependency or selector framework without demonstrated need.
- Add Destination -> Key -> Access method/options steps. Show Windows account only on Windows, new key name only for Create, and identity path only for private-key authentication. Clear inapplicable hidden values.
- Keep manual method and configure=no defaults. Explain public-only limits and bootstrap credential purpose; no secret field.
- Preserve create/import capabilities; defer inline import inside onboarding. Type-specific recovery forms expose no unrelated authentication/configuration fields for local operations.
- Validate all applicable fields, point to the first error, retain input, and reuse authoritative rules. Native execution remains outside this ticket's navigation implementation.

## Acceptance criteria

- [x] AC03 normal target selection requires no recalled ID; ID/fingerprint search works and duplicates are distinguishable.
- [x] Empty/ineligible targets explain next steps. Picker cancel preserves caller state; changed/missing IDs cannot silently retarget.
- [x] AC04 Linux/Windows and auth-mode transitions display applicable fields and remove hidden stale inputs from requests.
- [x] Existing/new key choice is exclusive; manual/configure defaults remain unchanged; public-only choices explain unavailable verification.
- [x] Invalid port/required host/user/OS/name/path errors locate the input, including validation of earlier steps from Ctrl+S.
- [x] Local resume omits auth/config; unregister asks fresh alternate credentials and describes local-file preservation. No safety rule is relaxed.

## Verification

Use public-entry journeys and focused shared-validator cases. Include existing/new key, Linux/macOS/Windows, manual/password/private-key modes, changed conditional values, public-only key, empty eligible set, selection refresh, first-invalid focus, Unicode labels, and no execution during preparation. Verify emitted requests, not only snapshots matching implementation internals.

### Implementation evidence (2026-11-07)

Completed on `feat/03-selectors-staged-forms`, clean start `dce6c7c6b9c8d71c92e7ed71fdb77b2445cce5c3`. Implementation commit `d032cd3` merged integration `93c27798ce9aa077ccc704bb19707938892fe281`; exact verified source is `33654a7240e8da7c0e6ddb0f60dc7e974d584a43`.

- Public `tui.NewProgram` journeys cover searchable fixed-ID/fingerprint selectors and duplicate labels (retained02 tests); Unicode creation/destination labels; existing/new exclusivity; Windows/Linux/macOS fields; manual/key/password transitions; identity/account clearing; public-only guidance; first-invalid port/host/user/OS/name/path focus; picker/help cancellation; cursor editing; long paths and compact errors at80x24,60x18,32x12.
- Fixed-ID rename/reorder keeps the selected key. Changed paths, missing keys and a changed destination after Review retain the draft and refuse handoff. Preparation uses only Snapshot: rejecting Runner journeys and unchanged saved-state bytes prove no dependency probe, command or mutation before handoff.
- Public NewProgram -> real app.Run -> fake external Runner tests verify the actual selected destination/port and key/password SSH probe arguments. Switching key authentication to password emits no stale identity argument. The fixture explicitly approves only its disposable registration and stops at the first fake probe; no real SSH runs. There is no production test-only Request accessor.
- Local creation/deletion continuations remain reachable with an integrity notice and omit remote auth/config. Unregister displays the labelled pair, requests fresh alternate authentication or manual guidance, preserves local files, and offers no configuration field.
- Meaningful red/green slices exposed the old mixed form, missing target re-resolution, missing local-resume wiring, cursor reset, compact error guidance, missing unregister labels and horizontal-only long-value access. Each corresponding focused test passed after the fix.
- On the exact merged source above: `go test ./internal/tui -count=1` PASS (3.356s); `go vet ./...` PASS; `go build -buildvcs=false -o <task-owned temporary executable> ./cmd/ssh-keyflow` PASS; `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'` PASS (app41.852s, ssh23.174s, TUI2.834s); `git diff --check` PASS. Windows amd64, Go1.26.2. Execution policy is process-only. Task-owned check executable and terminals removed.
- Standards and specification checks were performed separately; no unresolved03 finding remains. The detailed contracts, check limits and04/05/06 boundaries are in [handoff-03.md](../handoff-03.md). Full dirty-cancel/review/native reports remain04; Tasks grouping remains05; native/integrated verification and runtime docs remain06.

## Handoff

Starts after 02. Provide retained draft/step state and validated requests to 04. Keep changes focused in forms/selector helpers; coordinate TUI event-handler ownership.

## Comments

- 2026-11-07: Proposed ticket; the final UX choices require human review before implementation.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.

- 2026-11-07: Implemented and verified ticket03; all acceptance criteria checked with evidence above. Ready for human review and coordinator integration. Comments remains the final section. No push, main merge, release or remote source transfer.

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
