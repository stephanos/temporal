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

Integrated batch: follow the .8 amendment. Fresh baseline/candidate comparison uses the retained fn172 revision `15b491281cdafbfcd7e8b6c8ea4447ac8fbc6208` on the same current Linux/arm64 host; current fn176 is an additional incremental baseline, not a replacement for retained fn172. Resolve the diagnostic-cluster refusal versus matched release measurement deliberately; do not compare incompatible cost domains. Keep authorized wall, parent Go CPU and helper/compiler CPU separate. Preserve every original gate, resource bound and multi-seed schedule-diversity requirement. Root serializes final measurements after fn176's qualification exits.
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
Gates, budgets, speedup, docs (R4/R5/R8). In-process TestTemporalIndependentClustersPartitionHealWholeClusterCrash passed with prefix and full replay byte-equal World evidence; owner report two-cluster-owner.json names rust-cluster + helper digest. Multi-instance qualification success (qualified, within budget, cluster helper peak RSS 3.86 GB <= 8 GiB) and canary (intentionally unqualified, within budget) passed, plus every other primary cell, TestActualMandatoryWorkloadCorpus, 4 SQLite selected states, relocation and catalogue. Speedup (R4/R8) vs same-host retained fn172 matched baseline at seeds 7/19/42, 3 warm reps each: authorized execution median 237.4->36.9 s (6.43x), 256.4->34.9 s (7.35x), 166.5->50.8 s (3.28x), candidate max below baseline min on every seed; parent Go CPU median 161.0->26.2, 165.6->24.6, 124.7->34.5 s; R5 candidate cluster helper peak RSS 3.58 GiB, per-store linear-memory peak 172 MiB. Test fixes: simulation-model fidelity specs (54a60207c4), helper-only service checks (27822d5e86, b4aad2b1c7, 8c151b9952), QUALIFICATION.md (e75ac7df25). Record: .flow/artifacts/fn178-measurements-2026-10-15.md. Integrated batch .8–.11 (MILESTONES batch-first). Frozen qualified candidate `e75ac7df25fe59934630510dd3bc235f47557c92` (branch `gomad-fn178-batch-20261015`, joined with gomad and fn176-close 3d9c60813f). Fast frozen driver `/home/agent/wt/fn178-tmp/qual/run-qualification-fast.sh` (same 89 stages as the serial driver, asserted) PASSED: root `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual`, 3226 s, all stages EXIT 0 except 3 keyed REUSED phase-0 passes (lint-router, lint-router-vet from window 1; rust-fmt from window 2). Source digest `sha256:c80a0428…4d53`; authenticated release helper sha256 `5278fab3…0161`. Independent batch impl review (flowctl claude, claude-opus-5-5 high, base gomad): NEEDS_WORK -> SHIP; focused re-reviews of post-SHIP commits: SHIP (27c51cc28a..27822d5e86, P3s fixed in b4aad2b1c7) and NEEDS_WORK -> SHIP (b4aad2b1c7..e75ac7df25). Handover: `.flow/tmp/handovers/fn178-batch-summary.md`. Native execution remains deferred (fn-128/fn-149).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 69f11f696f, 00a5246ef7, 54a60207c4, 27822d5e86, b4aad2b1c7, 8c151b9952, e75ac7df25, e6af3af015
- Tests: frozen driver /Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual PASS (clusters, workload-* cells, workload-corpus, sqlite-selection-0..3, catalogue), TestMatchedReleaseClusterBaseline/Candidate (exit 0)
- PRs:
### Detail (re-appended after flowctl done, 2026-10-15)
- Qualified frozen candidate e75ac7df25; driver stages log `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/stages.log`, per-stage `.command`/`.log`, lanes A/B/C 1319/2291/2165 s, total 3226 s. Toolchains: stock Go 1.27.1 (sha256 1675694e…), rustc 1.94.1. Host Linux/aarch64, boot e1418388.
- Clusters gate: `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/clusters.log` + owner report; old-assertion red `.flow/tmp/fn178-resume2/redgreen/clusters-old-assertion-red.log`; SQLite split-assertion red `.flow/tmp/fn178-resume2/redgreen/sqlite-split-assertion-red.log` (window 4), negative controls `.flow/tmp/fn178-resume2/redgreen/negctl2.log`.
- Workload results `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/workloads/wasm-qualifications/v1` (10 primary: 5 qualified successes, 5 intentionally unqualified canaries, all within budget; multi-instance success cluster helper peak 3.86 GB), selections `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/sqlite-selections/`. Terminal-to-reap success 37,230,125 ns, refusal 5,494,958 ns.
- Speedup record bound to b4aad2b1c7 (measured candidate); `git diff --stat b4aad2b1c7..e75ac7df25 -- tools/` = qualification/sqlite_durable_test.go + QUALIFICATION.md only (19+/33-), never executed by the measured scenario (conductor-accepted). Baseline same boot (e1418388).
- Docs: protocol.md, README.md, QUALIFICATION.md, CLI.md, ARCHITECTURE.md, SPEC.md, tools/gomad/README.md cluster updates (batch commits, see handover).
