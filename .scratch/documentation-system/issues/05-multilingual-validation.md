Status: ready-for-human
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
- 2026-11-13: `check_docs.py` checks locale coverage, language links, declared
  fallbacks and stale markers; `test_docs.py` has negative cases. Per-language
  reviews and the re-check are recorded in docs/verification.md (2026-11-13).
  Integration is pending PR review.
- Follow-up: the reviews flagged possible inaccuracies in the Japanese sources
  that were left unchanged:
  - troubleshooting's Windows `UNPROTECTED PRIVATE KEY FILE` explanation;
  - the contract's Portable release note that the README documents manual
    maintenance;
  - the contract's "third step" wording for the connection step;
  - the installation guide limiting the old-installer note to v0.2.0 archives;
  - the tutorial omitting the local-OS condition for pinning steps.
  Fix them in all three languages in a later PR.
