---
satisfies: [R4]
---
# fn-253-reusable-deadline-countdowns-for-models.9 Activity World: maximum_attempts as a Finite wrapper

Touches: [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/expected/**, model/temporal/realization/**]

## Description
Try making the attempt bound a type, as `Ticks` already is for timeouts: `World::maximum_attempts` becomes a one-field `Finite` wrapper with `#[inputs(1..=3)]` if the existing authoring surface admits the required comparisons. Preserve native invalid-World validation. Owner-filed 2026-10-25 from the activity quick-win review.

**Cross-spec gate:** after fn-246.15 and fn-246.13 implementations and Flow closures are committed in `umpire`, and after .7 is integrated. This changes the World schema that fn-246.13's shared parameter contract validates against. Update the exact parameter contract and affected feature/protocol adapters in the same change without changing their meaning. Resolve and integrate this task before .10; do not overlap their activity-file edits.

**Size:** S
**Files:** activity `model.rs`, `tests.rs`, `policy_checks.rs`, `proof_checks.rs`, `expected/`, and realization World constructors if they build the field.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/expected/**, model/temporal/realization/**]

### Approach
- Add `Attempts(#[inputs(1..=3)] pub u8)` beside `Ticks` (`model.rs:15-16`); retype `World::maximum_attempts`. Update `attempts(world)`, the `attempt < maximum` comparisons and the World constructors in tests and realization.
- `fair_worker` currently selects on primitive `world.maximum_attempts == 1/2`. Retyping the field requires compilable wrapper comparisons; the direct literal equality no longer typechecks, and payload comparisons such as `world.maximum_attempts.0 == 1` encounter the current refusal of nominal payloads in assumption guards (`model/umpire/README.md`, World-selected assumptions). Also probe the `#[domain] fn attempts(world)` payload read used to construct action domains; do not assume its admission follows ordinary handler expressions. Record both compiler/admission results. If either refuses, keep the field primitive and record non-adoption, or defer adoption until a separately approved front-end change admits it. Do not add comparison tricks or invent a workaround in the Model.
- Replace primitive range arithmetic with the wrapper's checked predicate only if native invalid-World validation remains explicit, for example `Attempts::contains(world.maximum_attempts.0)` in `policy_in_range`. `World::new` and `Finite` do not automatically validate native forged values. Mirror `configured_deadlines_reject_zero_and_binding_bypass` with 0/4 controls proving both source-binding refusal and native invalid-start rejection or invariant refutation. Binding enforcement alone does not justify deleting the native check.
- **Equivalence pin:** compare valid selected Worlds against .7's integrated, corrected fn-246.15/fn-246.13 baseline. State/edge/fact/outcome sets remain identical after accounting for the World's nominal Debug spelling. Review regenerated `expected/states.txt.snap`, `rows.txt.snap` and `tlc.txt.snap` for those representation changes; invalid native Worlds remain caught.

### Investigation targets
**Required:**
- `model/temporal/feature/activity/model.rs:15-45` — `Ticks`, `World`, `attempts`
- `model/temporal/feature/activity/policy_checks.rs` (`configured_deadlines_reject_zero_and_binding_bypass`) — binding refusal control to mirror
- `model/umpire/README.md` (World-selected assumptions) — guard admission rules
- `.flow/tasks/fn-246-realization-as-a-checked-refinement-a.13.md` — parameter contract coordination

## Acceptance
- [ ] Either `maximum_attempts` is a `Finite` wrapper with binding and native forged-0/4 controls, or the primitive field remains with the comparison refusal and non-adoption recorded.
- [ ] Native attempt-domain validation remains effective; a wrapper adoption uses its checked predicate where admitted.
- [ ] Both `fair_worker` comparisons and the nominal payload read in `#[domain] fn attempts(world)` compile and are admitted unchanged in meaning, or their concrete refusal is recorded without adding front-end scope here.
- [ ] Equivalence pin holds; fixtures pass with `UMPIRE_BACKENDS=require`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
