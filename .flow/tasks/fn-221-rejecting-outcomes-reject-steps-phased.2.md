---
satisfies: [R2]
---
# fn-221-rejecting-outcomes-reject-steps-phased.2 Phased, Phase and closable laws; the mini activity Model rejects cancel after closure

## Description
Add `Phase`, `Phased` and `closable::{terminal_remains, closed_rejects}` to `umpire_dsl` with the archive's caveats as docs. Convert the mini activity Model: `Phase::closed` names the terminal set once, `Client::Cancel if self.closed() => reject!(Rejection::NotFound)`, `terminal_remains` delegates to `closable`, `closed_rejects` is registered. Regenerate the activity goldens; update the fixture controls (hypothetical reopening and hypothetical accepted/wrong-rejection steps from closed phases).

**Size:** S
**Touches:** [model/umpire/dsl/src/**, model/temporal/features/activity_mini/model.rs, model/temporal/features/activity_mini/expected/**, model/temporal/testing/**]

### Approach
- Hand-written `impl Phase` and `impl Phased` inside the module; no attribute.
- The test module's `TERMINALS` reads `Phase::closed`.
- Keep cancel in Scheduled/Started atomic; restate the six-phase restriction in the module docs.

## Acceptance
- [ ] `closable` laws registered by delegating postconditions; both refuted by the hypothetical controls.
- [ ] Activity goldens show the four rejecting self-loops and the new claim; Kettle untouched.
- [ ] `make model-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
