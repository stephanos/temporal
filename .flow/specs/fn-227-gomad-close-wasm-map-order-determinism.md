# Gomad: close WASM map-order determinism escapes

## Goal

Close the three map-order escapes retained by fn-188's determinism inventory.
The existing Go `wasi` implementation and tests are the behavioral reference. Under the 2026-10-20 Rust-host decision, fn-232.8 owns the final Rust shutdown and crash-validation ordering; fn-227.1 owns the transferable regression corpus, inventory reconciliation and receiving-evidence check. No new Go-host implementation or native execution is in scope.

## Acceptance Criteria

- **R1:** The Rust cluster shutdown path (behavioral reference: Go `ClusterSession.Shutdown`) sends stop control frames in stable `(ID, Incarnation)` order when two or more nodes remain live. Fresh same-seed shutdowns preserve World and network evidence under varied map insertion order.
- **R2:** The Rust storage-crash selection validator (behavioral reference: Go `ValidateStorageCrashSelection`) visits `StorageCrashContext` and `StorageCrashSelection` volumes in sorted volume order. Multiple invalid entries yield identical unsupported refusal messages under varied insertion order, before execution.
- **R3:** Reconcile the three originally reported sites in `determinism/inventory.json` with the current implementation and named behavioral regressions. Already-controlled Go rows are historical/current-source facts, not proof of the Rust port. Admit controlled Rust rules and remove corresponding caveats only after the Rust behavioral corpus passes.
- **R4:** Retain actual model/implementation identity changes and reconcile the runtime, workload and retained-replay qualification received by fn-232 R12/task .13 (the green 96-stage cycle on current `gomad`; no formal freeze under the 2026-10-24 opportunistic-merge decision). Historical qualification does not qualify the changed model.

## Verification

Task .1 supplies regression cases with multiple insertion orders, verifies inventory completeness and architecture checks, and links fn-232.8 behavior proof plus fn-232.13 final qualification. Keep this spec open until R1–R4 are evidenced; a transfer or source-only test is not completion. This is a receiving dependency, not an extra Go qualification cycle. fn-188's source audit alone does not close these defects or extend determinism to live-node shutdown scenarios.


## Receiving contract

fn-232.8 explicitly owns the linked-core cleanup and sorted crash-selection implementation, the affected helper surfaces and the transferred corpus acceptance. Preserve malformed-plan refusal while adding the admitted live-session shutdown path. Final qualification remains with fn-232.13; fn-227.1 reconciles both receiving proofs before closure.

## Status (2026-10-24)

R1 and R2 are implemented in Rust by fn-232.8 (merge f16f53e59d, RED/GREEN receipts in `.flow/artifacts/fn232.8-identity-handoff-2026-10-22.txt`). What remains for task .1: name the Rust tests in the two determinism-inventory rows (ClusterSession.Shutdown, execution.Shutdown) and receive the fn-232.13 cycle for R4. Nothing else is open here; this spec closes with .1.
