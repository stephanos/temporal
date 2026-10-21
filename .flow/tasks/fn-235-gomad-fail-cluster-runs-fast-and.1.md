---
satisfies: [R1, R2, R3, R4]
---
# fn-235-gomad-fail-cluster-runs-fast-and.1 Classify broker capacity exhaustion and end manual cluster runs promptly with retained partial evidence

## Description
Rust helper owner (wasmhost/src/cluster/handoff/broker.rs, cluster/execution/manual.rs): add the golden two-node scenario with a tiny observation_bytes first (hangs to watchdog before the fix), then map world::Error::Capacity from broker World calls to Failure::Capacity(dimension) and emit the terminal result/abort to the controller as soon as an instance fails, stopping the rest through the existing terminate path and retaining each instance partial streams and abort virtual time. Go consumes it with the smallest change; note the R-ID for fn-232s Rust Runner. Merge after coordinating with the fn-232 helper-as-library batch (helper identity rotates). Reproducer and site analysis: /home/agent/wt/evidence/fn224/scenario/DIAGNOSIS.md; worker prototype branch fn224-cluster-stall-proto (instrumentation only, do not merge as is).

## Acceptance
- [ ] Golden tiny-budget scenario ends as capacity naming payload-bytes within 10 s wall with --execution-timeout>=3m
- [ ] inspect shows each node partial streams and the limit; replay reproduces the capacity outcome
- [ ] Other manual-cluster instance/broker failures also end without waiting for the watchdog
- [ ] Existing scenario corpora (durable, network, config) and fn-224 ringpop scenario stay green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
