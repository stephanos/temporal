---
satisfies: [R2, R16]
---
# fn-141-shrink-the-ir-generator-one-description.2 Retire the spellings no Model and no kit file uses

## Description
Part D. A spelling that only the lifter's own fixtures use still costs a matcher, a refusal and fixtures. Recount, list, confirm, remove.

**Size:** M
**Files:** `model/umpire/**`, `model/irgen/**`, `model/irgen/testdata/**`, `model/README.md`
**Touches:** [model/umpire/**, model/irgen/**, model/README.md, model/SEMANTICS.md]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Recount first. For each construct that has more than one spelling, count uses under `model/temporal` (Models and kit) and under `model/irgen/testdata`. Put the table in the done summary.
- Candidates on 2026-10-06, each to be re-checked: hand-bound `Bindings`, the hand-written `Typed*` realization constructors, `Instruction.readUntil`, the receiver-less `capabilities(limits)(…)`, the chained `.overriding(…)`, `.total(n)`. The DSL batch removes or reshapes several of them.
- Keep a spelling that a kept sugar's definition is written in: task 3 makes sugar bodies the lowering, so their core forms must stay liftable.
- Show the owner the final list before deleting anything (R2 lists candidates, the recount decides).
- Delete each retired spelling from the framework, the lifter and the fixtures together, with its lift and reject fixtures.

### Investigation targets
**Required:**
- `model/irgen/Declarations.scala` (`coreBinding`, `stepBinding`), `Realizations.scala` (`written`), `Capabilities.scala` (`declaration`, `peel`), `Claims.scala` (`totalOf`)
- `model/umpire/Machine.scala` (`Bindings`), `model/umpire/realize/*.scala`, `model/umpire/Claims.scala`
- `model/irgen/testdata/lifts/*.scala` (which fixtures exist only for a retired spelling)

## Acceptance
- [ ] A use-count table covers every multi-spelling construct; the owner confirmed the removal list.
- [ ] Each retired spelling no longer compiles; no fixture, doc or lifter arm names it.
- [ ] A spelling with a use under `model/temporal`, or one a kept sugar is written in, is still there.
- [ ] `scala-cli test model/irgen` and `make umpire-check-model` pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.
- [ ] Line counts before and after are in the done summary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
