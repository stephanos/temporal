---
satisfies: [R6]
---
# fn-156-wasm-gomad-productionization.8 Move runtime decisions and clock handling into the helper

## Description
The fn-151 prototype spends most Temporal wall time on JSON round trips between the Wasmtime helper and the Go model process: every scheduler decision, select result, clock read, poll and log write (~2,100 calls per virtual second). Handle seeded and forced decisions, observations, the frozen clock, idle advances and network-model decisions inside the helper, stream bounded traces and transcripts to the Go owner once, and keep identical evidence bytes and replay semantics. Measure with `make -C tools/gomad_wasm prototype`.

## Acceptance
The fn-151 prototype gates produce byte-identical evidence before and after the move for the same seeds and tapes, and the Temporal workflow and `./tests` suite guests run at least 5x faster on the same host. Model identity, divergence classification and capacity outcomes are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
