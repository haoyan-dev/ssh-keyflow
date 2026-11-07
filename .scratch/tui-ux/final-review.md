# Independent implementation review

Reviewed source: `7dcb8025de0e6a68d356749847d29fc200950389`.
Fixed point: `89340aebbc0b3ab7f781b822aaafc2130ae76f73` (main fetched at task start).
Commands: `git diff 89340aebbc0b3ab7f781b822aaafc2130ae76f73...HEAD` and `git log 89340aebbc0b3ab7f781b822aaafc2130ae76f73..HEAD --oneline`. Full nonempty 510,054-byte diff and commit list were captured before independent fresh-context parallel reviews on 2026-11-08 JST. Both reviewers remained read-only; neither reran suites or used real keys/servers. The axes are kept separate below. The findings below are the original review checkpoint. All resolutions and final rechecks are recorded separately at the end of this file.

## Standards

No documented-standard violations found across implementation01–06, including reporting, TUI, tests and documentation. Standards sources: AGENTS.md, README development/release contract, docs/application-contract.md, GLOSSARY.md, docs/agents/domain.md, issue-tracker.md and triage-labels.md; no CONTRIBUTING/CODING_STANDARDS/docs/adr exists. Fowler smell heuristics were reviewed with repository rules taking precedence and tooling-enforced checks excluded.

- **S1 / P3 — Possible Repeated Switches (judgement, not a documented violation).** `internal/tui/forms.go:115`, also353/394. Field construction, validation and request construction independently enumerate add/register/unregister/remote resume to decide whether authentication/configuration options apply. A missed update could make displayed controls, validation and emitted request values disagree. Extract one shared eligibility predicate, retaining Add step-two visibility at the rendering call site. Existing conditional-form/local-resume journeys verify the refactor. No current behavior failure is claimed.

## Spec

- **P1 — Interrupted destination setup cannot resume.** AC09: “If key creation incomplete, choose onboarding continuation.” `internal/tui/targets.go:84` requires the onboarding reserved KeyID to already exist in managed Keys. Interrupted creation legitimately has no managed key yet, accepted by `internal/state/state.go:405`. Tasks offers continuation, but Ctrl+S rejects it before Review. Validate the linked incomplete Creation/reserved identity without requiring an already-created key; preserve normal key checks and backend guards.
- **P2 — Bracketed paste disappears from forms and search.** AC04: “Preserve valid entries.” AC05: “Text-input keys, including `q`, `?`, and action letters, remain text.” `internal/tui/tui.go:122` routes only KeyPressMsg through form/search, so PasteMsg falls through to list/viewport. Native evidence reproduces empty pasted Management name and lost host-17 search. Route paste to the active input and update draft/filter state with execution/size guards; test native paths.
- **P3 — Recovery Review omits known target identity.** AC06: “Review displays action, resolved destination/login, key name/full fingerprint or new-key intent.” `internal/tui/actions.go:200` resolves resume identity only via Registrations. Creation/deletion resumes reach `forms.go:448` with opaque operation ID/generic text and omit creation intent or deletion key name/fingerprint. Resolve review context from every supported operation type and display applicable identity/effects.
- **P4 — Refresh drops existing result-linked details.** AC01: “A refresh preserves the selected ID if present.” Result→View entity can open registration details while originating tab is Destinations. `internal/tui/navigation.go:344` searches only that tab, so F5 returns to list and clears result origins despite unchanged state; View task also mismatches. Refresh by explicit target kind/ID while retaining origin/scroll.

No concrete scope-creep finding. All Spec findings are source-traced; native evidence additionally supports P2.

Standards:1 judgement finding, worst S1 (P3); Spec:4 correctness findings, interrupted onboarding continuation is the most consequential within that axis. No ranking across axes.

## Resolution record

The original pending gates were resolved as follows. Native Linux/macOS and real remote SSH remain outside local verification.

### Standards resolution and independent recheck

Hilbert independently rechecked S1 at `ba0f69d81c3345ed2bd78062521bcd63adda7c69`: `usesRemoteOptions()` is shared by rendering, validation and request construction; Add step-two visibility is retained. S1 is resolved; no new material Standards findings. The final Standards-only recheck reviewed `ba0f69d..26269049b747db02988ad00db2b62dcff22119f0` at exact product/test SHA `26269049b747db02988ad00db2b62dcff22119f0`. The Value/Position condition and public delayed-response regression introduce no material documented-standard or heuristic concern. S1 remains resolved. The reviewer was read-only and did not duplicate tests/native checks.

### Specification resolution and independent recheck

Epicurus independently rechecked P1–P4 at ba0f69d: valid linked incomplete creation can reach onboarding Review before a managed key exists; paste and input replies reach the active form/search/picker with generation guards; recovery Review resolves all supported saved identities/effects; Refresh preserves explicit result-linked entity/task identity, origin and scroll. All four are resolved.

That recheck found **P5 / AC11**: cursor-only replies reached an unconditional `GotoTop()` in forms.go and reset a scrolled long value. The same fresh post-review implementer fixed it in `26269049b747db02988ad00db2b62dcff22119f0`; no additional fixer was introduced. Value or caret-position changes now reveal the field, while cursor/unrelated replies preserve scroll. Meaningful RED/GREEN and opaque natural/stale reply regression evidence are in [handoff-review-fixes.md](handoff-review-fixes.md).

Epicurus's final Spec-only source/test recheck at `26269049b747db02988ad00db2b62dcff22119f0` concluded **P5 resolved, no further material Spec findings**. Tab/Shift+Tab and page navigation retain their paths; old-generation replies remain rejected; P1–P4 remain resolved. No concrete scope creep was found. This independent inspection is distinguished from the worker's executed suites and coordinator's native checks.

### Final integration and verification

One fresh post-review implementer fixed S1 and P1–P5; fresh mergers integrated both batches. Final merge `c155e2adf6c0b0963fb68085c9d036286f188539` has the same tree as worker handoff `d279463bdb41d0e8cd2184b4259880e222ac2473`. Full uncached Go, focused TUI, vet, native build/version, actual six-target packaging and archive/extracted-Windows checks passed at `26269049b747db02988ad00db2b62dcff22119f0`; focused TUI passed again on the exact merge: package8.910s / elapsed10.448s, 2026-11-08 01:13:11.764–01:13:22.216 JST, Windows amd64 / Go1.26.2 / process-only PowerShell policy bypass in Herdr t72 (removed). [Native evidence](native-verification.md) records paste/caret/help/real resize, saved recovery Review, result-linked refresh, actual ssh-keygen interruption/restart at ba0f69d, and affected P5 scroll at2626904. The P5 production delta is only conditional viewport reset; unchanged native paths retain their explicitly pinned prior evidence.

Final unresolved material findings: **Standards 0; Spec 0**. Execution boundaries are listed in [coverage-06.md](coverage-06.md).
