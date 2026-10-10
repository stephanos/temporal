# Combined task 64/65 source handover

The six admitted scripted campaigns now pass their original assertions, and the watchdog readiness errcheck is removed. The frozen source remains red overall. Task 63, task 64, task 65 and fn-112.10 retain their source acceptance.

The execution candidate is isolated HEAD `b88c7dcb3c2681e58b2692eee9c49b18026d36fe`. Task 64 maps original `8d8131a66910087879792735eb2940deb6115d86` to isolated `8f584e80e37eb073009fb892d57f45f3fe99f581` and primary `8b13b026302e06b0d356065443b579f35b2006ca`. Task 65 maps original `505f34e627a984b7c4f8bd4d9938beefb79d4cc4` to isolated `b88c7dcb3c2681e58b2692eee9c49b18026d36fe` and primary `b6288b43f248e811990affd7a4a524d33ecc37ec`. Primary `6af88c48ac6ac5c934dfc5e938b380504bbb5c2a` adds their integration notes. Root Git comparison exits 0 across all three Gomad source trees, lint routing, Makefile, config and root module files between that primary commit and the frozen execution candidate; no working changes exist in those compared paths.

| Observed command | Exit | Result |
| --- | --- | --- |
| Ordinary Runner with test_dep/count=1/JSON | 1 | 379 pass, 282 fail, 12 skip across all named levels. |
| Original-base Make integrated lint | 2 | 52 findings; integrated errortype unreached. |
| Darwin/arm64 affected static vet with test_dep | 0 | Runner and execution packages. |
| Linux/amd64 affected static vet with test_dep | 0 | Runner and execution packages. |

The four exact argv, elapsed times, log hashes, terminal handles and freeze checks remain in their named JSON receipts. Root ran no overlapping Go command. Each command ran once, all handles are terminal, and the worker released the shared execution lane.

[outcome-comparison.json](outcome-comparison.json) compares all 658 original Runner names against 673 current names. Exactly the six task-65 admitted tests change from fail to pass. All 652 other original names remain unchanged, none disappears, and no original subtest becomes newly reached. Fifteen outcomes from five new control roots pass. The old all-level counts were 358 pass, 288 fail and 12 skip; neither set of counts denotes top-level tests or full-host qualification.

[lint-comparison.json](lint-comparison.json) retains all original and current complete diagnostic blocks. Exactly the readiness-write errcheck disappears from the original 53; all 52 remaining messages, source lines and carets match, with only the source-proven Runner line relocation 405 to 401. There are zero introduced findings and zero mapping gaps. The raw Make output retains unfiltered `diff: 52/52` and stops before integrated errortype. Worker diff-filtered fast lint and standalone errortype do not close this failing aggregate gate.

Before/after source manifests are byte-identical, SHA-256 `ef815e9f74282cfd44da7698c4a1e5b5d756b660d43fc1fe7a79a1f8aec8e784`. The original 1,070-path scope plus six new source files yields 1,076. Materialized simulation sources and the historical local owner spec expand it to 1,130; eleven supplemental absolute consumed inputs yield 1,141 bindings. All 24 lint-reference blobs match retained task-63 source hashes. Root freshly verified every manifested current file, tool binary, raw receipt hash and before/after equality. The executed wrapper hash is `c83d15c6549da0d81e1efbe15fb6fc18dc9d9642a501ec7c845a29ffc3f911fd`.

The primary owner spec remains bound at `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`; joined-local `0866...` is historical. Root preserved all 26 unrelated owner/Turbo files through primary integration. [index-cache-research.md](index-cache-research.md) records the supported alternate-index continuation and filesystem-cache limits without deleting either primary index path.

Fresh [correctness](independent-correctness-review.md) and [standards/evidence](independent-standards-review.md) reviews accept this packet for bounded source progress. Correctness reports zero critical, important or minor findings. The evidence axis reports no critical or important finding and retains two minor capture limits. Both literal reports remain separate. Writer and requested reviewers are GPT Sol at high effort, from the same GPT family; actual model telemetry is unavailable.

The environment capture is selective while commands inherit additional variables, so the packet establishes no hermetic environment. Original execution binding does not directly seal current Go settings or derived comparisons. The separately labelled post-capture seal preserves their current bytes without proving execution-time binding, historical origin or continuous locking. Historical ordinary effective CGO remains unknown; current default is 1, and only cross-source vet sets it to 0. Historical ordinary argv also selected CLI/campaign packages and an eight-minute timeout. The named comparison is an observed source result, not an identical-environment experiment. Its historical raw log lacks an original execution-time raw hash; current binding preserves consumed historical bytes only.

Tool binaries and resolvable compiler executables are hashed. Complete Go installation, caches, C headers and libc inputs remain uninventoried. These stock linux/arm64 observations supply no patched-toolchain or supported-native test-host qualification. fn-128/fn-149 and native soak bounds remain deferred and unverified. No formal SHIP, Done, native execution, CI, PR or push follows.

The [next explicit fixture slice](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/next-scripted-slice.md) identifies ten remaining calls eligible for an explicit subsequent admission. Its current preparation failures do not predict downstream assertion passes. The [acceptance map](../../fn-112-gomad-determinism-assurance-and-test/task-10/post-task65-acceptance-map.md) records the earlier, pre-batch observation boundary; this packet supplies its later observed counts without rewriting that report.
