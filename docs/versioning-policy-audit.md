# Versioning policy audit

Date: 2026-11-11

Scope: the AGENTS.md versioning policy and its agreement with the existing
application, packaging, installer, and release contracts. This change does not
publish a release or alter release automation.

## Decisions

- Keep SSH KeyFlow on the component track: 0.x development, then SemVer after
  an explicit 1.0.0 readiness decision.
- Reserve CalVer for an explicitly identified umbrella distribution. No such
  distribution is introduced into SSH KeyFlow by this policy.
- Distinguish compatibility, maturity, release channel, support, and state-schema
  version. In particular, forward migration does not imply downgrade support.
- Preserve the existing tag format, immutable artifacts, authorization boundaries,
  and verification requirements.

## Standards compliance

The policy keeps each version-selection rule in one section and refers to it
from the release procedure. Existing branch and release approval requirements
remain intact. The application contract continues to own concrete CLI, JSON,
state, and installation behavior.

No changes are needed to historical ADR 0001 or ADR 0004: their 0.x breaking
change and rollback decisions remain compatible with the new policy.

## Specification compliance

| Scenario | Required result |
| --- | --- |
| Compatible fix after 0.5.0 | 0.5.1 |
| Feature or breaking change after 0.5.1 | 0.6.0; breaking changes require migration notes |
| Compatible feature after 1.2.3 | 1.3.0 |
| Breaking promised state-upgrade path after 1.2.3 | 2.0.0 |
| First umbrella series in November 2026 | 2026.11.0 |
| November-series maintenance published in December | 2026.11.1; actual date recorded separately |
| Preview 2026.12.0-rc.1 | Prerelease; excluded from latest |
| Publish 2026.11.1 after 2026.12.0 | latest stays on 2026.12.0 |
| Publish component 1.9.1 after 1.10.0 | latest stays on 1.10.0 |
| Normal release request while on 0.x | Remains 0.x unless graduation is explicitly authorized |

The existing package and installer validators accept three numeric fields with
an optional prerelease suffix. SSH KeyFlow retains that shape. This inspection
does not establish that the validators fully implement SemVer or validate
calendar dates, nor does it certify another project's CalVer integration.

## Implementation finding and follow-up

`.github/workflows/release.yml` unconditionally passes `--latest` for every
non-prerelease tag. It does not compare the release with the current latest
version. This predates the policy update and violates the existing requirement
to advance latest only to a newer stable release.

Tracked in `.scratch/versioning-policy/issues/01-guard-latest.md`. Policy now
requires verifying that guard before pushing a tag that could violate it.
The user subsequently authorized implementation, isolated on
`fix/01-release-latest` in `.worktrees/release-latest`. The fix adds numeric
comparison, fail-closed API handling, and a shared publication queue. Consult
that branch's `docs/verification.md` for evidence and limitations. Until that
branch is integrated, main still has the original workflow behavior.

## Verification evidence

Reviewed on Windows PowerShell against base commit
`662d7b7b4999a3569b71fd5abbae2d121fb5d3e8`, with the uncommitted documentation
changes on `docs/versioning-policy`. Evidence applies to that base plus the
reviewed diff, not to a claimed committed or CI-tested release candidate.

Checks: final diff and whitespace review; manual scenario review above; inspection
of AGENTS.md, README.md, docs/application-contract.md, ADRs 0001 and 0004,
scripts/package_release.py, install.sh, install.ps1, and the release workflow.
No application source or executable workflow changed; runtime tests and release
publication were not performed.

## Reusable skill audit

The companion `li-agent-skills` change creates
`skills/haoyan-dev/project-development-policy/` on
`feat/project-development-policy`, based on
`a63627d48a50ca91112f578d7a724d8a0c6bd6b6`. It supplies the skill, an adaptable
AGENTS.md asset, supporting-doc conventions, and manual audit scenarios.

Standards review: the skill preserves existing project conventions, keeps
template resources inside its folder, resolves all local Markdown links, and
does not require another installed skill. Its README entry explains discovery.
Existing autonomous-task defaults remain unchanged; that skill already defers
to the target project's documented policy.

Specification review: component, umbrella, monorepo, existing-project, dirty-tree,
and unversioned-directory paths are covered. Explicit authorization boundaries
prevent a policy setup invocation from starting publication. A manual walkthrough
of the bundled cases found no unresolved documentation conflicts.

Validation: skill-creator's quick_validate.py passed; local-link, whitespace,
and unfinished-scaffold checks passed across all four skill Markdown files;
git diff --check passed in both repositories. This is static validation and a
manual decision review, not an independent execution of the skill in a new
project. The skill is stored in its source repository; global installation was
not part of this task.
