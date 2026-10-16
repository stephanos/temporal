---
satisfies: [R4, R5]
---
# fn-182-gomad-smaller-wasm-guests.3 Measure trimmed guests and qualify behavior and replay

## Description
Complete matched measurements and affected qualification for R4/R5, and publish supported build usage.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasi/integration_test.go, qualification/onebox_test.go, qualification/temporal_cluster_test.go, wasmhost/src/{volume,network,broker.rs,model/inspection.rs}, CLI.md, QUALIFICATION.md, README.md
**Touches:** [tools/gomad/internal/backends/wasm/wasi/*test.go, tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/wasmhost/**, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/README.md]

### Approach
- Run baseline and trimmed candidates on the same real Linux/arm64 host using the task .1 collector, workloads, seeds, capacities and repetitions. Report medians/ranges and raw receipts for all R4 metrics; report startup/execution/memory regressions or noise honestly rather than assuming gains from bytes saved.
- Compare functional success and intentional-failure witnesses across builds; establish repeated execution and fresh exact retained replay within each candidate's own identities. Preserve existing capacity/resource acceptance.
- Run affected portable default-build controls and actual WASM workload gates; native execution qualification stays deferred.
- Document the build selection, excluded-provider refusal, measured tradeoffs and new candidate identities together with one retained report and rerun procedure.

### Confirmed inspection-cost regression (2026-10-16)

The corrected matched batch passed six one-box cells and failed its first multi-instance baseline. A separate retained-module diagnostic preserved the production helper/module, seed, inputs, scenario, fuel, memory and 15-minute deadline; it confirmed watchdog termination with both guests making progress to virtual time 32 seconds. Four bounded native stack samples located inspection volume hashing, network-history hashing and World serialization. Full evidence and identities are in `.flow/artifacts/fn182-current-batch-2026-10-16.md`.

Repair this prerequisite without changing execution policy, workload, canonical evidence bytes or budgets:
- Encapsulate file bytes so a lazy content digest is invalidated before every mutable access; preserve fresh metadata, namespace, recovery, operations and history.
- Cache the exact bounded World wire independently by the existing authoritative transition revision, publish only after bounded success, retain at most one wire and drop stale bytes before replacement. Continue fresh network evidence and preserve combined commitment framing; streaming must be byte-equivalent, not a substituted hash.
- Remove duplicate network-record digest work without caching endpoint evidence or weakening validation/refusal behavior.
- Write and observe failing cost/invalidation regressions first; compare complete snapshots, operation evidence, recovery images, identities, errors, capacities, canonical network permutations, peer restart and terminal cuts against uncached behavior. No naive rolling SHA, schema change or new unproved freshness epoch.
- Qualify the integrated changed helper against existing conformance/inspection/runtime gates, rotate affected identities consistently, then use a fresh matched measurement root. Prior-helper results do not qualify the repair. Keep all failed evidence.

### Confirmed residual recovery commitment cost (2026-10-16)

The integrated 9ec helper passed source/runtime/retained-inspection gates and all six new matched onebox success cells, but its first cold multi-instance cell failed overall_timeout at the original 900-second bound. The exact retained-module warm diagnostic then completed both crash/reopen sequences in 751.6575654 seconds, without watchdog. Its adapter discarded the simulation result, so recovery/convergence oracle outcomes remain unverified: nil RunWASM error alone does not prove them. This is nonacceptance diagnostic evidence, not a cold budget pass. Four bounded native samples found recovery commitment hashing twice, persistence-operation observation hashing once and World state hashing once. Offline current-helper disassembly identifies the repeated Recorder::snapshot call as the recovery domain; no statistical CPU share is inferred.

Next implement only bounded exact-byte reuse of recovery commitments in model/inspection.rs, not an unproved storage/namespace freshness epoch:
- Continue freshly encoding all namespace/recovery/log/network/metadata inputs. Reuse a recovery digest only when the current encoded recovery bytes exactly equal the retained bytes for that handle/incarnation; equal length alone is insufficient.
- Keep retained raw cache bytes explicitly bounded independently of evidence admission, with bounded entry ownership. Drop stale bytes before replacement. An oversized material or cache pressure uses the original uncached hash path or safe eviction, never a new semantic refusal or relaxed capacity.
- Preserve canonical recovery/state framing, all evidence bytes, storage/network/witness metadata and original refusal precedence. No profile/schema/dependency/engine/fuel/scheduler/budget changes. Leave persistence-operation and World paths unchanged in this one repair; their audit is supporting evidence, not authority to bundle changes.
- Observe a real test-first repeated-hash RED before production changes, then prove byte-identical hashes/evidence with unchanged material, same-length changes, changing surrounding metadata, multiple handles/incarnations, capacity/exhaustion, replacement/eviction and oversized-cache bypass. Use an independent literal-framing uncached oracle and retain actual RED/GREEN logs.
- Run Rust tests/format/clippy and required source gates. Before another long full acceptance batch, root must run an explicitly nonacceptance first-record cold retained-module diagnostic on the candidate with an empty private engine cache and the same original 15-minute bound. Retain the full returned simulation result and explicitly require completed outcome and passing original oracles, not merely nil Go error. That establishes critical-path feasibility only; current-helper runtime/Temporal qualification and a fresh full matched16-cell run, spread reporting, source review and all final task/spec gates still remain mandatory.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/wasi/integration_test.go
- tools/gomad/internal/backends/wasm/qualification/onebox_test.go
- tools/gomad/internal/backends/wasm/qualification/temporal_cluster_test.go
- tools/gomad/internal/backends/wasm/wasmhost/src/volume/{state.rs,hash.rs,image.rs}
- tools/gomad/internal/backends/wasm/wasmhost/src/network/{mod.rs,history.rs}
- tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs
- tools/gomad/internal/backends/wasm/wasmhost/src/model/inspection.rs
- tools/gomad/internal/backends/wasm/QUALIFICATION.md
- tools/gomad/internal/backends/wasm/CLI.md

### Quick commands
`make -C tools/gomad test-wasm GOMAD_STOCK_GO=/absolute/pinned/go1.27.1` and the selected actual qualification/performance drivers with explicit inputs; confirm no acceptance selector skipped. Run helper formatting, tests, clippy, generated/pin validation and source review for the repair before actual requalification. Root serializes heavy host work and owns final review, integration and completion.
## Acceptance
- [ ] Matched Linux/arm64 measurements cover all R4 metrics with identities, repetitions, ranges and rerun commands
- [ ] Existing resource budgets pass; measurement spread and regressions are reported explicitly
- [ ] Success and canary failures remain observable, with repeated execution and fresh retained replay under the new candidate
- [ ] Build documentation and qualification report reflect actual results and preserve native deferrals

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
