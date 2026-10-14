---
satisfies: [R1, R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.4 Machines construct their messages: header, rules with captured functions, monitors, refinement, derivations

## Description
Depends on full .1, full .3 and the committed .16 core-binding and .17 header/type source checkpoints. `Rules.on` and the other machine-level capture points (`effect`/`is`, `end`, monitor `next`/`after`, refinement maps, evidence) call `Reify.function` on their `inline` argument and store the `ir.Function` on the rule, monitor or refinement value. `Machine` builds `ir.Machine` with `StepBinding`s from its rules and the functions list from what its values hold; rule lowering (`where`/`always`/`~>` to the step function's `if` chain) is done on constructed `Rule` values, as fn-141.10 planned. Monitors, refinement and derivations follow. The lifter's machine arms and its rule lowering go.

Task .16 source `17706a4127` retains captured core binding propagation, exact Action declaration references and compiler-owned named/anonymous roots through native tables and scoped derivations. Private ordered StepBinding/function/type contributions preserve native callable and table timing, initial duplicate raw admission and pre-reduction operation refusals. The frozen 273-case source run and independent committed-source checks (13 focused cases and five compiled cases from a checksum-identical temporary snapshot) establish only that isolated capability; the first root compiled attempt was disk-exhaustion setup failure, not semantic evidence. Type duplicates ignore only top-level Position while retaining the first original message. Copy/function-alias controls retain actual old-refusal qualifications.

Task .17 source `cb871e4406` retains compiler-owned Machine identity and exact S/O/F witness selection with a private five-field header/type contribution. Frozen checks pass 232 cases and configured formatting on 12 Scala files. Root independently verifies nine runtime cases, four compiled header cases and three Actor cases, plus all committed-source hashes and retained old artifacts. Conversion excludes init/end/rules/catalog reads and production emission. Malformed roots/fields/shapes, wrong keys and missing/conflicting provenance or closure refuse; scalar blanks and Nothing omission retain historical projection. Nested Inbox ownership requires exact selected nested witnesses; complete supported Machine construction must retain those dependencies without changing author spellings. Actual old-refused state/shared-witness/helper controls remain qualified. This is source preparation, not full Machine acceptance.

Complete Machine headers/init/end/evidence, ordinary Rules capture/lowering, monitors, refinement, complete derivations, automatic supported nested-witness propagation, G01 channel delivery/loss checks, consumer integration, production activation, reader retirement, full R1/R3/R4/R5 and every original full .1/.3 prerequisite remain here. The isolated collectors neither construct a complete Machine nor activate emission. Independent review and broad gates remain at source-batch end.
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
