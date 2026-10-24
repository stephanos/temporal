---
satisfies: [R10]
---
# fn-159-qualified-action-names-in-rules.4 Remove disabled(process.stop) from the activity product

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Remove `disabled(process.stop)` from the standalone activity's product machine. It binds the worker process's stop without ever enabling it, and the System's stop row (`always ~> effects.keep`) is a stutter, which the refinement check accepts without any product row. An experiment on 2026-10-10 found nothing that needs the binding: with the line and its `import actors.worker.worker as process` removed, the models compile and package, and every model and framework test passes except one pin test that lists the product's bindings. The lift could not be run then, because the tree's in-flight IR `Duration` change broke the lifter's build. Confirming the lift is part of this task.

**Size:** S
**Files:** `model/temporal/features/activity/standalone/product/Product.scala`, `model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala`, `model/ir/activity-standalone*.json` (only if the lift changes them)
**Touches:** [model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala, model/ir/activity-standalone.json, model/ir/activity-standalone-race.json, model/ir/activity-standalone-record.json, model/ir/*.lint.json]

### Approach
- In the product's `rules`, delete `disabled(process.stop)` and its comment ("The worker stopping is a fault the Run records and the activity does not feel."). Delete the now-unused `import actors.worker.worker as process`; `-Wunused:all -Werror` fails the build without this. Keep the meaning: if the product rules or the header comment have a natural place for it, say that a worker stop is invisible at the product level and handled by the System, its composition with the worker, and the realization.
- Update the pin test "the activity machines bind actions in actor-group order": remove `process.stop` from the product's expected bindings and keep it in the System's.
- Run the full gate. If irgen, lint or refinement refuses the missing binding, keep the line, record the reason the binding is needed (the refusal text) in a comment beside it, and close the task with that finding. If the gate passes, check whether the activity-standalone IR changed. Any change should be limited to the product's binding of `process.stop`, e.g. a disabled row or the product's action list. Regenerate, and record what moved in the done summary.
- This task removes the only `disabled(...)` in the standalone activity's product. The spec's parked question, whether `disabled(...)` targets need qualification, stays open for the other models' `disabled` uses.

### Prerequisite
- The working tree must build the lifter and pass the Cases check: the IR `Duration` migration in `irgen/Realizations.scala` and regenerated Go protos (`make protoc`). Until then, only the compile and test half of the verification can run.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] The standalone activity product no longer binds `process.stop` and no longer imports the worker actor; the pin test expects the product's bindings without it and the System's with it.
- [ ] `make umpire-check-model` and `make umpire-check-lint` pass; any activity-standalone IR change is limited to the product's `process.stop` binding and is stated in the done summary.
- [ ] Or, if irgen, lint or refinement requires the binding: the line stays, with a comment stating why, and the done summary quotes the refusal.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: