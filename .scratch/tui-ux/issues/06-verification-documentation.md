Status: ready-for-human
Blocked by: 02, 03, 04, 05

# 06: Verify complete journeys and document the shipped interaction contract

## Problem and intended behavior

Individual component tests cannot establish that selection, editing, terminal handoff, reporting, and recovery form a usable whole. Verify the integrated implementation at a named commit and update user documentation only when the behavior exists.

Specification: [AC11 and AC12](../spec.md#ac11-terminal-size-and-accessibility), cross-journey AC01-AC10. Baseline: [planning verification](../verification.md).

## Scope and seams

- Run implementation acceptance journeys through public TUI entry plus fake Runner/confirmation fixtures, avoiding test suites that merely mirror internals.
- Native checks at 80x24, 60x18, 32x12 and below minimum, including long/Unicode aliases, fingerprints/paths, many rows, wrapped errors/instructions, focus/scroll, dynamic resize, and no color-only meaning.
- Exercise review-edit-cancel-help and native confirmation/interruption/restore. Separate fake SSH evidence from actual native OpenSSH behavior and explicitly record untested platforms.
- Run README's applicable build/test/vet requirements against the final source commit. Recheck affected work if that commit changes.
- Update README help and `docs/application-contract.md` for shipped behavior, preserving historical specs. Record verification SHA/environment/results, review findings/resolutions, and limitations.

## Acceptance criteria

- [x] AC01-AC12 coverage is mapped to tests/native checks with pass/fail/untested conditions, not assumed from individual ticket completion.
- [x] AC11 focused content/controls remain usable at the stated sizes. Below-minimum mode retains drafts and prevents starting mutation; resize restores focus without execution.
- [x] Manual, unknown, failed verification, partial cancellation, stale reload, deletion blocker/recovery, and stale onboarding after unregister are covered end to end. Include manual Register with unrecorded/already verified pair and restart, plus alias-generation denial versus Include denial/failure with saved alias preserved.
- [x] No real credentials/servers are used in unattended fixtures; destructive/remote native checks require a separately authorized disposable setup.
- [x] Existing CLI/safety contract checks pass at the final implementation commit. Linux/macOS native behavior is tested or explicitly left unverified.
- [x] Standards and specification compliance are reviewed separately, material findings resolved, documentation matches actual behavior, and verification is tied to the tested SHA.

## Verification and completion evidence

Record commands/results and native reproduction sequences with environment/platform. Do not add redundant unit tests for cosmetic details. Use broader tests only for changed behavior or unresolved concerns. A passing fake Runner test cannot certify real SSH or platform terminal behavior. No PR push, merge, or release is authorized by this ticket; follow repository authorization rules.

## Handoff

Starts on the integrated outcome/navigation/forms/results state after 02-05. Return a concrete coverage map and tested commit for implementation review; required checks/reviews must pass before requesting merge.

## Comments

- 2026-11-07: Draft implementation verification ticket. Planning artifact checks are recorded separately and do not satisfy these implementation acceptance boxes.
- 2026-11-07: Following the user's 「いい感じ」 on the interaction walkthrough, the design direction is accepted and this ticket is ready-for-agent. Earlier proposal comments record the pre-acceptance state. Honor Blocked by before starting; implementation criteria remain unchecked and no implementation has started.

- 2026-11-08:06 implementation/tests/docs are locally ready-for-human at tested source `284a00bcf7078042f072804a5ea36da725a4fd11`; latest integration `a731973` merged. [Coverage](../coverage-06.md) maps every AC to named tests/native evidence/limits; [handoff](../handoff-06.md) records source, environment, all checks and artifact hashes. Full Go/vet/native build, Portable5/Installer50/Profile30 and actual6-target7-asset package inspection PASS. Windows final05 navigation/resize supersedes earlier02/04 source checks; no native file was edited by06. The final review acceptance box intentionally remains unchecked: independent Standards/Spec integration reviews and reserved post-review PasteMsg fix/native retest are coordinator-owned and pending; AC04/AC05/AC12 are partial. No all-spec completion, real SSH, Linux/macOS native, push/main merge or release is claimed.

- 2026-11-08 final integration: all applicable acceptance criteria complete; ready-for-human. Product/tests `26269049b747db02988ad00db2b62dcff22119f0`, tested merge `c155e2adf6c0b0963fb68085c9d036286f188539`. Independent Standards and Spec rechecks have zero unresolved material findings; [coverage](../coverage-06.md), [review](../final-review.md), [native evidence](../native-verification.md) and [fix handoff](../handoff-review-fixes.md) preserve exact conditions and platform limits. Earlier pending-gate comments are historical. No push/main merge/release.
