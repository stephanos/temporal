# fn-178-gomad-multi-instance-clusters-in-one.10 Parallel versus serialized instances: measure and decide

## Description
Resolve the parked parallelism question with data, now that the harness and the serialized scheduler exist. Should instances run in parallel on host threads, or stay strictly serialized? The answer is a recorded decision. Any parallel implementation would be a follow-up task, added only if the decision says so.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** S
**Files:** the fn-178 spec (Decision Context, via `flowctl spec set-plan`), `.flow/artifacts/` (measurements)
**Touches:** [.flow/specs/fn-178-gomad-multi-instance-clusters-in-one.md, .flow/artifacts/fn178-*, tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasi/cluster*.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go]

Integrated batch: follow the .8 amendment. The write surface admits bounded measurement instrumentation needed for this decision, not a parallel scheduler implementation. Retain telemetry outside replay-visible semantic evidence unless explicitly part of the existing metrics contract. Measure on the real Linux/arm64 host without competing qualification/performance workloads; historical spike figures remain historical. A positive parallelism decision requires a separately scoped follow-up before implementation.

### Approach
- Measure on the two-cluster Temporal selection with the serialized scheduler:
  - per-instance runnable overlap: how often more than one instance is runnable between World-visible operations
  - the share of authorized execution spent in guest code versus scheduler handoff
- Use these to bound the best-case parallel speedup.
- Assess determinism constraints: World delivery order and clock settling require a single global order at visible operations. Parallelism is possible only between visible operations, and only if per-instance local effects stay instance-private. That holds for namespace and volume, but not for network sends.
- Recommendation: stay serialized unless the bound shows ≥ 1.5× and the scheduler seam from fn-178.1 supports it without changing evidence. Record the decision and, if it is positive, the follow-up scope.
- Threading revisit (fn-178.1 design record item 1): count baton handoffs (guest idles) per cluster run and multiply by the measured thread-baton switch cost. If that exceeds 5% of authorized execution, record a decision on moving to Wasmtime async fibers (wasmtime `async` feature, async handoff-capable imports, `call_async`), with its `Cargo.lock` change; otherwise record that threads stay.

## Acceptance
- [ ] Measurements retained under `.flow/artifacts/`; decision recorded in the fn-178 Decision Context, replacing the serialized working assumption note.
- [ ] Handoff count, the thread-baton cost share and the threads-versus-fibers decision are recorded.
## Done summary
Measured and decided (no implementation): instances stay strictly serialized (runnable overlap ~51% of World-visible operations but guest share 15–19%; best-case two-store bound 1.08–1.106x < 1.5x) and threads stay (current-host Condvar baton 11.2 µs/switch, fibers 152 ns; 2,586–2,614 handoffs/scenario = 0.05–0.09% of authorized execution < 5%). Decision recorded in the fn-178 spec Decision Context ("Scheduling decisions (fn-178.10)"), replacing the serialized working-assumption note. Measurements: .flow/artifacts/fn178-measurements-2026-10-15.{md,json}, fn178-baton-spike-2026-10-15.json; quiet real Linux/arm64 host, candidate bound to b4aad2b1c7 (delta to e75ac7df25 is one qualification test file + QUALIFICATION.md, conductor-accepted). Integrated batch .8–.11 (MILESTONES batch-first). Frozen qualified candidate `e75ac7df25fe59934630510dd3bc235f47557c92` (branch `gomad-fn178-batch-20261015`, joined with gomad and fn176-close 3d9c60813f). Fast frozen driver `/home/agent/wt/fn178-tmp/qual/run-qualification-fast.sh` (same 89 stages as the serial driver, asserted) PASSED: root `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual`, 3226 s, all stages EXIT 0 except 3 keyed REUSED phase-0 passes (lint-router, lint-router-vet from window 1; rust-fmt from window 2). Source digest `sha256:c80a0428…4d53`; authenticated release helper sha256 `5278fab3…0161`. Independent batch impl review (flowctl claude, claude-opus-5-5 high, base gomad): NEEDS_WORK -> SHIP; focused re-reviews of post-SHIP commits: SHIP (27c51cc28a..27822d5e86, P3s fixed in b4aad2b1c7) and NEEDS_WORK -> SHIP (b4aad2b1c7..e75ac7df25). Handover: `.flow/tmp/handovers/fn178-batch-summary.md`. Native execution remains deferred (fn-128/fn-149).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e6af3af015
- Tests: TestMatchedReleaseClusterBaseline (window1, exit 0, 3514s), TestMatchedReleaseClusterCandidate (window4, exit 0, 902s), store-handoff-spike 200000 7 (exit 0)
- PRs:
### Detail (re-appended after flowctl done, 2026-10-15)
- Qualified frozen candidate e75ac7df25; driver stages log `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/stages.log`, per-stage `.command`/`.log`, lanes A/B/C 1319/2291/2165 s, total 3226 s. Toolchains: stock Go 1.27.1 (sha256 1675694e…), rustc 1.94.1. Host Linux/aarch64, boot e1418388.
- Raw: `.flow/artifacts/fn178-measurements-2026-10-15.md` / `.json`; baton `.flow/artifacts/fn178-baton-spike-2026-10-15.json`. Baseline report sha256 825ccc4c…; candidate report sha256 82012ba2…; reports under the window roots named in the artifact.
