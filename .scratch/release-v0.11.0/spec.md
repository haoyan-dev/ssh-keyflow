# Release v0.11.0

Status: ready-for-human
Published: v0.11.0 at `848c5f286a02dc05e7fddff0893cff35b3fae547`; audit PR
pending
Previous release: v0.10.0 (tag `v0.10.0`,
`0af26316c9589203402c46fa6f4f97871f46c414`)
Tested source: `f718b817527bec8ef4b6a6d27fe56aee33cc931e`

## Objective and scope

Publish the next supported stable SSH KeyFlow release from integrated `main`,
following all preparation, candidate verification, publication, and audit gates.
On 2026-11-28 the user asked to "prepare the v0.11.0 release", then said "merge
it once CI passes, then tag and release". Together these authorize:
- the preparation branch and PR, and merging it after CI passes;
- the annotated tag, pushing it, and the automatic release.

On 2026-11-28 the user then said "also merge the audit PR when it's ready", which authorizes
merging the publication-audit PR after its checks and reviews pass.

Select v0.11.0. Since v0.10.0 the release contains breaking changes:
- commands and flags are retired (every `hosts` command, the guide, pin/unpin,
  the TUI, public-only import);
- the `keys list --json` shape is reduced;
- the state moves to schema v3 through an archived migration.

In 0.x development a breaking change increments MINOR and resets PATCH. No
1.0.0 graduation is requested. Local and remote tags and releases were refreshed
on 2026-11-28, and v0.11.0 was unused. Recheck immediately before tagging.

## Acceptance criteria

1. Merge a preparation PR containing these tracked notes and a
   `docs/verification.md` record. The record states the version rationale, the
   previous release, the full tested source SHA, the environment, results,
   evidence reuse and limits.
2. Select refreshed `origin/main` after preparation merges, and account for
   every difference from the tested source. Verify a clean state, the required
   CI and reviews, and the latest-channel safeguards.
3. Report the version and final source SHA. Create an annotated v0.11.0 tag and
   push only that tag. The release workflow must pass on that exact source.
4. Replace the generated notes with the prepared notes plus the final source
   and comparison, preserving asset IDs and digests. Confirm stable and latest
   status.
5. Download all seven assets and verify:
   - exact filenames and hashes;
   - archive members and notices;
   - Unix permissions;
   - binary build metadata and version;
   - a native Windows amd64 version check and read-only smoke test.

   Record native targets that weren't available.
6. Merge the publication-audit PR, preserve immutable release assets and tags,
   clean up task-owned branches and files, and report publication and audit
   status.

## Dependencies and limits

01 preparation → 02 candidate/tag/publication → 03 downloaded-asset audit.
No open scope questions.
A live interactive confirmation and native runs on macOS, Linux and arm64
remain unverified, and must be stated as limits.
