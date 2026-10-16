---
satisfies: [R1, R4]
---
# fn-185-gomad-wasm-host-side-execution-cost.1 Attribute host imports to exact ops and model owners in the diagnostic profile

## Description
Extend the diagnostic-profile feature in wasmhost: record each host import's exact module.op and the model owner that served it (namespace, volume, network, clock, World, storage transport, cache, scheduler handoff, transcript/evidence) with count, total wall, exclusive wall (nested encoding/hash/remote-write spans subtracted) and a bounded per-call distribution (min, median, p99, max; fixed-size histogram, no allocation in the hot path beyond what the feature already does). Add the reconciliation row (attributed host wall + guest residual versus execute wall) and keep the overlap error counter. Emit `abi_ops` and `owners` in the profile JSON while keeping phases/abi/nested buckets unchanged. Extend the Go collector in wasi/integration_test.go to read the new fields and fail on a reconciliation miss or any unattributed import. Prove the release helper is unchanged: empty-features build, `TestHelperRuntimeCostProfile` on it, no Cargo.lock change. Early proof: one single-guest activity run on seed 0 whose tables reconcile within tolerance with the same choice witness as the release run. Land or stash the fn-180 working-tree edits to abi.rs/engine.rs/model.rs first.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/{diagnostic.rs,abi.rs,namespace_call.rs,storage_transport.rs,clock.rs,cache.rs,network/mod.rs,world/mod.rs,cluster/mod.rs}, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/README.md

## Acceptance
- [ ] Every host import appears under an exact module.op and an owner; unattributed imports fail the collector
- [ ] Reconciliation within the stated tolerance and zero overlap errors on the proof run
- [ ] Choice witness of the profiled run equals the release run at the same seed
- [ ] Release helper behavior unchanged: TestHelperRuntimeCostProfile passes, no Cargo.lock change

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
