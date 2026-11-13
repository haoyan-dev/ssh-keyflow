# 01: 起動・環境確認・管理対象外の一覧・既存鍵の取り込み

Status: ready-for-human
Resolution: completed
Blocked by: None (can start immediately)

## What to build

起動・環境確認・管理対象外の一覧・既存鍵の取り込みを対話CLIから完了できる。spec.md、requirements.md、plan.md の該当要件を満たす。

## Acceptance criteria

- [x] 起動時に依存コマンド、書き込み、JSONを確認し、不足の導入案内と再確認ができる — `TestStartupShowsMenu`, `TestMissingDependencyCanBeRechecked`, `TestStartupReportsUnwritableStorage`, `TestInvalidStateStopsStartupWithoutOverwriting`
- [x] 対話メニューから一覧と明示的な既存鍵取り込みを実行できる — `TestPublicOnlyImportPersistsAndListsStableManagedIdentity`, `TestImportRequiresExplicitConfirmationAndDoesNotDuplicateKeys`; native public-only/pair smoke reported below
- [x] 再帰Include探索はMatch execを実行せず、管理対象外の元ファイルを変更しない — `TestListDiscoversRecursiveIncludesWithoutExecutingOrChangingThem`, `TestImportValidatesPairForEitherSelectedHalf`
- [x] 固定ID、version、バックアップ、排他、不一致の要確認、公開鍵のみと鍵ペア確認が機能する — `TestStateValidatesIDsRelationshipsAndLifecycleFields`, `TestSuccessfulUpdateKeepsPreviousValidStateBackup`, `TestMutationLockCoversExternalValidationAndConfirmation`, `TestManagedMismatchIsVisibleAndBlocksFurtherImports`, `TestMismatchedPairNeverBecomesManaged`
- [x] CLI境界で正常経路と破損・競合・対象外保護をTDDで検証する — `internal/app/app_test.go` exercises exported `app.Run` only with temporary Home and injected process runner; `cmd/ssh-keyflow/main_test.go` exercises the actual version CLI, including release injection

## Comments

2026-11-04: 前回提案した5チケットの分割に従い公開。

2026-11-04: Implementation started in `.worktrees/01-foundation`, branch `ticket/01-foundation`; HEAD and merge-base with `implement/ssh-keyflow` are `a8f67ba`. Approved seam: exported `app.Run` with temporary Home, input/output and injected external-command runner. One failing CLI behavior then minimal implementation per cycle. Credential input remains owned by child processes; menu input must not read ahead. No real SSH home or remote operations in tests.

2026-11-04: Red/green CLI cycles completed for startup/menu, dependency recheck, unsupported/corrupt JSON, duplicate fields and state relationships, recursive Include discovery, public-only import, previous-valid backup, failed-save reporting, pair import, managed mismatch, source replacement during validation, release version injection, import provenance, operation-wide exclusion, and private digest validation. Supplementary CLI regressions cover cancellation, duplicate import, mismatched pairs, credential handoff, write failure and honest unavailable menu choices. `state.WithLock` holds exclusion across validation, confirmation and save; its `LockedStore.Load/Save` API permits later tickets to journal stages while retaining the same lock. No remote functionality added.

2026-11-04: Parent reports native Windows OpenSSH smoke passed with isolated USERPROFILE: public-only import at `.tmp/native-smoke-1793780924142` and actual PTY private-selected pair import at `.tmp/native-pair-smoke` (default `.pub`, interactive `ssh-keygen -y`). Fingerprints, provenance, retained source files and managed listing confirmed. Parent records these in `docs/verification.md`; this implementer does not edit the parent's checkout.

2026-11-04: Completion verification: `gofmt -l cmd internal` produced no paths; `go test ./... -count=1` passed both cmd and app packages (state intentionally has no private-boundary tests); `go vet ./...` passed; `git diff --check` passed. Final schema regression rejected ambiguous `version`/`Version` input after an observed red/green cycle. No inline code comments or single-letter Go identifiers remain. Tests/vet ran with sandbox escalation for Windows fixture/cache access. Patches used the installed `codex.exe --codex-run-as-apply-patch` entry point because the batch wrapper did not pass patch stdin correctly.

2026-11-04: Merged current integration tip `c579a1f` into the assigned branch (fast-forward, including `9ed731f` and `docs/verification.md`); implementation remains entirely in this worktree. The public app/runner, state schema, full-operation locking contract, version injection, and followup limits are documented in `docs/application-contract.md`. Remaining functionality belongs to tickets 02–05: generation/registration/SSH aliases/remote verification, unregister/local deletion, installers/distribution. Menu paths for unavailable operations say so explicitly. Linux/macOS native behavior is not claimed by this ticket.

2026-11-13: v0.1.0（`ee3a00f`）で公開済み。起動・環境確認・既存鍵の取り込みは現行版に残る。
