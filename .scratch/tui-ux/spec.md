# TUI UX redesign specification

Status: ready-for-human

The [design direction](design.md#discussion-decisions) was accepted on 2026-11-07 after the interaction walkthrough. All six implementation tickets are implemented, integrated, verified and ready-for-human; their acceptance criteria are checked. Current evidence is in [coverage-06.md](coverage-06.md), [final-review.md](final-review.md) and [native-verification.md](native-verification.md). On 2026-11-07 the user invoked implement-spec and authorized implementation of all six tickets on the integration branch implement/tui-ux.

Source baseline: `89340aebbc0b3ab7f781b822aaafc2130ae76f73`, reviewed 2026-11-07. Evidence: [interaction review](review.md). Verification record: [verification.md](verification.md).

## Objective and boundaries

Reduce recall, lost input, hidden recovery, and misleading completion when operating SSH KeyFlow through the terminal. Users must be able to identify a target, prepare an operation, revise or cancel it, complete native approvals, and understand saved effects and remaining work.

Include list/detail navigation, selectors, conditional forms, review/back/help, native handoff, structured result facts, and a Tasks projection. Keep `app.Run`, structured requests, Snapshot, existing state schema, and the existing safety/ownership model as the foundation. The only shared-boundary extension is additive operation reporting; it must preserve existing CLI behavior.

The original planning change excluded implementation; the later implement-spec request authorized the implementation recorded here. Continue to exclude new SSH capabilities; bulk actions; destination rename/edit; inline import inside onboarding; new persisted workflow schema; migration; installer/release work; standalone connection-check or config-only operations; changing exact confirmations; collecting secrets in the TUI.

## Domain and state rules

Use the repository [GLOSSARY](../../GLOSSARY.md). Destination is machine plus login user. Managed keys and discovered unmanaged files remain separate. Unregister changes remote registration intent/state; delete local key concerns managed local files and preserves history. Recorded evidence is not a live server assertion.

Current registration, destination-specific connection check, configuration, and operation progress are independent facts. Do not collapse them into a green/red key-wide badge. Show `Registration.ConnectionCheck` for a destination. Public-only keys show verification unavailable and the saved unverified value; do not treat lack of private key as an execution failure.

## Interaction requirements and acceptance criteria

### AC01: Categories, identity, search, and empty states

- Categories are Destinations, Keys, Registrations, Tasks. Lists use names, login target, and relevant status rather than fixed ID as the primary label. Fixed IDs remain inspectable and are used in requests.
- `/` searches supported display fields plus fixed IDs. Search is case-insensitive for names/aliases; fingerprint/ID substring search works without altering identifiers. Query, selected ID, and scroll are retained per category across detail/help/back and category switches.
- A refresh preserves the selected ID if present; otherwise selects the nearest remaining row and announces the change. Empty filtered results preserve query and offer Clear search. No selection produces no implicit target action.
- Fresh home explains add destination and create/import key. Empty Tasks means no recorded follow-up, not universal verification. Read-only startup/search/navigation never create storage or probe SSH.

### AC02: Readable details and action eligibility

- Destination, key, and registration details join existing references and clearly separate registration, connection evidence, configuration evidence, and history. IDs/paths/full fingerprints remain accessible. Missing references produce requires-review details rather than an invented relationship.
- The action target comes from the visible entity/result/task, never an underlying list row. A related picker is opened if more context is needed. Ineligible actions explain their blockers in Actions/help; shortcuts do not open blank ID forms.
- Register from destination selects key; from key selects destination; from registration uses the displayed pair. Unregister selects a current eligible registration and shows its destination/key. Delete applies only to a managed key. Historical events and tombstones are read-only.
- Eligibility presentation is advisory; the operation revalidates against locked current state. The TUI must not bypass guards or claim preflight guarantees.

### AC03: Eligible selectors

- Pickers show alias/login or key name/fingerprint/private-key availability and search by the AC01 fields. Duplicate-looking labels remain distinguishable by target and fixed ID.
- Empty eligible sets explain why and provide the appropriate navigation to create/import/add; no raw-ID entry is required for normal operation. Existing IDs remain usable as search terms.
- Esc returns to the caller with draft values and focus intact. Selection stores the fixed ID and display label; later rename/reorder or refresh never silently retargets a draft.
- Before handoff, re-resolve the selected IDs. Missing/changed targets return a useful error and retained draft; locked backend checks still decide execution safety.

### AC04: Conditional forms and validation

- Add uses Destination -> Key -> Access method/options -> Review. Destination contains alias, host/IP, user, port22, explicit supported OS. Windows account is shown only for Windows. Existing key versus new key is a mutually exclusive choice.
- Manual instructions remains the default authentication method. Identity path appears only for private-key authentication. No password/passphrase field exists. SSH configuration defaults no and is an explicit choice with eligibility explanation.
- Create/import retain their existing capabilities through focused forms. Register/unregister use the same selectors and method choices. Local creation/deletion resume has no irrelevant authentication/configuration fields.
- Inapplicable hidden values are cleared from the draft when the controlling choice changes. Only applicable inputs enter the request. Going back/help/review preserves all applicable values.
- Validate on step continuation and review; name the field, explain the accepted value, and focus the first invalid input, including earlier steps. Port must be a number in 1..65535; required target/user/OS/key-name/path errors receive specific guidance. Preserve valid entries. Reuse shared validation rules; do not relax backend checks or invent a competing target grammar.

### AC05: Back, edit, cancel, and help

- Esc from Review and the Edit control return to the last edit position without clearing values. Esc moves back one step; from first step it invokes Cancel. Ctrl+S validates all applicable fields and opens Review or focuses the first invalid field.
- A dirty draft is one with user changes to values or choices relative to entry. Explicit Cancel or Ctrl+C prompts Keep editing / Discard draft; default Keep editing. Clean drafts return directly. Discard runs no operation.
- Help saves and restores origin, query, selection, draft focus, and scroll. Text-input keys, including `q`, `?`, and action letters, remain text. A labelled keyboard-focusable Help control is available in forms.
- Tab/Shift+Tab cycle categories on list/entity pages, fields in forms, and controls in overlays. They do not discard a draft. Search Enter accepts the filter; Esc leaves search editing while retaining the query; explicit Clear removes it.
- Outside drafts, help, and native execution, `q`/Ctrl+C exits. In help, Esc/`?` returns to origin and Ctrl+C closes help first. Native execution retains its own interrupt behavior. Every screen shows the applicable back/continue/help controls within the size contract.

### AC06: Orientation review and native execution

- Review displays action, resolved destination/login, key name/full fingerprint or new-key intent, bootstrap method, configuration request, expected local/remote effects, and unresolved runtime checks. Paths not yet determined are described as subject to terminal review.
- Enter is labelled Continue to terminal. Review itself does not execute, write storage, contact SSH, or replace exact backend approvals. Enter is accepted once per handoff; repeated input cannot launch duplicate execution.
- Handoff retains original terminal stdin/stdout for OpenSSH and native confirmations. Passwords/passphrases never enter form state, reports, fixtures, logs, or documentation. Native confirmation denial/interruption does not imply rollback.
- On return, keep typed outcome facts and diagnostics, refresh Snapshot, and restore the TUI. Native control sequences/output do not become keyboard actions on the returned result screen.

### AC07: Additive outcome facts

- Extend the shared operation boundary with an optional typed report sink on `app.Options`, consumed by TUI execution. `app.Run(ctx, Options) error`, requests, CLI text, exit semantics, and persisted schema remain compatible when the sink is absent.
- Report only authoritative facts from operation/checkpoint branches: resolved target IDs, committed local/remote evidence, manual-guidance requirement, verification outcome/unavailability, managed alias generation and main-config Include installation/reconciliation independently (applied/skipped/declined/requires-review), cancellation, and uncertainty. Report saved effects after successful persistence, with no secrets. A read-only Snapshot supplements these facts; it cannot establish every current-operation effect on its own.
- Reports are informational, never a second operation executor or approval callback. No new workflow engine, stdout/error-message parsing, native prompt replacement, or duplicate state machine. Existing confirmation callbacks stay separate.
- Lack of a report or interrupted execution is unknown, not success/no changes. Existing records may predate reporting: absent facts are not recorded. Retain the returned error and diagnostics even when some effects succeeded.

### AC08: Results and stale state

The primary summary names remaining work and saved effects. The following required cases are distinct:

| Execution evidence | Required presentation |
| --- | --- |
| All requested applicable work confirmed complete | Completed summary with individual facts; private-key verification is not claimed for public-only keys. |
| Explicit manual register/unregister guidance, including nil error | Manual action required; show recorded state separately from what was not checked this time; instructions and supported follow-up route. No record means not recorded, not a fabricated pending registration. |
| Registered saved, specified-key verification failed | Registered; connection check failed; configuration skipped when that is the branch taken. |
| Registered saved, alias generation approval declined | Stopped after registration; managed alias not generated this time. Preserve any prior config evidence separately. |
| Registration and managed alias saved, Include approval declined or later Include step fails | Name both saved effects; main config remains unchanged on denial, Include/manual setup remains; on failure report exact checkpoint/uncertainty rather than configuration not applied. |
| Confirmation declined with authoritative no effects | Cancelled before changes. |
| Error/interruption with uncertain effects | Stopped; outcome unknown where appropriate, preserved checkpoints, reconcile/recovery guidance. |
| Snapshot reload fails | Preserve operation facts/error/diagnostics; mark prior saved-state views stale, expose Refresh/Exit, disable mutations based on stale state until refresh succeeds. |

- Manual Register for a previously unrecorded pair may save no registration/creation/onboarding intent; for an already verified pair it may leave previous evidence unchanged. Test both, including restart. The immediate typed result labels manual follow-up but does not overwrite recorded evidence or promise a Task. If no persisted follow-up exists, show No saved recovery task, retain instructions while the result is open, and direct the user to Register again after exit/restart. No session-only fact is silently persisted by this redesign.
- Include denial before an edit intent is saved can also lack durable task evidence. Distinguish the generated alias from activation through Include and explain the session-only follow-up; no assumption that a missing ConfigEdit proves activation or non-activation.
- Results provide a concise primary action and target-bound View entity / View task links where applicable; diagnostics/manual commands are scrollable and remain available until leaving the result.
- Refresh retries Snapshot only, never execution. A later refresh does not automatically replay an operation or confirm remote state.

### AC09: Tasks projection and precedence

Derive Tasks from one loaded state, without writes, dependency probes, new IDs persisted to disk, or inferred remote access. List ordering is deterministic: requires-review, uncertain/interrupted work, manual follow-up, then verification follow-up; stable source identity breaks ties. Task identity derives from saved IDs (or config path for review entries). Multiple reasons on one work item appear as substeps rather than duplicate primary rows.

Apply these rules in order:

| Existing evidence | Projection and continuation |
| --- | --- |
| State cannot load | Show load error; no empty Tasks or mutation. |
| Missing/contradictory links in otherwise loadable data | Requires review with source IDs; no guessed continuation. |
| Pending local deletion | One deletion task per saved deletion ID, separate from historic onboarding. Resume that deletion with local reconciliation and original checks; no authentication form. |
| Current pair has unregister intent | Registration task uses saved unregister operation if unfinished/unknown/failed. If removed/complete, no remote follow-up. Suppress stale onboarding/verification prompts for that pair. Never offer onboarding resume that would re-register. |
| Linked onboarding, creation, and register work | One grouped task, not three rows. If key creation incomplete, choose onboarding continuation and explain that it can continue into registration. If creation complete and current registration exists, prefer its saved registration ID; if registration not yet exists, use onboarding ID. |
| Standalone incomplete creation | One local creation task by creation ID. It creates/reconciles the key only and promises no remote registration. |
| Register operation pending/unknown/failed or manual | Show interrupted/unknown/manual follow-up with saved pair and supported resume; remote retry must reconcile first. Where persisted data cannot distinguish manual from unknown (including data saved by the new runtime), use neutral Outcome unknown wording. A current-session typed result can explain manual guidance without changing saved evidence. |
| Registered/complete + private key + failed/unknown/unverified connection | Verification follow-up under the same grouped pair. Offer Continue registration and explain existing reconciliation/verification; do not invent a Verify-only operation. |
| Registered/complete + public-only key | Show verification unavailable in entity details; no perpetual verification-only task. Other unknown registration/manual work still remains follow-up. |
| Registered/complete + verified connection, or complete creation | No operation-pending task merely because onboarding is planned/manual. Other explicit pending config/deletion evidence is still shown. |
| Config pending digest / incomplete config edit | Requires-review config entry grouped with a safely known parent, otherwise by path. No fabricated generic resume ID. Show supported configuration pathway and its effects when identifiable; otherwise read-only recovery guidance. |
| Complete deletion / deleted-key tombstone / history event | History only, not an executable task. Retain relation visibility for audit. |

- Suppress only redundant presentation, never delete records or infer onboarding completion by `stage != complete`. A completed creation linked to unfinished onboarding does not by itself complete the group.
- Finished current registration does not prove all past requested configuration ran: no durable Configure intent is stored on onboarding. Show not recorded where unavailable. An empty Review string is not full config-integrity clearance.
- A general integrity warning is visible; it does not blanket-disable dedicated recovery. Expose recovery controls that can invoke existing guarded paths; failed checks explain blockers and preserve state.
- A manual operation with no saved pending intent is not a durable Task. After restart projection follows saved evidence only, including when an earlier registered/verified pair was left unchanged. Current-session result follow-up and persisted Tasks are explicitly different; no task is invented for an Include refusal before intent persistence.
- No Mark done shortcut for manual guidance and no dismissal that falsely verifies remote state.

### AC10: Destructive-operation safeguards

- Unregister requires fresh alternate authentication excluding the target fingerprint, or explicit manual guidance; remote unknown resumes inspect before retry. Local key files are preserved.
- Local deletion retains all registration-removal, digest/path/ownership, config, pending Include, other-key, and reappeared-file checks. A blocked operation names the blockers and supported next route; it never offers force/delete-all.
- Target selection, edit/back, search, task projection, history, and discovery cannot cause mutation. Refusal and runtime changes preserve saved checkpoint truth.

### AC11: Terminal size and accessibility

- Verify 80x24, 60x18, 32x12 layouts with focus, long/Unicode aliases, full fingerprints/paths, many rows, wrapped errors, and instructions. The existing measured layout already fits these sizes; the goal is usable hierarchy and controls, not an unproven overflow fix.
- Keep focused field/error/primary next control visible. Body scrolls; full values are accessible. Text labels supplement color. No mouse requirement or horizontal-only access to critical text.
- Below 32 columns or 12 rows, show the minimum size requirement plus Exit, retain drafts, and prevent starting a mutation. Resize restores the previous screen/focus without execution. Native handoff retains native terminal behavior.

### AC12: Regression verification and documentation

- Use public `tui.NewProgram` event tests for journeys and pure projection tests for Tasks. Fake Runner tests assert command intent and denial behavior; never access real keys/servers from fixtures.
- Test shared reporters across success, explicit manual nil-error, partial failure, cancellation after prior writes, unknown outcome, and reload failure. Confirm reporting-disabled CLI behavior and safety tests remain compatible.
- Perform native terminal checks of focus, rendering, handoff/input, interruption, resize, review edit, and results. Run applicable tests/build/vet in README at the implementation commit; record platform, SHA, conditions, and untested platforms. Do not claim fake SSH proves actual server behavior.
- At implementation time update README help and application contract to match shipped behavior. Preserve historical specifications. The original proposal did not change runtime documentation; the authorized implementation has now updated README and the application contract.

## Traceability and sequencing

| Finding | Acceptance criteria | Ticket |
| --- | --- | --- |
| F01 misleading completion | AC07, AC08 | [01 outcome facts](issues/01-operation-outcomes.md), [05 Tasks/results](issues/05-tasks-results-recovery.md) |
| F02 lost draft | AC05, AC06 | [04 review/back/handoff](issues/04-review-edit-native-handoff.md) |
| F03 contextless actions | AC02, AC03, AC10 | [02 navigation](issues/02-navigation-details.md), [03 selectors/forms](issues/03-selectors-staged-forms.md) |
| F04 scattered/stale recovery | AC09, AC10 | [05 Tasks/results](issues/05-tasks-results-recovery.md) |
| F05 eleven-field setup | AC04 | [03 selectors/forms](issues/03-selectors-staged-forms.md) |
| F06 recalled IDs | AC01, AC02, AC03 | [02 navigation](issues/02-navigation-details.md), [03 selectors/forms](issues/03-selectors-staged-forms.md) |
| F07 generic validation | AC04 | [03 selectors/forms](issues/03-selectors-staged-forms.md) |
| F08 raw JSON | AC02 | [02 navigation](issues/02-navigation-details.md) |
| F09 lost origin | AC01, AC05 | [02 navigation](issues/02-navigation-details.md), [04 review/back/handoff](issues/04-review-edit-native-handoff.md) |
| F10 weak empty/compact guidance | AC01, AC11 | [02 navigation](issues/02-navigation-details.md), [06 verification/docs](issues/06-verification-documentation.md) |
| Cross-cutting runtime compatibility | AC06, AC10, AC12 | [04 handoff](issues/04-review-edit-native-handoff.md), [06 verification/docs](issues/06-verification-documentation.md) |

Design direction is accepted. Tickets 01 and 02 have no blockers and can run independently in separate worktrees (shared app boundary versus TUI navigation). 03 follows 02; 04 follows 01 and 03; 05 follows 01, 02, 04; 06 verifies the integrated final state after 02-05. Each implementation ticket includes focused meaningful tests; 06 owns cross-journey/native acceptance and shipped documentation. No parallel edits to the same TUI files without an explicit handoff.

## Comments

- 2026-11-07: Draft created for interaction discussion before implementation. All tickets remain ready-for-human; no approval, release, PR, or merge is implied.
- 2026-11-07: User feedback 「いい感じ」 accepted the interaction direction following the Japanese walkthrough. Promoted the design, specification, and six tickets to ready-for-agent; dependency blockers and unchecked implementation criteria remain in force. This records design acceptance only, with implementation left for a subsequent task.

- 2026-11-07: Implementation authorized through implement-spec for the complete task graph. The implementation uses the agreed public test seams; no additional seam confirmation is needed. Local ticket completion is recorded with checked criteria, verification evidence, and ready-for-human for final review, since the tracker has no closed label. No remote push, main merge, or release is included.

- 2026-11-08: All six tickets completed on implement/tui-ux. Tested product `26269049b747db02988ad00db2b62dcff22119f0`; integration merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards/Spec rechecks resolved S1/P1–P5 with no remaining material findings. Final affected native scroll PASS; earlier exact-source native evidence and platform limits are preserved. Status ready-for-human reflects the local tracker workflow; no push/main merge/release.
