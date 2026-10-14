---
satisfies: [R4, R5, R8]
---
# fn-178-gomad-multi-instance-clusters-in-one.11 Multi-instance gates, in-process memory budgets, speedup record and docs

## Description
Pass the multi-instance gates with in-process clusters, move qualification budgets to the in-process memory measure, show the speedup against the fn-172 baseline, and update the docs (R4, R5, R8). The network gate and cost items moved here from former fn-177.6, adapted to clusters.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/qualification/budget.go`, `tools/gomad/internal/backends/wasm/qualification/cost.go`, `tools/gomad/internal/backends/wasm/qualification/cluster.go`, `tools/gomad/internal/backends/wasm/qualification/*_test.go` (budget tests), `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (gate assertions), `tools/gomad/internal/backends/wasm/README.md`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md` (finalize the Cluster execution section), `tools/gomad/internal/backends/wasm/QUALIFICATION.md`, `tools/gomad/internal/backends/wasm/CLI.md`, `tools/gomad/ARCHITECTURE.md`, `tools/gomad/SPEC.md`, `tools/gomad/README.md`, `.flow/artifacts/`
**Touches:** [tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/README.md, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/ARCHITECTURE.md, tools/gomad/SPEC.md, tools/gomad/README.md]

### Approach
- Gates (run alone; memory budget):
  - `TestTemporalIndependentClustersPartitionHealWholeClusterCrash` (integration_test.go:3594; `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT`), including `temporalReplayClusters` prefix and full replay with byte-equal World evidence
  - `make -C tools/gomad/internal/backends/wasm qualify-workloads`, multi-instance `TestActualTemporalClusterQualification` `success` and `canary` cells, then `TestActualMandatoryWorkloadCorpus`
- Network gates (R8): run the independent-clusters gate and the broker network-stream replay fixture (`TestBrokerIndependentGuestStreamsReplay`) on the Rust network owner. Require distinct choice traces across seeds (`distinctChoiceTraces`). Each gate's retained report names the network owner that ran (in-process Rust or Go reference) and its implementation digest; the independent-clusters gate has no such report field today, so add one.
- Budgets (R5, per fn-178.1):
  - Per-instance guest linear-memory peak against each instance's `memory_bytes`.
  - The cluster helper's peak RSS against the existing conservative resident bound (budget.go:76-82), replacing per-incarnation `HostPeakRSSBytes` samples (`cost.go:116 recordHelperPeakRSS`), mapped as the R5 amendment states (one cluster sample covers every resident guest; warm/cold by compile-cache hit; the 8 GiB resident limit applies, the per-helper 4 GiB warm and 6 GiB cold limits do not).
  - Keep `EvaluateBudget`'s nonzero-sample rule.
  - Update the budget tests.
- Performance (R4): compare cluster authorized execution against fn-172's retained multi-instance baseline at the same seed. If fn-172 retained none, measure the pre-change per-process broker build at the same seed, record that fact, and use it. Record per-seed numbers, including the parent Go process cost (R8).
- Docs:
  - README "isolated engine helper" and "modeled broker" text; V2/broker "Go host model" statements; crash/stop prefix text
  - protocol.md cluster frames, `peak_memory_bytes` semantics, crash/stop interrupt text
  - QUALIFICATION.md multi-instance rows, per-incarnation RSS wording, "broker's single exact clock"
  - protocol.md also: the "one execution and exits" statement (line 5), per-instance versus cluster frame and callback limits, the `getrusage` fields now being cluster-wide (lines 192-194), and "one store instance is admitted" (line 249)
  - QUALIFICATION.md also: the 41-57 and 88-93 per-process RSS and upper-bound prose
  - CLI.md: multi-instance replay (153-183), with whether retained artifacts record the cluster path; cost and budget text (237-267); fixture cost table note (362-380)
  - an ARCHITECTURE.md note that World and the clock have a Rust duplicate kept equal by differential tests, while Go stays authoritative for native and the runner. The single-helper WASM cluster claims Model Fidelity only, not Hard Isolation (ARCHITECTURE.md 62-64, 104-106; confirm SPEC.md 106, 340)
  - `tools/gomad/README.md:10-11`: shared clock sources bound through the facade; add the Rust duplicate and how the helper digest binds it

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:3594-3830`
- `tools/gomad/internal/backends/wasm/qualification/temporal_cluster_test.go:153`, `tools/gomad/internal/backends/wasm/qualification/cluster.go:228`
- `tools/gomad/internal/backends/wasm/qualification/budget.go`, `tools/gomad/internal/backends/wasm/qualification/cost.go:100-140`
- `tools/gomad/internal/backends/wasm/Makefile:55-80`

## Acceptance
- [ ] The independent-clusters partition/heal/whole-cluster-crash gate passes in-process, with byte-equal World evidence across prefix and full replay.
- [ ] Multi-instance qualification `success` and `canary` cells pass within the updated budget.
- [ ] Network gates pass on the Rust owner with distinct schedules across seeds, and reports name the network owner.
- [ ] Cluster authorized execution is lower than the recorded baseline at the same seed on every measured seed; numbers, including parent Go process cost, retained under `.flow/artifacts/`.
- [ ] Docs updated; no stale "one helper per instance" or "broker uses the Go host model" statements remain for in-process clusters.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
