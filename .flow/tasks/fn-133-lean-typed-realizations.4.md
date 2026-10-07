---
satisfies: [R6, R13]
---
# fn-133-lean-typed-realizations.4 Coverage report, class-pattern rule, deadlines bound once, action-level onPath

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part B, R6, R13.

1. **Settle the class-pattern rule first.** Read the lifter and the Go lowering to find whether `caller.start(scheduleToStart := expires)` names one exact class or every class with that input. Record the answer in the spec (closing its Parked unknown) and in README.
2. **Coverage report.** The gate or lint reports a `perform`/`onPath` binding of a class no realizable path takes (an unreachable binding, e.g. the activity's `deadline.scheduleToClose` await), and a class of a performed action that a Query's path takes but no binding covers. One fixture each.
3. **`deadlines(input -> field, …)`.** One declaration per realization generates the binding of every class of the start or schedule action, combined classes included, and marks classes the server refuses as unrealizable (they show in the report). The activity's and the Nexus caller's per-class variants go.
4. **Action-level `onPath`.** The Nexus caller's three-class list becomes `onPath(caller.schedule)`.

Combined classes may become realizable and add Cases. List each one; no existing Case may change.

## Acceptance
- [ ] The class-pattern rule is recorded in the spec and README.
- [ ] The report's two fixtures fire; the three realizations have no unreachable binding and no uncovered class that is not marked unrealizable.
- [ ] `deadlines(…)` replaces the per-class variants; a declaration naming a non-`Timeout` input is refused at its line.
- [ ] A before/after projection: existing Cases are identical; new Cases are listed.
- [ ] The spec's Verification gates pass.

## Done summary
Settled the class-pattern rule: a class pattern is exact. Added umpire-lint's coverage report: `unreachable-binding` and `uncovered-class`. Deadlines are now bound once with the kit's `deadlines` declaration, and the Nexus caller's three-class `onPath` is written `onPath(caller.schedule)`. A scratch lift shows each existing Performance byte-equal to the baseline's once positions are stripped; the only IR changes are the listed additions and the removal.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### Class-pattern rule (closes the spec's parked unknown; please record it in the spec)
The pattern is **exact**:
- `start(scheduleToStart := expires)` lifts (irgen Syntax.scala `named`) to the positional class with every omitted input at its domain's first value: `start(unset, expires, unset)`.
- Go keys a class by its action and every input value (interp.ClassKey, check/claims.go `classKey`).

It is recorded in model/README.md.

### What changed
- **umpire-lint** (`tools/umpire/lint/bindings.go`, kinds registered in lint.go). A realizable path takes only steps of bound classes (perform or activity start) or of system actions. The report has two kinds:
  - `unreachable-binding` (owner: the realization): a perform or onPath class that no state on a realizable path enables.
  - `uncovered-class`: a class of a performed non-system action that a realizable state enables and no binding performs.
  - Tests: `bindings_test.go` (TestUnreachableBindings, TestUncoveredClasses), built over checked-in IR plus mutations, so they hold after the regeneration. `testdata/coverage.golden` is regenerated for the two new count lines.
- **Kit** (`Modules.scala`): `deadlines[M](action, call, value, unset = Some(input -> command))(input.sets(_.field), …)`, plus `DeadlineField` and the `sets` extension.
- **Lifter** (`Realizations.scala` `deadlinesItem`):
  - It emits one perform item. Its classes follow a binary count over the declared inputs, with the first declared input as the lowest bit: `{}`, `{sts}`, `{stc}`, `{sts, stc}`. This reproduces the existing order and adds the combined class last.
  - Each command is the call (or the `unset` entry's command) with the expiring inputs' deadline fields set to `value`. On an rpc, the fields become assignments `<path>.seconds`; on a workflow command, they are appended to the protobuf at the path.
  - Command names are the base's: `start-activity` and `start-nexus-operation`.
  - Refusal: an input that is no `Timeout` of the action (ScriptRejects.scala:181).
  - `onPath(classes: (ClassRef | Action[?])*)`: an action with inputs expands to the classes the realization's `deadlines` declaration of it bound. It is refused when there is none.
- **Realizations:**
  - activity: `deadlines[StartActivityExecutionRequest](client.start, startActivity, duration(deadlineSeconds), unset = Some(startToClose -> startUnreached))(scheduleToStart.sets(…), startToClose.sets(…))`. The timed-out await's `onPath` drops `deadline.scheduleToClose`, which was an unreachable binding. `Timeout.expires` is no longer imported.
  - nexus caller: `deadlines[ApiCommand](caller.schedule, startNexusOperation, requestDeadline)(…)` and `onPath(caller.schedule)(awaitNexusOperation)`. The local `scheduling` factory is gone.
- **Accepted findings** (model/ir/*.lint.json, author-written, hand-added):
  - activity `standalone`: the four `start-expires-*` classes (schedule-to-close is unrealizable: no start sets it).
  - `asyncNexus` / `forgedCompletion`: the four `schedule-expires-*` classes, plus `complete-canceled` and `reply-operationCanceled` (the Case's handler realizes no canceled answer).
  - Checked: `go run ./tools/umpire/cmd/umpire-lint` over the scratch-lifted IR, with these lint.json files copied beside it, exits 0. It reports no unreachable binding and no unaccepted uncovered class.

### Declared IR delta (batch regeneration)
- activity-standalone.json, realization `standalone`, script `controller`:
  - The start perform gains `start(unset, expires, expires)`, whose command is start-activity with `schedule_to_start_timeout.seconds = 2` and `start_to_close_timeout.seconds = 2` appended.
  - The timeout await's `when` drops `deadline.scheduleToClose()`.
- nexus-workflow.json (`asyncNexus`) and nexus-workflow-control.json (`forgedCompletion`), script `workflow`:
  - The schedule perform gains `schedule(unset, expires, expires)`, whose command is start-nexus-operation with both deadline fields appended.
  - The await item's `when` gains that class.
- Everything else: positions only. Existing Performances are byte-equal with positions stripped.
- **New Cases:** none expected. No Query's witness takes a combined class: the manifest shows no unperformed standing, and searches do not depend on bindings. The batch's gen-cases run confirms it; any new Case there comes from these two classes.
- **After the regeneration:** rerun `go test -tags test_dep ./tools/umpire/lint -run TestCoverageSummaryIsPinned -update-coverage`. The golden pins model/ir.
- **Until the regeneration, `make umpire-check-lint` over the stale model/ir fails.** The new acceptances do not match the baseline IR's findings: unreachable scheduleToClose, and the combined classes uncovered.
- The lifter fixture `rejects.txt` gains ScriptRejects.scala:181.

### Tests
- `go test -count=1 -tags test_dep ./tools/umpire/lint/` passes; `./tools/umpire/cmd/umpire-lint/` passes.
- `mise exec -- scala-cli test model/irgen`: 96 passed.
- The model unit tests pass.
- `--check-syntax` and `--check-comments`: clean.
- A scratch lift compared per script item (/tmp/laneE-tools/items.py) shows exactly the deltas above.
- `gofmt` and `go vet` on the lint package: clean.

### For later tasks
- `deadlineClasses`, `commandOrigins` and `factsNamed` are reset per realization in `realizationOf`.
- The kit's `deadline`/`unreachedDeadline` operands are still unused by realizations; only lifts/Scripts.scala uses them (fn-133.5).
- In the new lint kinds, a Tally's owner is the realization name, not the machine.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 800fe504d0
- Tests: go test -count=1 -tags test_dep ./tools/umpire/lint/, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-lint/, mise exec -- scala-cli test model/irgen, mise exec -- scala-cli test model/project.scala model/umpire model/temporal, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, go run ./tools/umpire/cmd/umpire-lint over scratch-lifted IR with the new acceptances: exit 0, scratch lift --ir; per-item diff shows only the declared deltas
- PRs: