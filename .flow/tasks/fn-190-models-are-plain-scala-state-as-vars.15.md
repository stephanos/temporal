---
satisfies: [R6]
---
# fn-190-models-are-plain-scala-state-as-vars.15 DSL completion: move the native runtime out of framework.dsl into a test-support module

## Description
**Carried by fn-196.1 (owner, 2026-10-17):** the same move, but the destination is the product checker module `model/framework/checker`, since the Scala checker is now the oracle (fn-196); fn-190.5 is re-cut by fn-196.4 to delete the Go check instead of this runtime.

**DSL completion phase (owner, 2026-10-17).** The DSL half of R6, split from fn-190.5 so the DSL executes nothing before the IR evaluator exists. Move the native execution (`effectOf`, `guardHolds`, input-effect adapters, `Finite` enumeration, `StateData`, the `Domain` runtime, native budget runtime, `NativeCheck` and the var-snapshot runtime) out of `model/framework/dsl` into a test-support module used only by the sample and DSL test suites; `framework.dsl` keeps declarations and the references the plugin reads. The DSL's public surface does not change for authors; tests import the runtime explicitly. fn-190.5 later deletes that module once fn-190.3's evaluator carries the suites.

**Size:** L

### Verification
- No production source in model/framework/dsl invokes an effect or guard callable (grep recorded).
- The native dsl/realize/samples suites pass with the same test count; lint RuleSuite and boundary suites pass.

## Acceptance
- [ ] Native execution lives outside framework.dsl in a test-support module
- [ ] Native suites pass with unchanged counts

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
