---
satisfies: [R2]
---
# fn-221-rejecting-outcomes-reject-steps-phased.2 Phased, Phase and closable laws; the mini activity Model rejects cancel after closure

## Description
Add `Phase`, `Phased` and `closable::{terminal_remains, closed_rejects}` to `umpire_dsl`, carrying the archived closure laws and their caveats. Convert the current mini activity Model at `model/temporal/features/activity/model.rs`: `Phase::closed` names the terminal set once, closed `Client::Cancel` emits `Rejected(NotFound)`, and delegating postconditions register both shared laws. Regenerate only the activity goldens; retain the existing six phases and atomic Scheduled/Started cancellation.

**Size:** S
**Touches:** [model/umpire/dsl/src/**, model/umpire/dsl/tests/**, model/temporal/features/activity/model.rs, model/temporal/features/activity/expected/**, model/temporal/testing/**]

### Approach
- Hand-write `impl umpire_dsl::Phase for Phase` and `impl Phased for Model` inside the module; no new attributes or capability registry. Use the current standard formatting contract: `Phase: Copy + Eq + Hash + Debug`, not removed `Spell`. `Phased::closed` delegates to `self.phase().closed()`.
- `terminal_remains(after, before)` checks that an emitted step from a closed phase preserves that phase, including rejecting moves between different closed phases. `closed_rejects(after, before, outcome, expected)` additionally requires identical state and `Rejected(expected)`. Keep the unit-returning `check!` registration style; callers evaluate these laws inside claim frames.
- Preserve the archived caveats: terminal preservation alone promises neither rejection nor whole-state immutability; uniform closed rejection is entity-specific and does not model repeated-request-ID deduplication. Generic docs must not mention Temporal-specific entities or numeric RPC codes.
- Enumerate phases in the activity controls, deriving the terminal set by the single `Phase::closed` classification instead of duplicating a hand-maintained terminal list. Retain positive checks for all six phases and every action, and exercise the registered callbacks directly where actions are disabled so the laws are not merely vacuously green.
- Test genuine emitted rejected cancel self-loops from all four terminal phases. Hypothetical controls refute reopening and transitions between closed phases; accepted, wrong-rejection and state-changing edges from closure refute `closed_rejects`. Add a generic multi-field control distinguishing phase preservation from whole-state preservation.
- Kettle remains unphased and existing generic goldens unchanged. This task changes the independent activity Model contract according to R2; later tasks carry outcomes through the protocol and judge real cancel errors. Do not fit these laws to runtime behavior.
- Verify the parent Quick commands using the existing Make-provided `UMPIRE_TEMPORAL_WASM` guest path for the direct Go integration command; preserve evidence that an unconfigured invocation is not an implementation regression. No protocol or realization changes in this task.
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
