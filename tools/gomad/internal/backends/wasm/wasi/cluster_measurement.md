# fn178 matched release measurement preparation

Source preparation only, 2026-10-15. No Go/Rust build, test, execution gate or performance measurement has run. No measurement or review pass is claimed. The conductor owns measurement approval and Flow lifecycle. The code-style skill guided reuse of existing admission, fixture, observation and choice-diversity helpers.

Workspace: `/Users/stephan/Workspace/skunkworks/gomad-fn178-measurement-20261015`.
Retained base: `15b491281cdafbfcd7e8b6c8ea4447ac8fbc6208`.
This is an instrumented descendant, not the unchanged historical binary. `process.go` is a model identity input. Its host-only endpoint patch rotates the model from `sha256:42c1f68b5ffb266c50fe93d0cdf73bcfec7afa2e01fa6fb5cb95838a06d56e46` to `sha256:9d41b55ab401bc8424a3a3b266d065658768f424a63f89b3c9963cff568d6ff2`. The identity test pin was updated by reproducing the source algorithm, first checking that the unmodified base exactly reproduces its existing pin. The compiled identity test still must run after the hold. Helpers, modules and evidence must be freshly authenticated under the instrumented source identities.

## Producers and boundaries

`TestBuildTerminalExitReleaseHelper` is the existing authenticated release build producer: pinned compiler, locked Cargo inputs, no default features, empty helper features. The new `TestMatchedReleaseClusterBaseline` authenticates that release receipt and the raw Cargo execution/version evidence. It deliberately does not change `diagnosticHelperCommand`, whose nonempty-feature rule remains intact. `TestDiagnosticClusterBaseline` and diagnostic-profile clusters are not used for release comparison.

The actual `temporalIndependentClustersPartitionHealWholeClusterCrash` fixture runs unmodified assertions and the same SQL/SQLite workload, including directional partition/heal, commit, whole-cluster crash, durable reopen, interrupted-prefix replay and full replay. The only fixture change assigns `brokerConfig.Seed = request.Runtime.Seed` after `temporalBrokerConfiguration`, so seeds 7, 19 and 42 change the broker as well as the guest. Seed 7 preserves the historical fixture inputs.

Each seed gets a fresh isolated compile-cache directory, one cold-or-warmup scenario (repetition 0), and three intended warm scenarios (1–3). Report actual per-helper `CacheHit`/compile metrics when available; modeled killed helpers may have no terminal telemetry, so repetition labels alone do not prove warm cache hits. Every observed warm terminal must report a cache hit and zero compilation. The producer reports the three warm measurements, median and min/max for each seed; it retains all raw pair costs and identities. It uses existing `distinctChoiceTraces` on the same active node's reopened recording across three seeds and requires at least two distinct traces. The original fixture supplies exact replay and World equality checks.

Primary comparison for each complete scenario:

```
minimum(actual helper authorization across every phase)
    -> maximum(final classified AND owned-reaped result across every phase)
```

This is a single elapsed envelope, not a sum of overlapping helper durations or of separate phase envelopes. It includes preparation, oracle, publication and control gaps between phases. Baseline authorization uses the existing `frameSink.authorizedAt`, immediately before sending `authorize`. A new outermost `Run` defer records `executionClassifiedAt` after all existing deferred classification, `Broker.Finished`, partial-evidence capture and owned process reaping. Missing real validated-terminal timing on a killed modeled crash stays missing; it is never replaced with a fabricated validated terminal. Endpoint fields are private and explicitly `json:"-"`; they do not enter canonical semantic evidence, though their source changes correctly rotate the model identity.

Whole-scenario elapsed and parent `RUSAGE_SELF` before/after snapshots surround the fixture plus its cleanup. Pair dispatch wall/CPU and exact pair authorization-to-classified envelopes are secondary observations. Parent CPU includes orchestration and helper preparation waits, but excludes child/compiler CPU. Helper process metrics and wait usage are separate. Helper wait RSS can inherit an earlier compiler high-water; use the helper's own metrics for actual image RSS when available.

## Candidate binding

`cluster_measurement_candidate_test.go` is a typed collector enabled with the explicit `gomad_cluster_candidate` tag after the client joins. It accepts an actual `CompletedResult()` plus absolute actual cluster-authorization and final classified/reaped endpoints. It retains `AuthorizationToTerminalNanos`, `WallElapsedNanos`, `Metrics`, `HelperResourceUsage`, `Frames` and every `Scheduler` field without changing replay-visible evidence. It refuses missing process/scheduler observations and any guest callback.

The client owner must bind that collector to the migrated actual fixture. The current producer uses two persistent sessions: one helper for crash plus reopen, and one helper for prefix crash replay plus full reopen replay. This keeps both recovery chains inside their Rust owner; the client does not admit externally reconstructed recovery images. The candidate envelope adapter accepts any nonempty completed process list, while the scenario producer proves all four phases actually ran. Candidate primary is minimum **actual cluster helper authorization** across those processes through maximum final classification/reaping, including final World validation. Per-instance `Start` send timestamps are secondary conservative bounds that include command admission, not the primary authorization boundary. Never compare a sum of baseline pairs to one candidate session or omit interphase gaps from only one side.

Use identical seeds, repetitions, fixture arguments, workload data and output checks on both adapters. Host/boot/CPU, full source and instrumentation patch, parent executable, helper/module/runtime/engine/model identities, exact commands and raw rows remain required. The candidate's network owner is `in-process-rust`, baseline is `go-reference`; bind the implementation digest to each report. Do not present the typed collector as a completed candidate scenario producer before that binding is implemented and exercised.

For task .10 retain `handoffs`, `handoff_samples`, `handoff_nanos`, `active_nanos`, `overlap_nanos`, `operation_samples`, `runnable_samples`, `guest_nanos` and `host_nanos`. Guest residual includes ABI and unattributed runtime wall and is only an upper bound; it is not guest-only CPU. Grant-to-wake timing divides by matching `handoff_samples`, never by guest idle count unless those domains are shown equal. Re-measure current host baton-switch cost during the isolated window and multiply that cost by guest idle handoffs. If its share exceeds 5% of primary authorized execution, record the fibers decision and its async-import/`call_async`/Cargo.lock scope. The historical 13 microsecond spike is historical only. A best-case parallel speedup of at least 1.5x can justify a scoped follow-up recommendation, never a parallel scheduler change in this batch.

## Commands after explicit resource release

Every shell call first enters the assigned batch root to satisfy the host's BASH_ENV guard, then the measurement workspace. The worktree is currently sparse (`tools/gomad`, `.flow`); actual fixture preparation also needs the retained Temporal repository and schema. Expand checkout only after disk approval. No cache or target tree was created by this preparation.

First run focused source controls with the retained stock Go (not run yet):

```bash
cd /Users/stephan/Workspace/skunkworks/gomad-fn178-batch-20261015
cd /Users/stephan/Workspace/skunkworks/gomad-fn178-measurement-20261015/tools/gomad
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go \
  test -tags test_dep -count=1 -timeout=10m \
  -run '^(TestImplementationIdentityBindsCompiledModelDigest|TestClusterMeasurementEnvelope)$' \
  ./internal/backends/wasm/wasi
```

For the heavy producer set the following explicit absolute paths in the isolated measurement shell. `TMPDIR` must name the conductor's admitted native/tmp directory. `GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT` and `GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT` must be fresh external evidence directories with sufficient disk. `GOMAD_WASM_TEST_HELPER` and `GOMAD_WASM_CLUSTER_HELPER_ADMISSION` must be the actual immutable staged paths emitted by the release build, not guessed aliases. Also set explicit `GOMAD3_STOCK_GO` to the retained stock executable above. Preserve every environment value alongside captured command/exit receipts.

```bash
cd /Users/stephan/Workspace/skunkworks/gomad-fn178-batch-20261015
cd /Users/stephan/Workspace/skunkworks/gomad-fn178-measurement-20261015/tools/gomad
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go \
  test -tags test_dep -count=1 -timeout=90m \
  -run '^TestBuildTerminalExitReleaseHelper$' ./internal/backends/wasm/wasi

env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go \
  test -tags test_dep -count=1 -timeout=6h \
  -run '^TestMatchedReleaseClusterBaseline$' ./internal/backends/wasm/wasi
```

Capture each actual command's output once and retain its exit code. Do not infer success from a summary string or a zero-test selection. Candidate execution command and full correctness/resource gates remain the conductor/client owner's integration work; this preparation supplies no performance acceptance.
