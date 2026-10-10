---
satisfies: [R10]
---
# fn-159-qualified-action-names-in-rules.4 Remove disabled(process.stop) from the activity product

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
