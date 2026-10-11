---
satisfies: [R4]
---
# fn-156-wasm-gomad-productionization.4 Complete network model and reference checks

## Description
Extend the prototype to the full declared network model: stream bytes, stable listener/connection/incarnation identities, backpressure, deadlines, half-close, cancellation, bounded queues, drain/block/reset policy per action, generated reference-sequence checks and negative/capacity cases.

The provisional batch generates a host projection from unchanged native gomadio network semantics, adapting only blocking/readiness and runtime-domain boundaries. Pending operations return immediately and the guest parks; virtual time and explicit completion ordering control readiness. Existing prototype network evidence remains on its v1 profile; the full native-compatible stream model has a distinct profile because partition behavior differs. A broker can reuse its run/node/incarnation-bound handles and completions.

**Touches:** `tools/gomad3/simulation/network/**`, `tools/gomad_wasm/toolchain/net_wasm.go.txt`, `tools/gomad_wasm/toolchain/overlay.go`, `tools/gomad_wasm/wasi/network_v2*.go`, `tools/gomad_wasm/testdata/network_streams/**`

Typed full-stream helper/protocol integration is serialized with the helper owner. Final admission follows `.2`; provisional implementation is permitted by the batch-first strategy. Gates cover generated reference sequences, capacity/negative operations, actual guest partial I/O/deadline/half-close/cancel, and named Temporal retry/recovery fault with matching choice/I/O/model/outcome replay.
## Acceptance
Original fn-151 Phase 7 exit gate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
