---
satisfies: [R2]
---
# fn-249-gomad-retire-the-go-reference-models.2 Capture Go oracle outputs as Rust corpus goldens

## Description
At one recorded revision, run each convert-to-corpus Go oracle and check in its inputs and expected outputs as Rust corpus tests in `wasmhost` (or the fn-240 crate that owns the model). Prove each converted test catches a representative mutation of the Rust model.

## Acceptance
- [ ] Each converted conformance role has a Rust corpus test with a recorded source revision and an executed catching mutation.

## Done summary
Blocked:
Cross-spec ordering (owner throughput amendment, 2026-10-24): fn-249.1 (analysis-only inventory) may run while fn-242 is in flight. Code changes start only after fn-242.8 completes, because fn-242's cutover still runs the Go model tests and helper-built stages this spec deletes. Unblock when fn-242.8 is done. flowctl only supports intra-spec task dependencies, so this block records the edge.
## Evidence
- Commits:
- Tests:
- PRs:
