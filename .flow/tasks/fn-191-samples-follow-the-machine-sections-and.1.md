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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
