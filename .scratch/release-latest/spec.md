# Preserve latest across maintenance publications

Source: `.scratch/versioning-policy/issues/01-guard-latest.md` on the policy
worktree's `docs/versioning-policy` branch.

## Scope and acceptance

Implement the follow-up's numeric version comparison, first-stable-release
handling, fail-closed lookup errors, preview exclusion, and publication
serialization. Keep the actual repository, verified tag, generated notes, and
built assets in the existing publication flow. Preserve existing releases by
using creation only; do not retry by deleting or replacing assets.

No tag push, remote release, or change to existing published versions is part
of this implementation request.

## Diagnosis and design

The old workflow's publication shell was executed with a mocked gh command.
For candidate v1.9.1 it emitted `--latest` without any lookup, so the regression
assertion failed. The cause is the unconditional stable-release flag; there is
also no cross-tag lock around publication.

Move publication into a standard-library Python helper. Use gh api --include to
distinguish HTTP 404 from transport/authentication/server errors. A missing
latest is accepted only after a successful paginated listing with no published
stable release. Compare the three numeric fields of validated stable component
tags. Preview tags explicitly use --prerelease and --latest=false.

Place a constant repository-wide concurrency group on the publication job,
with cancel-in-progress false and queue max. Resolve latest inside that job,
immediately before creation. Queue order need not match version order because
each publication compares the state left by its predecessor.

The group coordinates runs using this workflow. Manual publishers, workflows
with different group names, and tags pointing to old workflow definitions do
not participate in the lock. Use the guarded workflow for future publications.
GitHub's queue is bounded at 100 waiting jobs; queue overflow requires operator
attention rather than silently assuming every tag was published.

## Sources consulted

- GitHub Actions workflow syntax, concurrency and queue max:
  `https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax`
- GitHub CLI api, include/pagination/slurp:
  `https://cli.github.com/manual/gh_api`
- GitHub CLI release create, verify-tag and latest:
  `https://cli.github.com/manual/gh_release_create`
- GitHub REST releases, latest and release listing:
  `https://docs.github.com/en/rest/releases/releases`

## Verification

See the dated entry in docs/verification.md. Local tests mock GitHub publication;
GitHub-hosted scheduling and live release mutation require subsequent CI/release
verification and are not claimed by the unit tests.
