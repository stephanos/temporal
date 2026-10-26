---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.1 Spike: generate admitted Models and run them through every backend

Touches: [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

## Description
Settle how to generate Models for the differential test, at mandatory source level and optionally tree level (spec Architecture R1), before building the harness. The throwaway probe is retained as reproducible evidence; the verdict is committed. No prototype implementation is merged.

**Size:** M
**Files:** `model/design/generated-differential.md` (verdict); probe, README and `run.sh` under `.flow/tmp/fn255-spike/`.
**Touches:** [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

### Approach
- Tree level: write a small proptest strategy for admitted trees (two enum fields, one bounded int, one `Option<Finite>`, one guarded handler with `reject!`, one invariant). Check whether interpreter and TLC can consume a tree built without the front end, and whether that breaks the independence constraint (spec Edge Cases).
- Source level is required: print the same shapes as `#[machine]` source. Compile N per scratch crate; measure rustc time per batch and per Model. Choose a nonzero bounded source count for required-backend runs; tree-only coverage cannot replace macro/front-end coverage.
- Demonstrate bounded extraction/comparison of complete starts and labeled transitions (action, successor, outcome and facts) from TLC. Existing state-set/verdict and counterexample-replay helpers do not establish complete edge agreement. Account for extraction launches/work in R7 budgets.
- Require successful backend completion before comparing results; probe matching refusals and confirm they fail rather than pass as zero-state agreement.
- Measure cases per minute at each level, with TLC per-run cost from fn-254's baseline, and test batching several generated Models into one TLC module.
- Seed a swapped comparison and an outcome/fact-only mutation in scratch copies. Confirm both are caught and shrunk, including the latter when reached states and claim verdicts remain unchanged.
- Propose the bounds for R7: maximum states/edges/depth per generated Model, cases per tier, TLC launches per case (with batching), and a per-case timeout, each derived from the measurements.
- Answer the spec's three Parked unknowns, or name them as owner questions.

### Investigation targets
**Required:**
- `model/umpire/ir/src/lib.rs` — admitted tree types
- `model/umpire/backends/native/src/interpret.rs` — interpreter entry
- `model/umpire/backends/tla/` — emitter entry
- `model/umpire/backends/native/tests/main/oracle.rs` — existing proptest use
- `model/umpire/dev/testing/src/backend.rs::{check_exploration, agreement}` and `interpreted.rs::compare` — current agreement coverage and matching-refusal behavior

## Acceptance
- [ ] Verdict retains mandatory source coverage, chooses any additional tree layer, harness home, nonzero required-mode source count and TLC batching, with measurements.
- [ ] Probe demonstrates bounded complete starts/labeled-transition extraction; same-state/verdict outcome/fact mutations fail, and matching refusals fail.
- [ ] Seeded lowering bug caught and shrunk at the chosen level.
- [ ] Probe reproducible from `.flow/tmp/fn255-spike/run.sh`.
- [ ] Proposed R7 bounds (Model size caps, cases per tier, TLC launches per case, per-case timeout) with the measurements behind them.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
