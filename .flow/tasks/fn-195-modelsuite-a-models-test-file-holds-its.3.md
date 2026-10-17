---
satisfies: [R5, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.3 DSL completion: the Model asks nothing of itself: retire queries, Query, witness, .live, the capability live switch and Realizes.runs

## Description
**DSL completion phase (owner, 2026-10-17).** Remove from the author surface `object queries`, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`, `.live`, `.explore`, `.records`/`.ends`, `Exploration` (its `runs`, `edits`, `dropPrefix`, `priority` retire with it), the capability expectation binding's live switch and `Realizes`' `runs` argument; retire `SampleClaims` and `NativeControls`. Add the section lint rules: a `test` only inside one of the four sections, liftable-subset bodies in `examples` and `conformance`, `any` only in `explorations`, no `find`/`sample` in `examples`, conformance steps performed or awaited by a realization, unique slugs, non-empty `expect:`. Coordinate with fn-190.16 (the record-state vocabulary deletion) so each name is deleted once; this task owns the query vocabulary, fn-190.16 the record-state one.

**Size:** L

**Files:** `model/framework/dsl/Claims.scala`, `Syntax.scala`, `Capabilities.scala`, `model/framework/realize/Realize.scala`, `Exploration.scala` (deleted), `model/framework/lint/*` (new section rules), DSL tests that exercised the retired forms (rewritten or deleted), `SampleClaims.test.scala` and `NativeControls.test.scala` (deleted).

### Verification
- A grep shows none of the retired names reachable outside `framework`; the lint suite covers each new rule with a refused and an admitted case.
- Quick: the dsl and lint native suites.

## Acceptance
- [ ] Retired names unreachable by authors; Exploration deleted; Realizes has no runs
- [ ] Section lint rules with a refused and an admitted case each
- [ ] Split with fn-190.16 recorded on both tasks

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
