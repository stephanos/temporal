---
satisfies: [R8, R11]
---
# fn-139-actor-grouped-rules-per-rpc-actions.6 Nexus and the shared worker on the shared Outcome; alreadyCompleted becomes rejected(failedPrecondition)

## Description
Moves Nexus (product, standalone and workflow Systems) and the shared worker onto the shared `Outcome`/`Rejection` (R8, R11). Their rules keep their shape. The `in` to `when` rename for them is task .7. The task queue's `QueueOutcome` and the close policy's `Answer` keep their types (spec, Edge Cases).

**Size:** M
**Files:** model/temporal/features/nexus/Nexus.scala (remove `Outcome`, `given Ok`), features/nexus/standalone/Standalone.scala and features/nexus/workflow/Workflow.scala (`given Ok[Outcome]` lines), features/nexus/product/Product.scala, features/nexus/standalone/system/System.scala, features/nexus/workflow/system/System.scala, shared/worker/Worker.scala, tools/umpire/check/grouping_spike_test.go, tools/umpire/check/nexus_kind_test.go
**Touches:** [model/temporal/features/nexus/**, model/temporal/shared/worker/**, tools/umpire/check/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Delete each feature's own `enum Outcome` and `given Ok[Outcome]` (Nexus.scala:20-21 and :48, nexus/standalone/Standalone.scala:45, nexus/workflow/Workflow.scala:94, shared/worker/Worker.scala:28 and :41), and import the shared ones explicitly from the framework's namespace (task .1). TrustingCaller.scala:26 and other files that read `Outcome` from their package switch through that import. Keep the comment at Nexus.scala:18-19, reworded to say that both levels share the framework's outcome.
- `reject(Outcome.notFound, s)` (nexus/product/Product.scala:64, nexus/workflow/system/System.scala:189) becomes rows `~> rejects(Rejection.notFound)`. `def closed(s) = reject(Outcome.alreadyCompleted, s)` (nexus/standalone/system/System.scala:88) becomes `rejects(Rejection.failedPrecondition).because("<server message>")`. Take the message from the server source the existing citation names (chasm/lib/nexusoperation/operation.go) and quote it exactly. The comment at :140 already records that it is a FailedPrecondition.
- Capability arguments: `rejected = cited(Outcome.alreadyCompleted, …)` (:150) becomes `cited(Outcome.rejected(Rejection.failedPrecondition), …)`.
- Go tests that pin outcome names (grouping_spike_test.go:35,84,102,118 and nexus_kind_test.go:21) change to the shared outcome's names as the lifter writes them (task .1 established the encoding). The tests that read checked-in IR will only pass after the batch regeneration, which is expected mid-batch. Note it in the commit message.
- Declared IR delta: the outcome type and names, and the new `because` text on the `alreadyCompleted` rows. List both in the commit message for the batch diff check.
- Equivalence pin: before editing, diff each machine's set of rejecting outcomes, then confirm that the converted machine rejects in exactly the same (action, class, state) cells (memory: consolidated extractor dropped a rejection). A munit test over each machine's step function, against a table recorded from the pre-task tree, pins it.

### Investigation targets
**Required:**
- model/temporal/features/nexus/Nexus.scala:1-60
- model/temporal/features/nexus/standalone/system/System.scala:80-155
- model/temporal/shared/worker/Worker.scala:20-45
- tools/umpire/check/grouping_spike_test.go:20-120
## Acceptance
- [ ] Nexus and the shared worker declare no outcome enum. `alreadyCompleted` is `rejects(Rejection.failedPrecondition).because(<the server's message>)`, and `notFound` is `rejects(Rejection.notFound)`.
- [ ] A munit test shows each converted machine rejecting in exactly the same (action, class, state) cells as before.
- [ ] The Go outcome-name expectations name the shared outcome. The commit message notes that they pass at the batch regeneration.
- [ ] The Models compile, and the Nexus and worker Scala tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
