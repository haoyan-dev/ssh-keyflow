# OSS documentation research

Observed: 2026-11-11. Target: SSH KeyFlow documentation, not a new application.
Repository base: `1d2f44eb0449b389629e01fe71b970f715badd47`.

## Primary sources

| Source | Finding | Adaptation |
| --- | --- | --- |
| [Memos README at 9ee06fd](https://github.com/usememos/memos/blob/9ee06fdc4722b45cc7efe1311767407fc86fec66/README.md) | A short product introduction and quick start lead to detailed documentation and contribution guidance. | Make README the entry point; move detailed operating procedures into linked pages. |
| [Documentation index](https://usememos.com/docs) | Popular Paths complements a categorized topic inventory. | Provide task-based navigation and a complete maintained-document inventory in `docs/README.md`. |
| [Getting started](https://usememos.com/docs/getting-started) | Prerequisites, ordered actions, checks, and first-run help form a path to a first useful result. | Add one end-to-end tutorial with expected results and explicit manual SSH work. |
| [Development](https://usememos.com/docs/development) | Setup, building, testing, and contribution have distinct destinations. | Add root contributor guidance and dedicated documentation/release maintenance pages. |
| [Contributing](https://usememos.com/docs/development/contributing) | Scoped changes, realistic examples, linked pages, and documentation updates reduce review friction. | Define page types, sources of truth, update triggers, and an automated offline link check. |
| [Operations](https://usememos.com/docs/operations) | Backup/restore and upgrades are explicit operating concerns. | Preserve state migration, rolling-backup, downgrade, and installer ownership guidance in task-oriented pages. |

## Local findings and decisions

- README currently combines seven long sections: install, operation, state safety,
  Portable distribution, manual maintenance, old installer migration, and release.
  Split these without dropping the existing operational details.
- User guidance is primarily Japanese. Keep that language rather than creating an
  incomplete translation. This initial scope decision was superseded by the
  user's later requirement for `zh-CN`, English, and Japanese documentation;
  see spec criteria 10–13 and tickets 04–05. Keep concise, factual writing.
- `docs/application-contract.md`, `GLOSSARY.md`, ADRs, and
  `docs/verification.md` already own contract, language, decisions, and evidence.
  Link them; do not create a second normative contract.
- Keep old README heading anchors as short navigation sections. Preserve existing
  document paths and historical records; mark history in the index.
- Use repository-native Markdown rendered by GitHub. A hosted site, new branding,
  Docker deployment, Memos' version scheme, and a license change are outside scope.
- Publication for this change means a review PR and normal documentation
  integration. Documentation-only changes do not require binary publication.
  A release tag requires the explicit authorization described in `AGENTS.md`.

## Evidence limits

The research helper inspected the pinned Memos README and official website.
The parent independently fetched the official index and contributor page.
Website pages are mutable observations; only the README reference is pinned.
No claim is made that SSH KeyFlow commands were exercised on live remote servers.
