---
satisfies: [R2, R19]
---
# fn-115-make-the-scala-model-the-model-and.8 Split the TASTy lifter through a shared typed context

## Description
Split the TASTy lifter through a shared typed context. Implements R2, R19 using the reviewed parent contracts.

**Size:** M
**Files:** model/lifter entrypoint, typed context and concern modules; lifter focused tests
**Touches:** [model/lifter/**, model/gate/**]

### Approach
- The conductor owns shared module-map and migration-manifest updates after this task returns; return any required changes in the task-specific handover instead of editing those shared files.
- Extract the shared Quotes/symbol/type state from the current nested liftAll structure into one typed context. Separate types, expressions, declarations, realizations, compositions/claims and the entrypoint according to the map.
- Preserve recursion, symbol resolution, exact supported subset and source diagnostics. No ScalaPB conversion, root discovery redesign, compiler-name normalization or new author syntax belongs here.
- Keep the existing gate working during extraction; do not migrate fixture orchestration yet. Compare every checked and fixture IR against the exact current baseline and exercise refusals after each extraction.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/lifter/Lift.scala:76`
- `model/scalav2/lifter/Lift.scala:167`
- `model/scalav2/lifter/Lift.scala:364`
- `model/scalav2/lifter/Lift.scala:772`
- `model/scalav2/lifter/Lift.scala:1278`
- `model/scalav2/lifter/Lift.scala:1964`

### Quick commands
The renamed model check command from task 1; make lint-model; the semantic golden test

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Lifter entrypoint and concern modules share one typed context and preserve the admitted language and diagnostic locations.
- [ ] Every selected and fixture lift is unchanged relative to the post-relocation baseline; refusal checks still fail for the intended reasons.
- [ ] No later-spec ScalaPB, DSL or root-declaration changes are introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
