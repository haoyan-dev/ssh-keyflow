# TUI interaction design

Status: ready-for-human

Accepted interaction direction, based on the [current interaction review](review.md). The [specification](spec.md) defines the acceptance criteria. The subsequent implement-spec request authorized implementation. All six tickets are now integrated and verified; [coverage-06.md](coverage-06.md) and [final-review.md](final-review.md) record final acceptance and execution limits.

## Direction

Make the destination, key, intended action, and remaining work visible throughout a task. Users select familiar names; fixed IDs remain the internal identity. Review helps users understand effects, editing keeps their draft, and execution results distinguish saved effects from things that still need attention.

Keep the native OpenSSH handoff. Passwords and passphrases are entered in the original terminal; the TUI does not collect them. Existing exact confirmations and safety checks remain authoritative after the orientation review.

## Orientation and navigation

Use four categories: **Destinations | Keys | Registrations | Tasks**. `/` opens search within a category. Search accepts alias, address, login user, key name, fingerprint, and fixed ID as appropriate. Each category keeps its query, selected fixed ID, and scroll position. A result opens the affected entity or task explicitly; it never borrows the hidden list selection for actions.

Entity lists contain current records. History is available inside the relevant entity, with deleted-key history reachable from Keys. Discovery stays a separate read-only view (`g`), with an explicit Import action only when a supported import can be started. Discovery never makes unmanaged files eligible for destructive operations.

An empty home explains the two useful beginnings: add a destination with an existing/new key, or create/import a key first. Empty search results offer Clear search. Empty Tasks says there is no recorded follow-up; it does not claim every server is currently reachable or verified.

At 80x24, a destination detail would read approximately:

```text
SSH KeyFlow
[Destinations] | Keys | Registrations | Tasks (1)

build-box
alice@build-box.example:22   Linux

Key               Laptop   SHA256:RXm/...YnA
Registration      Recorded registered
Connection        Verified for this destination (recorded)
Configuration     Not requested in last operation

Registrations and history  -> Enter

r Register another key   u Unregister selected registration
/ Search   ? Help   Esc Back
```

This is a content hierarchy sketch, not a claim of an exact rendered layout. If several registrations exist, `u` opens an eligible registration picker. Configuration labels describe current-operation evidence where available; absent persisted evidence is shown as not recorded, never guessed from an old onboarding request.

## Keyboard contract

| Context | Interaction |
| --- | --- |
| Category list | Tab/Shift+Tab cycle categories; arrows select; Enter opens; `/` searches; `?` opens help. |
| Search input | Text keys type; Enter accepts filter and focuses results; Esc closes input while retaining query; explicit Clear resets it. Global shortcuts do not steal input. |
| Current entity | Contextual `r`, `u`, `d`, or `s` acts on the displayed eligible target or opens a related eligible picker. Missing actions have visible explanations in Actions/help. |
| Creation entry | `a` adds a destination, `c` creates a key, `i` imports a key from lists/entity views. Forms never treat those letters as global commands. |
| Choice picker | Arrows choose; typing in search filters; Enter selects a labelled item; Esc returns with the draft unchanged. |
| Form | Tab/Shift+Tab move fields; Enter advances or continues the step; Ctrl+S validates and opens review, focusing the earliest invalid field across steps. |
| Review | `e` or Esc returns to the last editing position; Enter hands off to native execution; explicit Cancel opens discard confirmation when dirty. |
| Help | Esc or `?` returns to the exact origin, including draft focus and scroll. In a text field, `?` is text; a labelled Help control opens help. |
| Draft exit | Explicit Cancel or Ctrl+C opens Keep editing / Discard draft if dirty; Keep editing is default. A clean draft returns directly. |
| Quit | `q` quits from lists and ordinary details/results; Ctrl+C quits outside drafts/help/native execution. Help returns to origin first. Native controls apply during handoff. |

Esc moves back one level, never silently discards a dirty draft. On the first step Esc invokes the same cancel flow. Picker Esc cancels the selection only. Category switching is available from category/entity pages, not as an implicit way to abandon a draft. Read-only discovery/history screens expose back/navigation, not mutations based on their underlying row. Result actions are explicit target-bound actions.

## Add a destination and set up key access

1. **Destination**: alias, host/IP, login user, port (22), and OS choice. OS has no guessed default. Only Windows reveals standard/administrator account choice. Validate here with field-specific guidance.
2. **Key**: searchable existing managed keys, or Create a new Ed25519 key. Show name, fingerprint, and private-key availability in existing choices. Only Create reveals the new management name. A separate Import journey remains available before starting this flow; inline nested import is deferred.
3. **Access method and options**: Manual instructions (existing default), Password, or Another private key. Show identity path only for private-key authentication. Explain that these are bootstrap credentials; no password/passphrase field is added. SSH configuration is an explicit yes/no choice, default no. Public-only keys explain that specified-private-key verification is unavailable; configuration may depend on backend eligibility and is never promised blindly.
4. **Review**: show resolved alias/login, selected key/name/fingerprint or new-key intent, method, optional config request, expected local/remote effects, and what remains subject to native checks. `e` returns to edit with all values. Enter means Continue to terminal; it is not the final destructive approval.

Switching OS or authentication mode clears inapplicable hidden values before request creation. Switching back shows blank inapplicable values rather than silently reusing a hidden identity. All applicable values survive back, validation, review edits, and help.

For a common Linux destination with an existing key and manual instructions, the user makes three focused choices/steps rather than traversing eleven mixed fields. The number of backend checkpoints and approvals does not change.

## Register, unregister, and delete

From a destination, Register chooses an eligible managed key. From a key, Register chooses a destination. From a registration, the current pair is already resolved. Pickers show names and destination login; fixed IDs are secondary. No eligible target yields an explanation and the relevant create/import entry, not an empty ID form.

Unregister always identifies the registration and destination/key fingerprint being removed, asks for fresh alternate authentication or manual instructions, and states that local key files remain. It does not offer the target key as an alternate credential. Native backend validation remains the final authority, including changed state since the view loaded.

Delete local key is available for a managed key and shows recorded blockers before review. It requires all recorded registrations removed and all existing integrity checks. A failed eligibility check is explained rather than bypassed. Deleted keys and historical events are read-only. Pending deletion recovery remains reachable even if the general integrity banner is present.

## Tasks and recovery

Tasks is a read-only projection of existing saved work and evidence, not a new persisted job engine. It groups linked onboarding/creation/registration work into one primary row with substeps. A row states the destination/key, why attention is needed, saved operation, and the exact available continuation. Where saved unresolved intent exists, manual instructions remain a follow-up until authoritative evidence resolves it; re-opening instructions alone does not mark a task done. Manual Register can also leave no registration record, or leave an already registered/verified record unchanged. In those cases the current result says Manual action required and explains that no new resumable task was saved. Instructions stay available in that result; after restart the user starts Register again. Tasks does not fabricate durable work or downgrade prior evidence.

Current registration intent supersedes old onboarding intent. After unregister, an old planned/manual onboarding must not offer re-registration. Completed creations and deletion tombstones appear in history. Registered records with failed verification remain follow-up with an explanation: the available continuation re-enters registration/reconciliation; there is no standalone Verify command. Public-only verification is unavailable, not a retry loop. Missing or contradictory links require review.

Config issues are visible as review items, with read-only details and supported registration/configuration pathway where known. Do not label these as config-only Resume. A generic integrity warning does not disable all recovery routes; each operation still validates its own preconditions.

## Results with honest next steps

A manual result should make the pending work prominent:

```text
SSH KeyFlow
Register Laptop on build-box

Manual action required
Instructions prepared; remote changes were not verified.

Registration record   Unknown
Connection check      Unverified (public key only)
Configuration         Not requested

Next: Review manual instructions
      View destination / View follow-up task

Enter Instructions   Esc Back   ? Help
```

At 32x12, show one focused block with scroll:

```text
SSH KeyFlow
Register on build-box
Manual action required
Remote state unknown

Connection: unverified
Public key only

Enter Instructions
Esc Back   ? Help
```

Use a small typed shared outcome boundary plus refreshed saved state. Outcomes distinguish requested-work completion, manual follow-up, partial/stopped/uncertain execution, and cancellation before execution. Detail includes saved effects, registration and destination-specific connection facts, configuration outcome, blockers, and next supported action. Diagnostics remain scrollable; text does not determine success classification.

If registration succeeds and verification fails, say **Registered; connection check failed** and indicate configuration was skipped. If the user declines alias generation after registration, say **Stopped after registration; managed alias not generated**. If the user declines the later Include edit, say **Registration and managed alias saved; Include not installed** and show the remaining manual setup. A failure after alias persistence likewise preserves that fact. Report alias-file generation and main-config Include installation/reconciliation separately; declining one does not roll back the other. An Include refusal before an edit intent is saved may have no durable task, so its remaining work is explicit in the current result. If snapshot reload fails, retain operation facts/diagnostics, mark displayed saved state stale, and offer Refresh/Exit before further affected mutation. A failure or interruption never implies rollback.

## Layout and accessibility

Target 80x24 first, retain compact use at 60x18, and use focused stacked content at 32x12. At smaller sizes show a resize requirement and Exit without interactive mutation controls. Use text labels as well as color. Keep focused fields, inline errors, and the primary next action visible; scroll the remaining body. Full fingerprints, paths, error output, and instructions must remain accessible without horizontal truncation being the only representation.

## Discussion decisions

| Decision | Accepted choice | Rationale / considered alternative |
| --- | --- | --- |
| Recovery placement | Fourth Tasks category | Makes interrupted/manual work discoverable. A badge plus per-entity recovery is smaller visually but keeps recovery scattered. |
| Setup layout | Three editing steps plus review | Separates destination, key, and method; a single conditional form is possible if users strongly prefer it. |
| Back from review | Edit preserved draft | Reverses the reproduced loss; discard becomes an explicit decision. |
| Manual result wording | Manual action required | Names the remaining work; generic completion currently overstates it. |
| Follow-up persistence | Tasks reflect existing saved intent; other manual follow-up stays in the current result | Durable tracking of every manual action would require a separate backend behavior change. |
| Scope | Interaction and reporting changes only | No new remote capabilities, persisted task schema, bulk operations, or config-only/verify-only APIs. |

The interaction direction is accepted. The six decisions above define the implementation ticket scope; the dependency graph was completed through separate implementer worktrees and fresh mergers. The implementation is ready for human review.

## Comments

- 2026-11-07: After the Japanese interaction walkthrough, the user replied 「いい感じ」. Recorded agreement to the four categories, retained drafts and review/edit, conditional setup, and separate outcome/remaining-work presentation. Existing-state Tasks and the documented backend/safety boundaries remain the scope. The specification and six tickets are ready-for-agent; no application implementation has started.

- 2026-11-08: Authorized implementation completed; product/test SHA `26269049b747db02988ad00db2b62dcff22119f0`, merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Final acceptance and platform boundaries are recorded in coverage/review/native evidence. Historical discussion comments above retain their original planning status.
