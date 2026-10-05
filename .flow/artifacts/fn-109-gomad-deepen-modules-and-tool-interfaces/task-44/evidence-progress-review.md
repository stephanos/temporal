# fn-109.44 independent evidence-progress review

Assessment is `SOURCE_PROGRESS_COMMIT`, conditional on root completing the final Flow/MILESTONES status synchronization and retaining this source freeze in the separate progress commit. I found no introduced P1, P2 or P3 evidence-integrity or overclaim finding. Required lint remains red. This bounded review supplies no formal implementation-review receipt, SHIP, merge or DONE verdict.

## Review identity and boundary

AGENTS.md selects `gpt-6.1-sol` at high for the reviewer. The dispatch explicitly selected that reviewer in a fresh context, intentionally from the writer's model family. This statement describes the explicit selection; I make no independently verified execution-backend claim.

The independent freeze-review interval is `2026-10-05T19:33:18.500Z` through `2026-10-05T19:37:16.770Z`. Complete source/tool checks ran at its start, before writing at `2026-10-05T19:35:51.948Z`, and after writing at its end. Branch is `gomad`. HEAD at all three checks was `6f7631ac560a117f1300c1e64b0a9fd5d3e22e63`. Admitted source BASE is `13df4f16f90d49938ea123a29859da62ad2cab9f`.

| Candidate under tools/gomad3/internal/compatibilitypack | SHA-256 at all three checks |
| --- | --- |
| schema.go | 449fb233c61b6a3691c913dde04e1630cfb1f4e9b2fffea82dacfeb18251f0ed |
| mutation_test.go | b6552e443ae5df2b1c6e3e8f97e52a96a06ea61a6afaead86103549c8b3ba185 |
| schema_timezone_test.go | 5a055e7ce61f18b9fbd3fb16f68daddbe7cb1a66c6fa039973aecebf9b53c2eb |

I read AGENTS.md, `flowctl usage`, current task 44, the full parent spec, task 11/21 ownership and acceptance, and relevant README, Makefile and MILESTONES contracts. I inspected admission, capture/verification/comparison scripts, all command captures, before/final manifests, preservation proof, worker handover/evidence, root integrated handback and the separate source review. Read-only Git/Flow/file operations and Node comparisons in memory supplied the checks below. I launched no Go, test, lint, build, generator, download or cache operation. This artifact is my sole write; root retains source, Flow, index, history and commit ownership.

## Independent preservation checks

I compared all 1,264 before.json file hashes with the complete corresponding Git blobs at the admitted BASE using `git cat-file --batch`. Every hash matched, including the three candidates, protected source/config/dependency inputs, and historical task 42/43 evidence. This verifies the complete selected inventory against Git, beyond a protected-file sample.

I independently hashed every current selected file and all five executables at all three freeze checks. Current file hashes match final.json; the five current executable hashes match before.json and final.json. Exactly the three candidate files differ between the 1,264-entry manifests. All 1,261 protected entries are unchanged. The documented selection predicate identifies exactly 51 generated/pin paths, whose before/final/current bytes match. This is the selected-input inventory, not a full-repository or full-toolchain qualification closure. gci is the configured embedded analyzer bound by the golangci-lint executable hash; no standalone gci executable is claimed.

I reconstructed each complete candidate from its BASE Git blob using only the three admitted conversions and one import regrouping. Each replacement matched exactly once; all reconstructed files equal current bytes. Together with the manifests and independent raw comparisons below, this reproduces all 13 preservation-proof checks rather than accepting their stored booleans.

The source/destination structs have the same ordered string fields, differing only in JSON tags. The expressions still produce `Source` or `ForeignSource` values. Allocations, capacities, indexed/append loops, nil/empty handling, activation order and detached element storage retain their original code. The foreign-source `Kind + ":" + Name` projection and DigestSources' NUL framing remain unchanged. Exact reconstruction also preserves every existing comment, error, fixture and assertion outside the four admitted sites. Protected policy/admission/grant files retain their bytes, including the five-import ban and the three uppercase `Go` ST1005 error strings.

## Raw command observations

For every capture I checked raw argv/cwd, HEAD, status, null signal, timestamps, measured elapsed seconds and before/after identities. All six per-command file hashes and five tool hashes match the appropriate BASE or final manifest. BASE lint/packages ran at source HEAD 13df; later BASE target controls ran at admission HEAD 6f7631ac56 with the identical BASE source hashes. Every capture's before/after identities match. Recorded worker/root commands are serialized by their capture intervals.

| Capture | Actual exit and terminal test events | Measured seconds |
| --- | --- | --- |
| baseline-lint.json | 1; seven diagnostic blocks | 2.094436 |
| baseline-packages.json | 0; 469 pass, 0 fail, 1 skip; three packages pass | 0.611822 |
| baseline-target-consumers.json | 1; 38 pass, 1 fail, 0 skip; target fails, capabilitypolicy passes | 0.565159 |
| baseline-target-controls.json | 0; 37 pass, 0 fail, 0 skip; two packages pass | 0.269458 |
| final-validate.json | 0; check-only validation | 6.045494 |
| final-packages.json | 0; 469 pass, 0 fail, 1 skip; three packages pass | 1.038801 |
| final-target-controls.json | 0; 37 pass, 0 fail, 0 skip; two packages pass | 0.944342 |
| architecture-purity.json | 0; six pass, 0 fail, 0 skip; root package passes | 46.061186 |
| formatting.json | 0; empty stdout/stderr | 0.018118 |
| errortype.json | 0; empty stdout/stderr | 1.109771 |
| final-lint.json | 1; three diagnostic blocks | 2.727705 |
| source-diff-check.json | 0; empty stdout/stderr | 0.022582 |
| root-integrated.json | 2; 319 diagnostic blocks | 132.948745 |

I parsed raw Go JSON lines without discarded nonempty lines and recomputed run and terminal events. BASE/final packages each contain 470 run events and 469 pass/one skip events. Their sorted Package/Test/Action terminal multisets match exactly. The bounded target captures each contain 37 run/pass events, no failures/skips, and exactly matching terminal multisets. Package terminal statuses also match. These event counts include subtests and do not claim 469 independent top-level tests.

The actual package skip is `TestHostPacksBindCurrentProfile`. Both outputs say `no deterministic profile for linux/arm64`. The broad BASE target failure is `TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests`; its raw output retains the failed fork/exec of the absent patched `.toolchain/bin/go`. The later bounded selection uses `Test(Evaluate|BuiltInSimulation|CapabilityReviewGoldenCanonicalBytes|PreparedCacheDigest)`, distinct from the failed broad `Evaluate|Simulation|Canonical|PreparedCacheDigest` selection. Its green result does not qualify the failed command or full affected-consumer acceptance.

The architecture capture has exactly six passing named tests covering package architecture, public aliases, host effects, private Runner injection, exact module edges and public wire framing. The package captures retain existing generated mutation, complete-inventory selection, DecodePackV2, governance canonical/error-priority, policy-copy and five-import controls. Existing passing tests are preservation controls. The actual configured analyzer findings provide the causal RED for this mechanical correction; no new behavioral regression test is claimed.

Final validation is the first captured command with candidate hashes, running `19:24:22.645Z` through `19:24:28.796Z`, before final package and consumer captures. Makefile and stdout agree on version/protocol/boundary `-check`, compiler-test check, patch/script validation, compatibility-pack `check` and qualification-manifest `-check`. No regenerate/pin-refresh command appears. Its ordinary profile-test `ok` output supplies no native proof; the complete package JSON retains the profile skip. The retained diff-check exit is a tracked-source check and does not establish whitespace validation of then-untracked artifact content.

## Actual lint comparison and stage reachability

I independently split the scoped and integrated raw outputs into complete diagnostic blocks and compared block/header multiplicities. Scoped lint actually changes seven to three, retaining the same three ST1005 blocks byte-for-byte. Integrated lint actually changes task 43's 323 to 319. Both comparisons remove exactly these four headers and complete bodies.

- mutation_test.go:98:23 S1016
- mutation_test.go:102:28 S1016
- schema.go:292:29 S1016
- schema_timezone_test.go:5:1 gci

There are zero added blocks, zero shifted headers and identical multiplicities for every retained header. All 319 retained integrated blocks are byte-identical. Independently recounted residuals are 252 errcheck, three exhaustive, 11 forbidigo and 53 staticcheck. The three compatibility-pack ST1005 blocks and error strings remain present.

The independently hashed task-43 baseline log is `358ea9f49ea93999fc2f6fca63ebad6d8beda3cd81731b12851f509facbd3cd7`; root candidate stdout is `3fb126eb1e38a595c2812cdf0e01e9eb87704585f560ec6fb1adce5263e7ba8f`. These hashes/counts agree with root-integrated-comparison.json. The script's stored output was corroborated by the independent complete-block comparison.

Root's exact captured argv invokes `make --trace lint-code-gomad3` with original comparison base `951c5516e9e7b3066e7e069adda9565cfd68844c`, fix=false, and the pinned golangci-lint/errortype paths. Raw Make output selects exactly 55 ordinary host-package arguments and retains `disable_grpc_modules,,test_dep,`, the repository config and the same base used by task 43. Stderr names golangci-lint 2.13.0 built with Go1.27.1 and the configured embedded gci analyzer. No analyzer suppression/config/pin change belongs to this candidate.

The capture interval is `2026-10-05T19:28:16.381Z` through `2026-10-05T19:30:29.743Z`; monotonic command elapsed is 132.948745314 seconds. The 133.362-second UTC interval also includes capture identity/hash overhead, consistent with capture.mjs. Raw stdout/stderr retain recursive Make error 1, module-router exit status 2, and actual top-level Make exit 2. The Make recipe stops before its later errortype `go vet` invocation. Integrated errortype is `UNREACHED`; the scoped standalone errortype exit 0 remains separate evidence.

The retained worker terminal-process capture exits 0 with `live_lane_commands: []` before the root gate. All reviewed captures have terminal statuses and null signals. Root's final verification at `19:30:52.678Z` binds the same 1,264 selected inputs and five executables after that gate; my independent complete current rehashes confirm those bindings.

## Ownership and remaining acceptance

Current Flow state read during this review has task 44 `in_progress` and task 21 `blocked`, with task 44 a direct dependency of task 21. MILESTONES currently reports task 44 in progress. Root must synchronize final blocked/open acceptance and measured progress before committing. Worker handover/evidence and the separate source review retain earlier pending-root snapshots. root-integrated-lint.md supersedes their pending gate/review status only; it does not replace their raw results or close qualification.

Task 11 keeps R17, task 42 keeps exhaustive correction, task 43 keeps R21, and task 21 consumes task 44 under R18/R19. Original matched-first-baseline, predecessor, preservation/fixed-identity, full/default/functional/affected-consumer, native Darwin, static both-source-set and formal requirements remain required and open wherever unproved under their owner. The selected preservation proof does not substitute for original aggregate preservation acceptance. Native Linux remains unverified and nonblocking under fn-128. Developmental linux/arm64 checks supply no native qualification. Historical task 42/43 bytes remain protected.

No introduced actionable integrity/overclaim finding remains in the reviewed candidate. Root may commit this bounded source/evidence progress after the final status synchronization. Required lint still fails; formal implementation review and completed acceptance remain deferred.
