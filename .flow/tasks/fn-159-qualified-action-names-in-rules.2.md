---
satisfies: [R8]
---
# fn-159-qualified-action-names-in-rules.2 Rename the By-ID actor service to completionClient

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Rename the standalone activity's By-ID actor from `service` to `completionClient`, after the Java SDK's `ActivityCompletionClient`. `service` reads as if the Temporal server itself were settling the activity. In fact the caller is any client that completes an activity asynchronously by its id, without the task token (Go SDK `client.CompleteActivityByID`), and it is independent of the worker process.

**Size:** S
**Files:** `model/temporal/features/activity/standalone/{Standalone.scala,product/Product.scala,system/System.scala,system/ResponseByID.scala,system/Realization.scala}`, the standalone activity tests that name `service.*`, `model/ir/activity-standalone*.json`
**Touches:** [model/temporal/features/activity/standalone/Standalone.scala, model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/system/*.scala, model/temporal/features/activity/standalone/*.test.scala, model/ir/activity-standalone.json, model/ir/activity-standalone-race.json, model/ir/activity-standalone-record.json, model/ir/*.lint.json]

### Approach
- Rename `object service extends Client` to `object completionClient extends Client` and give it a comment saying who it is: a client that settles an activity asynchronously by activity id, without a task token and independent of the worker process. Keep the action names (`respondCompletedByID`, `respondFailedByID`, `respondCanceledByID`).
- Update every `service.respond…ByID` reference: the product and system rules, the By-ID derived models (`ByIDCompletion`, `ByIDFailure`, `ByIDCancellation`), the realizations, and the regression and pin tests. Update the comments that say "service" for this actor, e.g. "By-ID service actions" and "by-ID service settlement". Leave `WorkflowService` / gRPC references alone.
- If fn-159.1 has landed, the `from(service) { ... }` block becomes `from(completionClient) { on(completionClient.respondCompletedByID) ... }`. Otherwise rename within the old form.
- Regenerate the IR with `make umpire-gen-model`. The actor name (`"actor": "service"`) and the qualified action ids (`temporal.features.activity.standalone.service.…`) change in the activity-standalone IR files. Check that the diff contains nothing but the rename. The generated Cases and the Go testpilot don't use the actor name; confirm this with a grep, then run conformance.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/Standalone.scala`: the actor declaration
- `model/temporal/features/activity/standalone/system/ResponseByID.scala` and the `Realization.scala` By-ID realizations

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] The By-ID actor is `completionClient`, with a comment saying who it is; no `service.respond…ByID` reference or "service" naming of this actor remains in the standalone activity models, tests or comments.
- [ ] The regenerated activity-standalone IR differs from the previous IR only in the actor name and the qualified action ids.
- [ ] `make umpire-check-model` and `make umpire-check-lint` pass; Cases and testpilot conformance are unaffected.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: