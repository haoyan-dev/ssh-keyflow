Status: ready-for-agent
Blocked by: 04

# 05: Validate locale coverage and translation quality

Implement [specification](../spec.md) criterion 13 after ticket 04.

- Extend offline checks and CI to validate all `zh-CN`, `en`, and `ja` pages,
  links, fragments, equivalent-page language navigation, and translation coverage.
- Exercise failures for missing translations, broken locale links, and invalid
  fragments; keep declared fallbacks distinguishable from complete translations.
- Review terminology, command examples, safety conditions, and compatibility
  promises in each language separately from link validation.
- Record the tested source SHA, environment, coverage, results, and limitations;
  update the PR's scope/readiness before requesting integration.

Completion: locale checks pass with meaningful negative-case coverage; language
reviews are recorded and findings resolved. Do not claim language quality from
link-check success alone.

## Comments

- The current checker covers the original maintained pages only. Locale checks
  and translation reviews remain pending.
