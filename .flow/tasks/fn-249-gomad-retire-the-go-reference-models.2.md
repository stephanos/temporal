---
satisfies: [R2]
---
# fn-249-gomad-retire-the-go-reference-models.2 Capture Go oracle outputs as Rust corpus goldens

## Description
At one recorded revision, run each convert-to-corpus Go oracle and check in its inputs and expected outputs as Rust corpus tests in `wasmhost` (or the fn-240 crate that owns the model). Prove each converted test catches a representative mutation of the Rust model.

## Acceptance
- [ ] Each converted conformance role has a Rust corpus test with a recorded source revision and an executed catching mutation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
