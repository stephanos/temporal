---
satisfies: [R3, R5, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.3 DSL completion: the Model asks nothing of itself: retire queries, Query, witness, .live, the capability live switch and Realizes.runs

## Description
**DSL completion phase (owner, 2026-10-17).** Remove from the author surface `object queries`, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`, `.live`, `.explore`, `.records`/`.ends`, `Exploration` (its `runs`, `edits`, `dropPrefix`, `priority` retire with it), the capability expectation binding's live switch and `Realizes`' `runs` argument; retire `SampleClaims` and `NativeControls`. Add the section lint rules: a `test` only inside one of the four sections, liftable-subset bodies in `examples` and `conformance`, `any` only in `explorations`, no `find`/`sample` in `examples`, conformance steps performed or awaited by a realization, unique slugs, non-empty `expect:`. Coordinate with fn-190.16 (the record-state vocabulary deletion) so each name is deleted once; this task owns the query vocabulary, fn-190.16 the record-state one.

**Size:** L

**Files:** `model/framework/dsl/Claims.scala`, `Syntax.scala`, `Capabilities.scala`, `model/framework/realize/Realize.scala`, `Exploration.scala` (deleted), `model/framework/lint/*` (new section rules), DSL tests that exercised the retired forms (rewritten or deleted), `SampleClaims.test.scala` and `NativeControls.test.scala` (deleted).

## Required refinement before implementation admission

This retained L-sized retirement/lint task must be refined against the completed public checkpoints
before it starts. Preserve R3/R5/R7 and split genuine independent mechanisms if needed. It cannot retire
native realization or capability declarations before their replacement is committed.

The existing Machine source admission excludes `*.test.scala`. Add a bounded typed suite-admission
pass and wire its actual native caller so a malformed suite with no test invocations still refuses.
Keep per-mechanism public source/library and macro refusals from the earlier checkpoints; they do
not replace this complete corpus admission. Preserve direct suites and held-source exclusions.
Replace the checker-internal author-word dependency in `checker/Execution.scala` when retiring
`verify`. Still-used private metadata for held consumers remains until its own replacement, with
existing ownership/copy/provenance controls intact.

**Touches:** [model/framework/dsl/**, model/framework/realize/Realize.scala, model/framework/realize/Exploration.scala, model/framework/lint/**, model/framework/check/**, model/framework/checker/Execution.scala]

### Verification
- A grep shows none of the retired names reachable outside `framework`; the lint suite covers each new rule with a refused and an admitted case.
- Quick: the dsl and lint native suites.

## Acceptance
- [ ] Retired names unreachable by authors; Exploration deleted; Realizes has no runs
- [ ] Section lint rules with a refused and an admitted case each
- [ ] Split with fn-190.16 recorded on both tasks

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: