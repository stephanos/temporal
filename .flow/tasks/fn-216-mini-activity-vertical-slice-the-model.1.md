---
satisfies: [R1]
---
# fn-216-mini-activity-vertical-slice-the-model.1 Mini activity Model as a shared fixture

## Description
Fix the independent finite product abstraction and its evidence before any server run (R1).
This task owns the pure Model and fixture crate. Task .6 supplies the Temporal realization.

**Size:** M
**Files:** mini activity Model and expected files, Temporal fixture crate, workspace membership.
**Touches:** [model/temporal/features/activity_mini/model.rs, model/temporal/features/activity_mini/expected/**, model/temporal/testing/**, model/Cargo.toml, model/Cargo.lock]

## Approach
- Reuse `umpire-testing` discovery and runner from the existing shared fixture crate. Make a Temporal-side fixture crate under `model/temporal/testing/`, rooted at `../features`; do not add a second runner or discovery mechanism.
- Author the six-phase, six-changing-action abstraction in the parent contract with the existing attribute surface. A default Scheduled enum variant supplies the initial field value through State's existing Default support. No new starts/input/fact/end attribute is needed.
- Read the archived Dispatching and Completion facets as references, preserving the documented restriction rather than claiming byte/behavior parity with seven-phase retry-aware Scala.
- Register terminal_remains and reuse its registered callback for the exhaustive fixture checks. Check every terminal phase against every catalog action, including disabled actions. Use a test-only hypothetical reopening edge to refute the same predicate. Do not alter production transitions for coverage or duplicate the predicate body in tests.
- Produce report/states/rows through existing recognized Files outputs and deterministic double rendering. Run enum derives through existing Spell/State behavior, without Debug-based canonical output.
- Record encountered DSL gaps for .4. Preserve the abstract cancellation/deadline behavior even though the live scenarios do not exercise it. Keep realization/protobuf/SDK concerns out of this task.

## Investigation targets
**Required:**
- `model/umpire/tests/build.rs:3` - shared discovery reuse.
- `model/umpire/tests/fixtures.rs:7` - fixture runner registration.
- `model/umpire/tests/fixtures/sample_kettle/model.rs` - attribute authoring and fixture hook.
- `model/umpire/dsl/macros/src/state.rs:27` - Default-backed starts for enum fields.
- `model/archived/temporal/features/activity_mini/product/Dispatching.scala.disabled` - read-only archive reference.
- `model/archived/temporal/features/activity_mini/product/Completion.scala.disabled` - read-only closure/retry reference.

## Quick commands
From model/, run the new Temporal fixture crate's focused Cargo tests using its actual package name. Then run make model-check and record make model-time from the repository root.

## Acceptance
- [ ] The new fixture crate discovers the mini activity through the existing runner and pins its deterministic report/states/rows.
- [ ] Scheduled is the initial state; all six specified phases and six changing actions have the contract's exact transitions and disabled-action behavior.
- [ ] Terminal closure checks enumerate every terminal/action pair and the hypothetical reopening control fails the same preservation predicate. The postcondition's vacuity limit is disclosed.
- [ ] Focused tests and the full model gate pass with the existing generic fixtures unchanged; raw timing and DSL gaps are recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
