Status: ready-for-human
Blocked by: none

# 05 Platform-specific terminal snippets

## Objective

Give users scripts they can copy directly into the correct local or destination
terminal, respecting its platform and Windows account type.

## Scope

Shared guide generation, TUI per-step navigation/copy, CLI guide output, regression
tests and the corresponding application/domain documentation.
Specification: `.scratch/platform-copy-paste-snippets/spec.md`.

## Acceptance criteria

- [x] Destination POSIX and PowerShell snippets append public keys with appropriate permissions.
- [x] Local config snippet matches the local OS; connection is a separate step.
- [x] Each step identifies the terminal; TUI y copies only its executable script.
- [x] Existing data, duplicate handling, literal paths, validation and stale-copy guards are verified.
- [x] Applicable Go checks, separate standards/spec reviews and verification records are complete.

## Comments

- 2026-11-09: User clarified that copy-paste means executable snippets for the
  corresponding terminal/platform. Started from refreshed origin/main
  `e5e826c32cb79b5267cd5ee4f327ad516ffb659c`.
- 2026-11-09: Implemented on `feat/05-platform-copy-paste-snippets`;
  tested source commit `e67d05d18c8ddafacaaf6a63271b11a1db0cdc3f`.
  Full `go test ./...` and `go vet ./...` passed on Windows amd64.
  Native Git Bash, Windows PowerShell 5.1 and PowerShell 7 snippet checks passed
  against temporary files. Standards and specification reviews completed separately
  with no unresolved findings. Limits and test conditions are recorded in
  `docs/verification.md`. Ready for review; no push, merge or release requested.
- 2026-11-09: Follow-up request authorizes merge to main and a version tag.
  Refreshed origin/main and tags; latest stable is v0.5.2. Selected v0.6.0
  for the backward-compatible terminal snippet feature. Release preparation
  and reusable verification evidence are recorded in `docs/verification.md`.
- 2026-11-09: PR #13 merged into main as
  `1227e27e610f750804417a8e475cdf4952c49407` after all push/PR checks passed.
  Published annotated v0.6.0 from that commit; release workflow 42496973891
  passed all five jobs. The release is stable/latest. All seven downloaded
  assets passed metadata, checksum and archive audits; native Windows amd64
  version/read-only smoke and public PowerShell install/update/uninstall passed.
  Feature branch cleaned locally and remotely. Candidate and publication
  evidence, limits and separate standards/spec reviews are in
  `docs/verification.md`; post-release records are integrated through a
  documentation PR without changing the release tag or binaries.
