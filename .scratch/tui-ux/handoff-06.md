# Ticket06 verification/documentation handoff

Status: ready-for-human

## Source and changes

Worker `test/06-tui-journeys`, based clean on `58148f57f0ca1ad336ed210be0fabba3dd35f39d`. Latest integration `a7319738c6adf050fd632e111f90dd8069a2a5a8` is merged; final merge check returned Already up to date. Tested clean source is **`284a00bcf7078042f072804a5ea36da725a4fd11`**. Later evidence-only documentation does not change runtime/test source. Product remains identical to **`0be6ae68a2f983877fa89c21b6cc019f6e5b354d`**; only `internal/` addition is the06 journey test.

Added `TestJourneyUnregisterReconciliationUnlocksDeletionWithoutRevivingOnboarding` through public `tui.NewProgram`, reusing existing result/handoff fixtures. It connects deletion blocker -> target-bound unregister Review/Help/Edit -> external absence reconciliation -> restart without stale onboarding -> fresh-session guarded local deletion -> preserved tombstone/history. No product change, extra dependency, broad duplicate matrix or PasteMsg fix. Initial fixture EOF failures were corrected by using an actual fresh session for the second operation; they are not presented as a product red/green cycle.

README and `docs/application-contract.md` now document the implemented lists/search/target selectors, conditional forms, review/edit/cancel/help, original-terminal handoff, independent outcome facts, honest manual/restart behavior, Tasks precedence and minimum-size guards. Historical planning/specification documents are preserved. [coverage-06.md](coverage-06.md) is the complete AC01–12/test/native/untested map and local artifact manifest; [docs/verification.md](../../docs/verification.md) records the successful final checks.

## Checks

2026-11-08 JST, Windows amd64, Go1.26.2, PowerShell5.1.26100.9711, existing CPython3.12.13. All final checks exited0 in task-owned Herdr terminals:

- Full uncached Go suite using process-only policy bypass: cmd2.239s / app46.640s / cli2.564s / ssh27.857s / tui11.024s; state no test files. New journey also passed focused1.252s before commit.
- `go vet -buildvcs=false ./...` PASS.
- Portable5 tests5.376s; Installer50 tests226.021s; Profile30 tests109.056s PASS.
- Native build/version PASS `SSH KeyFlow dev`.
- Actual package build with local smoke version `v0.2.0`:6 targets/7 assets PASS. Independently checked every digest, archive members/notices, Unix modes and all target/build metadata. Extracted Windows amd64 `--version` = `SSH KeyFlow v0.2.0`, help and read-only JSON PASS without initializing isolated home. Other5 binaries were not executed natively.

Coordinator [native-verification.md](native-verification.md) remains the authoritative native record and was not edited by06. Final05 native manual/restart/tasks, Ctrl+C/denial and navigation/real resize PASS at0be6. The a731973 final navigation/resize supersedes older02/04 source checks. Actual ssh-keygen passphrase-prompt interruption is recorded separately at earlier6fdc973; final05 inspected its saved checkpoint at restart, without executing creation.

## Review and integration gates

06 Standards self-review and Specification self-review are recorded separately in the coverage file. **Independent integration Standards/Spec reviews remain pending coordinator.** The known **PasteMsg loss remains unresolved** and reserved for the single fresh post-review fixer. AC04/AC05/AC12 are partial; the ticket's final review box remains unchecked. Coordinator owns final native paste/dynamic-resize checks after that fix and the corresponding exact-SHA evidence.

Linux/macOS native, Windows arm64 execution, actual remote SSH/server behavior and elevated Windows administrator shared-file execution remain unverified. No source transfer, hosted CI, push/main merge, tag or publication occurred. Fake Runner results do not certify real SSH. Merge06 locally through the coordinator's merger, preserving these pending gates.

## Cleanup

Successful logs/checksums are captured in the coverage file. All12 own finished ux06 terminals, `.tmp/ux06-verification` and own Python bytecode cache were removed after capture. Coordinator native fixtures, terminals, worktrees and native evidence were preserved. Local Markdown links,61 coverage test names, gofmt, Comments-last and diff whitespace checks passed. The worker stays committed/clean for coordinator integration; coordinator owns eventual worktree removal after confirmed integration.
