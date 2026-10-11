---
satisfies: [R7]
---
# fn-151-wasm-gomad-execution-backend.7 Simulated in-guest network (prototype)

## Description
Add an interception seam in the guest net implementation before fake TCP delivery, reusing gomadio network semantics through a narrow guest adapter. Model stable listener/connection identities, deterministic ordered delivery, delay, partition/heal and reset, driven by the same virtual clock and seed. Single guest only; reference-model conformance, backpressure/half-close/deadline completeness, storage faults and multi-node brokering move to fn-156.

## Acceptance
Real gRPC/SDK traffic in the one-box guest flows through the model; same seed gives identical delivery transcripts. Injected delay and a connection reset/partition during a workflow produce a Temporal retry/recovery that replays with matching choice, network and outcome evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
