---
satisfies: [R4, R6, R10, R11, R12, R14, R16]
---
# fn-141-shrink-the-ir-generator-one-description.8 The exporter; realizations exported from constructed values (size proof)

## Description
The first declaration kind moves. The gate constructs each IR file's roots, an exporter emits their realizations, and the lifter's realization reader goes. This task also decides whether Part B continues.

**Size:** M
**Files:** a new exporter in `model/umpire`, `model/irgen/{Lift,Lifting,Realizations,Syntax}.scala`, `model/check/Gate.scala`, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/check/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Construct the roots from the same compiled classes whose TASTy the lifter reads, in the lifter's run, as `IrFile.construct` walks them today.
- Realizations are data: one generic mapping from a constructed value to the IR message of its name, plus the few constructs that need more.
- Assemble one Model per IR file from both sources. No kind comes from both (R11).
- Determinism (R10): export each lifter fixture in two separate JVM runs and compare bytes. No output may read identity, hash order or initialization order.
- Init order (R14): the lint's initialization rules run before construction; a `null` or half-made value reached during export is refused naming the declaration that holds it.
- Refusals: follow the ledger's realization rows.
- Remove the request-scope and script-helper arms task 4 left, and their lint exemptions.
- Size proof (R16): count the realization lifter's lines removed against every line this task and task 7 added. Below half removed, net, stop and ask the owner.

### Investigation targets
**Required:**
- `model/umpire/IrFile.scala` (`construct`, `queriesOf`) and `model/temporal/IrFiles.test.scala`
- `model/irgen/Realizations.scala` (`emit`, `Message`, `written`, `valueOf0`, `declaration`)
- `model/irgen/Lift.scala` (`liftRoots`: where the Model is assembled) and `model/check/Gate.scala` (how the lifter is run)
- `model/irgen/Order.scala` rules (a) and (b), as moved by task 1

## Acceptance
- [ ] Every realization in `model/ir` and in the fixtures comes from the exporter; the lifter's realization reader is deleted.
- [ ] The two-run determinism test passes and fails on a seeded nondeterminism.
- [ ] Each realization row of the ledger has its outcome: reject fixtures still refuse at their lines, or became lift fixtures.
- [ ] A `null` reached during export is refused naming its declaration.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.
- [ ] The size proof is in the done summary, with the owner's decision if the bound was missed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
