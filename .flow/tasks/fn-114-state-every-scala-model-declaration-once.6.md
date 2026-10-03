---
satisfies: [R1, R2, R6, R10]
---
# fn-114-state-every-scala-model-declaration-once.6 Restate the worker Model and the lifter fixtures with captured names and named choices

## Description
Convert the remaining consumers of string-named forms and unnamed branching outside the Nexus folders: the worker entity Model and every lifter fixture. fn-120.2 turns on the unnamed-branch refusal only after this spec has migrated all fixtures, so this task owns that fixture migration.

**Size:** M
**Files:** `model/temporal/worker/Worker.scala` (76 lines; `Family("temporal.worker")` :17, `action("workerStop", party)` :52-54, `machine(..., "polling")` :71) -> `worker/Model.scala` (plus Properties/Queries only if it has such declarations); `model/lifter/testdata/**` and `lifts/expected` IR; `model/lifter/test/Fixtures.test.scala`.
**Touches:** [tools/umpire/internal/golden/config.json, model/temporal/worker/**, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala, model/ir/nexus-caller.json]

### Approach
- Worker: captured names, `given` family, DefinitionScope so `worker.polling` keeps its ID (it is lifted into `nexus-caller.json`). Rename `Worker.scala` to `Model.scala`; its functions `stopStep`, `resumeStep`, `serveStep` change from `Worker$package$` to `Model$package$` in `nexus-caller.json`, recorded as `function_name_substitutions` entries.
- Positive fixtures: rewrite every string-named declaration to the captured form and every multi-result branch to fn-120's named choices, keeping expected IR identical except for inert choice names. Refusal fixtures that deliberately exercise a form task 7 retires stay for task 7.
- Fixture roots stay fully qualified in the test (spec: fixtures are the only place outside Models naming Scala sources); update any root a rename changed. Depends on task 2 (both edit fixture roots in `Fixtures.test.scala`) and task 5 (both add entries to the golden `config.json`).

### Investigation targets
**Required:**
- `model/temporal/worker/Worker.scala`
- `model/lifter/test/Fixtures.test.scala:76-245`
- `model/lifter/testdata/lifts/`
- `.flow/tasks/fn-120-adopt-what-quint-does-well-named.2.md` - what fn-120.2 expects to find migrated
**Optional:**
- `.flow/memory/bug/integration/feature-modules-must-import-temporal-2026-09-28.md` - pin worker IDs after a move

### Quick commands
```bash
scala-cli test model/lifter
make umpire-check-model
```

### Execution constraints
- Expected fixture IR and `nexus-caller.json` change only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows).
## Acceptance
- [ ] Worker declarations take names from `val`s; the folder uses `Model.scala`; `worker.polling` ID unchanged.
- [ ] Every positive lifter fixture uses captured names and names every multi-result branch; expected IR otherwise unchanged.
- [ ] Lifter tests, model gate and R1 goldens pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
