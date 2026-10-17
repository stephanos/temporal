---
satisfies: [R1, R2, R3, R4]
---
# fn-195-modelsuite-a-models-test-file-holds-its.2 DSL completion: ModelSuite, the three sections, expect blocks, spaces and expectations in the DSL

## Description
**DSL completion phase (owner, 2026-10-17).** Land the suite as the prototype decided: `ModelSuite(instance)` with `examples`, `explorations`, `conformance` and `unit` (plain munit) sections, nothing but sections and shared helpers at the top level; `test(name, from, expectation, through)`; bodies of named action calls, whole actions, `oneOf` families and `expect:` blocks replayed on the instance by the checker from `from` or the single start (refused without `from` when init is a set); `explore(steps)` spaces, `space.find(steps)` with an `expect:` target, `space.sample(count, seed)`, `refuted(machine, claim)`; `conformance` as the only declarations the Case generator reads, each replayed on the instance first, with the expectation defaulting to `satisfied` for a path and `conformant` for a sample, `through` required only with several realizations and refused for another Model's. Move `satisfied`, `inconclusive(reason, because)` and add `conformant` as constructors in the realize package. Case names `<instance>.<slug(test name)>`, a sample's `<instance>.<slug>.<seed>.<index>`. Refusals as the spec's API Contracts list, each located at the test.

**Size:** L

**Files:** `model/framework/dsl/ModelSuite.scala`, `Sections.scala`, `Expect.scala`, `Spaces.scala` (new), `model/framework/realize/Expectations.scala` (constructors moved from export testdata), `model/framework/checker` call sites, tests beside each (`ModelSuite.test.scala`, `Expect.test.scala`, `Spaces.test.scala`).

### Verification
- DSL tests cover each refusal in API Contracts with `compileErrors` or a located lint assertion; the defaults for `expectation`; `through` binding with one, several and a foreign realization; the Case-name slugs.
- Quick: the dsl native suite via the README Native checks recipe.

## Acceptance
- [ ] ModelSuite with the four sections registers and runs as specified; a top-level test is refused
- [ ] expect: bare Booleans are assertions with line and state; from, oneOf, any, find, sample, refuted as specified
- [ ] conformance defaults, through binding and Case names as specified; satisfied/inconclusive/conformant in realize
- [ ] Every API Contracts refusal has a test

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
