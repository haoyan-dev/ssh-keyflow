# TUI UX implementation coordination

Integration: `implement/tui-ux` in `C:\Users\haoyan-ts\repo\ssh-keyflow-tui-ux`.
Source base: `89340aebbc0b3ab7f781b822aaafc2130ae76f73` (origin/main fetched 2026-11-07).
Planning commit: `bb20d99`.

The user invoked implement-spec and requested completion of all implementation tickets. Use a fresh implementer and separate branch/worktree per ticket. Merge through a fresh merger into integration; never edit main. The existing specification defines the public test seams and safety boundaries.

Frontier: 01 and 02 independently; 03 after 02; 04 after 01/03; 05 after 01/02/04; 06 after 02/03/04/05. Each implementer records its API/behavior handoff under `handoff-NN.md`, meaningful red/green checks, and completion evidence in the ticket. Completed work uses ready-for-human with acceptance boxes checked; the tracker has no closed label. Final standards and specification reviews are separate.

## Baseline verification

- `go test -buildvcs=false ./...` in Herdr t14 failed on unchanged source because Windows native fixture scripts could not run under the inherited default execution policy.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command 'go test -buildvcs=false ./...'` in Herdr t15 passed (exit 0). This affects only the test process and its children. No persistent execution policy was changed.
- Default Python commands resolve to Windows Store aliases. Further inventory with uv python list found an existing CPython 3.12.13 at C:\Users\haoyan-ts\AppData\Roaming\uv\python\cpython-3.12.13-windows-x86_64-none\python.exe; use this exact path for local packaging/installer checks. No install is needed.
- Native TUI verification will use disposable home directories and Herdr terminal.run initial size. No real SSH targets, credentials, user keys, or unmanaged data may be modified.

## Delivery boundary

Implement and review locally, keep integration branch for handoff. No remote push, main merge, release, or destructive real-server test is included. Remove only task-owned, clean implementer worktrees after integration; preserve the integration worktree and all user terminals/worktrees.

A proposed source transfer to connected Linux was rejected by automatic approval review because destination trust and source-transfer authorization were unverified. No transfer ran. Continue locally; do not retry remote transfer indirectly. Linux/macOS native behavior remains unverified unless separately authorized.

Local Python baseline completed using the existing CPython 3.12.13 executable above and task-owned Herdr terminals: test_portable.py -v: PASS 5 tests (3.548s); test_installers.py -v: PASS 50 tests (214.588s); test_installer_profiles.py -v: PASS 30 tests (100.078s). These ran before product changes were integrated, against unchanged application/packaging source from 89340ae (integration documentation commits 4e63567/c544197). Final implementation verification remains ticket06.

## Integrated checkpoints

- Ticket02 integrated at dce6c7c6b9c8d71c92e7ed71fdb77b2445cce5c3 by fresh merger; focused TUI tests and diff checks passed. Ticket03 started from that integration tip in its own worktree.
- Ticket01 integrated at 2501db9ca1a333f0e2dd21365cab7d072f34cb5b by fresh merger; focused Report tests passed (5.171s) and diff checks passed. Worker handoff records full uncached suite, vet, build against b9cf0b9 (same final product source).
- Coordinator native navigation smoke used the exact dce6c7c source binary built with buildvcs=false, Windows amd64, Go1.26.2, Herdr0.1.18. Disposable USERPROFILE contained20 fictitious Linux destinations with long Japanese aliases; no SSH credentials, keys, servers, or real home were used. Herdr terminal.run initial size80x24: list rendered, fixed-ID search host-17 and Enter opened the matching resolved destination. At32x12, labels remained readable, details wrapped vertically, End reached the final configuration/history text with visible Back/Exit controls. At31x11 only Resize: minimum32x12 and q quit appeared; a/Enter did not start a form or operation. This does not establish native help, dynamic resize, handoff, interruption, or final03-05 behavior;06 must verify those separately. All coordinator test terminals were exited and removed. Ignored .tmp/native-navigation holds disposable fixtures/binary for optional reuse; remove after final verification.
- Ticket03 integrated at cdd0c5b67cfd9c3df909d8ed6901079482c20569 through a fresh merger; focused TUI tests and diff checks passed. Its final worker commit is f802c7cfb592cd4ee636265510ad15dbcadd85cc, verified product33654a7240e8da7c0e6ddb0f60dc7e974d584a43.
- Ticket04 integrated at1f90c4101f525e677d5cc09b98541da948657026 through fresh merger Faraday, exact parents cdd0c5b and dd582b2d43bf7d755dbef6c71c3fc915547c6f8f. Focused `go test -buildvcs=false ./internal/tui -count=1` passed3.449s; merge tree equals worker final. Verified product6fdc973d6908fd32e4b3d73ae9272225bb753569 passes full suite/vet/native version/six cross-builds. [Native evidence](native-verification.md) records the baseline Ctrl+C failure and04 restoration, repeat denial, minimum-size input suppression and actual dynamic resize. The state hash remained unchanged, with no approval/SSH execution. All coordinator and merger test terminals were removed.
- Ticket05 starts from36e61375a651846654644b41c564fd9a54aa9543 on feat/05-tasks-results-recovery in sibling ssh-keyflow-ux-05, fresh implementer Galileo.01-04 are ready-for-human;05 owns full Tasks/results replacing the interim04 view.06 remains blocked until05 is integrated, then owns final cross-journey verification and shipped documentation; coordinator owns final native checks.

### Ticket06 implementation verification checkpoint (2026-11-08 JST)

- Worker `test/06-tui-journeys` confirmed clean base `58148f57f0ca1ad336ed210be0fabba3dd35f39d`; integration through `a7319738c6adf050fd632e111f90dd8069a2a5a8` merged, final merge check Already up to date. Tested clean source `284a00bcf7078042f072804a5ea36da725a4fd11`; later evidence-only docs do not change runtime/test source. Runtime remains frozen `0be6ae68a2f983877fa89c21b6cc019f6e5b354d`.
- Added one public NewProgram journey reusing existing fixtures: deletion blocker -> unregister absence reconciliation -> restart suppresses stale onboarding -> guarded local deletion -> retained tombstone/history. No product change or PasteMsg fix. README/application contract updated for implemented interaction; historical plans preserved.
- Full uncached Go suite PASS with process-only PowerShell policy bypass (app46.640s/ssh27.857s/tui11.024s), vet/native dev version PASS; Portable5/5.376s, Installer50/226.021s, Profile30/109.056s PASS. Local v0.2.0 smoke package:6 targets7 assets, all hashes/members/notices/Unix modes/Go target metadata PASS; extracted Windows amd64 version/help/read-only JSON PASS. No release/tag/publication.
- [coverage-06.md](coverage-06.md) maps AC01–12 to named tests, exact-source coordinator native evidence, local artifact hashes and untested boundaries. [handoff-06.md](handoff-06.md) is the merger/reviewer entry. Final05 native results/restart/Tasks, Ctrl+C/denial and final navigation/real resize are PASS at0be6; a731973 supersedes earlier02/04 navigation/resize evidence. Actual child interruption remains separately identified at earlier6fdc973. Coordinator native file is untouched.
- Ticket06 is locally ready-for-human. Independent final integration Standards/Spec reviews, material-finding resolution and the single fresh post-review PasteMsg fixer/native retest remain coordinator-owned gates. AC04/AC05/AC12 are partial; ticket06's final review box remains unchecked. Native Linux/macOS, other5 target execution, real remote SSH and elevated administrator shared-file behavior remain unverified. Do not infer them from fake Runner or cross-build success.

## Final implementation checkpoint (2026-11-08 JST)

All six dependency-ordered tickets are implemented, integrated and ready-for-human with every acceptance checkbox checked. One fresh post-review implementer resolved Standards S1 and Spec P1–P5, including the cursor-scroll issue found during recheck; separate independent final rechecks report zero unresolved material findings. Fresh merger integrated d279463 at `c155e2adf6c0b0963fb68085c9d036286f188539` with identical tree and passing focused TUI. Full tested product/test SHA is `26269049b747db02988ad00db2b62dcff22119f0`; later changes are completion evidence only.

Full uncached Go (cmd2.028s/app41.354s/cli2.343s/ssh23.251s/tui10.282s), focused TUI9.037s, vet, native dev version, actual six-target package and archive/metadata/extracted-Windows smoke PASS. Unchanged Portable5/Installer50/Profile30 PASS at284a00b remains applicable. Exact commands/package hashes are in [handoff-review-fixes.md](handoff-review-fixes.md). Current acceptance is in [coverage-06.md](coverage-06.md); original findings and separate resolutions in [final-review.md](final-review.md).

[Native verification](native-verification.md) records final paste/caret/help/resize, pending onboarding/local recovery Review, explicit result entity/task refresh, actual ssh-keygen interruption/restart at ba0f69d and final affected cursor-scroll check at2626904. No real SSH server or passphrase was used. Native platform/remote/elevated/hosted-CI limits remain explicit. The source-transfer rejection above remains applicable; no transfer occurred.

All seven task-owned implementer worktrees and branches were verified clean and fully integrated, then removed without force. Original clean main89340ae, integration implement/tui-ux and planning docs/tui-ux-redesign are preserved. Task-owned ignored native/review fixtures and finished terminals were removed after evidence capture; user terminals and remote terminals were preserved. No push/main merge/tag/release occurred.

Final artifact audit:149 local Markdown links and all ticket Comments-last positions PASS; all6 ready-for-human,0 unchecked acceptance boxes. All71 named coverage/handoff tests exist (TestJourney is a -run filter), committed Go blobs are gofmt-clean (checkout CRLF accounts for gofmt -l output), diff whitespace PASS. Completion commit changes documentation only. Exact-merge focused TUI package8.910s / elapsed10.448s, Herdr t72 removed.

## PR preparation checkpoint (2026-11-08 JST)

The user authorized proceeding with push, PR and integration after CI/review. origin/main was refreshed and remains89340ae. GitHub rejected the first implement/tui-ux push because two disposable generated reserved key IDs in native evidence matched Mailgun API-key patterns. No remote branch was created and no exemption was requested. Verified app.newID uses crypto/rand for these state identities.

The original unpublished integration history remains locally on implement/tui-ux for audit. Public delivery uses feat/tui-ux in sibling ssh-keyflow-tui-ux-pr, squashed from the latest origin/main with those two identifiers omitted from the evidence. All runtime/test, scripts, README and application-contract bytes match the tested integration; only this publication note and the evidence notation differ. Earlier full source/merge SHAs refer to retained local history, not commits published by this delivery branch. Hosted CI validates the public PR head independently. No published history is rewritten.

## PR CI fixture correction (2026-11-08 JST)

Public PR: https://github.com/haoyan-dev/ssh-keyflow/pull/4. Initial public head `55f92ae6da8116e974444369ddb3cd21afb1bad0` reproduced macOS TUI failures in push run `42391651145` and PR run `42391960644`: creation/configuration guards rejected the `/var` symlink ancestor in `t.TempDir()` paths before the intended test scenario ran. Ubuntu and crossbuild passed in the push run.

The correction resolves only the handoff/result fixture homes and interrupted-creation fixture to their physical paths with `filepath.EvalSymlinks`, matching the existing app fixture pattern. Production guards, behavior, test assertions and fake external boundaries are unchanged. Windows local `go test -count=1 ./internal/tui` PASS (9.447s), `go vet ./internal/tui` PASS, Herdr t73 exit0. The correction commit and full hosted CI results are tracked in PR #4; previous product/native evidence remains applicable, while the published test source now additionally contains this fixture correction. No release/tag publication.