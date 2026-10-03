---
satisfies: [R1, R2, R5, R6, R10, R11]
---
# fn-114-state-every-scala-model-declaration-once.2 Restate the Nexus caller Model with captured names, derivation and the four-file layout

## Description
Apply fn-112's settled declaration surface to the Nexus caller Model files (not its realization, which is task 3) and give the folder fn-112's file names, folding `Nexus.scala` and `Control.scala` into them.

**Size:** M
**Files:** `model/temporal/nexuscaller/{Nexus.scala (328), Model.scala (184), Claims.scala (289), Control.scala (97)}` -> `Model.scala`, `Properties.scala`, `Queries.scala`; `model/lifter/test/Fixtures.test.scala` (the `temporal.nexuscaller.Claims$package$.syncCompletion` fixture root); the task-1 IR-file declarations for `nexus-caller` and `nexus-control`.
**Touches:** [tools/umpire/internal/golden/config.json, model/lifter/testdata/**, model/temporal/nexuscaller/Nexus.scala, model/temporal/nexuscaller/Model.scala, model/temporal/nexuscaller/Claims.scala, model/temporal/nexuscaller/Properties.scala, model/temporal/nexuscaller/Queries.scala, model/temporal/nexuscaller/Control.scala, model/temporal/nexuscaller/*.test.scala, model/lifter/test/Fixtures.test.scala, model/ir/nexus-caller.json, model/ir/nexus-control.json, model/cases/**]

### Approach
- Take names from `val`s for every declaration kind in R2; for names that differ from their `val`, either rename the `val` to the IR name or use fn-112's one explicit-name form. Computed Query names keep their spelling. Keep IDs exact with one `DefinitionScope` pin per former owner (fn-112.2/.8 pattern), never per-declaration ID strings.
- Family as a `given`; derive copied machines with `rebind`/`extend`/`refining`/`assuming`/`unmonitored`; typed composition members and syncs; canonical step helpers; captured typed action inputs with `:=` (e.g. the `schedule(unset, expires, unset)` call); existing branches use fn-120's named choices.
- Delete the identity evidence lines of `nexusProduct` (`Model.scala:97-104`) and `nexusProtocol` (:132-140), keeping `pendingAttempts.name`; replace `Reads.through(nexusProtocol, nexusProduct)` (:155) with the refinement read.
- R10 layout: move `Nexus.scala`'s domains and step functions (`object Product` :52, `object Protocol` :162) into `Model.scala`; split `Claims.scala` into `Properties.scala` and `Queries.scala`; fold `Control.scala`'s forged machine into `Model.scala` and its Query into `Queries.scala`, keeping `object Control` as its vocabulary and `nexus-control.json` as its own IR file through the task-1 declaration. The forged machine keeps its own family `temporal.nexus.control` (`Control.scala:21`) through a `given` scoped inside `object Control` (or the one explicit family form), never the package's caller `given`; its `val` stays inside `object Control` so it cannot collide with the `forgedCompletion` Query at package level. Delete `Nexus.scala`, `Claims.scala` and `Control.scala`.
- A construct that does not fit a Nexus declaration stays in the old form for that declaration only and goes in the done summary with the reason (R6).

### Investigation targets
**Required:**
- `model/temporal/nexuscaller/Model.scala:90-160` - evidence lines and `Reads.through`
- `model/temporal/nexuscaller/Claims.scala`, `Nexus.scala`, `Control.scala`
- `model/temporal/standaloneactivity/` after fn-112 - the target style to mirror
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.8.md` - DefinitionScope-on-move procedure
**Optional:**
- `model/lifter/test/Fixtures.test.scala:105-112`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model
make lint-model
```

### Execution constraints
- Lifted meaning frozen: IR deltas only from the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows), with this task's moves added as entries (including the `source` label of the lifter fixture's `expected/realizations.json` when `syncCompletion` moves); tables, IDs, fingerprints, answers and Case bytes exact. Any other diff stops the task (R1 errors clause).
- Behavior-neutral: do not add validation while moving code (memory: behavior-neutral refactors must not strengthen validation).
## Acceptance
- [ ] Every R2 declaration in `nexuscaller` (excluding `Realization.scala`) takes its name from its `val` or the one explicit form; action inputs use captured tokens.
- [ ] No identity evidence line and no `Reads.through` remain; `pendingAttempts.name` is the only evidence exception.
- [ ] The folder has `Model.scala`, `Properties.scala`, `Queries.scala`, `Realization.scala` and no `Claims.scala`, `Nexus.scala` or `Control.scala`; Definition IDs are unchanged.
- [ ] Any construct kept in old form is listed with its reason in the done summary.
- [ ] R1 goldens and fn-112.1 equivalence pass under the allowed deltas only; model gate and lint-model pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
