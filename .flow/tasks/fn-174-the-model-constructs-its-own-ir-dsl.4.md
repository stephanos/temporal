---
satisfies: [R1, R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.4 Machines construct their messages: header, rules with captured functions, monitors, refinement, derivations

## Description
Depends on full .1, full .3 and the .16 core-binding source checkpoint. `Rules.on` and the other machine-level capture points (`effect`/`is`, `end`, monitor `next`/`after`, refinement maps, evidence) call `Reify.function` on their `inline` argument and store the `ir.Function` on the rule, monitor or refinement value. `Machine` builds `ir.Machine` with `StepBinding`s from its rules and the functions list from what its values hold; rule lowering (`where`/`always`/`~>` to the step function's `if` chain) is done on constructed `Rule` values, as fn-141.10 planned. Monitors, refinement and derivations follow. The lifter's machine arms and its rule lowering go.

Task .16 prepares captured core binding propagation and isolated ordered StepBinding/function/type contributions after committed .15. It retains exact compiler-owned root provenance and pre-reduction operation evidence, preserves initial duplicate raw admission and qualifies copy/function-alias controls against actual old refusals. It neither constructs a complete Machine nor activates emission. Full header/init/end/evidence, ordinary rule capture/lowering, monitors, refinement, complete derivations, G01 channel delivery/loss checks, reader retirement, full R1/R3/R4/R5 and all original .1/.3 dependencies remain here.
## Acceptance
- Every machine, its rules' functions, monitors, refinements and derivations come from constructed values; the lifter's machine arms are deleted (R3).
- Function IR equals the tree path's with positions stripped, for the samples and `activity_mini` (R1).
- Names equal today's; function positions present (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
