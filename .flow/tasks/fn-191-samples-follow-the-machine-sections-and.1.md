---
satisfies: [R2]
---
# fn-191-samples-follow-the-machine-sections-and.1 Move the five remaining samples and NativeInputs into properties and queries sections

Touches: model/framework/samples/Bank.scala, Voting.scala, TwoPhaseCommit.scala, ReliableBroadcast.scala, TwoLayeredCache.scala, NativeInputs.scala, the native sample tests, model/framework/export/ReifySamples.test.scala

## Description
**Phase 1 (DSL only), owner phasing 2026-10-17.** Change model/framework/dsl, realize and samples only; the gate is the native checks. Move the samples into their sections in the section order fn-190 settles on (`invariants`, `postconditions`, `monitors`, `progress`, `queries` where still needed); this lands before fn-190.7 so that rewrite changes bodies, not layout.

Follow the Kettle layout applied on capture: each Machine gains `object properties` (its Properties, through the Machine's own `property`) and `object queries` (scenarios, Limits and queries as named vals, no Vector). A Property goes in the machine it is about and a Query in the machine of its Scenario. Update the native sample tests to reach properties through `<Machine>.properties`. Change the reify sample test's discovery roots to the machines' `queries` sections; do not regenerate its oracle (export suite re-anchoring is fn-164.3).

**Size:** M

### Verification
- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests` passes.
- Formatting check passes on the changed files.
## Acceptance
- [ ] TBD

## Done summary
Phase 1 (DSL only). The five remaining samples and NativeInputs keep claims, scenarios, limits and queries in the machine sections in fn-190's order (states, refinement, rules, effects, invariants, postconditions, monitors, progress, queries); queries are named vals, the Vector is gone, negative controls own their sections and borrow the shared Limits. Kettle's progress claim lives in a HeatingOnly derivation restricted to pressButton/heat/failover (the spec's restriction alone never heats from the only start, so the claim would be vacuous); a native test walks its 81-step path. Native checks: compile exit 0, 468 tests (one new), 29 lint rule tests, scalafmt clean. Phase 2 owns: ReifySamples roots, IrFile discovery of the new sections, structure/order lints and SEMANTICS (fn-191.2, fn-190.9).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f8a81391bd
- Tests: scala-cli compile dsl realize samples: exit 0, scala-cli test dsl realize samples --require-tests: 468 passed, lint *RuleSuite: 29 passed
- PRs: