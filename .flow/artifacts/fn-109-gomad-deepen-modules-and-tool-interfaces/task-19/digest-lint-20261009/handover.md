# Task19 bounded architecture digest repair

The two admitted source-digest writes are lint-clean and preserve their identities. `checkStartupSource` and `checkMemorySource` now use `_, _ = digest.Write(fmt.Appendf(nil, "%x  %s\n", sha256.Sum256(data), entry.Name()))`. Each production file differs from HEAD21b30b4604788c9e2e54b6c33f3e47045725daf9 by that statement alone. Existing comments, enumeration, selection, read errors, diagnostics, pins, public APIs and generator inputs remain unchanged.

The additive `TestSourceDigestFraming` exercises both real loops with files created in reverse order, a non-Go file and a directory ending in `.go`, then changes one included file and requires rejection. Its literal expected digest is `67ad170a9788e8b8d82ab27c22a54c6f98af43713537fa5775a11f1424d4bcd5`. [preservation.mjs](preservation.mjs) independently checks the complete ordered literal framing and file hashes with Node SHA-256. The test uses a private fixture import key, restores its temporary map entries, and changes no checked-in pin. These controls catch order, framing, omitted-entry, incorrectly included-entry and changed-source errors.

## Verification

Every receipt binds exact argv, cwd, environment, source/tool/control manifests, output hashes, elapsed time and terminal result. [evidence.json](evidence.json) indexes all 21 write-once receipts, including failed and supplementary results. Final source manifest is `source-eb4136469addda8629a61bab47a4a95c015ca60f1547dc17b10edb048b93ecfd.json`. Actual execution is stock Go1.27.1 linux/arm64 on aarch64, supplying developmental source evidence.

| Check | Actual result |
| --- | --- |
| Six existing identity/purity/initialization controls plus literal fixture, before and after | Each exit0, seven top-level tests, 22 named passes, zero failures/skips |
| Complete architecture checker package | Exit0, 27 top-level tests, 183 named passes, zero failures/skips |
| Root architecture/public-signature/purity/edge/ownership checks | Exit0, seven top-level tests, 14 named passes, zero failures/skips |
| HostPackageVet inventories | 55 complete packages each for darwin/arm64, linux/amd64 and actual linux/arm64 |
| Explicit affected-package source listings | Exit0 for darwin/arm64 and linux/amd64 |
| Check-only `make -C tools/gomad3 validate` | Serialized replacement exit0; generators, patch/overlay, script ownership, current packs/profile and qualification manifest checked |
| Standalone errortype | Exit0 on `./internal/gomadtool/architecture` |
| Unfiltered scoped pinned analyzer | Actual RED exit1 with two errcheck findings; GREEN exit0 with zero findings |
| Actual `make lint-code-fast`, base21b30b4604 | Exit0, 55 host packages, zero reported issues; errortype reached |
| Actual original-base integrated lint, base951c5516e9e7b3066e7e069adda9565cfd68844c | Exit2 before/after, 215 to 213 findings; errcheck160 to158, exhaustive2, forbidigo9 and staticcheck44 unchanged; errortype unreached |
| Actual later-base integrated lint, based635e23f00d926a43b942f25a9d05bd0ccb72025 | Exit2 before/after, 24 to22 findings; errcheck20 to18, forbidigo1 and staticcheck3 unchanged; errortype unreached |
| Formatting and `git diff --check` | Exit0 |
| Corrected preservation proof | Exit0; both complete integrated stdout reports equal their baselines after removing only the two exact diagnostic blocks and updating aggregate counts; 401 original test/pin/module inputs match baseline21b30b4604, with the additive fixture bound separately to the frozen final source manifest |

The original-base 215/213 result is authoritative. The 24/22 comparison uses a later filter and retains only that narrower meaning. Each integrated command discovers 55 host packages, runs the pinned full configured analyzer with `--fix=false` and its explicit `--new-from-rev` filter, and fails before its Make errortype recipe. Full stdout/stderr reports remain retained. Standalone architecture errortype does not establish complete integrated errortype coverage.

The preservation proof also confirms the old statement occurs at architecture introduction4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7. `initialization.go` matches that checkpoint before this repair; `standard.go` has later owned changes. This is bounded statement provenance, without a claim of complete original first-baseline acceptance. Earlier task40 default92-green controls remain retained; this developer-checker-only correction required no repeat of those unrelated Go packages.

## Sequencing corrections

An early check-only validation overlapped the package test from 2026-10-09T02:39:42.581Z through02:39:45.397Z, 2.816 seconds. Package handle69978 and validation handle24142 both ended with exit0 and unchanged source/tool/control bindings. The overlapped validation receipt is supplementary. The authoritative validation rerun started after the package's terminal result and passed.

The remaining required gates ran through one foreground synchronous driver, handle32968. It verified a numeric expected child exit plus unchanged source/tool/control bindings before starting each successor. The driver stopped after its final preservation artifact incorrectly expected empty clean-lint stdout; the analyzer actually prints `0 issues.\n`. The original failed proof receipt and byte-exact [script preimage](preservation-preimage.mjs) are retained; its SHA-256 matches the failed receipt's archived control manifest. The corrected artifact also matches its fresh control-manifest SHA-256 and passed under a new write-once receipt, handle99904. No production or test source changed during this correction.

Root then found that the proof selected protected files from the current index. Staging the additive fixture would incorrectly request that new path from the baseline; deleting an original protected file could silently omit it. The authoritative proof now selects the 401 original protected paths through `git ls-tree -r --name-only 21b30b4604788c9e2e54b6c33f3e47045725daf9 -- tools/gomad3`, checks every original candidate path, and separately verifies the additive fixture against the unchanged final source manifest. The prior [index-dependent script](preservation-index-dependent.mjs), [control configuration](controls-before-baseline-inventory.json), historical control snapshots and raw receipts remain retained. New `digest-preservation-baseline-inventory-receipt.json` is exit0 and handle96766 is terminal. Its source/tool/control bindings remained unchanged. Only the evidence script and packet changed; no Go, build, lint or generation gate was rerun.

All unified handles are confirmed terminal. Source/test files stayed frozen for the final gates; the execution lane is free. No staging, commit, Flow lifecycle write, push, PR, CI or toolchain/native execution occurred. The two pre-existing Turbo documents are untouched.

## Acceptance retained by root

Root coordinates an independent source-progress review and any separate progress commit. Formal SHIP and task19 completion remain open while original-base integrated lint is red. No worker source-acceptance verdict is supplied.

Task18's dependency and retained qualification remain intact. Root found 13 of14 entries matching its historical final manifest; the Makefile has later owner changes, so complete current task18 qualification is unproved. All inherited task19 R8/R18/R19, preservation, affected-consumer, first-baseline and formal acceptance obligations retain their original meaning. Task40's genuine pipe-Close fault proof and prior upgrade-publication ENOENT proof remain unproved. Native fn128/fn149 remain deferred and unverified.

The worker requested implementer gpt-6.1-sol/high. Root supplied the task-aware selector result `Tier: session`, with `jev-unavailable(no_key)`; actual executed-model metadata is unavailable. No formal reviewer ran in this worker.
