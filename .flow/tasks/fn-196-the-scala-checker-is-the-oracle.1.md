---
satisfies: [R1]
---
# fn-196-the-scala-checker-is-the-oracle.1 DSL completion: the checker module outside framework.dsl, with one Result shape

## Description
**DSL completion phase (owner, 2026-10-17).** Supersedes fn-190.15's destination: the same move of the native runtime (`effectOf`, `guardHolds`, input-effect adapters, `Finite` enumeration, the `Domain` runtime, the budget runtime, `NativeCheck` and the var-snapshot runtime) out of `model/framework/dsl`, but into a product checker module (`model/framework/checker`, package `framework.checker`) rather than test support, because the Scala checker is the oracle. `framework.dsl` keeps declarations and the references the plugin reads and executes nothing. Expose the questions as `Checker.explore`, `verifyAll`, `monitor`, `progress` with one `Result` carrying verdict (`verified`, `refuted`, `found`, `notFound`, `unknown`), budget, completion and the counterexample or witness path with states and outcomes; a `Report` wraps results and warnings. Refuse a budget under one. Record in fn-190.15 that this task carries it.

**Size:** L

**Files:** `model/framework/checker/*.scala` (new), `model/framework/dsl/NativeCheck.scala` and the runtime files it moves, `model/framework/dsl/project.scala` and the checker module's `project.scala`, every dsl/samples/realize suite that imports the runtime.

### Verification
- A grep records no production source in `model/framework/dsl` invoking an effect or guard callable.
- The dsl, realize and samples native suites pass with unchanged test counts; lint and boundary suites pass.
- Quick: the README Native checks recipe.

## Acceptance
- [ ] Checker module outside framework.dsl; framework.dsl executes nothing
- [ ] One Result shape with verdict, budget, completion and path; budget refusal
- [ ] Native suites pass with unchanged counts; fn-190.15 annotated as carried here

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
