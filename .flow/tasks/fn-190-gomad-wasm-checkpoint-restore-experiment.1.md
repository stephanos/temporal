---
satisfies: [R2, R4]
---
# fn-190-gomad-wasm-checkpoint-restore-experiment.1 Dormant re-entry prototype in the runtime overlay and recorded-suffix fixture

## Description
Kill switch first. In toolchain/runtime_wasm.go.txt and overlay.go, using the fn-180 g0 handoff: when a stop with image capture is requested, record the pending decision request in a reserved linear-memory slot, unwind to the run loop and trap to the host instead of blocking inside the import; on resume re-enter the run loop and read the delivered decision from the slot. Dormant otherwise: recording and ordinary replay take the existing import path with identical evidence. Fixture guest: two goroutines on a channel, one timer, one select with two ready cases, a few hundred bytes of fake network, one read-only mount read. Prove recorded-suffix equivalence at a run-queue decision, a select decision and a boundary inside a clock run, using an in-process harness (image stays in memory; .2 adds the on-disk dump). If re-entry cannot be made to work, record why and go to .5. Sequencing: after fn-240.7 (spec-level dep on fn-240).

## Acceptance
Three boundaries show equal choice trace, WASI transcript digests, network and storage digests, stdout/stderr, terminal outcome and runtime witness; cooperative recording/replay evidence byte-identical before and after the overlay change (fixture corpus and the determinism lane); or a retained no-go note naming the state that blocks re-entry.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
