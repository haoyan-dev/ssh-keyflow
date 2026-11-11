Status: ready-for-agent
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
