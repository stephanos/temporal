---
satisfies: [R8]
---
# fn-190-models-are-plain-scala-state-as-vars.7 Rewrite the six samples; budget moves to the checker; regenerate oracles; equivalence

Touches: model/framework/samples/*.scala, model/framework/samples/testdata/native-migration, model/framework/export/testdata/reify/samples/*.json

## Description
**Phase 1 (DSL only), owner phasing 2026-10-17.** Change model/framework/dsl, realize and samples only; the gate is the native checks (dsl/realize/samples compile, their native suites pass, block-form lint). The plugin, lifter, export and IR are untouched: rewrite the six samples in the plain form (after fn-191.1 puts them in sections); they compile under the native strict flags and their native suites pass. Lifting, the Go budget, start list, vacuity, oracles and equivalence move to fn-190.10.

Prerequisite: fn-191 has put the samples into sections. Rewrite Kettle, Bank, Voting, TwoPhaseCommit, ReliableBroadcast and TwoLayeredCache in the plain form; compile with the plugin under the native strict flags; lift. R14 and R15's consumer side lands here: the Go explorer iterates the start list (verify it reads more than the head today), reports vacuity (actions that never fire, after(action) postconditions never triggered) in the spec's shapes, the derived negative-control machines (Leaky, Naive, EagerCommit) and refuted claims move to the sample tests as assertRefuted, remove the bounds field from the IR `Query` message, give the Go checker a budget argument (default plus per-feature overrides in the gate's configuration, none in a Model), make every result carry the budget and completion in the spec's shape with a cut exploration answering unknown, and record the budget in the inventory of intended answers. Regenerate the retained oracles once. Prove equivalence: the reachable graphs of the six samples are unchanged, and every Query answer, now on the default budget, is unchanged from the inventory. A changed answer is a finding to fix, not a regeneration.

**Size:** L

### Verification
- Six samples compile and lift; the four native suites pass on the evaluator.
- Graph comparison per sample recorded; the samples model-check stage (fn-191.3) passes on the default budget with every answer matching the inventory.
## Execution status

The six samples use plain vars, transitions and assignment effects. Kettle declares all temperatures
from -40 through 40 as starts and has no query section; redundant verification queries are removed
from the other samples. Five negative designs and their special effects live only in test sources,
with prior answer cases retained as default checks or explicit test queries. The six sample
cases now live in `<Sample>.test.scala`, each with its matching control objects; the shared controls
file and `Expect` suffix are removed. Quint references are removed from authored sample comments.
The owner's effect-definition follow-up uses inferred, multiline method bodies and multiline Model
val/def object members, including constants; vars, method/function locals and declarations outside
Models are exempt. The native DSL/realize/sample run passes all 538 tests. Suite discovery (8 tests),
block-form lint (7 tests), boundary checks (3 tests) and scoped formatting also pass. This records a
native source checkpoint; the .10 lifting, schema/Go/oracle and equivalence obligations stay open.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
