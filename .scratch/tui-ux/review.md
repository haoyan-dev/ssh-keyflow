# TUI interaction review

Status: ready-for-human

Reviewed: 2026-11-07. Source: `89340aebbc0b3ab7f781b822aaafc2130ae76f73`.

This review concerns the current SSH KeyFlow TUI, its operation boundary, and the difficulty of completing and recovering ordinary tasks. It accompanies a [proposed design](design.md), [draft specification](spec.md), and [implementation tickets](issues/01-operation-outcomes.md). No application implementation is included.

## Objective, scope, and completion criteria

Find interaction difficulties using the current application, explain their cause, redesign the affected journeys, and leave reviewable requirements and local tickets before implementation. Cover orientation, entity selection, destination setup, review/edit/cancel, terminal handoff, results, and recovery. Preserve the existing SSH safety and ownership rules.

Planning is complete when findings distinguish reproduced behavior from source analysis, the design explains keyboard journeys and failure paths, requirements have testable acceptance criteria, and tickets identify dependencies and verification. Product approval and implementation remain subsequent work.

Open decisions are recorded in [design.md](design.md#discussion-decisions). Current proposals are recommendations for discussion, not approved behavior.

## Evidence method

- **Native**: built the executable and interacted through a Windows PTY, including task-owned Herdr terminals. Tested an empty temporary home and an isolated populated fixture home. No actual SSH server was contacted.
- **Event**: temporary probe drove public `tui.NewProgram` with key events and a captured renderer. Fixture creation used the shared app boundary; only fingerprint validation was stubbed. No SSH or real key generation was performed.
- **Source**: examined TUI event handling and app/state contracts. Independent read-only review checked backend constraints at the same source commit.
- Severity below describes the interaction risk, not incident frequency. There is no user study, timing study, or claim of observed production damage.

## Findings

| ID | Priority | Difficulty and concrete evidence | Proposed response |
| --- | --- | --- | --- |
| F01 | High | Manual guidance is labelled **Operation complete**. Native: choose manual registration, continue, confirm `yes`; result heading says complete while the body provides commands still to perform. In the onboarding fixture, explicit manual registration returns nil while the saved registration is unknown/unverified; a separate Register can leave no new record or leave prior verified evidence unchanged. | Distinguish manual follow-up, recorded registration, connection verification, and configuration. |
| F02 | High | Going back from review discards a draft. Native: `c`, enter `demo`, Enter to review, Esc, `c` again; the name is blank. Event probe confirms. Review has no Edit action. | Esc returns to the draft; explicit Cancel handles discard. |
| F03 | High | Contextual actions are accepted from unrelated rows and result screens. Event: `d` from a destination opens a blank key form. Source: global actions use the underlying list selection even while a result or discovery page is visible. | Offer eligible actions tied to the visible target; use a picker when more context is needed. |
| F04 | High | Recovery is mixed with entities and history. Source: every onboarding record is listed, incomplete creation/deletion rows live among keys, and registration events live among current registrations. An onboarding never gains a complete stage. | A derived Tasks view groups linked work, follows current intent, and separates history. |
| F05 | Medium | Destination setup exposes eleven fields at once as a sequence, including Windows account on Linux, identity path for manual auth, and mutually exclusive existing/new key fields. Native: `a` shows `Field 1/11`. | Separate destination, key choice, method/options, and review; reveal conditional inputs only. |
| F06 | Medium | Register/unregister/resume depend on recalling IDs or unique names. Source/event: registration review contains opaque host/key IDs; rows use registration IDs as titles. Filtering is disabled. | Search aliases, login, names, fingerprints, and IDs; display resolved labels and retain fixed IDs internally. |
| F07 | Medium | Validation does not tell users where to fix the destination. Native: fill alias only then Ctrl+S; `invalid destination fields` appears without pointing to the missing host/user/OS. | Inline messages, first-invalid-field focus, and retained input. Backend remains authoritative. |
| F08 | Medium | Details are raw JSON and require mental joins to interpret destination/key relationships, status, and history. Event/source: Enter opens serialized record fields. | Labelled human-readable summaries; diagnostics and fixed IDs remain available. |
| F09 | Medium | Help loses the page it was opened from. Event: destination details, `?`, Esc returns to list. One list also serves all categories without explicit per-category context restoration. | Save origin, selection, query, focus, and scroll for back/help navigation. |
| F10 | Medium | An empty home shows generic empty lists and the full action footer. Native: no explanation of the first useful task. At narrow widths many actions move behind help. | Explain empty states and show the next useful actions in the body; compact context-specific footer. |

## Source anchors

Links refer to the reviewed source; line numbers below are evidence anchors at the recorded commit.

- F01: [TUI completion handling](../../internal/tui/tui.go), lines 189-208; [manual registration](../../internal/app/registration.go), lines 42-79; [manual unregister](../../internal/app/revocation.go), lines 79-88; [execution boundary](../../internal/tui/execution.go).
- F02/F07: [review and form handling](../../internal/tui/forms.go), lines 18-21, 87-141; [review escape](../../internal/tui/tui.go), lines 114-120.
- F03: [action dispatch](../../internal/tui/tui.go), lines 184-188; [target prefilling](../../internal/tui/forms.go), lines 27-51. Result content is set independently of the underlying selected row.
- F04/F06/F08/F09/F10: [list construction and JSON details](../../internal/tui/tui.go), lines 273-318; filtering at line 65; help navigation at lines 153-173; chrome at lines 225-254.
- F05/F06: [form definitions](../../internal/tui/forms.go), lines 53-82. Basic validation calls `Request.Validate`; operation-specific authority remains in [app operations](../../internal/app/operations.go).

## Backend constraints that shape the redesign

1. **Linked records are not independent work items.** Onboarding links creation and destination/key work; creation resume alone does not register. Registration resume follows the saved `register` or `unregister` operation. Onboarding resume pursues registration, so it must not be offered after a later unregister. See [creation](../../internal/app/creation.go), lines 105 onward; [onboarding](../../internal/app/onboarding.go), lines 169-213; [resume dispatch](../../internal/app/operations.go), lines 347-386.
2. **Completion cannot be inferred from onboarding stage.** Valid stages are `key_pending`, `planned`, `manual`. Completion must use linked current evidence; contradictory joins require review. See [state validation](../../internal/state/state.go), around line 388.
3. **Registered and connected differ.** Registration is checkpointed before specified-key verification. A failed verification can coexist with recorded registration and skipped configuration. Use `Registration.ConnectionCheck` for that destination; `Key.ConnectionCheck` is not destination proof. Public-only keys cannot provide private-key verification. See [registration](../../internal/app/registration.go), lines 202 onward, and [state](../../internal/state/state.go), lines 19 onward.
4. **Unknown outcomes need reconciliation.** An error does not prove no remote change. Resume must retain read-only inspection before retry. Unregister requires fresh alternate authentication excluding the removed fingerprint. See [registration](../../internal/app/registration.go), lines 154-200, and [revocation](../../internal/app/revocation.go), lines 98 onward.
5. **Config edits do not have generic resume IDs.** Recovery uses recorded digests/backups inside the existing configuration pathway. Do not promise a config-only Resume or an independent connection-check operation: neither exists. See [config](../../internal/app/config.go), lines 46 onward, 241 onward; [request operations](../../internal/app/operations.go), lines 18-40, 201-225.
6. **Integrity warnings must preserve recovery.** A partially completed deletion can itself generate a general review warning. Dedicated deletion resume reconciles partially absent files; disabling all mutations on every warning would remove that route. Never weaken deletion checks. See [deletion](../../internal/app/deletion.go), lines 40-110 onward; [integrity checks](../../internal/app/import.go), lines 233 onward.
7. **Manual follow-up is not always durable.** Registration fallback occurs before allocating a new registration record. Without an onboarding/current pending record it may leave no resumable task; on an existing verified pair prior evidence can remain unchanged. A typed result can name manual work without promising restart persistence. See [registration](../../internal/app/registration.go), lines 48-79 and 141 onward.
8. **Stopped does not mean rolled back.** Declined alias generation may follow successful registration; declined Include can follow saved managed alias/binding; interruption may follow saved checkpoints and created files. Keep exact native approvals and credential input. See [confirmation](../../internal/app/import.go), lines 20-39; [creation](../../internal/app/creation.go), lines 145 onward; [application contract](../../docs/application-contract.md), lines 100, 127-131.
9. **Outcome facts need a small shared boundary addition.** `Snapshot` provides state plus advisory Discovery/Review; `Run` returns error and text, not typed outcomes. Error/output-string parsing cannot reliably classify partial success. Retain text diagnostics and reload state; a load error must not become an empty list. See [operations](../../internal/app/operations.go), lines 18-41, 228 onward; [app entry](../../internal/app/app.go), lines 56 onward.

## What the evidence does not show

The current views fitted the measured 80x24, 60x18, and 32x12 windows in the event probe. This review does not claim a reproduced overflow bug. The design work at small sizes concerns clarity, focus, and access to actions; Unicode, long content, real terminal resize behavior, and every operation still need implementation-stage verification.

No destructive deletion, real SSH authentication, remote registration/unregister, real interrupted key generation, macOS terminal, or Linux terminal was exercised. Those are source-backed design constraints and future verification cases, not demonstrated live outcomes. See [verification](verification.md).
