---
satisfies: [R4]
---
# fn-253-reusable-deadline-countdowns-for-models.7 Activity Model uses countdowns and the deadline pattern

Touches: [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/expected/**]

## Description
Migrate the activity to `countdown` and the deadline pattern, starting from .1's cleanup. Behaviour must stay exactly the same.

**Cross-spec gate:** start only after fn-246.15 and fn-246.13 implementations and Flow closures are committed in `umpire`. Compare against that corrected baseline after .1's integrated cleanup; do not race or undo the independent protocol work. Integrate .7 before .9, then .10; these tasks share the activity files.

**Size:** M
**Files:** activity `model.rs`, `proof_checks.rs`, `policy_checks.rs`, `tests.rs`, `expected/`.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/expected/**]

### Approach
- Replace every `is_some() && unwrap_or(Remaining { ticks: 0 })` deadline expression with `countdown` calls; the expiry chain becomes one priority list using the pattern's effects.
- Express `countdowns_within_budget` (split out in .1) with `countdown::within`.
- Reanchor source mutants (`proof_checks.rs` `global_mutants` and the others), keeping each one discriminating; do not drop any.
- **Equivalence pin:** `expected/states.txt.snap`, `rows.txt.snap` and `tlc.txt.snap` stay byte-identical. `Activity.tla.snap`/`lift.txt.snap` may change and are reviewed. `compatibility/checkpoint.rs` passes unchanged.

### Investigation targets
**Required:**
- `model/temporal/feature/activity/model.rs` — corrected fn-246.15/fn-246.13 baseline with .1 integrated
- `model/temporal/feature/activity/proof_checks.rs:457-540` — forged-countdown and source-mutant controls
- `model/temporal/feature/activity/policy_checks.rs:749-800` — invariant-name lookups
- `model/temporal/feature/activity/compatibility/checkpoint.rs` — independent lifecycle pin

## Acceptance
- [ ] No `unwrap_or(Remaining { ticks: 0 })` left in deadline code.
- [ ] Equivalence pin holds: states, rows and TLC snapshots unchanged; checkpoint passes.
- [ ] Every existing mutant still discriminates under `UMPIRE_BACKENDS=require`.
- [ ] `mise exec -- cargo test -p umpire-temporal-tests --test fixtures` passes with TLC required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
