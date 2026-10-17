---
satisfies: [R6]
---
# fn-190-models-are-plain-scala-state-as-vars.5 Remove in-process execution from the DSL

Touches: model/framework/dsl/Machine.scala, RuleBindings.scala, CoreBindings.scala, DerivationBindings.scala, Budget.scala, Domain.scala, Crash.scala, InputExecution

## Description
**Re-scoped (owner, 2026-10-17, DSL completion phase):** fn-190.15 moves native execution out of `framework.dsl` into a test-support module now; this task, in the later IR/plugin phase, deletes that module once fn-190.3's evaluator carries the suites. The DSL-side text below is satisfied by .15.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after phase 1 (fn-190.7) against the final DSL, batched with fn-190.9/.10; one full framework gate at the end. Deleting native execution waits until the IR evaluator (fn-190.3) carries the sample suites.

Delete effectOf, guardHolds, the input-effect adapters, Finite enumeration, StateData, the Domain runtime and native budget runtime; a binding holds a reference to its effect for the plugin only; derivations (`rebind`/`extend`) stay declarations that compose at the IR level. Remove the DSL tests that executed effects or enumerated states and replace them with evaluator-backed tests where the assertion still matters.

**Size:** M

### Verification
- No production code in model/framework/dsl invokes an effect or guard callable.
- Focused dsl and realize suites pass; the evaluator-backed suites pass.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
