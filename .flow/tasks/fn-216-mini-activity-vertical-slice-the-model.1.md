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
The mini activity Model now supplies an independent six-phase fixture for the SDK slice. The new `umpire-temporal-tests` crate uses the shared discovery and checker consumer at `../features`, with report, states and rows goldens rendered twice for determinism.

The graph starts in Scheduled and contains all six catalog actions, six reached phases and seven edges. The transition-matrix test checks all 36 phase/action pairs, including 29 disabled actions. `registered_terminal_preservation_checks_every_terminal_action_pair` invokes the registered terminal_remains callback for all 24 terminal/action pairs, including disabled actions; `hypothetical_reopening_refutes_the_same_registered_preservation_callback` refutes that callback with test-only reopening edges. The report's emitted-step check never exercises the terminal antecedent because terminals have no outgoing edges. Its `vacuity: none` line describes the checker's action-trigger accounting, and does not establish nonvacuous terminal preservation. The direct controls supply that additional evidence.

This Model deliberately restricts the seven-phase archive to atomic cancellation and fatal-only failure. Successful Start establishes the initial Scheduled condition outside the phase graph. Realization remains assigned to task .6; no server behavior influenced this Model.

baseline: green. Pre-edit `make model-check`, the existing `umpire-tests` fixture suite and `make model-time` passed. Baseline timing was 11.412s full gate and 2.469s checker edit/test. Logs are `.flow/tmp/fn216-1-baseline-{model-check,focused,model-time}.log`.

Verification passed. The new focused suite ran four tests, and the final full `make model-check` passed with the generic fixtures unchanged. The test-first log `.flow/tmp/fn216-1-red.log` records the missing Scheduled cancellation edge and a placeholder predicate incorrectly accepting reopening. The fixture runner also failed before snapshot metadata existed. A Clippy single-variant wildcard finding was fixed with an explicit Client::Cancel fallback before review. The final gate log is `.flow/tmp/fn216-1-verify-model-check.log`.

The isolated `make model-time` run used `UMPIRE_INNER_CRATE=umpire-temporal-tests` and `UMPIRE_INNER_SOURCE=temporal/features/activity_mini/model.rs`. Full gate 10.287s, Model edit/test 0.991s, both within the 30s/10s soft targets. Raw per-step timings are in `.flow/tmp/fn216-1-model-time-isolated.log`. An earlier overlapping observation returned 4.292s/0.916s and is retained in `.flow/tmp/fn216-1-model-time.log`; it was excluded from the isolated measurement.

DSL findings for task .4. Default-backed enum field initialization works through State without new attributes. General state enums have no dedicated Spell derive, so Phase uses standard value derives and an explicit canonical Spell implementation. The report detects untriggered action filters but does not report an unexercised conditional antecedent. No DSL extension was needed. The known legacy generated-Case failure remains separately recorded in `.flow/tmp/fn216-legacy-baseline.md`; this task does not claim a passing Case/server gate.

Review used three fresh Codex contexts with the explicit `codex:gpt-6.1-sol:high` override. Correctness, contracts and integration each returned SHIP with zero findings, and the finalizer recorded SHIP. Writer and reviewers share the Codex family. Receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.1.json`; fanout evidence `.flow/review-fanout/e955786548184f9085d611a027ba5252/`. Reviewers' fresh Cargo invocation was sandbox-blocked; the worker's full gate supplies compilation and execution evidence.

stage: impl-review - ran [2026-10-19T17:08:20Z..2026-10-19T17:09:38Z]

Tier: session (jev-unavailable(no_key)); explicit project routing preserved.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 45760a8d20d058e26bef54f86875c242b8453792
- Tests: cd model && mise exec -- cargo test -p umpire-temporal-tests, make model-check, UMPIRE_INNER_CRATE=umpire-temporal-tests UMPIRE_INNER_SOURCE=temporal/features/activity_mini/model.rs make model-time
- PRs: