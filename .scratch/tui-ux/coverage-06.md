# Ticket06 coverage and local verification

Status: ready-for-human

## Current final acceptance (2026-11-08 JST)

All AC01–AC12 are implemented and accepted within the explicit local verification boundary. The older tables below retain the pre-fix checkpoint and are superseded by this section for current status. G1 paste and G2 independent review are closed; the later P5 cursor-scroll regression is also resolved.

- Tested product/tests: `26269049b747db02988ad00db2b62dcff22119f0`; final tested integration merge: `c155e2adf6c0b0963fb68085c9d036286f188539`. Later completion documentation does not change runtime/test source.
- AC01–AC03 and AC07–AC10: existing full-suite matrices PASS, plus post-review exact-target/paste/recovery regressions; P1/P3/P4 native evidence confirms saved identity and refresh behavior.
- AC04–AC05: **PASS**. `TestReviewFixPasteEditsAtCaretAndRetainsDirtyDraftThroughHelp`, `TestReviewFixPasteFiltersListAndPickerAtCaret`, hidden/fixed-control, below-minimum and dirty-draft regressions pass. Actual ba0f69d native paste/caret/Help/resize is recorded in [native-verification.md](native-verification.md).
- AC06: **PASS within boundary**. Original terminal approvals, denial and interruption/restoration are tested. Actual Windows ssh-keygen interruption and restart were rerun at ba0f69d, with no passphrase supplied. Real remote authentication/server changes remain unverified.
- AC11: **PASS on Windows plus event tests**. Existing80x24/60x18/32x12 and real31x11 resize evidence is supplemented by final2626904 natural cursor-scroll native check and `TestReviewFixScrolledFormSurvivesCursorReplies` (natural/unrelated/stale replies, actual edits/caret).
- AC12: **PASS within boundary**. Full uncached Go/vet/build, actual six-target package/metadata/checksums and extracted Windows smoke PASS at2626904; unchanged Portable5/Installer50/Profile30 PASS at284a00b remains applicable. Independent Standards and Spec rechecks show zero unresolved material findings. README and application contract describe shipped behavior.

[Final review](final-review.md) keeps the axes separate. [Review-fix handoff](handoff-review-fixes.md) records exact commands, timings, meaningful RED/GREEN and final package hashes. Final focused TUI ran again at the integration merge. All six tickets are ready-for-human with checked criteria; no closed label exists.

Native Linux/macOS/Windows arm64, the other five packaged-target executions, real remote SSH/host-key/server registration/removal, elevated Windows administrator shared targets and hosted CI remain unverified. Cross-builds and fake Runner success do not establish those behaviors. No push/main merge/release or source/binary transfer occurred.

## Historical ticket06 checkpoint: source and execution conditions

- Worker: `test/06-tui-journeys`, `C:\Users\haoyan-ts\repo\ssh-keyflow-ux-06`.
- Clean starting base: `58148f57f0ca1ad336ed210be0fabba3dd35f39d`, the then-current integration. No remote refresh was requested or performed for this dispatched worker.
- Tested clean commit: **`284a00bcf7078042f072804a5ea36da725a4fd11`**. README/application contract and the additional journey test are included. Subsequent evidence-only documentation does not change this tested runtime/test source.
- Product source remains `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`. The only `internal/` difference from that source is `internal/tui/journeys_test.go`; no product fix is included in06.
- Integration documentation through `a7319738c6adf050fd632e111f90dd8069a2a5a8` is merged. It includes `9a077d8` final result interruption and `a731973` final navigation/resize evidence. A final merge check returned Already up to date.
- Execution: **2026-11-08 JST**, Windows amd64, Go1.26.2, Windows PowerShell5.1.26100.9711, existing CPython3.12.13; native evidence uses coordinator Herdr0.1.18. Earlier coordinator native sessions are dated2026-11-07 in their own record.
- Python below means `C:\Users\haoyan-ts\AppData\Roaming\uv\python\cpython-3.12.13-windows-x86_64-none\python.exe`. Independent checks ran in local Herdr terminals with `--bg --no-parent` and unique ux06 names.
- Go tests use process-only `-ExecutionPolicy Bypass` because unchanged Windows native fixture scripts otherwise fail under the inherited policy. No persistent policy was changed. Homes, HTTP fixtures and installation/profile paths are disposable; real user keys/profiles and remote servers were not used.

## Historical ticket06 acceptance coverage

PASS below means the named automated behavior and explicitly listed native evidence passed, within those boundaries. It is not an independent final review or a claim that every platform was exercised. **AC04/AC05/AC12 remain partial because bracketed paste is a known unresolved defect.**

All TUI tests are under [internal/tui](../../internal/tui); report tests are in [reports_test.go](../../internal/app/reports_test.go). Public event journeys use `tui.NewProgram`; pure projection tests use `tui.ProjectTasks`. Runner replaces the external process boundary, not the operation implementation.

| AC | Named automated evidence, all included in the full PASS run | Native evidence and remaining condition | Status |
| --- | --- | --- | --- |
| AC01 categories/search/identity/read-only | `TestNavigationSearchKeepsQueryAndCategoryIdentity`; `TestNavigationRefreshPreservesFixedIDAndAnnouncesRemoval`; `TestNavigationReadOnlyJourneysKeepStorageAndCategoryScroll`; `TestTUIStartsAtReadOnlyListsAndQuits`; `TestTasksSearchAndCompactRecoveryControls` | N3 exact host-17 ordinary-key search, retained category query/selection,20 Japanese aliases at80x24/32x12. Paste is excluded. | PASS for named cases |
| AC02 details/visible targets/eligibility | `TestNavigationDetailsJoinRecordedEvidenceAndHelpReturnsToOrigin`; `TestNavigationActionsUseVisibleTargetAndRelatedSelection`; `TestNavigationMissingJoinFailsReadOnlyInsteadOfInventingRelationship`; `TestNavigationSelectedTargetCannotBeReplacedByLegacyIDInput`; `TestNavigationHistoryKeepsTombstonesOutOfCurrentKeys` | N1 results/tasks exact identities; N3 details/End expose full target. Backend remains authoritative. | PASS |
| AC03 selectors/fixed IDs | `TestNavigationRelatedUnregisterPickerSearchesFingerprint`; `TestFormsKeyPickerCancelRetainsDraftAndExclusiveKeyChoice`; `TestFormsReresolveFixedIDsAndRetainDraftOnChangedTargets`; `TestFormsReresolveAgainBeforeHandoff` | N1 saved task uses exact registration/creation ID in Review without execution. No real remote eligibility certified. | PASS |
| AC04 staged conditional forms/validation | `TestFormsDestinationStepValidatesAndKeepsEarlierInputs`; `TestFormsConditionalChoicesEmitOnlyApplicableRequestValues`; `TestFormsValidationNamesFirstInvalidInput`; `TestFormsHandoffEmitsChosenAuthenticationIntent`; `TestFormsLocalResumeSelectorsOmitRemoteOptions`; `TestTUIAddDestinationRequiresExplicitOSBeforeReview` | N1 local recovery has no remote options; N3 ordinary qar retained. Native bracketed paste into form/search is pending G1. | PARTIAL |
| AC05 edit/back/cancel/help/text | `TestReviewDirtyCancelKeepsEditingByDefault`; `TestReviewControlsAndHelpRestoreExactEditingPosition`; `TestReviewRepeatedEditingLiteralKeysAndCleanCancel`; `TestReviewPickerDirtyCancelRestoresSearchAndCaller`; `TestFormsEditingAndHelpRetainCursorAndDraft` | N1 Review/Help/back/cancel; N3 literal qar, Keep editing restores Help focus. Paste pending G1. | PARTIAL |
| AC06 review/original terminal/restore | `TestHandoffAcceptsContinueOnlyOnceBeforeNativeReturn`; `TestHandoffInterruptedCreationRetainsSavedCheckpointAndOriginalInput`; `TestHandoffRetainsManualFactsAndExplicitTarget` | N2 exact manual-unregister approval Ctrl+C restores live final result; same process denial returns Cancelled before changes. N4 actual ssh-keygen passphrase-prompt interruption is earlier04 source evidence, not a rerun on05. Real SSH untested. | PASS within boundary |
| AC07 additive typed outcome facts | `TestReportsRegistrationAndDestinationVerification`; `TestReportsManualRegisterPreservesPriorEvidence`; `TestReportsIndependentConfigurationApprovals`; `TestReportsUnknownRemoteAndFailedRegistrationSave`; `TestReportsConfirmationDenialBeforeEffects`; `TestReportsCreationInterruptionAndResume`; `TestReportsOnboardingSavedBeforeCreationCancellation`; `TestReportsUnregisterManualRemovalAndUnknown`; `TestReportsDeletionCheckpointsAndPartialFailure`; `TestReportsIncludeRecoveryAndLateFailure`; `TestReportsDoNotChangeOutputApprovalsOrCommandIntent` | N1/N2 exercise consumed manual/denial/unknown facts. Reporting-disabled CLI/safety tests also pass; persisted schema remains unchanged. | PASS |
| AC08 results/manual/partial/stale | `TestResultManualHasHonestSessionAndRestartFollowup`; `TestResultManualVerifiedPairPreservesPriorEvidenceAfterRestart`; `TestResultAutomaticRegistrationOutcomeMatrix`; `TestResultAliasDenialKeepsPriorBindingAndIncludeEvidence`; `TestResultRemoteUncertaintyKeepsStoppedSummaryAndExactContinuation`; `TestHandoffDenialRetainsFactsThroughStaleReloadAndRefresh`; `TestNavigationRepeatedRefreshFailureReturnsToDetailsAfterRecovery` | N1 both manual pairs/restarts, full instructions and no fabricated Task; N2 unknown versus no-effects denial. Fake verification success does not certify server access. | PASS |
| AC09 Tasks projection/precedence | `TestProjectionAC09Precedence`; `TestProjectionMultipleOnboardingReferencesKeepStableSources`; `TestProjectionIncompleteCreationUsesReferencingOnboarding`; `TestProjectionConfigurationUsesOnlyUnambiguousSavedParent`; `TestProjectionContradictoryOnboardingLinksNeverGuessResume`; `TestTasksGroupSavedOnboardingAndContinueExactRegistration` | N1 unknown registration32x12 and genuine pending local creation60x18, exact supported continuations; manual restart Tasks empty. | PASS |
| AC10 unregister/deletion safeguards | `TestJourneyUnregisterReconciliationUnlocksDeletionWithoutRevivingOnboarding`; `TestResultManualUnregisterContinuationNeverOffersRegisterAgain`; `TestTasksPendingDeletionSurvivesIntegrityWarningAndUsesLocalGuards`; shared app/CLI safety regression suite | N2 denied/interrupted native unregister leaves fixture bytes unchanged. New06 journey proves fake external absence reconciliation, guarded local deletion and preserved tombstone/history. No real SSH removal performed. | PASS within boundary |
| AC11 size/accessibility | `TestNavigationLayoutsAndSmallWindowMutationGuard`; `TestNavigationFullValuesRemainReachableByVerticalScroll`; `TestFormsLongValuesAndErrorsAreVerticallyAccessible`; `TestFormsCompactValidationAndUnregisterGuidance`; `TestReviewResizeRetainsDraftAndPreventsContinueBelowMinimum`; `TestResultCompactSizesAndUndersizeGuard`; `TestTasksSearchAndCompactRecoveryControls` | N1/N3 cover80x24/60x18/32x12 and real31x11->60x18 restoration, labels/controls, Unicode, scroll, guarded input and unchanged state. Linux/macOS native untested. | PASS on Windows + event tests |
| AC12 regression/docs/native | Full commands below; public journey/report suites; README and application contract now describe implemented navigation/forms/results/Tasks, separate registration/verification/configuration and known limits. Historical planning preserved. | N1–N4 explicitly scoped. G1 paste fix/native retest and G2 independent final Standards/Spec reviews remain coordinator-owned. | PARTIAL |

