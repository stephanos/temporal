---
satisfies: [R8]
---
# fn-159-qualified-action-names-in-rules.2 Rename the By-ID actor service to completionClient

## Description
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
Batch 1d (Scala only): `object service` is `object completionClient`, documented as the client that settles an activity asynchronously by activity id without a task token, independent of the worker process (Java ActivityCompletionClient, Go CompleteActivityByID). Every rule, derived model, realization, test and comment that named the actor moved; `effects.serviceRetry` (the server's retry) keeps its name. Declared IR change for 5b: the actor name and the qualified action ids `...standalone.service.*` become `...standalone.completionClient.*`; the step tables rename `service.respond*ByID` to `completionClient.respond*ByID` (activity.map). Outside the activity: model/testdata/binding-order.tsv updated.

Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits: ebc9a2a698, ea62e94b16 (branch w-1d-activity)
- Tests: activity test classes (framework.Activity*, StandaloneActivityPins, RoleRefinements, CapabilityRetryDeadlineFeatures) 66 passed, 0 failed (.flow/tmp/1d-activity/test-step1.log); full model suite: only MachineTablePins' stale-IR subject count fails (IR regenerates in 5b); ExportedMachineLawsTest passes after the binding-order pin update.
- Equivalence: `make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d/activity.map STEPS_ONLY=1` equivalent (.flow/tmp/1d-activity/equiv-step1.log, equiv-step2.log).
- PRs:
