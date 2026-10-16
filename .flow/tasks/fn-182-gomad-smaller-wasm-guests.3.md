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
