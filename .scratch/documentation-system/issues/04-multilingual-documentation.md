Status: ready-for-human
Blocked by: none

# 04: Provide Simplified Chinese, English, and Japanese documentation

Implement [specification](../spec.md) criteria 10–12.

- Choose a locale layout for `zh-CN`, `en`, and `ja` while preserving existing
  root document paths and published anchors.
- Translate the README, documentation index, tutorial, operating/troubleshooting
  guides, contributor guidance, and maintenance/release guidance.
- Make public reference explanations accessible in each language without creating
  competing contracts or glossaries.
- Add equivalent-page language links labeled 简体中文, English, and 日本語.
- Preserve commands, flags, JSON keys, paths, safety conditions, and migration
  limits. Mark missing/stale translations and fallback language explicitly.

Completion: all required maintained pages have technically equivalent versions
in the three languages, with working navigation and no undisclosed coverage gaps.
Historical research, tickets, ADRs, and verification records need not be translated.

## Comments

- Added at the user's request after the initial Japanese documentation structure.
- Requirements are recorded; translation implementation has not started.
- 2026-11-13: Implemented on `docs/multilingual-documentation`. The Japanese
  source stays at the existing paths; English and Simplified Chinese live in
  `*.en.md` / `*.zh-CN.md` and `docs/en/` / `docs/zh-CN/`, and the contract and
  glossary get non-authoritative overviews. All 13 page sets are complete.
  Integration is pending PR review.
