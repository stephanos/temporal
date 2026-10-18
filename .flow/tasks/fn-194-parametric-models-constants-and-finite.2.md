---
satisfies: [R1, R2, R3, R4]
---
# fn-194-parametric-models-constants-and-finite.2 DSL completion: the class form of a Model in the DSL and the lint

## Description
**DSL completion phase (owner, 2026-10-17).** Admit `class M[D: Finite, ...](constants...) extends Machine[O, F]` as decided by the prototype: parameters readable by var initialisers and `oneOf` starts; one `VarSchema`, var runtime, start set and reachable graph per instance; an instance named by the `val` or suite holding it, with a located refusal when a check asks the name of an unheld instance. The value-parameter kinds are `Finite` types, `Int`, `Boolean`, and `Set`/`Map` of those; a type parameter without a `Finite` bound or a value parameter of another kind is a compile error naming the parameter. The structure lint admits the class form with the object form's section rules and extends its refusal text to name both forms. `object M extends Machine` is unchanged.

**Size:** L

**Files:** `model/framework/dsl/Machine.scala`, `model/framework/dsl/Vars.scala` (or where the schema given is derived), `model/framework/lint/Order.scala`, new DSL tests beside them (`ParametricMachines.test.scala`, a lint case in the lint suite).

### Verification
- DSL tests: a parametric Machine checks from parameter-dependent starts; two instances do not share state; the naming refusal fires; the `Finite`-bound and parameter-kind compile errors are asserted with `compileErrors`.
- Lint tests: the class form with sections passes; declarations outside sections are refused with the extended text.
- Quick: the dsl and lint native suites via the README Native checks recipe.

**Touches:** `model/framework/dsl/**`, `model/framework/checker/**`, `model/framework/lint/**`, `model/framework/check/lint/**`.

**Prototype-driven native seams (conductor, 2026-10-18).** The committed .1 verdict (`9648caeb9d`, closure `f74ab768fe`; `model/design/parametric-models.md`) refutes direct generic state/input schemas, instance-dependent `before`, class-owned claim acquisition, holder naming and constructor/lint admission. Implement the intended Finite-only class API, not the prototype's integer-address fallback, additional author evidence bounds or .equals substitution. Reuse selected structural evidence lawfully and preserve native collection/Inbox/opaque validity admission; do not blanket-admit arbitrary Finite leaves as collection members. Preserve exact instantiated receiver types in schemas and before, including generic class instances used as composition members. Evaluate class claim sections on the exact owning instance without accepting foreign/facet sections or running guards/effects/catalogs during metadata acquisition. Automatic holder naming must reach public result subjects; refuse unheld constructor expressions contextually, preserve existing object names, and define/test aliases/rechecks without guessing internal helper variable names. Do not introduce a manual naming argument as a substitute. Constructor checks must locate/name the invalid parameter and cover nested Set/Map leaves, missing Finite bounds and unsupported callable/object values; positive controls include the spec's Int/Boolean and Set[Int] constants. Preserve existing object/derived/native admission controls. The unchanged-count requirement means existing controls remain covered; add new tests instead of suppressing or deleting tests to retain a number.

**Verification boundary.** Existing native baseline is600 tests and lint31. No baseline handoff is valid after conductor milestone edits; run an own focused baseline. Red-first tests should demonstrate the actual prototype gaps, then test the public API in source and cache-isolated separate-library clients under unchanged strict flags. Run native dsl/realize/samples and lint/check controls affected by header or ownership changes, fresh declaration-only/core/plugin compilation and realization-mode sample clients where their paths are touched. Keep the historical prototype/verdict as pre-change evidence, not an author API compatibility requirement or a target to weaken. The three sample migrations and instance realization fixture conversion belong to .3; prose migration belongs to .4. IR/export/testdata/Go/Temporal, schema generation, broad GuideSourcesTest and two real-Go sample bodies remain held. No Make green claim: model-check-changed routes to held Temporal sources. Record authorized native substitutes and all holds. Source-free user-facing REFERENCE stays free of Scala-file pointers.
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
