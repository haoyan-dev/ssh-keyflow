# SSH KeyFlow documentation system

Status: ready-for-agent
Base: `1d2f44eb0449b389629e01fe71b970f715badd47`
Research: [research.md](research.md)

## Objective

Give users and contributors a professional, maintainable path from discovery to
first use, daily operation, precise reference, and contribution. Apply the
reader-oriented structure of Memos to this native CLI/TUI project.

## Scope

- Concise root README retaining quick installation, purpose, boundaries, and
  existing section anchors.
- Documentation index with common tasks, maintained pages, normative sources,
  and clearly labeled historical material.
- First-use tutorial; installation, TUI, CLI, data/recovery, and troubleshooting
  guides. Reuse the complete existing README procedures where appropriate.
- Root CONTRIBUTING with development checks and local-ticket workflow.
- Documentation maintenance rules and a release guide that points to, rather
  than replaces, the existing policy.
- Offline documentation validation in CI, plus review/verification evidence.
- Maintained documentation in Simplified Chinese (`zh-CN`), English (`en`),
  and Japanese (`ja`), with equivalent navigation and synchronized content.

Runtime behavior, state schemas, versioning policy, licensing, hosted websites,
and application UI/CLI localization are outside scope. Documentation translation
is in scope following the user's additional language requirement.
Do not promise native validation on targets that were not exercised.

## Acceptance criteria

1. README identifies purpose, prerequisites, installation, first-use tutorial,
   documentation, contribution, and published releases in at most 100 lines.
2. Each old README section remains reachable under its original heading anchor;
   all substantive procedures survive in a linked maintained page.
3. A novice can follow the tutorial using real supported commands, identifiable
   placeholders, confirmation expectations, and expected results. Distinguish
   generated guides from manual execution and externally verified SSH success.
4. Installation/update/removal, TUI controls, CLI use, backup/migration/downgrade,
   and troubleshooting each have an indexed destination.
5. Contract, glossary, ADR, and verification paths retain their authority.
   Historical specifications/reviews are not presented as current guidance.
6. Contributors can find setup, applicable checks, documentation conventions,
   local tickets, PR expectations, and the authorized release flow.
7. A documented offline check validates maintained Markdown links and heading
   fragments, rejects broken links, and runs in CI. Historical documents are
   validated as link targets, not recursively rewritten.
8. Standards and specification reviews are recorded separately. Verification
   identifies the tested source SHA, environment, results, and limitations.
9. Open a review PR after verification. Preserve a clean committed work branch.
   Report PR/CI status and any pending publication phase accurately.
10. Provide Simplified Chinese (`zh-CN` / 简体中文), English (`en` / English),
    and Japanese (`ja` / 日本語) versions of the README, documentation index,
    tutorials, operating/troubleshooting guides, contributor guidance, and
    documentation/release maintenance guidance. Explain public reference material
    in each language while preserving one authoritative contract and glossary.
11. Provide visible language links on the README and documentation pages. Link
    to the equivalent page in the selected language; identify missing translations
    and their fallback language explicitly. Preserve current root paths and
    published anchors when introducing the locale layout.
12. Keep prerequisites, examples, safety conditions, compatibility/migration
    limits, and release guidance equivalent across languages. Keep executable
    commands, flags, JSON keys, paths, and product identifiers unchanged.
    Update corresponding translations in the same PR, or explicitly track missing
    or stale content; do not claim complete language support while gaps remain.
13. Extend documentation checks to cover all three locale trees, internal
    links/fragments, equivalent-page language links, and translation coverage.
    Review terminology and technical accuracy in each language; link validation
    alone does not establish translation quality.

## Dependencies

Research → specification → ticket 01 (structure and migration) →
ticket 02 (checks and review) → ticket 03 (review PR).

Additional language requirement: tickets 01–03 establish the original structure
and PR. Ticket 04 (translations and navigation) → ticket 05 (locale checks and
language review) must complete before claiming the expanded scope is implemented.

## Decisions and open questions

The user clarified that “document system” means OSS project documentation.
The user subsequently required `zh-CN`, English, and Japanese documentation,
superseding the initial Japanese-only scope. These requirements are recorded;
translations and locale validation are not yet implemented. Choose the locale
file layout during ticket 04 while retaining existing links and one source of
truth for the public contract. Historical research, tickets, ADRs, and verification
records do not require retrospective translation.
The request explicitly authorizes creating the review PR. Leave integration
pending review; do not infer authorization for an application release tag.
