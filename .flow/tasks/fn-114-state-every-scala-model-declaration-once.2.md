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
Restated the Nexus caller Model in fn-112's declaration surface and four-file layout. Commits: 7191eb9909 (lifter fix), f73fa17494 (restatement), 4bef3fae3b (review follow-up: fixture, README wrap).

**What changed**
- **Layout (R10).** `nexuscaller/` now has `Model.scala`, `Properties.scala`, `Queries.scala`, `Realization.scala` and `IrFiles.scala`.
  - `Nexus.scala`, `Claims.scala` and `Control.scala` are deleted.
  - Domains, `object Product`, `object Protocol` and `object Control` (inspect, step functions, forged machine) moved into `Model.scala`.
  - The control's Property, Scenario, Limits and Query are package-level `forgedSuccess`, `inspectedFailure`, `control` and `forgedCompletion`. `nexus-control.json` keeps its own `irFile` declaration.
- **Names (R2).** Every machine, action, timer, Property, Scenario, Query, Limits, party, entity and observation outside `Realization.scala` takes its name from its val. Three vals were renamed to their IR names: `Control.forged` to `forgedCompletion`, `inspected` to `inspectedFailure`, and the inline `"control"` Limits to `val control`. Computed names: none here.
- **Constructs (R6).**
  - The family is a given: `object CallerFamily`, plus `object Control`'s own `temporal.nexus.control` given.
  - Input tokens live in `object Inputs`, so a path writes `schedule(Inputs.scheduleToStart := expires)` or `schedule()`.
  - Step functions use `accept`/`stay`/`disabled`/`in`, and the helpers `moves` and `productStep` are gone.
  - `restrict(workerStop, serve)`; the composition and its syncs are keyed by field.
  - The cross-entity claim and Scenario use `synced`/`own` instead of key strings.
  - Scenarios take the machine's start, and the transition claim is `once(...).keeps(_.phase)`.
- **Evidence (R5).** Every identity line is gone. Only `nexusOperationTimedOut(_)` and `pendingAttempts.name` remain: the framework requires a line for a fact with fields. No `Reads.through` remained.
- **Removed.** The dead `enum Delivery` and `val workerStop = worker.workerStop`.
- **R11.** The four caller files went from 898 lines and 114 literals to 780 and 21. The nexuscaller folder went from 2,774/324 to 2,657/231 (`metrics-{before,after}.txt`). Every remaining literal is an id the IR needs (family roots, IR file names, entity key and refer role, examples, results text, read field, exploration ids), prose, or the timed-out evidence line. The counter calls the exploration ids `once`/`none` "repeated name"; they are ids written once.

**Old form kept, with reasons (R6)**
- **`Control.forgedCompletion` is declared, not derived.** `nexusProtocol.unmonitored.rebind(...).extend(...)` lifts its source machines, which would add `nexusProtocol`, `nexusProduct`, three types and an action to `nexus-control.json` and fail the original baseline.
- **`forgedComplete` keeps the unnamed `++` of two `completeStep` results.** A `choose` alternative must be a written-out step (fn-120.1 refuses helper calls), and an unscheduled operation yields no step. This is a finding for fn-120 before it refuses unnamed branching.
- **`ProtocolState.attempts` stays `Int`, bounded by `given Finite[ProtocolState]`.** `UpTo[2]` would rewrite the require preconditions and is not an R6 construct.
- **`repliedThenStopped` keeps its explicit start.** The default start would take the worker's position in `Worker.scala`.

**Decisions taken autonomously**
- **Lifter fix outside Touches** (`model/lifter/Constants.scala`). `Entity(key = ..., refer = Map(...))` compiles to a block that binds `refer` first, and constant folding refused it. `resolve` now reads such a block as the call. The fixture is `Captured.scala`/`Spelled.scala`.
- **Step function names are kept** (`handlerReplyStep` and so on). Renaming them would have meant editing config.json's existing function substitutions and the Go tests, and config edits were to stay append-only.
- **Golden config, appended only:**
  - six per-file `source_path_merges` (Nexus, Model, Claims, Control, Properties and Queries to `model/temporal/nexuscaller`). There is no directory merge because it would swallow `closepolicy/`.
  - five `source_root_moves` (`Claims$package$.{functionalQueries,terminalHolds,stoppedWorkerRepliesNothing,syncCompletion}`, `Control$.forgedCompletion`).
- **Harness fix.** In `model/migration_golden_test.go`, `locationProjection` let a merged file match only with an extra character after its name. Now a merged file names itself.
- **Canary.** The Case bytes differ only in Scala source paths, and the canary Case identity includes them. It moved from `27d65f3b…` to `d20aef7f…`, so the pin was updated and `make umpire-rerecord-pinned-runs` re-recorded `tools/canary/assessment/testdata/nexus-caller-syncCompletion-run.json` live. The control record stayed current.
- **Regenerated:** `make umpire-gen-fixtures` and `make canary-gen-case`.
- **Doc updates:** README, `IrFile.scala` doc and the spec's IR-file example now name the root `forgedCompletion`.

**Verification.** Tables, Definition IDs, fingerprints, answers and totals are unchanged. The Case diffs are paths only. All gates were run under the heavy lock (`gates.status`):
- gen-model, OriginalBaseline and migration goldens;
- check-model, lint-model ×2 and lifter tests;
- the full Go suite;
- lint-code-fast.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with two P3s, both fixed in 4bef3fae3b: the lifter fixture for the block form, and the README wrap. FYI items are addressed above.

**For fn-114.3 (realization).**
- `Realization.scala` was touched minimally: `(using CallerFamily.family)`, `import worker.workerStop`, and `Control.forgedCompletion`.
- It still uses positional `schedule(unset, …)`; tokens `Inputs.*` are available.

**For fn-114.4 (close policy).**
- closepolicy is a chained subpackage and sees the new package-level names: `Inputs`, `satisfied`, `inconclusive`, `explanationsDisagree`, `neverEvaluated`, `control`, `forgedSuccess`, `inspectedFailure`, `forgedCompletion`, `given Accepted[nexuscaller.Outcome]`.
- The caller family given is not package-level, so a closepolicy given will not clash.
- An `Entity(..., refer = ...)` now lifts with a captured name.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7191eb9909, f73fa17494, 4bef3fae3b
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0; lower+golden in run 2, model in run 3 after the projection fix), make umpire-gen-fixtures && make canary-gen-case (exit 0), make umpire-rerecord-pinned-runs (exit 0; canary run re-recorded live, control record current), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0, twice), scala-cli test model/lifter (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: