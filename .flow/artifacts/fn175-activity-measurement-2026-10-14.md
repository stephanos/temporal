# fn-175.6 activity authorized-execution reduction (Linux/arm64, 2026-10-15 UTC; host local date 2026-10-14)

Owner policy: MILESTONES "WASM performance acceptance host". Baseline and candidate ran on the same real
Linux/arm64 host and boot (`ad1ec4fe-2bb8-4c96-bf7f-a714910117e6`) with the host quiet (other workers suspended),
alternating paired fresh `go test` processes in their own trees.

## Verdict

- **R4: PASS on every seed.** Candidate median authorized execution (authorization to validated terminal) is
  0.583-0.622 × the pre-fn-175 baseline (37.8-41.7 % faster; bound ≤ 0.75 ×, i.e. ≥ 25 %).
- **Identical choice traces / evidence:** every measured sample of a seed (5 baseline + 5 candidate) has the
  same `EvidenceSHA256`, `EnvironmentIdentity`, `Calls` and `TimeAdvances`; each seed has a baseline-tape replay
  on both trees (8 replays), all accepted.
- **R1 Go filesystem counter:** `GoFilesystemCalls` is present and empty (`{}`, `CounterPresent=true`) on every
  candidate run.
- `measurement.json`: `accepted=true`, `failure=""`.

## Per-seed results (median of 5, after one warm-up per role)

| seed | baseline median ns (min-max) | candidate median ns (min-max) | ratio (≤ 0.75) | reduction | result |
| --- | --- | --- | --- | --- | --- |
| 0 | 2,464,085,168 (2,272,640,376-2,573,244,126) | 1,476,075,084 (1,402,822,167-1,566,047,918) | 0.599 | 40.1% | PASS |
| 1000003 | 2,498,067,876 (2,185,113,126-2,789,748,085) | 1,457,553,542 (1,415,784,334-1,588,081,126) | 0.583 | 41.7% | PASS |
| 2000006 | 2,570,710,377 (2,511,031,835-2,967,104,084) | 1,545,112,501 (1,332,388,125-1,631,698,584) | 0.601 | 39.9% | PASS |
| 3000009 | 2,544,137,042 (2,186,918,834-6,134,312,378) | 1,583,435,792 (1,361,601,876-2,313,726,167) | 0.622 | 37.8% | PASS |

Evidence per seed (EvidenceSHA256 / EnvironmentIdentity / Calls / TimeAdvances, identical across all 10 measured samples):
- seed 0: 1 distinct tuple(s): `8d3ef930df665480f4def2b259a508b4703cba3069b6cd2fc235defa3ab45474` / `776330d5e037effbbaa02532a37b1a16b445deaac1601743792204201c900e9a` / 209441 / 264
- seed 1000003: 1 distinct tuple(s): `6c93493eefaaf3cc2a210316d3cba5b8c4a2b5bdd0e8558e3b29d656539f2e5b` / `776330d5e037effbbaa02532a37b1a16b445deaac1601743792204201c900e9a` / 221077 / 270
- seed 2000006: 1 distinct tuple(s): `31394ce8abc08b1259074bdb4056dfca07c09d6759acb81fd57810c922985df3` / `776330d5e037effbbaa02532a37b1a16b445deaac1601743792204201c900e9a` / 218663 / 268
- seed 3000009: 1 distinct tuple(s): `f32e43f647601264266eddbadfbc8a4ed468c2cc9b724941be8ef53d85373170` / `776330d5e037effbbaa02532a37b1a16b445deaac1601743792204201c900e9a` / 215730 / 213
- candidate GoFilesystemCalls: ['{}'], CounterPresent: [True]
- replays: 8 (baseline@0, baseline@1000003, baseline@2000006, baseline@3000009, candidate@0, candidate@1000003, candidate@2000006, candidate@3000009)
- collection order entries: 56

The ranges overlap only in the seed 3000009 baseline/candidate maxima (one slow sample each: 6.13 s baseline,
2.31 s candidate); every candidate median is below every baseline minimum for seeds 0-2000006, and the seed
3000009 candidate median (1.58 s) is below its baseline minimum (2.19 s).

## Host

`Linux 7.0.14 aarch64`, 12 CPUs, 15 GiB, no swap; cpuinfo CPU architecture 8, implementer 0x61,
Features `fp asimd … i8mm bf16`, bound in both `*_source` records of `measurement.json`.

## Identities

| | Baseline (pre-fn-175) | Candidate |
| --- | --- | --- |
| Source | `fast-baseline175` `fdbe843c5c` (= fn-174 base `3858b64690` + the two collector files) | `measure` `15b491281c` |
| Model sha256 | `sha256:fa1652bd6175bbc5ee79d4b4ed73c2239fcd8195d61f8387236a84aa8d328efa` | `sha256:42c1f68b5ffb266c50fe93d0cdf73bcfec7afa2e01fa6fb5cb95838a06d56e46` |
| Release helper sha256 (admission receipt) | `4cfadc2107a77b0c05b09d299b5e434e121ae3886868748b1bbb0015fb2f0185` | `52053e57a4f5e1b623977b3631ee160128667e008004d5669fae0e50a36d5903` |
| Module (shared) | `5835781d9896516a4a6cf1b3ab5dc10fc7fe8cb14a92297776d7c93e13d0f56c` (`/home/agent/wt/perf-measure-20261014/module/temporal-module-1750913189`) | same |
| Toolchain build key | `1d17508fb2df7192771aaf163e092682f39b347bb52ae2be1ee0bf3170f97f7e` | same |
| Warm cache | `cache-baseline/9c37efc2a0d1aa1a1ea6ba8e8a6311a528fb46e7230e376b2f5c300b17ab5551.gomad-cache-v3` | `cache-candidate/e5f39153944288c6cdadde94f00eef69d9d6084dfd2875866cc44a815604849b.gomad-cache-v3` |

Selection `^TestActivityTestSuite$` (sqlite), seeds 0, 1000003, 2000006, 3000009.

The R4 numbers come from the source-matched namespace collector (`TestNamespaceActivityMeasurement`), not from
`TestHelperMatchedSuiteQualification` timings: that gate's baseline sets `remoteModel=true` (the fn-156
all-Go comparison), not the fn-174 local-model pre-fn-175 baseline. The matched gate was used only to prime the
two caches (workflow suite, its timings unused).

## Evidence paths

- `/home/agent/wt/perf-measure-20261014/fn175/namespace-paired-644753816/measurement.json` (per-sample receipts in `raw_receipts`, `collection_order` with all 56 entries)
- Log: `/home/agent/wt/measure-logs/fn175-paired.log`; helpers `/home/agent/wt/perf-measure-20261014/helper-{CAND,BASE175}/`

## Rerun

`/home/agent/wt/fast-RUNBOOK.md` "fn-175 R4" steps 1-4 (patched: step 2 needs `GOMAD_WASM_TEST_HELPER`; step 3
cache dirs must be mode 0700), or `RERUN=fn175-paired ONLY=fn175-paired /home/agent/wt/measure-all.sh`. Core command:

```sh
export GIT_DISCOVERY_ACROSS_FILESYSTEM=1 GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1
GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go
export PATH=$(dirname $GO):$PATH GOMAD3_STOCK_GO=$GO GOMAD_WASM_STOCK_GO=$GO; unset GOROOT GOMADSEED GOMAD3_CHILD_SEED
M=/home/agent/wt/perf-measure-20261014
cd /home/agent/wt/merge/tools/gomad && \
GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT=$M/fn175 \
GOMAD_WASM_NAMESPACE_BASELINE_SOURCE=/home/agent/wt/fast-baseline175 \
GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT=$M/helper-BASE175/release-helper-2255955692/admitted-helper-3561190554/helper-admission.json \
GOMAD_WASM_NAMESPACE_HELPER_RECEIPT=$M/helper-CAND/release-helper-87990216/admitted-helper-3193458758/helper-admission.json \
GOMAD_WASM_NAMESPACE_BASELINE_CACHE=$M/cache-baseline GOMAD_WASM_NAMESPACE_CACHE=$M/cache-candidate \
GOMAD_WASM_NAMESPACE_MODULE=$M/module/temporal-module-1750913189/prepared/module.wasm \
GOMAD_WASM_NAMESPACE_MODULE_RECEIPT=$M/module/temporal-module-1750913189/temporal-producer.json \
GOMAD_WASM_NAMESPACE_STOCK_GO=$GO \
$GO test -p 1 -tags test_dep -count=1 -v -timeout 4h -run '^TestNamespaceActivityMeasurement$' ./internal/backends/wasm/wasi
```

Inputs: the helper receipts come from `TestBuildTerminalExitReleaseHelper` in each tree
(`GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT=$M/helper-<tree>`); the module from `TestPrepareTemporalTerminalExitModule`
(`GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT=$M/module`, `GOMAD3_STOCK_GO=/home/agent/wt/merge/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go`,
`GOMAD_WASM_TEST_HELPER=<candidate release helper>`); the caches (mode 0700) were primed by
`TestHelperMatchedSuiteQualification` with `GOMAD_WASM_MATCHED_SUITE=workflow`.

## After the measurement

- Test-file fold (fn-175.6 follow-up): commit `6be76036ed` moved `host_call_profile_test.go`,
  `namespace_measurement_test.go` and `namespace_measurement_controls_test.go` into `wasi/integration_test.go`
  and `namespace_conformance_test.go` + `namespace_routing_test.go` into `wasi/namespace_test.go`. Test names are
  unchanged (`go test -list '.*'` identical, 255 names); wasi has 13 test files.
- `make -C tools/gomad test-wasm` on `6be76036ed`: exit 0 (log `/home/agent/wt/measure-logs/test-wasm.log`).
- Docs: `wasmhost/protocol.md` and `README.md` describe the helper-owned namespace ("Go owns descriptors and files
  except in helper-owned namespace runs (see "Helper-owned namespace")"); no stale unqualified statement remains.
