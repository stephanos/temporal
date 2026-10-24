---
satisfies: [R12, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.22 Receive current runtime and mandatory workload qualification evidence

Touches: fn-232 runtime/workload receiving evidence and task receipt; common identity ledger remains closeout-owned

## Description
Size M. Review outputs produced by the already-in-progress fn-232.13 driver; do not start another producer or depend on .13 being done. Reuse runtime_gate.py:167/:214, adapters.py:158/:207/:255 and receiving.py:1603. Inventory 32 indices times25 fresh instances,32 forced full replays and exact ten mandatory workload cells with raw Results/publication receipts. Trace transferred fn228R7/fn219R5/fn227R4/fn188/fn223 requirements to their real outputs and original assertions/bounds. Same-cycle R4 consumers may use published workload outputs before this review is complete. Preparation and interim cycle5 inventory snapshots are historical input evidence, not successful execution. Deliver a bounded receiving receipt to the closeout owner rather than editing its common ledger concurrently.

## Acceptance
- [ ] Authenticate every required runtime instance/replay and all ten mandatory cells to one actual candidate/config/source/helper/runtime/model/module/compiler/host context, with numeric producer/reader exits and raw Results/publication evidence.
- [ ] Independently verify counts, fresh-instance separation, forced replay, expected canary versus infrastructure failure, correctness/determinism/memory/capacity and all unchanged assertions/inputs/bounds. Missing, skipped, mismatched or failed obligations remain explicitly incomplete and receive no credit.
- [ ] Retain requirement-to-output mapping for all transferred obligations and a receiving receipt with exact raw paths/hashes. Parent .13 retains final96/145 composition; no second cycle, artifact migration, digest normalization or performance waiver beyond the existing timing decision.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
