# Gomad: close WASM map-order determinism escapes

## Goal

Close the three map-order escapes retained by fn-188's determinism inventory.
The WASM `wasi` package owns these fixes. No native execution is in scope.

## Requirements

- R1: `ClusterSession.Shutdown` sends stop control frames in stable `(ID, Incarnation)` order when two or more nodes remain live. Fresh same-seed shutdowns preserve World and network evidence under varied map insertion order.
- R2: `ValidateStorageCrashSelection` visits `StorageCrashContext` and `StorageCrashSelection` volumes in sorted volume order. Multiple invalid entries yield identical unsupported refusal messages under varied insertion order, before execution.
- R3: Replace the three `escape` dispositions in `determinism/inventory.json` with the actual controlled rules and named behavioral regression tests. Remove the matching README caveats only after those tests pass.
- R4: Retain the WASM model identity rotation and rerun the frozen-candidate runtime, workload and retained-replay qualification required by the qualification contract. Historical qualification does not qualify the changed model.

## Acceptance

The owner fixes all three sites, runs behavioral regression tests that exercise multiple insertion orders, verifies inventory completeness and architecture checks, and retains current-candidate qualification. fn-188's source audit alone does not close these defects or extend determinism to live-node shutdown scenarios.
