---
satisfies: [R1, R2, R6]
---
# fn-134-capabilities-own-their-properties.4 Remove Law, Catalog, Implements and the law sidecar from the framework and lifter

## Description
Delete the old Scala surface now that nothing uses it: `Law`, `LawRef`, `Catalog`, `Implements`, the `capabilities(m, limits)(…)` functions and helpers, the old value class, `cited(…)`, the `implements` section name, the transitional both-forms refusal, and the sidecar writer.

**Size:** M
**Files:** `model/umpire/{Catalog,Capabilities,Machine,IrFile}.scala`, `model/temporal/capabilities/Catalog.scala`, `model/irgen/{Capabilities,Claims,Lifting,Lift,Context,Structure,Order}.scala`, irgen fixtures and expected files, `model/check/test/{CapabilityVocabulary,Gate}.test.scala`, `model/temporal/IrFiles.test.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/capabilities/Catalog.scala, model/check/test/**, model/temporal/IrFiles.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Delete `model/umpire/Catalog.scala` but keep `CapabilityKind` (the lifter keys on it, Capabilities.scala:183), and delete `model/temporal/capabilities/Catalog.scala`.
- Lifter: remove `implementsObject`/`implementsOf`, `declaration`'s `capabilities(…)` forms (Capabilities.scala:200-291), `catalogOf`/`lawOf`/`lawText` (:528-623), `citedOf` (:440-480), `lawSidecar` and its types (:8-111), `Context.scala:199-201`, and the sidecar writes in `Lift.scala:49-50, 136, 252-265`. Drop `implements` from `formSections` and the order lint.
- Fixtures: delete `expected/capabilities.laws.json`; migrate or delete the `implements` fixtures (lifts, crossed, NoCatalog, initOrder, sectionOrder, irFileRefusals) and the expected messages in Fixtures.test.scala (:193, 453-479, 524, 849-897, 1743, 2368) and `rejects.txt`.
- `CapabilityVocabulary.test.scala`: update its comment ('laws and catalog').
- `Gate.test.scala:725-730`: keep a stray `*.laws.json` as the example of an unproduced file. `IrFiles.test.scala:29`: drop the sidecar filter.

## Acceptance
- [ ] None of `Law`, `LawRef`, `Catalog`, `Implements`, `cited`, the `capabilities(m, limits)` functions or the old value class exist in `model/`.
- [ ] The lifter writes no `*.laws.json`; a checked-in one is refused by the gate as unproduced.
- [ ] All irgen fixtures and expected messages use `capabilities`; `scala-cli test model/irgen` and `model/check` pass.
- [ ] `make umpire-check-model` passes with `model/ir` byte-identical to task 3's output.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
