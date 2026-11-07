# Local detail colors and panels: verification

Date: 2026-11-08 JST. Product/test source:
`6fb51e850ef39fa76afeb4024df37e2dfab87246`, local branch
`feat/entry-detail-ux`. Reused the existing task branch at `762acdc`;
original refreshed main base remains `76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.
No integration/merge or remote refresh was needed for this same-task follow-up.
Source working tree was clean during final verification. Subsequent changes
only record these results and complete the local ticket.

## Automated verification

Windows 11 amd64, PowerShell 7, Go 1.26.2. Workspace-local GOCACHE:
`.tmp/go-cache`. Existing native fixtures required execution outside the
filesystem sandbox and process-only `PSExecutionPolicyPreference=Bypass`.
No persistent policy changed. The sandbox-only initial TUI run failed at
native fixture access, then the authorized full run passed.

| Check on the exact source above | Result |
| --- | --- |
| `go test -buildvcs=false ./... -count=1` | PASS: cmd 3.892s, app 47.215s, cli 2.799s, ssh 25.705s, tui 13.790s; state has no test files |
| `go vet -buildvcs=false ./...` | PASS |
| `go build -buildvcs=false -o .tmp/entry-detail-ux/ssh-keyflow.exe ./cmd/ssh-keyflow` | PASS |
| Native executable `--version` | `SSH KeyFlow dev` |
| `git diff --check` and Go formatting | PASS |
| Local Markdown artifact links | PASS: all nine entry-detail artifacts resolve their relative references |
| Separate local standards/specification review passes | No unresolved findings; [review](panel-final-review.md) |

Final native executable SHA256:
`55661fba7db5cc71b624f5a12ac715ee7a26a986de26d725f6a3ed03e18f19d4`.

Public NewProgram journeys test all four overview/technical kinds at 80x24,
60x18 and 32x12. Added checks assert green registered/verified-previously,
yellow pending/unknown/unchecked and red failed/blocker evidence, neutral
removal and label legibility after stripping ANSI. The resize matrix covers
100→60, 80→59, 80→32 and 32→80 for all four kinds, directly and via Help and
technical document origins. Exactly two panels are present at >=60 columns,
none below. Complete wrapped Unicode names, frame bounds, scrolled origin
restoration and byte-identical saved state are checked. The rejecting Runner
asserts that navigation performs no external commands or dependency probes.
Existing refresh/action-safety and full-value journeys also pass.

## Native Windows console

Used the existing isolated `.tmp/entry-detail-ux/home` fixture, containing
synthetic public-only keys and `.test` destinations. No operation was
submitted. Only the launched process's USERPROFILE was redirected; real user
SSH data was untouched. Set console dimensions with Console.SetWindowSize
and read them back before each launch.

On the final source executable:

- 80x24: destination identity, both panels, green registration, yellow unknown
  evidence, red failed connection and complete technical disclosure.
- 60x18: key overview, bordered panels, wrapped action text, PgDown,
  red local-deletion blocker and related destination evidence.
- 32x12: registration overview with plain headings; resumable Task overview,
  PgDown to yellow follow-up evidence, technical details, Help, Esc to
  technical details, Esc to the same scrolled overview, clean quit.

Color inspection removed inherited NO_COLOR and set TERM=xterm-256color and
COLORTERM=truecolor only in the launched PowerShell process. An earlier
exploratory build also rendered the destination panels with inherited
NO_COLOR=1 / TERM=dumb: readable labels and borders remained, with styling
removed by the existing renderer. No forced color profile is added to product
code. Automated ANSI-stripped label checks run on the final source.

Fixture state SHA256 before and after every native session:
`e64f0615892799358ed2a1dbe8eb01ba45de6855f5395a8edb0372b6cb0bc576`.
All interactive sessions exited successfully. Ignored fixture/build artifacts
are retained under `.tmp/entry-detail-ux` for local review.

## Limits

Resizing through overlays is covered by automated event journeys; native
inspection used fixed console sizes for each session. Linux/macOS/Windows
arm64 native behavior, live SSH authentication or remote mutation, elevated
Windows shared administrator destinations, hosted CI and packaging/installer
suites were not exercised for this presentation-only change. No dependencies,
state schema, CLI or backend execution code changed. No push, merge, tag or
release occurred.
