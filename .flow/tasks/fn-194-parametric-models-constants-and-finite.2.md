---
satisfies: [R1, R2, R3, R4]
---
# fn-194-parametric-models-constants-and-finite.2 DSL completion: the class form of a Model in the DSL and the lint

## Description
**DSL completion phase (owner, 2026-10-17).** Admit `class M[D: Finite, ...](constants...) extends Machine[O, F]` as decided by the prototype: parameters readable by var initialisers and `oneOf` starts; one `VarSchema`, var runtime, start set and reachable graph per instance; an instance named by the `val` or suite holding it, with a located refusal when a check asks the name of an unheld instance. The value-parameter kinds are `Finite` types, `Int`, `Boolean`, and `Set`/`Map` of those; a type parameter without a `Finite` bound or a value parameter of another kind is a compile error naming the parameter. The structure lint admits the class form with the object form's section rules and extends its refusal text to name both forms. `object M extends Machine` is unchanged.

**Size:** M

**Files:** `model/framework/dsl/Machine.scala`, `model/framework/dsl/Vars.scala` (or where the schema given is derived), `model/framework/lint/Order.scala`, new DSL tests beside them (`ParametricMachines.test.scala`, a lint case in the lint suite).

### Verification
- DSL tests: a parametric Machine checks from parameter-dependent starts; two instances do not share state; the naming refusal fires; the `Finite`-bound and parameter-kind compile errors are asserted with `compileErrors`.
- Lint tests: the class form with sections passes; declarations outside sections are refused with the extended text.
- Quick: the dsl and lint native suites via the README Native checks recipe.

## Acceptance
- [ ] Class-form Machines compile, check and name as specified
- [ ] Compile errors for an unbounded type parameter and a non-liftable value parameter
- [ ] Lint admits the class form and refuses out-of-section declarations naming both forms
- [ ] Object-form Machines unchanged; existing dsl/lint suites pass with unchanged counts

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
