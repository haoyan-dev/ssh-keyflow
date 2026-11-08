## Development workflow

- Before starting, identify the objective, scope, acceptance criteria, and open questions.
- Proceed autonomously within the agreed scope. Confirm decisions that materially change the objective, scope, or risk.
- Map task dependencies and parallelize independent work.
- When delegating, define starting conditions and completion criteria. Hand off work through references to artifacts and their status.
- Document research findings and decision rationale needed by downstream work.
- Review standards compliance and specification compliance separately, and verify that findings are resolved.
- Tie verification results to the tested commit and execution conditions. Repeat verification according to the impact of changes.
- Complete applicable verification, review, issue updates, artifact checks, and workspace cleanup before reporting completion.
- Focus progress reports on decisions, findings, blockers, and completed work.

## Branch management

- Use `main` as the stable integration branch. Make changes on short-lived work branches and integrate them through pull requests.
- Before editing, check the current branch and working tree. Reuse a branch only when it belongs to the same task; otherwise create a branch from the latest fetched `origin/main`. If the remote cannot be reached, record the base commit and report that it was not refreshed.
- Name branches `<type>/<short-kebab-case-description>`, using `feat`, `fix`, `docs`, `refactor`, `test`, or `chore`; include a local ticket number when applicable.
- Keep each branch and pull request focused on one objective. Preserve existing uncommitted changes; use a separate worktree when switching tasks would mix unrelated work.
- For parallel work, give each worker a separate branch and worktree. Use a temporary integration branch such as `implement/<feature-or-version>` only when the task requires coordinated integration, and merge it into `main` through a pull request.
- Before requesting merge, resolve conflicts, review the final diff, and complete the applicable verification against the final branch state. Record the tested commit and results in the pull request; rerun affected checks after further changes.
- Update a work branch from `origin/main` before merging when needed to resolve conflicts or validate integration. Rebase only private, unshared commits; merge updates into shared branches to preserve collaborators' history.
- Merge only after required CI checks and reviews pass. Follow the repository's configured merge method, and obtain authorization before pushing, merging, or changing branch protection unless the agreed task already authorizes those actions.
- Keep `main` and shared branch history intact. Obtain explicit approval before rewriting published history or force-pushing; use `--force-with-lease` for an authorized force-push.
- After confirmed merge, delete only task-owned branches and worktrees whose changes are integrated and whose working trees are clean. Preserve unmerged work and follow the same authorization rules for remote branch deletion.

## Tags and releases

- Before preparing a release, read the development and release contract in `README.md`, the distribution contract in `docs/application-contract.md`, and `.github/workflows/release.yml`. Use those sources for verification commands, packaging, and publication behavior.
- An explicit request to create a release tag, including "mainにマージ、バージョンをタグする", authorizes tag creation, push, and the automatic release. Honor narrower instructions such as "local tag only" or "do not publish". A request to implement changes or review this policy alone does not authorize a release.
- Use version tags `vMAJOR.MINOR.PATCH`. When the version is unspecified, choose the next unused version from the latest stable release using these rules: increment PATCH for backward-compatible fixes and MINOR for backward-compatible features; increment MAJOR for breaking changes from v1.0.0 onward. While MAJOR is zero, put breaking changes in a new MINOR version and describe their migration impact. Honor an explicitly requested prerelease with a suffix such as `-alpha.1`, `-beta.1`, or `-rc.1`.
- Refresh remote branch and tag references first. By default, create an annotated release tag on the latest `origin/main` commit after the requested work is merged; honor an explicitly selected commit only if it is integrated into `main`. Confirm the version is unused locally and remotely, and record the chosen version, full target SHA, and rationale. Keep unreleased builds identified as `dev`.
- Verify the candidate from a clean working tree and complete the applicable README checks and required CI/reviews. Reuse passing evidence for an identical tree or unchanged check inputs, recording the tested SHA and comparison with the candidate; rerun affected checks when source, dependencies, build inputs, or execution conditions change. Required CI checks still apply. Let the release workflow perform its build matrix rather than duplicating that matrix locally without a concrete verification need.
- Prepare release notes covering changes since the previous release, breaking changes, migration or installation instructions, and known verification limits. Record the candidate SHA, verification environment, and results in `docs/verification.md`.
- Before pushing, report the chosen version and target SHA. For an authorized release, continue without another confirmation solely because the agent selected those values. Ask only when the user's requested version/target cannot satisfy this policy or proceeding would materially change the agreed scope or risk. Push only the intended tag.
- Publish through the release workflow, building from the tagged commit with that tag injected as the version. Verify that the tag, release, embedded version, and artifacts identify the same source commit and satisfy the distribution contract. Mark prereleases as prereleases and exclude them from `latest`; advance `latest` only to a newer stable version.
- After publication, confirm the workflow succeeded, download the published assets, and verify the asset set, checksums, archive contents, and available native version smoke checks. Check stable/prerelease and `latest` status, then record the release URL, tag and commit, workflow run, results, and untested targets in `docs/verification.md`. Track source-hosted installer verification against its own `main` commit.
- Integrate post-release verification records through a documentation PR. Those records do not require another version or moving the published tag.
- Keep published tags and binary assets immutable. Deliver code or packaging corrections under a new version; release-note corrections may update the existing notes. Retry failed publication only after confirming the tag still identifies the verified commit and existing published assets will be preserved. Handle rollback by directing users to a known-good version and documenting the affected release.

## Agent skills

### Issue tracker

Track issues and specifications in local Markdown.
Read docs/agents/issue-tracker.md when creating or consulting tickets.

### Triage labels

Use the five standard triage labels.
Read docs/agents/triage-labels.md when classifying issues.

### Domain docs

Maintain a single domain context.
Read docs/agents/domain.md before design work or code investigation.
