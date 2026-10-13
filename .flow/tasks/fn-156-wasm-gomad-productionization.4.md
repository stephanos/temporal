---
satisfies: [R4]
---
# fn-156-wasm-gomad-productionization.4 Complete network model and reference checks

## Description
Integrated MAIN affected check (2026-10-11): complete simulation/network package passed again on the integrated candidate, exit0, reasonnull/residuals[], source postcheck0e0b89b1f792d77ffa43b11e74e7a6db9b0b08b9d8c86b06b355e958421921e5. /tmp/fn156-main-affected-source-tests-evidence-v1/{0.log,journal.jsonl} retains the actual run. All32 generated negative/capacity cases and the original successful sequences ran. This is ordinary source/native-model coverage, not full native platform or guest qualification.


Current generated-network coverage (2026-10-11): integrated test-only adapter_test.go SHA2555fd734d6078287f9ff0c98ed6b1aecc6e841dcf61123ca535f9722bf74b6b preserves32seeds x64 successful stream operations and adds32x8 generated negative/capacity cycles. Literal errors/results, refusal/capacity transitions, state conservation, bounded queues, capacity recovery and exact native record/replay equality are asserted. The complete isolated network package (nine top-level tests, all32 added subtests) passed exit0 under ordinary source90s/1GiB supervision; peak sampled aggregate81477632B, reasonnull/residuals[], unchanged source postcheck. Raw /tmp/fn156-network-generated-evidence-v1/{0.log,journal.jsonl} and /tmp/fn156-minimization-completion-summary.md retain evidence. MAIN bytes match the tested file; integrated affected checks and independent review remain required. Current-helper stream guest and named Temporal reset/retry/fresh replay gates remain open; portable coverage is not native platform qualification.

Extend the prototype to the full declared network model: stream bytes, stable listener/connection/incarnation identities, backpressure, deadlines, half-close, cancellation, bounded queues, drain/block/reset policy per action, generated reference-sequence checks and negative/capacity cases.

The provisional batch generates a host projection from unchanged native gomadio network semantics, adapting only blocking/readiness and runtime-domain boundaries. Pending operations return immediately and the guest parks; virtual time and explicit completion ordering control readiness. Existing prototype network evidence remains on its v1 profile; the full native-compatible stream model has a distinct profile because partition behavior differs. A broker can reuse its run/node/incarnation-bound handles and completions.

**Touches:** `tools/gomad3/simulation/network/**`, `tools/gomad_wasm/toolchain/net_wasm.go.txt`, `tools/gomad_wasm/toolchain/overlay.go`, `tools/gomad_wasm/wasi/network_v2*.go`, `tools/gomad_wasm/testdata/network_streams/**`

Typed full-stream helper/protocol integration is serialized with the helper owner. Final admission follows `.2`; provisional implementation is permitted by the batch-first strategy. Gates cover generated reference sequences, capacity/negative operations, actual guest partial I/O/deadline/half-close/cancel, and named Temporal retry/recovery fault with matching choice/I/O/model/outcome replay.
## Acceptance
Original fn-151 Phase 7 exit gate.

## Done summary
Named Temporal network reset/retry with exact replay passes on 9d6012b52b.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9d6012b52b
- Tests: TestTemporalNetworkNamedResetRetriesAndExactReplay, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: