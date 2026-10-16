---
satisfies: [R6]
---
# fn-190-models-are-plain-scala-state-as-vars.5 Remove in-process execution from the DSL

Touches: model/framework/dsl/Machine.scala, RuleBindings.scala, CoreBindings.scala, DerivationBindings.scala, Budget.scala, Domain.scala, Crash.scala, InputExecution

## Description
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
