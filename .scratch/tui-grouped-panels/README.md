# A — Grouped panels in the native TUI

User decision: A, applied across the application.
Workflow: review → [spec](spec.md) → [tickets](issues) → implementation.

From the repository root, run the local development build in a real terminal:

```powershell
.\.build\ssh-keyflow.exe tui
```

This uses your normal saved SSH KeyFlow records. Start with Destinations,
press Enter on a destination, then try Keys, Registrations and Tasks with Tab.
Use `v` for the complete technical record and Esc to return. Arrow keys and
PgUp/PgDn scroll documents. Forms lead to Review before terminal execution.
The first detail viewport summarizes saved evidence; it does not run SSH.

Use an 80×24 or larger terminal to see status and next step side by side.
At 60×18 and 32×12 the panels stack with a concise summary first. To try the
same layout without color in PowerShell:

```powershell
$env:NO_COLOR = '1'
.\.build\ssh-keyflow.exe tui
Remove-Item Env:NO_COLOR
```

To rebuild:

```powershell
go build -buildvcs=false -o .build/ssh-keyflow.exe ./cmd/ssh-keyflow
```

The default embedded version is `dev`. Installation and release artifacts are
separate from this local build.

The original placeholder comparison is still available with
`node .scratch/tui-concepts/serve.cjs`, then
`http://127.0.0.1:4318/prototype.html?variant=A`.

Production frame evidence can be regenerated with:

```powershell
$env:KEYFLOW_CAPTURE = Join-Path (Get-Location) '.tmp/tui-grouped-panels'
go test ./internal/tui -count=1
Remove-Item Env:KEYFLOW_CAPTURE
```

Captures come from real model views reached through public keyboard events.
They cover three sizes and both ends of scrollable pages, including forms,
pickers, technical records, results and session instructions. These fixtures
use temporary saved data and injected runners. See [verification](verification.md)
for the tested source revision, review outcomes and native acceptance limits.
