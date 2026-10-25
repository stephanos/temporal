---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.1 Spike: generate admitted Models and run them through every backend

Touches: [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

## Description
Settle how to generate Models for the differential test, at tree and/or source level (spec Architecture R1), before building the harness. A throwaway probe plus a verdict; nothing merges.

**Size:** M
**Files:** `model/design/generated-differential.md` (verdict); probe, README and `run.sh` under `.flow/tmp/fn255-spike/`.
**Touches:** [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

### Approach
- Tree level: write a small proptest strategy for admitted trees (two enum fields, one bounded int, one `Option<Finite>`, one guarded handler with `reject!`, one invariant). Check whether interpreter and TLC can consume a tree built without the front end, and whether that breaks the independence constraint (spec Edge Cases).
- Source level: print the same shapes as `#[machine]` source. Compile N per scratch crate; measure rustc time per batch and per Model.
- Measure cases per minute at each level, with TLC per-run cost from fn-254's baseline, and test batching several generated Models into one TLC module.
- Seed one known lowering bug (a swapped comparison in a scratch emitter copy) and confirm the probe catches and shrinks it.
- Propose the bounds for R7: maximum states/edges/depth per generated Model, cases per tier, TLC launches per case (with batching), and a per-case timeout, each derived from the measurements.
- Answer the spec's three Parked unknowns, or name them as owner questions.

### Investigation targets
**Required:**
- `model/umpire/ir/src/lib.rs` — admitted tree types
- `model/umpire/backends/native/src/interpret.rs` — interpreter entry
- `model/umpire/backends/tla/` — emitter entry
- `model/umpire/backends/native/tests/oracle.rs` — existing proptest use
- `model/umpire/dev/testing/src/backend.rs:78-120` — three-way agreement helpers

### Acceptance
- [ ] Verdict chooses the level(s), harness home, case counts per tier and TLC batching, with measurements.
- [ ] Seeded lowering bug caught and shrunk at the chosen level.
- [ ] Probe reproducible from `.flow/tmp/fn255-spike/run.sh`.
## Acceptance
- [ ] Verdict chooses the level(s), harness home, case counts per tier and TLC batching, with measurements.
- [ ] Seeded lowering bug caught and shrunk at the chosen level.
- [ ] Probe reproducible from `.flow/tmp/fn255-spike/run.sh`.

### Acceptance
- [ ] TBD
- [ ] Proposed R7 bounds (Model size caps, cases per tier, TLC launches per case, per-case timeout) with the measurements behind them.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
