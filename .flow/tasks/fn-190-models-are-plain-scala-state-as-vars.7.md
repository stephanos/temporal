---
satisfies: [R8]
---
# fn-190-models-are-plain-scala-state-as-vars.7 Rewrite the six samples; budget moves to the checker; regenerate oracles; equivalence

Touches: model/framework/samples/*.scala, model/framework/samples/testdata/native-migration, model/framework/export/testdata/reify/samples/*.json

## Description
Prerequisite: fn-191 has put the samples into sections. Rewrite Kettle, Bank, Voting, TwoPhaseCommit, ReliableBroadcast and TwoLayeredCache in the plain form; compile with the plugin under the native strict flags; lift. R14 and R15's consumer side lands here: the Go explorer iterates the start list (verify it reads more than the head today), reports vacuity (actions that never fire, after(action) postconditions never triggered) in the spec's shapes, the derived negative-control machines (Leaky, Naive, EagerCommit) and refuted claims move to the sample tests as assertRefuted, remove the bounds field from the IR `Query` message, give the Go checker a budget argument (default plus per-feature overrides in the gate's configuration, none in a Model), make every result carry the budget and completion in the spec's shape with a cut exploration answering unknown, and record the budget in the inventory of intended answers. Regenerate the retained oracles once. Prove equivalence: the reachable graphs of the six samples are unchanged, and every Query answer, now on the default budget, is unchanged from the inventory. A changed answer is a finding to fix, not a regeneration.

**Size:** L

### Verification
- Six samples compile and lift; the four native suites pass on the evaluator.
- Graph comparison per sample recorded; the samples model-check stage (fn-191.3) passes on the default budget with every answer matching the inventory.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
