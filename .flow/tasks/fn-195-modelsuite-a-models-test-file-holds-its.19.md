---
satisfies: [R3, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.19 Implement immutable native exploration spaces and selectors

## Description
Implement R3's reusable native space declaration and its local R7 refusals. Consume committed .2 and the lifecycle/native runtime checkpoint `ff0dcaf55d460f887bf5fb56d7e31b4ecd92d7f1`; full .8 is still incomplete. This leaf supplies declarations to the question adapter leaf. Original .9 retains actual Registry integration.

**Size:** M, immutable selector capture adapting the existing typed action and run-position machinery, with four acceptance groups.
**Files:** new `model/framework/dsl/Spaces.scala`, `model/framework/dsl/ModelSuite.scala`, new `model/framework/dsl/Spaces.test.scala`, `model/framework/REFERENCE.md`; conditional `model/framework/dsl/InputSource.scala` only for typed per-input expansion.
**Touches:** [model/framework/dsl/Spaces.scala, model/framework/dsl/ModelSuite.scala, model/framework/dsl/Spaces.test.scala, model/framework/dsl/InputSource.scala, model/framework/REFERENCE.md, .flow/tmp/fn195-native-spaces/**]

### Approach
- Add the suite's `explore(steps = n)` declaration factory and DSL-owned immutable exact-instance space, using the existing `ExplorationSpace` marker admitted by `Sections.scala:146`. Keep question registration for the next leaf. `ModelSuiteMacros` lives inside `ModelSuite.scala:165`; no standalone file exists.
- Reuse `RunBody.scala:5` positions and `RunBodyMacros.roots` for exact owner checks. Its target helper rejects input-bearing whole actions at lines 104-109; give exploration capture an explicit separate admission path without weakening authored-run rules. Reuse `InputSource` finite domain metadata for per-input `any`, retaining named constraints, typed field identity and deterministic class ordering. Do not enumerate a replacement domain.
- Retain selectors and ordinary named expressions without executing actions, guards, effects or claims during capture. Evaluate each supplied constraint once at the deferred space acquisition boundary, in authored order, retain that immutable value across reuse and locate failures at its authored expression. Existing position-select callbacks do not restore reached state; do not claim per-reached-state reevaluation or add it accidentally. Test acquisition timing and unchanged surrounding Model state explicitly.
- Bound declared selector positions by space.steps and preserve existing search semantics for unspecified positions. Retain test-level `from`/single-start resolution for the later question adapter; do not invent an explore-from API. Foreign starts/classes/space receivers refuse, and a question missing `from` on a multi-start Model retains the parent's refusal.
- Define DSL-owned positioned find-target and seeded sample-request carriers/signatures here for the next adapter to consume. Retain every target condition and sample horizon/count/seed without installing execution early or importing checker types.
- Add minimum accurate user-facing REFERENCE coverage for the shipped declaration only, without Scala-file references or claiming installed questions/conformance. Run the existing ReferenceSurface gate. Final .5 still owns complete guidance.

### Investigation targets
**Required:**
- `model/framework/dsl/RunBody.scala:76` - capture validation, stable roots, positions and deferred targets.
- `model/framework/dsl/InputSource.scala` - existing finite typed input domains.
- `model/framework/dsl/ModelSuite.scala:165` - actual inline macro receiver and section context.
- `model/framework/dsl/Sections.scala:146` - immutable exploration-space member admission.
- `model/framework/checker/Spaces.scala:7` - exact starts and finite position selector contract.
**Optional:**
- `model/framework/check/lint/ReferenceSurface.test.scala` - existing public reference checks.

### Verification and handoff
Use focused native selectors and fresh framework-main plus authored test compilation, then a separately packaged framework/macro jar and outside-package client under unchanged `dsl/project.scala` strict flags. Allocate a fresh Scala CLI workspace for each complete recipe; same-path jar replacement is insufficient. Preserve commands, terminal exits, nonzero counts, original sources and artifact hashes under this task's scratch root. Prove declaration compile/load without checker initialization, zero capture callbacks and located intended refusals. The next leaf must consume actual retained declarations, not hand-built selector substitutes.

No Registry/gate caller, policy parsing, realization selection, capability production or Case eligibility changes belong here. Keep .8/.14/.11 and fn-196.2's ordering contract open. Temporal reads, actual repository Make, Go, export/lifting/IR/schema/backend/generated/regeneration and server/recordedRun/Case operations remain held. Export compile-only approval remains ungranted. Formal review stays phase-batch deferred.

## Acceptance
- [ ] A public reusable `explore` declaration captures whole actions with inputs and independently varying per-input `any`, preserving remaining named typed constraints, bounded selector positions, unspecified-position search, domain ordering/correlations and positive step bound. An admitted fixed-input neighbor pins unchanged ordinary authored-run behavior. Find-target/sample-request carriers preserve every condition and requested bound for the next adapter.
- [ ] Fresh exact Model instances retain separate spaces, starts and classes; aliases on the same exact instance remain valid. Foreign owners/starts/classes and nonpositive steps refuse at their authored locations. Repeated acquisition/use cannot mutate an earlier descriptor or leak Model state.
- [ ] Constraint expressions evaluate once at deferred space acquisition and remain immutable across questions; declaration capture/load executes no effect/guard/claim callback. Direct `any` outside exploration capture, including examples and conformance, refuses at the authored expression with an admitted exploration neighbor.
- [ ] Focused native, fresh source and separate-library controls execute nonzero counts under unchanged strict flags; declaration-only loading works without checker execution. REFERENCE and ReferenceSurface agree on the shipped subset, and the receipt assigns question execution to the next leaf and actual Registry integration to original .9.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
