---
satisfies: [R1, R4]
---
# fn-185-gomad-wasm-host-side-execution-cost.1 Attribute host imports to exact ops and model owners in the diagnostic profile

## Description
Extend the diagnostic-profile feature in wasmhost: record each host import's exact module.op and the model owner that served it (namespace, volume, network, clock, World, storage transport, cache, scheduler handoff, transcript/evidence) with count, total wall, exclusive wall (nested encoding/hash/remote-write spans subtracted) and a bounded per-call distribution (min, median, p99, max; fixed-size histogram, no allocation in the hot path beyond what the feature already does). Add the reconciliation row (attributed host wall + guest residual versus execute wall) and keep the overlap error counter. Emit `abi_ops` and `owners` in the profile JSON while keeping phases/abi/nested buckets unchanged. Extend the Go collector moved by fn-187 into internal/measure and cmd/wasmmeasure to read the new fields and fail on a reconciliation miss or any unattributed import. Prove the release helper is unchanged: empty-features build, the moved runtime-cost producer (formerly `TestHelperRuntimeCostProfile`) on it, no Cargo.lock change. Early proof: one single-guest activity run on seed 0 whose tables reconcile within tolerance with the same choice witness as the release run. Follow completed fn-187 and its final helper module ownership; fn-180 must already be integrated. Preserve the corpus contracts while extending the current dispatch modules.

Pending R4 receiving cross-reference: fn-232 R12/task .13's final frozen Rust window receives the actual runtime-cost-equivalent release-behavior and qualified choice/WASI/network/stdout/stderr preservation comparisons, with the original assertions, bounds and actual identity bindings. R4 stays OPEN in this task until that receiving proof exists and is reconciled here; older f172 raw early proofs remain historical and do not close fn-185.1 for the current candidate. This coordination authorizes no new Go-only profiling/full qualification, migrated or relabeled digests, acceptance waiver or status change.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/** (diagnostic owner and dispatch modules after fn-187), tools/gomad/internal/backends/wasm/internal/measure/**, tools/gomad/internal/backends/wasm/cmd/wasmmeasure/**, tools/gomad/internal/backends/wasm/README.md

## Acceptance
- [ ] Every host import appears under an exact module.op and an owner; unattributed imports fail the collector
- [ ] Reconciliation within the stated tolerance and zero overlap errors on the proof run
- [ ] Choice witness of the profiled run equals the release run at the same seed
- [ ] Release helper behavior unchanged: moved runtime-cost producer passes (formerly TestHelperRuntimeCostProfile), no Cargo.lock change

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