## Required cross-journey cases

| Required situation | Evidence and result |
| --- | --- |
| Manual Register with no prior intent | `TestResultManualHasHonestSessionAndRestartFollowup`, N1: Manual action required / No saved recovery task / Register again after exit/restart; restart does not invent a Task. PASS. |
| Manual Register with previously verified pair | `TestResultManualVerifiedPairPreservesPriorEvidenceAfterRestart`, N1: prior verification labelled recorded, current verification not checked; restart uses saved evidence only. PASS. |
| Unknown remote outcome | `TestResultRemoteUncertaintyKeepsStoppedSummaryAndExactContinuation`, N1/N2: unknown saved registration resumes exact ID; interruption retains diagnostics. PASS. |
| Registered but specified-key verification failed/public-only unavailable | `TestResultAutomaticRegistrationOutcomeMatrix` includes verify-failed and public-only; `TestReportsRegistrationAndDestinationVerification` distinguishes saved registration from verification. PASS under fake Runner. |
| Alias denial versus Include denial/failure | `TestResultAutomaticRegistrationOutcomeMatrix` covers alias-declined/include-declined/include-failed/configured; `TestResultAliasDenialKeepsPriorBindingAndIncludeEvidence` and `TestReportsIndependentConfigurationApprovals` preserve distinct prior/saved evidence. PASS. |
| Stale reload and recovery | `TestHandoffDenialRetainsFactsThroughStaleReloadAndRefresh`, `TestNavigationRepeatedRefreshFailureReturnsToDetailsAfterRecovery`: facts remain, mutation disabled, refresh does not execute. PASS. |
| Deletion blocker, unregister reconciliation, deletion and history | New06 `TestJourneyUnregisterReconciliationUnlocksDeletionWithoutRevivingOnboarding`: recorded registration first blocks deletion; Review/Help/Edit execute nothing; external absence reconciliation completes same unregister ID, preserves key/onboarding; restart Tasks empty; result links exact registration; fresh session deletes exact local key without auth/external execution; tombstone/history and registration remain. PASS. |
| Dedicated pending deletion despite integrity warning | `TestTasksPendingDeletionSurvivesIntegrityWarningAndUsesLocalGuards`: dedicated recovery remains available but local guards remain authoritative. PASS. |
| Stale onboarding after unregister | New06 journey plus `TestResultManualUnregisterContinuationNeverOffersRegisterAgain` and `TestProjectionAC09Precedence`: removed pair has no re-registration prompt; historical onboarding is retained. PASS. |

06 added one missing connected journey, reusing existing result/handoff helpers. Existing broad outcome/projection matrices were reused. Initial attempts failed because the test's extra confirmation bytes were consumed after terminal return; the fixture was corrected to open a fresh TUI session for deletion, as the restart scenario requires. This was a test-harness failure, not a product red/green claim. No new product behavior was implemented in06, and the reserved PasteMsg behavior was neither changed nor marked passing.

## Native evidence index

Authoritative sequences, fixture hashes and terminal cleanup are in coordinator-owned [native-verification.md](native-verification.md);06 does not edit that file.

- **N1:** Ticket05 native results, restart and saved-task continuation, product `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`: manual unrecorded and prior-verified pairs80x24, refusal/re-entry, independently restarted empty Tasks, instructions End/Home; unknown registration32x12; genuine interrupted local creation60x18. No SSH command/server contacted.
- **N2:** Ticket05 native interruption with final result renderer, same0be6 source; evidence appended in docs-only `9a077d8`: original manual-unregister approval interrupted with Ctrl+C, live TUI unknown result/EOF; same-process retry/no -> Cancelled before changes; q exit0; unchanged state hash. No approval or remote operation.
- **N3:** Ticket05 final navigation and dynamic resize, same0be6 source; evidence appended in docs-only `a731973`:20 long Japanese aliases, exact host-17/alice@build-17.example:22;80x24/32x12 End and retained query/selection; ordinary qar/Create/Help/Review/Edit/Keep; actual31x11 guard ignores Ctrl+S/Enter/a,60x18 restores Help focus/qar, discard and q exit0/hash unchanged. This supersedes earlier02 navigation and04 resize source checks for the final product.
- **N4:** Actual ssh-keygen child interruption, **earlier** `6fdc973d6908fd32e4b3d73ae9272225bb753569`: real Windows OpenSSH passphrase prompt interrupted without supplying any passphrase/empty passphrase; TUI restored, pending creation saved, no keyfiles and no lock retained. N1 later verifies this genuine checkpoint after restart at0be6. N4 is not a final-source child rerun and does not prove remote SSH authentication.

