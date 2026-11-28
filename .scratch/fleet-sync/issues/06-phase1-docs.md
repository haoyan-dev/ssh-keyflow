# 06 — Phase-1 contract and ja/en/zh-CN docs

Status: ready-for-agent
Blocked by: 05

Criterion 8 (docs half). Document what tickets 03–05 built on `main`: the
`node` and `fleet` commands, their flags, exit codes, terminal and `--dry-run`
rules, `node show --json` and `fleet show --json`, the roster files, the
acceptance rules, state schema v4 with the v3 → v4 migration and downgrade
limits, the fleet root key location and backup, and the OpenSSH 8.4 minimum.

- `docs/application-contract.md` (canonical, ja) gains a phase-1 section and
  updates the storage schema and migration sections. The en and zh-CN contract
  summaries follow it.
- A new workflow guide `guides/fleet.md` in ja, en and zh-CN, registered in
  `TRANSLATIONS`; `guides/cli.md`, `guides/data-and-recovery.md` and the docs
  index are updated in all three locales.
- Every new section says it describes `main` and is not yet in a release; the
  release that ships phase 1 (MINOR) removes that note. README is unchanged
  until then.
- No release tag; a release is a separate request.

## Comments

2026-11-28 — Drafted on `docs/06-fleet-phase1-docs` (base origin/main
61d19fe), uncommitted pending the user's go-ahead.

- ja contract: new section フリート同期フェーズ1（未リリース） (commands,
  terminal/confirmation/exit codes, roster and files, admin node operations,
  acceptance rules, JSON, OpenSSH 8.4, state schema v4 with migration,
  downgrade and root key backup); scope, CLI and storage sections point to it.
  The section states the canonical roster form itself rather than citing
  `roster-format.md`, since `.scratch/` is not contract.
- New guide `guides/fleet.md` (ja/en/zh-CN, in `TRANSLATIONS`); cli.md,
  data-and-recovery.md and the docs index updated in all three locales.
- Facts checked against the code by a separate spec review. It corrected:
  - the host key fallback directory, which is always the platform default;
  - why `node set` needs a terminal (confirmation, not a passphrase);
  - dry runs can exit 1 on a rejection;
  - `.bak` holds v3 bytes until the next save;
  - an unknown `node set --key` exits 2;
  - the future-time threshold is 5 minutes.
- The standards review's findings (scratch reference, migration trigger,
  downgrade caveats, en/zh-CN summary structure) are applied.
- Pre-existing, not changed: the v3 downgrade text in data-and-recovery.md
  says `.bak` "becomes v3 after migration"; strictly it does so only after the
  second save. The zh-CN glossary uses 密码短语 for passphrase while the zh-CN
  guides use 口令.
- `python scripts/check_docs.py` and `scripts/test_docs.py` pass.
