---
satisfies: [R4]
---
# fn-253-reusable-deadline-countdowns-for-models.9 Activity World: maximum_attempts as a Finite wrapper

Touches: [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/expected/**, model/temporal/realization/**]

## Description
Make the attempt bound a type, as `Ticks` already is for timeouts: `World::maximum_attempts` becomes a one-field `Finite` wrapper with `#[inputs(1..=3)]`. Then `policy_in_range` no longer checks it by hand. Owner-filed 2026-10-25 from the activity quick-win review.

**Cross-spec gate:** after fn-246.15's committed closure (same file). This changes the World schema that fn-246.13's shared parameter contract validates against, so coordinate with fn-246.13. If .13 is in flight, land after its closure and update its parameter contract in the same change.

**Size:** S
**Files:** activity `model.rs`, `tests.rs`, `policy_checks.rs`, `proof_checks.rs`, `expected/`, and realization World constructors if they build the field.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/expected/**, model/temporal/realization/**]

### Approach
- Add `Attempts(#[inputs(1..=3)] pub u8)` beside `Ticks` (`model.rs:15-16`); retype `World::maximum_attempts`. Update `attempts(world)`, the `attempt < maximum` comparisons and the World constructors in tests and realization.
- `fair_worker` selects on `world.maximum_attempts == 1/2`. Assumption guards cannot read nominal payloads (`model/umpire/README.md`, World-selected assumptions). Try the plain form first. If it refuses, record the refusal and either keep the field primitive (abandon this task, with the refusal as the reason) or widen assumption-guard admission to one-field `Finite` wrappers as a separate front-end change. Do not invent a workaround in the Model.
- Drop the hand-written `maximum_attempts` range check from `policy_in_range` once the binding enforces it. Confirm with a binding control that 0 and 4 are refused, as `configured_deadlines_reject_zero_and_binding_bypass` does for timeouts.
- **Equivalence pin:** `expected/states.txt.snap`, `rows.txt.snap` and `tlc.txt.snap` unchanged apart from the World's Debug spelling and the removed check.

### Investigation targets
**Required:**
- `model/temporal/feature/activity/model.rs:15-45` — `Ticks`, `World`, `attempts`
- `model/temporal/feature/activity/policy_checks.rs` (`configured_deadlines_reject_zero_and_binding_bypass`) — binding refusal control to mirror
- `model/umpire/README.md` (World-selected assumptions) — guard admission rules
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.13.md` — parameter contract coordination

### Acceptance
- [ ] `maximum_attempts` is a `Finite` wrapper; binding refuses 0 and 4 with a control test.
- [ ] `policy_in_range` no longer checks the attempt bound by hand.
- [ ] `fair_worker` is admitted unchanged in meaning, or the task closes as abandoned with the recorded refusal.
- [ ] Equivalence pin holds; fixtures pass with `UMPIRE_BACKENDS=require`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
