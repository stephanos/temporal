---
satisfies: [R4]
---
# fn-86-retire-hand-written-models-one.6 Worker-outage and get-system-info as command Models with workflow and RPC realizations

## Description
Produce the worker-outage and get-system-info Cases from command Models (R4): a workflow Model with `workerStop` and `workerResume` fault actions of the `worker` party through a workflow realization, and an RPC Model through an RPC realization; delete the two hand-written Case modules and their `register_case` lines; both live tests pass; fixture diffs listed. Carry fn-83 .5's dropped concern: the outage-order rule (same operation, terminal `resumed`, `rule_events` deadline) is emitted by the Producer for any Scenario carrying fault actions, as a separate commit from the migration.

**Size:** M
**Files:** `model/Temporal/Feature/Workflow/Outage/Model.lean` (new), `model/Temporal/Feature/System/Info/Model.lean` (new; final paths a task decision), their `Tests.lean`, `model/Temporal/Case/Realization/Workflow.lean` (workflow start plus fault bindings to the worker-stop and worker-resume fault kinds naming the task-queue role), `model/Temporal/Case/Realization/Rpc.lean` (new: an RPC-only realization), `model/Umpire/Case/Producer.lean` (outage-order rule emission for fault-bearing paths), `model/Umpire/Case/Tests/Producer.lean`, `model/Temporal/Testpilot/WorkerOutage.lean`, `GetSystemInfo.lean` (deleted), `model/Temporal/Testpilot.lean`, `model/Temporal/TestpilotTests.lean`, `model/Temporal/Tool/Testpilot.lean` (last two `register_case` lines removed; `Registry.register_case` deleted if unused), `tests/testcore/testpilot/testdata/{worker-outage,get-system-info}-case.json` (replaced by derived names), `tests/testcore/testpilot/worker_outage_artifact_test.go`, `artifact_test.go:139`, `tests/testpilot_worker_outage_case_test.go`, `tools/umpire/cmd/umpire-run/run_test.go`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate_test.go`
**Touches:** [model/Temporal/Feature/Workflow/**, model/Temporal/Feature/System/**, model/Temporal/Case/**, model/Umpire/Case/**, model/Temporal/Testpilot/**, model/Temporal/Testpilot.lean, model/Temporal/TestpilotTests.lean, model/Temporal/Tool/Testpilot.lean, tests/testcore/testpilot/**, tests/testpilot_worker_outage_case_test.go, tools/umpire/cmd/**]

### Approach
- Outage Model: entity `workflow`; actions `startWorkflow` (caller), `workerStop` and `workerResume` (worker); a machine whose rows put the stop before the start and the resume after, with `faultInjected` as the evidence of both fault rows (the Run Event catalog); the functional set binds `worker: driven`.
- Outage-order rule: the Producer derives the bounded-liveness rule (stop then resume, `rule_events` deadline) whenever a path carries two fault actions of one role; pin the rule ID, terminal state and deadline in the artifact test as today's hand-written Case does.
- RPC Model: one `getSystemInfo` action of the `caller` party with `schema: temporal.api.workflowservice.v1.GetSystemInfoRequest` and a result observation on the response; no workflow entrypoint.
- `Temporal.Testpilot.Conformance` and `CaseSupport` stay (Boundaries).

### Investigation targets
**Required:**
- `model/Temporal/Testpilot/WorkerOutage.lean:84-91,100-144,167` — the fault instructions, environment, the outage-order deadline prose
- `model/Temporal/Testpilot/GetSystemInfo.lean:39` — the RPC Case
- `tests/testcore/testpilot/worker_outage_artifact_test.go:22` — the rule ID, terminal state and Deadline pins
- `model/Temporal/Case/Realization/Nexus.lean` (fn-85) — the realization shape to follow for workflow and RPC
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md` — the outage-order rule design this task carries

**Optional:**
- `common/testing/testpilot/temporal/worker/outage.go:57-108` — the fault kinds' Driver

### Key context
- fn-85 Query 6 already drives `workerStop`; reuse its fault binding shape and the handler task-queue role lesson (name the role explicitly).

## Acceptance
- [ ] both Cases are produced from command Models through the workflow and RPC realizations; the hand-written modules and their `register_case` lines are gone; `register_case` itself is gone if nothing uses it
- [ ] the outage-order rule is Producer-derived for fault-bearing paths, pinned by a Producer unit test and the artifact test's rule ID, terminal state and `rule_events` deadline
- [ ] both live tests pass with the same Verdict shapes; fixture diffs listed in the receipt; the inventory rows are marked migrated
- [ ] `make umpire-check-regression` exit 0; `make lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
