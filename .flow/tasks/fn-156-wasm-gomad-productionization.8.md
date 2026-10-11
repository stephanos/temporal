---
satisfies: [R6]
---
# fn-156-wasm-gomad-productionization.8 Move runtime decisions and clock handling into the helper

## Description
The fn-151 prototype spends most Temporal wall time on JSON round trips between the Wasmtime helper and the Go model process: every scheduler decision, select result, clock read, poll and log write (~2,100 calls per virtual second). Handle seeded and forced decisions, observations, the frozen clock, idle advances and network-model decisions inside the helper, stream bounded traces and transcripts to the Go owner once, and keep identical evidence bytes and replay semantics. Measure with `make -C tools/gomad_wasm prototype`.

Execution batch: helper migration and scheduling qualification, followed by integration with the provisional Runner exploration surface in `.2`. The recorded prototype GO decision admits this work on branch `gomad`. Preserve unrelated worktree changes. No commits, push, PR or CI dispatch are part of this run; the spec's explicit publication constraint remains in force.

**Touches:** `tools/gomad_wasm/wasi/**`, `tools/gomad_wasm/wasmhost/**`, `tools/gomad_wasm/Makefile`, `tools/gomad_wasm/README.md`

Quick commands: `make -C tools/gomad_wasm test-helper test-model test-go`; Rust formatting and clippy; `make lint-code-gomad-wasm`. Retain matched same-host before/after `prototype` evidence and timings for the workflow and unchanged `./tests` suites. Complete only after byte preservation, typed divergence/capacity, and the 5x guest speed gate pass. Run `.1`'s 32x100 qualification on the integrated helper candidate.
## Acceptance
The fn-151 prototype gates produce byte-identical evidence before and after the move for the same seeds and tapes, and the Temporal workflow and `./tests` suite guests run at least 5x faster on the same host. Model identity, divergence classification and capacity outcomes are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
