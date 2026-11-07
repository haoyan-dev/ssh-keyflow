# Entry detail redesign verification

Date: 2026-11-08 JST.
Branch: `feat/entry-detail-ux`.
Refreshed base: `76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.
Final product and test source:
`73fb603a2760207845bac607a8d2dd7e8b86423e`.
The subsequent completion commit changes evidence and ticket status only.

Environment: Windows 11 amd64, NT 10.0.26300.0, Go 1.26.2 windows/amd64,
PowerShell 7 execution shell. `PSExecutionPolicyPreference=Bypass` is set only
in test processes so Windows PowerShell fixtures can run. No persistent policy
is changed. Native permission fixtures require execution outside the sandbox.
Go build cache is isolated at `.tmp/go-cache`.

## Final checks

The working tree was clean on the exact source commit for all final checks.

| Check | Result |
| --- | --- |
| `go test -buildvcs=false ./... -count=1` | PASS: cmd 1.722s; app 41.832s; cli 2.202s; ssh 22.301s; tui 10.972s; state has no test files |
| `go vet -buildvcs=false ./...` | PASS |
| `go build -buildvcs=false -o .tmp/entry-detail-ux/ssh-keyflow.exe ./cmd/ssh-keyflow` | PASS |
| Native executable `--version` | `SSH KeyFlow dev` |
| `git diff --check HEAD` | PASS |
| Standards review | No unresolved findings; separate pass in [final-review.md](final-review.md) |
| Specification review | AC01–AC07 covered; P01–P03 resolved in [final-review.md](final-review.md) |

Final native development binary SHA256:
`3c9d69822dbb3cbcfc0796cafc76b0ddd7aa87f0e3402cc8fef7fc343d92bb15`.

The public-event tests use `NewProgram` and a rejecting external Runner.
They verify independent saved status labels, configuration bound to another
key, public-only limits, empty states, contextual actions, read-only Task
guidance, technical disclosure, technical action rejection, Help/back
viewport restoration and no navigation writes. All four entry kinds and
their technical pages fit 80x24, 60x18 and 32x12. Existing tests retain long
Unicode values, complete vertical path/fingerprint accessibility, selection,
filters, refresh, below-minimum guards and result-linked journeys.

## Native console inspection

The fixture is isolated under `.tmp/entry-detail-ux/home`, with two synthetic
destinations, three synthetic public-only keys and registered/unknown/removed
registration records. Public blobs contain fixture bytes; no private key or
real server is used. Only the launched process's `USERPROFILE` is redirected.
The fixture generator and development build are retained as ignored local
review artifacts under `.tmp/entry-detail-ux`.

Native console dimensions were set with `[Console]::SetWindowSize` and read
back before launch. No browser emulator or renderer-only test is represented
as native execution.

- Source `f6b6427c984146fd897342fe03e783c633372b8d`: inspected destination
  overview and technical document at 80x24; key overview, deletion blocker,
  relationship evidence and PgDown at 60x18; registration overview, corrected
  category header and resumable Task at 32x12.
- That inspection exposed the Task footer splitting “scroll” at 32 columns.
  Source `73fb603a2760207845bac607a8d2dd7e8b86423e` changes that compact
  Task footer plus its regression assertion and review evidence. Rebuilt and
  repeated native 32x12 Task overview, `v` technical details, Help, Esc back
  to technical details and clean quit. Continue/technical, scrolling/Help and
  back/quit controls now occupy separate complete lines. The 80x24/60x18
  production paths from the preceding source are unchanged.
- Earlier exploratory sessions additionally exercised destination Help/back,
  overview scroll restoration, technical disclosure, category switching and
  key relationship scrolling. Final automated checks cover these paths on
  the exact final source.

Fixture state SHA256 before and after all native sessions:
`e64f0615892799358ed2a1dbe8eb01ba45de6855f5395a8edb0372b6cb0bc576`.
All interactive sessions exited successfully; no fixture operation was
submitted, and no real user SSH data was touched.

## Verification limits

Linux/macOS native TUI, Windows arm64, live SSH authentication, server-side
registration/unregister and elevated Windows shared administrator destinations
were not exercised. Hosted CI and packaging/installer suites were not run for
this TUI-only change. No packaging, installer, state schema, CLI or backend
execution code changed. No push, PR publication, merge, tag or release occurred.

The initial full run required process-only PowerShell policy for the existing
native fixtures. Subsequent test failures revealed the wrapped-value assertion
and 33-column category header described in the final review; these were fixed
and the complete final suite rerun successfully.
