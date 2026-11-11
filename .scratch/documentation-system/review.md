# Documentation system review

Reviewed source: `76dc5b2826468c0c744be18f49bc2e852115979c`.
Comparison: `git diff 1d2f44eb0449b389629e01fe71b970f715badd47...76dc5b2826468c0c744be18f49bc2e852115979c`.
These are separate self-review passes; PR review remains pending.

## Standards compliance

Sources: `AGENTS.md`, `docs/agents/{domain,issue-tracker,triage-labels}.md`,
`GLOSSARY.md`, existing ADRs, and the new contributor/documentation guidance.

- Work is isolated on `docs/project-documentation-system` from freshly fetched
  `origin/main`. No unrelated working-tree changes existed.
- Japanese user guidance and domain terms are retained. The contract, glossary,
  ADRs, publication policy, and historical verification remain at their paths.
- Research, spec, numbered local tickets, triage vocabulary, and dependencies
  follow the local tracker convention.
- Review finding, resolved before the reviewed commit: the inherited CLI example
  block removed the destination before showing its guide. Examples are now grouped
  by task and explicitly described as independent operations.
- Review finding, resolved before the reviewed commit: HTML anchors inside fenced
  examples must not satisfy fragment links. The checker excludes them and a
  regression test covers the failure mode.
- No remaining standards finding. Markdown checks and GitHub rendering pass.
  No published history, tag, binary, versioning scheme, or license was changed.

## Specification compliance

Source: [spec.md](spec.md), criteria 1–9.

- 1–2: README is 73 lines; all seven old second-level headings remain.
  Compared original long-form prose with the relocated guides/release page:
  all operating guidance survives with adjusted relative links. The product
  introduction is shortened at the root, not treated as an operating procedure.
- 3–4: tutorial identifies prerequisites, placeholders, confirmation, expected
  results, terminal ownership, and manual SSH verification. Indexed guides cover
  installation, TUI, CLI, recovery, and troubleshooting.
- 5–6: index distinguishes normative sources and historical records. Contributor
  and maintenance pages define setup, checks, update ownership, tickets, review,
  and existing publication gates.
- 7: offline checker is in CI; ten tests include real failure and success exit
  status, invalid paths/fragments, fenced examples, Japanese headings, duplicates,
  encoded paths, references, and repository boundary checks.
- 8: exact source, test conditions, results, and limitations are in
  [verification record](../../docs/verification.md).
- 9: implementation is ready for the authorized review PR. The publication ticket
  tracks its URL and CI status; merge remains subject to review.
- No remaining implementation discrepancy. A full live-server tutorial run,
  native validation on other targets, and human review are not claimed.
