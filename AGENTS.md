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

## Versioning policy

- Declare a versioning scheme for each independently released unit. SSH KeyFlow is a component: it uses `0.MINOR.PATCH` during initial development and SemVer from `1.0.0`. An umbrella is a distribution of independently versioned components; only an explicitly declared umbrella uses CalVer. Repository size or a monorepo layout alone does not make it an umbrella.
- Call the initial phase "0.x development"; it permits graduation to 1.0.0. Start a new component at `0.1.0`. In 0.x, increment PATCH for compatible fixes and MINOR for compatible features or breaking changes, resetting PATCH to zero. Identify breaking changes and migration steps explicitly.
- Adopt `1.0.0` through an explicit readiness decision when the public contract is documented, compatibility checks cover it, migration rules exist, and maintainers commit to preserving it within a major version. Feature completeness is not required; an ordinary release request does not itself authorize graduation.
- From 1.0.0, use SemVer: increment PATCH for compatible fixes, MINOR for compatible features or deprecations, and MAJOR for breaking public-contract changes. Reset lower fields after a MINOR or MAJOR bump; use the highest required increment when a release contains multiple kinds of change.
- SSH KeyFlow's public contract includes documented CLI commands, flags, exit codes, automation-facing JSON, configuration, supported state upgrades, and installation/runtime requirements. Evaluate compatibility against `docs/application-contract.md`. Supported forward migration and downgrade compatibility are separate promises; document backup, migration, and rollback limits. A change that violates a promised upgrade path is breaking even when command syntax is unchanged.
- An umbrella uses `YYYY.M.SEQUENCE`, with a numeric UTC year/month without leading zeroes and a sequence starting at zero. Default to a series for the current UTC calendar month; increment SEQUENCE for every subsequent release in that series. Explicit maintenance of an older supported series retains its date and records the actual publication date separately. The sequence conveys order, not compatibility.
- Umbrella releases record exact component versions and source commits in an immutable manifest, with integration verification, compatibility changes, migration instructions, and support status. Components keep independent versions. Any umbrella-owned API has an explicit compatibility contract; CalVer alone makes no compatibility promise.
- Release channel and contract maturity are independent: a normal 0.x release may be supported and eligible for `latest`. Use `-alpha.N`, `-beta.N`, or `-rc.N` for previews and exclude them from `latest`. Document supported series separately from version numbers.
- Changing an established unit's scheme requires an explicit migration decision covering tag/package ordering, dependency constraints, installers, and release automation. Preserve published versions. Keep saved-state/protocol schema identifiers separate from product release versions.

## Tags and releases

- Before preparing a release, read the development and release contract in `README.md`, the distribution contract in `docs/application-contract.md`, and `.github/workflows/release.yml`. Use those sources for verification commands, packaging, and publication behavior. Refresh remote branch and tag references before selecting a version or target.
- An explicit request to create a release tag, including "mainにマージ、バージョンをタグする", authorizes tag creation, push, and the automatic release. Honor narrower instructions such as "local tag only" or "do not publish"; stop at the authorized phase and report any remaining phases as pending. A request to implement changes or review this policy alone does not authorize a release.
- Prefix the selected unit's version with `v`; SSH KeyFlow tags remain `vMAJOR.MINOR.PATCH`. When the version is unspecified, apply the Versioning policy to the highest published non-prerelease version in the intended series, using all changes since that release. Components default to the current supported series; umbrellas default to the current UTC month's series. Maintain an older series only when explicitly requested and supported. Check local and remote reservations before selecting the next unused version; never reuse a published or reserved version. Honor an explicitly requested prerelease suffix.

### 1. Prepare and integrate

- Prepare a release PR containing the proposed release notes in tracked repository documentation and a preparation record in `docs/verification.md`; the notes may be part of that record. The record must identify the proposed version, previous release, version rationale, full tested source SHA, verification environment, results, and limits. Notes must cover changes since the previous release, breaking changes, and migration or installation instructions. The preparation may be included in the implementation PR or a separate documentation PR.
- Complete applicable checks and separate standards/specification reviews, then integrate the preparation PR into `main` before creating the release tag. Preparation is complete when the record and notes are merged and their required checks/reviews pass. A record left only in a local file or unmerged PR does not satisfy this gate. Follow the existing authorization rules for pushing and merging PRs.

### 2. Select and verify the candidate

- Refresh `origin/main` after preparation is merged. By default, select its latest commit as the final candidate; honor an explicitly selected commit only if it is integrated into `main` and the merged preparation record covers it. Compare the candidate with the tested source and account for all intervening changes. If they change release scope, version rationale, or required evidence, update and merge the preparation before selecting the candidate again.
- Verify the candidate from a clean working tree and complete the applicable README checks and required CI/reviews. Reuse passing evidence for an identical tree or unchanged check inputs, recording the tested SHA and comparison with the candidate; rerun affected checks when source, dependencies, build inputs, or execution conditions change. Required CI checks still apply. Let the release workflow perform its build matrix rather than duplicating that matrix locally without a concrete verification need.
- The preparation record references the tested source SHA. Record the final candidate SHA and the verification comparison in the tag annotation and published release notes, then in the publication audit. This keeps the preparation commit independent of its own SHA. Candidate selection is complete when the exact target, passing evidence, and any evidence reuse are accounted for. Keep unreleased builds identified as `dev`.

### 3. Tag and publish

- Recheck that the version is unused locally and remotely, then create an annotated release tag on the verified candidate. Record the chosen version, full target SHA, and rationale.
- Before pushing, report the chosen version and target SHA. For an authorized release, continue without another confirmation solely because the agent selected those values. Ask only when the user's requested version/target cannot satisfy this policy or proceeding would materially change the agreed scope or risk. Push only the intended tag.
- Publish through the release workflow, building from the tagged commit with that tag injected as the version. Verify that the tag, release, embedded version, and artifacts identify the same source commit and satisfy the distribution contract. Mark prereleases as prereleases and exclude them from `latest`. Compare non-prerelease versions numerically under the unit's scheme (SemVer precedence for components; year, month, sequence for umbrellas), not lexically or by publication date. Advance `latest` only within its designated supported channel to a newer version; an older-series maintenance release must not replace a newer series. Verify the workflow enforces this before pushing a tag that could violate it.
- Publish the prepared release notes with the final source SHA and verification comparison. If the workflow generates notes, replace them with the prepared content and confirm the edit preserves published asset IDs and digests. Publication is complete when the workflow succeeds and the published notes and source metadata match the verified candidate.

### 4. Audit and integrate the results

- After publication, confirm the workflow succeeded, download the published assets, and verify the asset set, checksums, archive contents, and available native version smoke checks. Check stable/prerelease and `latest` status, then record the release URL, tag and commit, workflow run, results, and untested targets in `docs/verification.md`. Track source-hosted installer verification against its own `main` commit.
- Integrate post-release verification records through a documentation PR under the existing push/merge authorization rules. The audit is complete when the checks and limitations are recorded and that PR is merged. Report publication and audit-integration status separately while the PR is pending.
- `main` may advance beyond a release tag with audit documentation. The tag continues to identify the source used to build the published binaries; documentation-only audit changes require neither another version nor moving the tag.
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
