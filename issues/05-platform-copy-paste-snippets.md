Status: ready-for-agent
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

- [ ] Destination POSIX and PowerShell snippets append public keys with appropriate permissions.
- [ ] Local config snippet matches the local OS; connection is a separate step.
- [ ] Each step identifies the terminal; TUI y copies only its executable script.
- [ ] Existing data, duplicate handling, literal paths, validation and stale-copy guards are verified.
- [ ] Applicable Go checks, separate standards/spec reviews and verification records are complete.

## Comments

- 2026-11-09: User clarified that copy-paste means executable snippets for the
  corresponding terminal/platform. Started from refreshed origin/main
  `e5e826c32cb79b5267cd5ee4f327ad516ffb659c`.
