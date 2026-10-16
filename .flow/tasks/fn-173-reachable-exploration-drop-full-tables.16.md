---
satisfies: [R3]
---
# fn-173-reachable-exploration-drop-full-tables.16 Retire native StepTables and catalog-based refinement/law walks

Touches: [model/framework/dsl/StepTables*, model/framework/dsl/Refine.scala, model/framework/dsl/*Laws*.test.scala, model/framework/dsl/IrFile.test.scala, model/framework/dsl/Compose.scala, model/framework/dsl/IrFile.scala, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Retire the native producer/walk portion of R3 after the completed six-sample fixture/map bundle. Original .3 retains Go replacements, orchestration, carriers and integration; this is not a replacement semantic seal.

**Size:** M, bounded deletion and concrete framework-test migration. Split before expanding into new checker or compatibility machinery.
**Files:** framework/dsl/StepTables.scala and its test; Refine.scala; MachineLaws.test.scala, MachineLawsHole.test.scala, IrFile.test.scala; comment references in Compose.scala/IrFile.scala; framework README/migrate README and model/README.md law/seal guidance.

### Approach
- Remove the StepTables producer, writeStepTables main and dedicated producer tests. Scoped pinned searches found no other executable framework symbol caller; refresh that audit first. Excluded Temporal callers are held debt, not a reason to retain aliases/stubs or inspect disabled sources.
- Remove Refinement.unclosed and MachineLaws all-catalog image/totality, role-closedness and repeated relation/state/outcome/fact membership walks. Keep Refinement/declaredBy and actual binding-order/duplicate-binding checks under clear concrete structural helpers, not a MachineLaws.check facade returning pass/empty/skipped semantic results.
- Re-anchor each IrFile.test.scala caller (pinned :286,322,335,351,389,400,408,432,451,463,477) and MachineLawsHole tests. Preserve discovery, owners, derivations, restrictions, actual Hole identity/exception examples and concrete callback cases where meaningful. Catalogue removed universal guarantees in held .3; do not rename the same catalog checker or delete assertions as supposed replacement proof.
- Preserve the completed old-source/map/expectation bundle. Leave private Query.total None and exporter/carriers held. Retain pure canonical comparison, RenameMap, lift comparison and synthetic negative controls; do not blanket-delete check/equivalence or disable Samples' lift-and-Go harness.
- ChangeCheck/Makefile historical equivalence orchestration and consumer reconciliation remain original .3 work. Document that the old source-stage seal cannot run after producer retirement; do not mark unavailable or unrun stages passed. No Makefile/exporter/schema/Go/Temporal changes in this task.

### Investigation targets
Required: dsl/StepTables.scala/test, Refine.scala, MachineLaws.test.scala, MachineLawsHole.test.scala, IrFile.test.scala; completed retained sample bundle; parent source-tail contract and original .3 proof inventory. Read scoped framework callers only, not model/temporal or archived Temporal source.

### Verification
Compile framework native roots and run real nonempty IrFile discovery, concrete binding/Hole and touched syntax selections, scoped lint/format, plus scoped retired-symbol/walk reference audit. No whole-state table/law execution or new explorer. Native count reductions must match explicit retirement, not broad skips. No old baseline/equivalence, exporter, Go, backend, Temporal or regeneration gate.

## Acceptance
- [ ] StepTables/writeStepTables and Refinement.unclosed/native catalog law evaluation are retired with framework callers reconciled; no compatibility facade or new semantic checker remains.
- [ ] Concrete binding-order, discovery/ownership, derivation and useful Hole/callback controls survive; actual native construction does not enumerate state or evaluate callbacks. Nonzero focused suites pass.
- [ ] Source/utility references and guidance distinguish retired native producers from retained pure comparison and unavailable historical orchestration; six old fixtures/maps remain available.
- [ ] Lost universal law/coverage/role/refinement guarantees and complete R3/R4 replacements, total carriers and consumer compatibility remain explicit held integration obligations, not passed seals.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