## Commands and results on clean 284a00b

All final commands below exited0. Times are program-reported, except package/build/inspection elapsed time is not used as a claim.

| Check | Command | Result |
| --- | --- | --- |
| Focused new journey | `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./internal/tui -run TestJourney -count=1 -v'` | PASS1.252s in t52 before commit; identical test source included in full run at284a00b. |
| Full Go | `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./... -count=1'` | PASS t53: cmd2.239s, app46.640s, cli2.564s, ssh27.857s, tui11.024s; state has no test files. |
| Vet | `go vet -buildvcs=false ./...` | PASS t54. |
| Portable contract | `python scripts/test_portable.py -v` | PASS5 tests,5.376s, t55. |
| Installers | `python scripts/test_installers.py -v` | PASS50 tests,226.021s, t56. Local HTTP/temporary targets; OS-specific fixture behavior is not native Linux/macOS execution. |
| Installer profiles | `python scripts/test_installer_profiles.py -v` | PASS30 tests,109.056s, t57. |
| Native dev build | `go build -buildvcs=false -o .tmp/ux06-verification/ssh-keyflow.exe ./cmd/ssh-keyflow`, then `--version` | PASS t59: `SSH KeyFlow dev`. |
| Actual cross-build | `python scripts/package_release.py --version v0.2.0 --output .tmp/ux06-verification/package` | PASS t58: six CGO-disabled target builds, six archives plus SHA256SUMS. Local smoke version only; no tag or publication. |
| Artifact inspection | Local assert-based scratch inspector, t5c | PASS six manifests/archives/build metadata; extracted Windows amd64 version/help/read-only JSON. Details below. |

## Local artifact evidence

Six targets: Linux/macOS/Windows × amd64/arm64. Packaging source invokes `go build -buildvcs=false -trimpath -ldflags '-s -w -X main.version=v0.2.0'` with CGO disabled. Inspection checked exactly7 assets, manifest6 unique lowercase SHA256 entries/two-space basename/LF/no CR, and independently recomputed every archive digest. Each archive contains exactly its regular executable and source-identical `THIRD_PARTY_NOTICES.txt`; no extra files. Unix executable0755/notices0644 and uid/gid0 passed. PE/ELF/Mach-O signatures and `go version -m` GOOS/GOARCH/CGO_ENABLED=0/trimpath metadata matched all6 targets.

Extracted Windows amd64 returned `SSH KeyFlow v0.2.0`; `--help` showed Usage, and `keys list --json` returned empty keys/deletions/deleted_keys in an isolated HOME/USERPROFILE without creating `.ssh`. The other five binaries were cross-built and inspected, **not executed natively**. Go metadata with trimpath does not expose the linker version flag here; embedded version execution is therefore certified only for Windows amd64. The first inspector's assumption about ldflags metadata was corrected, and the final inspector passed; packaging was not changed.

SHA256 of local `SHA256SUMS`: `f9ae24ff68870d1a37c02a2f9b49b231872bd53fd0008a28f329cf9304d93804`.

```text
6152bf7d7237fee71d49f01eb6af771aacb04556f1498d544c37e21ce4b1e890  ssh-keyflow_v0.2.0_darwin_amd64.tar.gz
f2a7865a1950caca1239db7ef4240d6d372e80de8e8b46975c65e5e3a841f870  ssh-keyflow_v0.2.0_darwin_arm64.tar.gz
13ee8725bc51df6d0fb837509a013cf9df44c88d820d772169dc26648bf8db7a  ssh-keyflow_v0.2.0_linux_amd64.tar.gz
b0d498e0c027c34b1e9eef0333b545577b8af55b323902b57b806279a0196dd9  ssh-keyflow_v0.2.0_linux_arm64.tar.gz
31b03b51ca0ba94c39b3bb936411f79f46f74d50de254dfeb10d66bf157e2ea3  ssh-keyflow_v0.2.0_windows_amd64.zip
d886a2dac0caf5530c0255f1076224cc475694c8edb44419055fb9f607a79c6f  ssh-keyflow_v0.2.0_windows_arm64.zip
```

These hashes identify this local test build, not published assets or a reproducibility promise; archive timestamps may differ in another build. After capturing this evidence, all12 own finished ux06 terminals (including earlier fixture/inspector failures), `.tmp/ux06-verification` and own `scripts/__pycache__/test_installers.cpython-312.pyc` were removed. Coordinator native fixtures/terminals were preserved. Local Markdown links, all61 named coverage tests, journey gofmt, ticket Comments-last and diff whitespace checks passed. Only documentation changed after284a00b, so the passing implementation checks were not repeated for evidence-only edits.

## Historical separate review checkpoints and remaining gates

### Standards self-review,06 scope

Checked public test seams, reuse of prior fixtures, no dependency/schema/product changes, focused branch ancestry, Japanese user documentation/domain vocabulary, exact-SHA verification, no credentials or real-server fixtures, Comments last in ticket, preserved historical planning, and no push/tag/publication. Formatting and local Markdown link checks are part of the final documentation check. This is implementer self-review; independent integration Standards review remains G2.

### Specification self-review,06 scope

Mapped every AC and required scenario above to named behavior evidence; reviewed README/application contract against implemented actions, forms, result facts, Tasks and size behavior. New connected journey fills blocker-to-unregister-to-local-delete/history coverage without duplicating prior matrices. Runtime documentation explicitly distinguishes saved registration from connection/configuration evidence and notes pending paste/verification limits. No material06 product change was needed. G1/G2 prevent claiming whole-spec final acceptance.

### Coordinator-owned pending work

- **G1 — known defect:** bracketed `PasteMsg` is lost in form/search routing. Fix only in the single fresh post-review fixer. Add meaningful public input regression coverage, rerun affected checks and coordinator final native paste/dynamic-resize checks on that exact fixed source. AC04/AC05/AC12 are not fully accepted at284a00b.
- **G2 — independent review:** separate final integration Standards and Specification reviews, followed by resolution and verification of material findings, remain pending coordinator. The ticket's final review acceptance box remains unchecked until then.
- Native Linux/macOS, Windows arm64 execution, real remote SSH authentication/host-key handshakes/server registration/removal, and elevated Windows administrator shared-file execution remain unverified. Fake Runner/fixture success cannot certify them. No source was transferred to ts-dev-3060 or another remote host.
- No hosted CI, PR push, main merge, release tag or publication was performed by06. Coordinator/merger owns integration and final gate closure.06 is locally ready-for-human, not an all-AC approval.
