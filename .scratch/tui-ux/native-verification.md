# Coordinator native TUI verification

## Ticket04 implementation checkpoint

Source: `6fdc973d6908fd32e4b3d73ae9272225bb753569` (clean product source).
Environment: Windows amd64, Go1.26.2, Herdr0.1.18, 2026-11-07.
Build: `go build -buildvcs=false -o <integration>/.tmp/native-handoff/ssh-keyflow-04.exe ./cmd/ssh-keyflow` from ticket04 worktree, Herdr t3f, exit0.

The disposable USERPROFILE contains one fictitious `fixture-user@example.test:2222` Linux destination, one public-only fixture key and one registered/complete/unverified registration. No private key, credentials or reachable SSH server is used. Never answer yes to the manual unregister prompt. The isolated state file SHA256 is `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0` before and after every sequence.

### Native confirmation and interruption

At80x24, `u`, Ctrl+S opens Review unregister with resolved Japanese alias, login, key label, full fingerprint, manual method and original approval caveat. Enter opens the native exact manual-unregister prompt on the original terminal.

Ctrl+C returns a usable TUI Result, with Error EOF, confirmation unknown and execution stopped. `hc get` confirms process still live. The result makes no no-changes assertion. Then in the same process, Esc/u/Ctrl+S/Enter opens the native prompt again. Typing no+Enter returns Result with confirmation declined. q exits0. State bytes unchanged. All task-owned terminals t3f/t3g removed.

Contrast: native baseline02 `dce6c7c6b9c8d71c92e7ed71fdb77b2445cce5c3` on the same fixture exited1 with `program was killed: program was interrupted` on Ctrl+C. Its state was also unchanged. This comparison establishes the terminal restoration fix, not actual remote SSH behavior.05 replaces interim result wording.

### Compact review/help/cancel and actual dynamic resize

At32x12, c/q/a/r writes literal qar to Management name. Tab/Tab/Enter opens the labelled Help control. Ctrl+C closes Help; Ctrl+S opens Review; Esc then Ctrl+C opens discard confirmation with Keep editing selected. Enter restores the form with the same Help focus.

To resize an existing terminal, launch another local Herdr terminal with argv `["hc","attach",target]`, no_parent=true and requested initial size. `hc get target` confirms actual dimensions change. This uses the public terminal API and a real PTY. Earlier standalone ctypes attach experiments did not resize and are not acceptance evidence.

Actual32x12->31x11 displays Resize: minimum32x12 plus q quit. Ctrl+S/Enter/a do not run an operation or alter the draft. Detach viewer with Ctrl+backslash, then attach from a60x18 viewer. Original Help focus is restored. Shift+Tab twice exposes unchanged qar. Ctrl+C/Tab/Enter discards the draft and returns the list without execution. q exits0. State bytes unchanged. Task-owned terminals t3h/t3j/t3k removed.

These are ticket04 source checks. Final05/06 source still requires a final native pass, especially Tasks and results. Native Linux/macOS and real SSH/server behavior are unverified.

### Actual ssh-keygen child interruption

The same04 binary was started at80x24 with a verified empty disposable home `.tmp/native-create-interrupt/home`. Native name characters were entered, Review opened, and Continue reached the exact local creation prompt naming only paths under that fixture. The coordinator approved this disposable local creation, then stopped at the actual Windows OpenSSH `Enter passphrase (empty for no passphrase):` prompt. No passphrase or empty passphrase was submitted. Ctrl+C returned a live TUI Result: creation interrupted / exit status255, saved pending checkpoints, creation unknown, execution stopped. State retained creation ID `creation-fe523012e7d4005a88c5269e42c9ee47` / key ID `fixture-reserved-ID-omitted`, stage pending. Managed keys remained empty, key directory contained no files, state.lock was removed. q exited0 and t3n was removed. Final fixture state hash: `89d203be99776b48a2c478db8496be448cc3c45edb614451aaab5c699601af1c`.

This adds actual native ssh-keygen credential-input interruption evidence on Windows, separate from fake Runner and confirmation denial. It does not prove remote ssh/password/host-key handling or native Linux/macOS. The saved fixture can test restart Tasks/local continuation without contacting a server or creating a real key.

## Known defect to resolve in final review

At04 source6fdc973, native bracketed paste into Create Management name is lost: c then `hc send <terminal> native-child-fixture` leaves name empty and Enter reports the required-name error. Ordinary named-key letters populate the same field and Review correctly. Source inspection: tui.Update routes KeyPressMsg to form/search input; PasteMsg falls through to the viewport/list instead. This is an input interaction defect under AC04/AC05/AC12 and must be fixed and checked for both form and search before final completion. It is intentionally pending the single fresh post-review fix implementer, not claimed as a passing acceptance case. No product change was made by the coordinator.

## Ticket05 native results, restart and saved-task continuation

Source: `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` (clean product source). Same Windows amd64 / Go1.26.2 / Herdr0.1.18 environment. Native build in coordinator terminal t4k passed exit0. Disposable fixtures only; manual authentication and configuration=no were inspected in Review before exact native Register approval. No SSH command was executed or remote server contacted.

- At80x24, manual Register for an unrecorded pair and a previously registered/verified pair both returned **Manual action required**, **No saved recovery task**, and **Register again after exit/restart**. The previously verified destination-specific connection remained explicitly labelled recorded, separate from this operation's verification:not-checked. End reached complete manual shell instructions; Home restored summary and primary Register again control. In the unrecorded fixture a preceding native refusal also returned **Cancelled before changes**, and the same process could start Register again normally.
- Both sessions exited0 and restarted independently. Tasks displayed **No recorded follow-up. Remote access is not verified.** Parsed state JSON was identical to its respective baseline (byte formatting changes are not treated as workflow changes). No pending registration, creation or onboarding was fabricated. The two baseline state hashes were `8c2bb838f1081e2ce5eb6108ae5b1ce14e8a04970bcebba05b0764f4113676b9` and `00d9cbbc4cbc16c717f0c18c3f5c337a1d08213344d0d785b7cd0153301a2d63`.
- At32x12, a saved unknown registration appeared in Tasks with the Japanese pair, exact registration-native-fixture ID and **Outcome unknown**. Enter/s opened the target-bound resume form; Ctrl+S opened Review without executing. The body scrolled with End while Continue/Edit/Cancel/Help stayed visible. Tab/Tab/Tab/Enter opened labelled Help, Ctrl+C returned to Review, Esc restored form, and Ctrl+C returned to Task details without executing. Saved state SHA256 stayed `1017446ae2183efe30ea3ccae6cba91b719a58783051ab7b7bbf62c83b83fd64`.
- At60x18, the genuine pending local creation from the earlier interrupted ssh-keygen session survived restart. Tasks described local creation only, and s/Ctrl+S opened Review for exact `creation-fe523012e7d4005a88c5269e42c9ee47`; there were no authentication/configuration inputs. Esc/Ctrl+C returned without executing; state SHA256 stayed `89d203be99776b48a2c478db8496be448cc3c45edb614451aaab5c699601af1c`.
- All coordinator terminals t4k/t4m/t4n/t4p/t4q/t4r/t4s were exited and removed. No credentials or private keys were supplied. Linux/macOS native execution and real SSH authentication/server behavior remain unverified. The known bracketed-paste defect still requires the final review/fix pass; this checkpoint does not certify its resolution.

### Ticket05 native interruption with final result renderer

The same `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` binary at80x24, fixture `.tmp/native-handoff/home`, was checked after results were implemented. u/Ctrl+S/Enter reached the original manual unregister intent confirmation. **No approval was given**. Ctrl+C returned **Operation stopped; outcome unknown or requires review**, retained EOF/confirmation:unknown/execution:stopped, and kept the process live. In that same process Esc/u/Ctrl+S/Enter reached the original prompt again; no+Enter returned **Cancelled before changes**, confirmation:declined, and retained earlier registered/unverified saved evidence separately. q exited0. State SHA256 remained `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0`. Coordinator terminal t4t was removed. This is real native confirmation/signal behavior, not a remote SSH execution test.

### Ticket05 final navigation and dynamic resize

Same frozen `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` binary,20 long Japanese destination aliases, no keys/registrations/servers. At80x24 and32x12, ordinary key input /host-17/Enter/Enter opened the exact Japanese alias and `alice@build-17.example:22` with destination ID host-17. End at32x12 reached the final configuration/history text vertically; Back/Help/Exit stayed visible. Esc/Tab/Shift+Tab at80x24 retained Search:host-17 and selection.

At32x12, Create accepted literal qar, labelled Help opened and Ctrl+C returned; Review/Esc/Ctrl+C/Keep editing restored Help focus. A real `hc attach` viewer resized the running terminal to31x11 (confirmed by hc get): only minimum32x12/Exit appeared, and Ctrl+S/Enter/a neither executed nor changed the draft. Detach, then attach at60x18 restored Help focus; Shift+Tab twice showed exactly qar. Discard returned without execution, q exited0. Fixture hash stayed `24eb5cb3259d3dfd073cbad4d986396595c84b909dcba6d1fb2d4c1ac84e8c31`. Own t4w/t4x/t4y/t4z terminals were exited and removed. Bracketed paste is deliberately not included in these ordinary-key passes and remains the recorded pending defect.

## Post-review native regression baselines (2026-11-08 JST)

Before applying the review fixes, the unchanged `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` native binary was tested locally under the same Windows/Go/Herdr conditions.

- P2 search:80x24, `.tmp/native-navigation/home`, `/` then `hc send host-17` (bracketed paste plus Enter) left the query empty and the list unfiltered at host-1. q exited0; SHA256 stayed `24eb5cb3259d3dfd073cbad4d986396595c84b909dcba6d1fb2d4c1ac84e8c31`. Own t5f removed. This extends the earlier lost-form-paste evidence to search.
- P1 onboarding:80x24, new synthetic fixture `.tmp/native-onboarding-review/home`. It combines one fictitious destination, one pending Creation with reserved key ID and absent keyfiles, and a matching key_pending Onboarding; managed keys and registrations are empty. This is deliberately schema-valid synthetic evidence, not an actual native interruption record. Tasks/Enter/s/Ctrl+S failed with `Managed key target key-native-reserved changed or is missing; draft retained, cancel and reselect` instead of Review. Ctrl+C/q exited0, no approval/SSH/key creation occurred, and state SHA256 stayed `9c17ba699e5132a7df37778345e29b7a4789199329281daac2d07d50967217bc`. Own t5g removed. Final-source retest is required for P1 and the P3 resolved creation intent.

### P4 native baseline before the review fixes

At the unchanged `0be6ae68a2f983877fa89c21b6cc019f6e5b354d` binary, 80x24, `.tmp/native-handoff/home`: u/Ctrl+S/Enter reached the native manual unregister prompt. Only no was submitted. Result showed Cancelled before changes; v opened registration details while the active category remained Destinations. F5 incorrectly returned to Managed destinations despite unchanged registration. q exited0. State SHA256 remained `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0`; t5s was removed.

## Post-review fixed native checkpoint (2026-11-08 JST)

Exact tested product/test commit: `ba0f69d81c3345ed2bd78062521bcd63adda7c69`. The coordinator built from the clean review-fixer worktree using `go build -buildvcs=false` through Herdr t67. Binary SHA256 `1275fabe720b9a7ffbe861fd74d9d1bf734f1183d947ae95818e8ce8e596b020` exactly matches the independently built worker binary. Windows amd64, Go1.26.2, Herdr0.1.18; disposable homes only. No real SSH server, credentials, user keys or deletion was used.

- **P2 form/caret/help/resize:** t69 at80x24, Create accepted actual `hc send` bracketed paste `q?確認`. Returning to the input, Left then pasting Z produced `q?確Z認` in Review. No action shortcut was triggered. Esc/labelled Help/Ctrl+C/dirty Cancel/Keep editing restored Help focus. A real `hc attach` viewer t6c changed the same terminal to31x11, confirmed with hc get. Paste ignored-q?, Ctrl+S, Enter and a were suppressed. Detach and60x18 viewer t6f restored the Help control; the same `q?確Z認` value and Review remained. Discard ran no operation. Pasted host-17 search then Enter opened exact Japanese alias17 / alice@build-17.example:22, and category switch/back retained query. q exited0. Navigation fixture state SHA256 stayed `24eb5cb3259d3dfd073cbad4d986396595c84b909dcba6d1fb2d4c1ac84e8c31`.
- **P1/P3 interrupted onboarding:** t6b80x24 used the synthetic valid pending fixture described above. Tasks/Enter/s/Ctrl+S now opened Review with exact onboarding-native-review, resolved fixture-user@onboard.example.test:2222, new-key name/reserved ID/creation ID, both saved paths, manual method/configuration=no and continuation into registration/manual guidance. Review caused no execution. Esc/Ctrl+C/q exited0; fixture hash remained `9c17ba699e5132a7df37778345e29b7a4789199329281daac2d07d50967217bc`.
- **P3 local deletion:** t6d32x12, Tasks searched deletion-native-review, selected the dedicated deletion row beneath the general integrity warning, then s/Ctrl+S. Review showed 公開鍵のみ・検証用, exact key ID and full vertically wrapped fingerprint `SHA256:RXm/ruZ0eTzRXKwi1AQEDynB0VgHQ2ac9KPSFdf/YnA`. Scroll/End reached local recovery effects and terminal guidance while controls stayed visible; no remote authentication/configuration fields. Esc/Ctrl+C/q exited0 without deleting. State hash remained `95ff3a3d237da66b7589a8dc17c0361c41dbaa8c1fade3a68277310ca5c5394f`; public file hash remained `0f8b90dbc8e0488c19f6985a624809e2a97a09b2401f92d019c7c9e64e84c43e`.
- **P2 picker/P4 entity:** t6g80x24, r opened a related key picker. Pasted key-native-fixture selected the exact public-only key; Review retained destination/key/full fingerprint/manual/configuration=no. Cancel returned without execution. u/Ctrl+S/Enter then no only returned Cancelled before changes. v/F5 now retained registration details under Destinations. A60x18 viewer t6k allowed distinct result and detail scrolling: F5 retained the detail's Connection line at the top, and Esc restored the result's Request line at the top. q exited0. State hash stayed `8fd8590f32439d40febe1d2f92c87aaab2395c495b7f927605b0bbb11e6e41a0`.
- **P3 genuine earlier creation:** t6h60x18 reopened the actual earlier ssh-keygen interruption fixture. Review now showed new-key name native/reserved ID, exact creation-fe523012e7d4005a88c5269e42c9ee47 and both saved paths/reconciliation intent, with no remote options. Cancel/q exited0; hash stayed `89d203be99776b48a2c478db8496be448cc3c45edb614451aaab5c699601af1c`.

### Final-source actual ssh-keygen interruption and restart

At the same ba0f69d binary, t6j80x24 used a newly empty `.tmp/native-final-child/home`. Pasted final-native-child, Review and Continue reached exact local creation approval naming only that fixture. Only this disposable local creation was approved. Actual Windows OpenSSH reached `Enter passphrase (empty for no passphrase):`; no passphrase or empty submission was sent. Ctrl+C restored a live Result with exit255, creation pending checkpoints, unknown/stopped facts and exact creation `creation-b439d13666a17022318e1855ed002432` / reserved key `fixture-reserved-ID-omitted`. The managed key list and key directory stayed empty and state.lock was absent. Result t/F5 retained the explicit creation Task while the category remained Destinations; Esc restored the stopped Result. q exited0.

New process t6n32x12 reopened the same home. Tasks/Enter/s/Ctrl+S showed the same local creation, name, reserved ID and known paths. End reached full guidance with controls visible. Esc/Ctrl+C/q exited0 without resuming execution. State SHA256 stayed `4d8b55aeb4dfe76c8fcd940190a68dd5121fbe7f1dea020fe4b64ca774512906`. This supersedes earlier04 child evidence for the reviewed product source, but does not establish remote SSH/password/host-key or other-OS behavior.

All own terminals t67,t69,t6b,t6c,t6d,t6f,t6g,t6h,t6j,t6k,t6n were exited0 and removed. The independent Spec recheck subsequently found cursor-tick form-scroll regression P5; this ba0f69d checkpoint does not certify its resolution. Long-path Import at32x12 in t6p could not sustain a lower viewport across page/point-in-time observations; no precise tick timing is claimed from that native attempt. Its draft was discarded without execution, and navigation state hash stayed unchanged. Final affected-source native scroll verification is recorded separately below.

## Final AC11 cursor-scroll check (2026-11-08 JST)

Exact source `26269049b747db02988ad00db2b62dcff22119f0`; Windows amd64, Go1.26.2, Herdr0.1.18. Clean-source `go build -buildvcs=false` in t70 produced SHA256 `8038c3a6537f128d4e69f083d39941f66f3d0562a3db2592bdc7153699342c1f`, identical to the worker's independent native build.

t71 used the unchanged navigation fixture at real32x12. Import accepted a long public-key path beginning C:/begin-part/, with25 directory segments and final-public-key.pub. With public-path input Field2/6 active, ordinary Left started caret activity; three PgDn presses exposed the path tail. Two `hc read` screens separated by1200ms were identical and retained final-public-key.pub and Field2/6 across natural cursor replies. Left restored Import existing key / Existing public key path / Full value at the top. After paging again, literal bracketed paste q? edited safely; hc send's automatic Enter advanced to private-path Field3/6 without launching an operation. Ctrl+C / Discard draft / q exited0. State SHA256 stayed `24eb5cb3259d3dfd073cbad4d986396595c84b909dcba6d1fb2d4c1ac84e8c31`.

Both own terminals exited0 and were removed. This final check certifies the affected viewport/edit path; automated public tests separately verify unrelated and stale replies. All other post-review native checks above remain explicitly pinned to ba0f69d. The only production change from ba0f69d to2626904 is the Value/Position condition in updateForm; handoff, native execution, target resolution, minimum-size guards, search/picker routing and result refresh are unchanged. Repeating a real child interruption for that viewport-only delta was not necessary. Linux/macOS and real remote SSH behavior remain unverified.

## Published evidence identifier notation

Two randomly generated reserved key identifiers from disposable interrupted-creation fixtures are omitted in this published record. Their spelling matched an API-key detector; they were produced by app.newID using crypto/rand, not provider credentials. Creation IDs, exact source identities, state hashes and observed behavior are retained. No detector exemption or credential was used.
