# fn-221 method-aware proof point

Observation on one candidate only. These seed-zero results are not support or qualification, and the capped root-package proof is not the full sweep.

The clean frozen source was `cc84570e2838dda3ec185764b5a099b284796ec9`. Both runs used fn-220's `temporal-install` configuration, one fresh guest per selected function, the four recorded slim build tags, a 3-minute host execution bound, an 8-minute preparation bound, a 30-minute global deadline per package, and a 2-GiB retained-exemplar bound. The Go global virtual timer was explicitly `-test.timeout=72h`; individual test contexts and host watchdogs were unchanged. Effective transcript capacity was explore's 64 MiB default, output 256 MiB, choices 64 MiB, guest memory 4 GiB, and fuel 2^50. The harness is `tools/gomad/internal/backends/wasm/cmd/wasmsweep`.

| Selection | Top-level parsed / list | Receiver methods | Isolated units | Pass | Not run | Sweep / verify |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| `./tests/gomadfunctional` | 1 / 1 | 0 | 1 | 1 | 0 | 0 / 0 |
| `./tests`, cap 10 units | 148 / 148 | 1,173 | 1,324 | 10 | 1,314 | 0 / 0 |

The method inventory records declarations and registration paths separately. A selected method must produce its own verbose PASS/SKIP; its wrapper's verdict cannot establish a pass. Every selected unit below received a PASS, one attempt, no skip, and exit 0. Other root units are explicitly not-run, not failures; their recorded reason is `not reached by this sweep (budget or --max-tests)`, here caused by the ten-unit cap. The exact names are selectors, not bulk suite executions.

| Selected unit | Wall seconds |
| --- | ---: |
| `TestFrontendSystemInfo` | 5.944 |
| `TestAcquireShardSuite/TestBlockedUpdateShard_AcquiresAfterRelease` | 9.057 |
| `TestAcquireShardSuite/TestDeadlineExceeded_DoesRetry` | 6.077 |
| `TestAcquireShardSuite/TestEventualSuccess` | 5.537 |
| `TestAcquireShardSuite/TestOwnershipLost_DoesNotRetry` | 6.470 |
| `TestActivityAPIBatchCancelClientTestSuite/TestActivityBatchCancel_Success` | 43.653 |
| `TestActivityAPIBatchCancelClientTestSuite/TestActivityBatchCancel_TargetsRunningAndPaused` | 15.254 |
| `TestActivityAPIBatchDeleteClientTestSuite/TestActivityBatchDelete_IncludesNonRunning` | 14.248 |
| `TestActivityAPIBatchDeleteClientTestSuite/TestActivityBatchDelete_Success` | 34.262 |
| `TestActivityAPIBatchResetClientTestSuite/TestActivityBatchReset_DontResetAttempts` | 17.882 |
| `TestActivityAPIBatchResetClientTestSuite/TestActivityBatchReset_Failed` | 7.939 |

Cluster-backed activity fixtures started, so the spec's fixture-start stop condition was not triggered. The two activity `Success` methods each additionally recorded three passing nested subtests. Successful explore bundles were not retained as failure exemplars; the journal and sidecar retain the classified selected verdicts and nested-subtest counts, not raw successful stdout.

## Identity and compile reuse

Both v2 sidecars retain full candidate identities. Launcher SHA-256 was `0e156379fe55beadc82ef304b3d364d8dc89853c2709d246d3834b3dc4711705`, pinned helper `5f04d229dc68dcebeaecb5e50be3de6c3aa148d8157910288217a4f90999757d`, stock Go 1.27.1 compiler `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`, and harness `0ce6067988c5ff56d4d3feea90f00f036fc9cf59d40e3437887cd2b4a086f348`. Engine identity was `wasmtime/47.0.3/cranelift-fuel-nan-canonical-nounwind-v2/linux/aarch64`.

The root candidate key was `sha256:37cdfabf2db01ac5477638f13cd7b855ed8dbd1f0611a138a14411797b9df456`. All ten selectors reused module `sha256:4e19470a4f6f68083f6bea28d3ff61d7ae7317d2799fdde7185e37e42506da78`, with a stable 1,881,896,288-byte preparation cache. Every timed method reported an engine cache hit, zero new engine entries, and 638,501,114 engine-cache bytes. Root warm-up selected no test, exited 0 in 263.010 seconds, and added two engine files; total root invocation wall time was 435 seconds. Functional-package warm-up exited 0 in 184.807 seconds; its timed test also hit the engine cache, with no added entries. Focused backend regression tests separately observe compiler command counts and prove selectors do not invoke compilation again while changed compile inputs invalidate reuse and runtime inputs retain distinct provenance.

## Resume and verification

The identical root command was rerun on the same clean source and pinned binaries after timestamp scratch cleanup. It returned 0 and reported all ten selected methods as already passed. Journal headers increased from one to two; warm-up records stayed one, executed test records stayed ten, and total test records stayed ten. No guest was started and no completed outcome was re-executed. The resumed sidecar verifier also returned 0. Not-run cap rows are synthesized in the sidecar, not journaled as executions.

The retained v2 sidecars are [functional](fn221-method-proof-gomadfunctional-2026-10-20.json) and [root methods](fn221-method-proof-root-2026-10-20.json). Verification commands are `wasmsweep --verify <sidecar>`. Focused sweep/command/installation tests and sweep/command lint returned 0 before the frozen run. Independent source reviews accepted both the compile-only cache and harness after journal-tail/source-identity repairs.

## Reproduction and limitations

The external runner command was:

```sh
env CONFIG_SOURCE=temporal-install PER_TEST_TIMEOUT=3m PREPARE_TIMEOUT=8m \
  OVERALL_BUDGET=30m GOMAXPROCS=2 CARGO_BUILD_JOBS=2 \
  bash /home/agent/wt/evidence/fn221/proof-point.sh /home/agent/wt/fn221
```

Each journal header retains its exact expanded `wasmsweep` argv, including explicit compiler, helper, runtime, output, scratch and artifact paths. For resume, the first root header's argv was replayed unchanged with its pinned harness executable. Raw journals, build logs, identities, host measurements, resume counters/log and binaries remain at `/home/agent/wt/evidence/fn221/proof-point/20261020T161527Z`. The runner returned 0. Timestamp scratch was removed; pinned tools and evidence remain outside the checkout.

Host: `dev-build`, Linux/aarch64, kernel `7.0.14`, 12 CPUs, 21,242,953,728 bytes RAM. Other builds/probes ran concurrently. These are loaded-host execution observations, not performance measurements or a quiet-host full survey. The ten root methods averaged 16.038 seconds; extrapolating that small, ordered sample to roughly 1,488 expected units suggests about 6.6 hours plus preparation, not a guarantee. The approved full-survey plan is 3 minutes per execution, 8 minutes preparation, 24 hours overall, no test cap, only after combined-source freeze and quiet-host coordination. No full sweep has run yet.

The [earlier wrapper-only proof](fn221-proof-point-2026-10-20.md) and its v1 sidecars are historical and cannot establish method coverage. `QUALIFICATION.md` is unchanged by this work.
