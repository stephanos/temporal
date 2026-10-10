---
satisfies: [R13]
---
# fn-159-qualified-action-names-in-rules.6 Drop Observation's free-text read field

## Description
Drop the free-text `read` field from `framework.Observation`, leaving `Observation(name, on)`. As far as we found, the lifter keeps only an observation's name (from its `val`) and the IR carries no `read` text. The typed read happens in the realization (`Observed[ActivityExecutionInfo](attemptCount.name)` reads DescribeActivityExecution's `info`). So `read = "attempt"` is an unchecked label that can drift from what the realization reads. Move the hint it gave into a comment beside the realization's typed read.

**Size:** S
**Files:** `model/framework/Action.scala`, `model/irgen/Constants.scala`, `model/irgen/testdata/lifts/Captured.scala` and its expected output if any, `model/README.md`, `model/temporal/features/activity/standalone/Standalone.scala`, `model/temporal/features/nexus/workflow/Workflow.scala`, and the two realizations that read these observations
**Touches:** [model/framework/Action.scala, model/irgen/Constants.scala, model/irgen/testdata/lifts/Captured.scala, model/irgen/testdata/lifts/expected/*.json, model/README.md, model/temporal/features/activity/standalone/Standalone.scala, model/temporal/features/activity/standalone/system/Realization.scala, model/temporal/features/nexus/workflow/Workflow.scala, model/temporal/features/nexus/workflow/Realization.scala]

### Approach
- **Check first:** confirm that `read` reaches nothing downstream. Grep the lifter (`irgen/Constants.scala` folds Observation names), the IR, the lowered Cases, and the Go tools (`tools/umpire/lint`'s unread-observation check and `tools/umpire/lower`) for any use of the text. If something reads it, stop and report what, rather than dropping it.
- **Framework:** `final case class Observation(name: String = "", on: Entity)`. Update its doc comment in `Action.scala`, whose example is `Observation(on = order, read = "attempt")`.
- **Uses:** `attemptCount` and `heartbeatDetails` in the standalone activity signature, and `pendingAttempts` in nexus workflow. Drop `read = …` from each.
- **Keep the hint in the right place:** beside each realization's typed read, add a one-line comment naming the response field the observation stands for. For example, at `publicAttemptCount` in the activity realization: "DescribeActivityExecution's `info.attempt`". The same goes for `rawHeartbeatDetails` (`info.heartbeatDetails`) and nexus's pending attempts (`attempts`).
- **irgen:** update the comments in `Constants.scala` that show `Observation(on = …, read = …)`, and the `Captured.scala` lift fixture (`Observation(on = entry, read = "lost")`).
- **README:** update the paragraph that documents `Observation(on = …, read = …)`.
- **IR stays the same:** expect no checked-in IR or Case change. If anything moves, report it before regenerating.

### Prerequisite
- The working tree must build the lifter (the in-progress IR `Duration` migration) for the IR check.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] `Observation` has no `read` field; its three uses, the lift fixture, the irgen comments and the README are updated.
- [ ] Each realization's typed read of an observation carries a comment naming the response field it stands for.
- [ ] A check confirmed that nothing downstream (IR, Cases, Go lint or lowering) read the `read` text; checked-in IR and Cases are unchanged.
- [ ] `make umpire-check-model` and `make umpire-check-lint` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
