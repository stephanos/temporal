---
satisfies: [R7]
---
# fn-190-models-are-plain-scala-state-as-vars.3 IR evaluator for tests and the four native sample suites on it

Touches: model/framework/check (new evaluator), model/framework/samples/Expect.test.scala, NativeHistory.test.scala, NativeMaps.test.scala, NativeVoting.test.scala

## Description
**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after phase 1 (fn-190.7) against the final DSL, batched with fn-190.9/.10; one full framework gate at the end. The evaluator needs lifted IR, so it follows fn-190.9; it replaces the phase-1 native var-snapshot runtime in the sample suites.

A Scala interpreter over ir.Model/ir.Function/ir values in model/framework/check: evaluate a guard, an effect (returning the steps with outcome, state and facts) and a predicate on a given IR state and input; support exactly the IR constructs the six samples use and fail a test naming any unsupported construct. States for tests are built from a Machine's lifted init plus named updates. Port the four native sample suites to the evaluator with their assertions preserved. It is a test tool: no CLI, no checker role.

**Size:** L

### Verification
- The four suites pass on the evaluator with the same case counts as before.
- A deliberately unsupported construct fails naming itself.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
