# Grouped-panel implementation verification

Date: 2026-11-08 JST
Source: `fbb2c377031d9526c1a35670973a7f3c2be8cc6c`
Branch: `feat/tui-grouped-panels`
Refreshed integration base: `76d392826f757e8aa4cfdb72d2d7b5216660f0fc`
Environment: Windows amd64, Go 1.26.2, PowerShell 7.

The user selected A from the three-direction gallery. Production changes are
in `b27b6ac`; the final picker-reference correction is in `fbb2c37`.
The completion follow-up changes documentation and ticket status only.

## Automated checks

All checks below ran against the identified source commit:

| Check | Result |
| --- | --- |
| `go test -buildvcs=false ./... -count=1` | PASS |
| `go vet -buildvcs=false ./...` | PASS |
| `go build -buildvcs=false -o .build/ssh-keyflow.exe ./cmd/ssh-keyflow` | PASS |
| Native `--version` | `SSH KeyFlow dev` |
| Native `--help` | PASS |
| Formatting and `git diff --check` | PASS |

Uncached package timings: cmd 1.921s, app 46.421s, cli 2.390s, ssh 27.420s,
tui 24.972s; state has no test files.

The first full-suite attempt was blocked by Windows PowerShell's script
execution policy in existing native SSH fixtures. The final passing run set
`PSExecutionPolicyPreference=Bypass` only in its process environment, inherited
by fixture child processes. No persistent execution policy was changed.

Regression coverage includes cell widths with ANSI/CJK, complete long values,
pagination and exact selected IDs, duplicate references, compact first
viewports, mixed pair evidence, form focus/caret, overlay restoration,
canonical-source reflow, typed results, stale/read-only guards and native
handoff behavior. No dependency was added.

## Production rendering

`KEYFLOW_CAPTURE` generated 240 production frames: 40 page/scenario families,
three sizes (80×24, 60×18, 32×12), and both document ends. Each has matching
plain text and ANSI data in ignored `.tmp/tui-grouped-panels/`.
The comprehensive public-event journeys assert supported frame bounds.

Reviewed contact sheets, rendered from those actual frames:

- `production-80.png`: identity/status/action columns, mixed relationships,
  registration, task, form, review, result and session instructions.
- `production-32.png`: compact details, picker, review/discard, forms,
  history, technical, result and instructions.
- `production-60-no-color.png`: stacked layout, continuation and unregister
  pickers, access options, deletion review, discovery and help.
- `production-pickers-32-no-color.png`: four categories and distinguishing
  operation references after the final correction.

The contact viewer uses a browser only to render captured terminal text.
It is not an alternate product implementation. Color-off contact sheets strip
formatting; the native check below separately used the real `NO_COLOR` setting.

## Native terminal smoke

The final executable opened in a Windows PTY at 80×24 with process-local
`NO_COLOR=1` and `USERPROFILE` pointing to an isolated synthetic home under
`.tmp/entry-detail-ux/home`. List → unregister picker → Esc → list → quit passed.
Both registration references remained visible. An earlier native check at
`b27b6ac` also inspected the destination overview and technical panels; those
renderers were unchanged by the final picker correction.

The synthetic state SHA256 remained byte-identical before and after:
`e64f0615892799358ed2a1dbe8eb01ba45de6855f5395a8edb0372b6cb0bc576`.
No operation was submitted in these terminal checks.

Final development binary:
`.build/ssh-keyflow.exe`

SHA256:
`a06ebffa3c5a2498d9edefcda238ef8e6fe3660c65a7fff4e97e3872df82bf1c`

## Separate reviews

Standards/correctness review (Astra/Lagrange): APPROVED for `fbb2c37`, with
no unresolved blockers. Independently checked the full TUI suite (24.384s),
TUI vet, formatting and diff checks. Reviewed geometry, source reflow,
pagination/focus, exact target references, data guards and test integrity.

Specification/visual review (Astra/Aristotle): PASS for `fbb2c37`, with no
blocking findings. Reviewed source and ANSI/text captures at all three sizes,
then inspected the final 32×12 color-off picker contact sheet. The distinct
registration references close the previous visual blocker.

Both reviewers treated the full-repository checks and native terminal smoke
as implementer-reported results; they did not independently repeat those.

Resolved findings:

1. A scrolling fixture detected a filename in the input rather than the full
   value below it. Strip ANSI before checking that the top heading has left
   the viewport; delayed-reply assertions now run.
2. Long picker headings could lose the position/count in a second truncation.
   Reserve the actual panel title capacity and test paging at 32 columns.
3. Saved continuations with duplicate labels were indistinguishable.
   Preserve a visible operation reference in both entry and result pickers;
   unregister pickers use the same suffix rule. Tests assert both references
   at every size and the exact second operation ID in Review.

## Acceptance limits

This is a local development build. No push, merge, tag or publication occurred.
Native runtime resize and native 60×18/32×12 inspection were not repeated;
public WindowSize journeys and rendered production frames cover these layouts.
Other OS/CPU native runs, external SSH acceptance, hosted CI and installer/
packaging suites were not repeated. Packaging and installer sources did not
change. The existing native handoff and Windows SSH fixture tests passed.

See [local test instructions](README.md) and [the specification](spec.md).
