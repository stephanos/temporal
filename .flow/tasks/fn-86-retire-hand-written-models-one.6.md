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
- Adjusted 2026-09-19 after fn-85 .5 and .7 landed (fn-85 .11 still open). **Faults are actions,
  not lines:** `Umpire.Case.Producer.Realization` today carries the template-era `hooks : List Hook`,
  `taskQueueRole` and `faultRuleId` (the rule "the Producer adds to order a Scenario's `fault`
  lines"), and fn-85 .11 removes `Hook` and `FaultLine` because the design's `workerStop` is an
  ordinary action of the `worker` party; so the Outage Model's `workerStop` and `workerResume` are
  `ActionBinding`s whose nodes are `InjectFault` instructions against the role the realization
  names, and the outage-order rule derives from two fault actions of one role on the path -- keep
  `faultRuleId` (or its successor) for that, and coordinate with .11 rather than rebuilding on the
  lines it deletes. **Realization shape:** `Temporal.Case.Realization.Nexus` (`plan`, `actions`,
  `setup`, `switches`); the workflow realization's plan is the `Program.workflow` entrypoint the
  `Template/Workflow.lean` arm builds today, which .11 also deletes. **Identity:** both Cases are
  produced by the Temporal `case … realizes <set>` block under `temporal.case.<set>.<query>` and
  `<set>-<query>-case.json`, so `worker-outage-case.json` and `get-system-info-case.json` and their
  Case ids move; `tests/testcore/testpilot/artifact_test.go:168-236` loads `get-system-info` by
  name and `fixture_table_test.go` picks the new fixtures up without a table entry. `register_case`
  is used only by `model/Temporal/Tool/Testpilot.lean:20-27` (four lines), so it goes with the last
  of them.
- Adjusted 2026-09-20 by fn-86 .1 after fn-85 closed. **Faults are keyed action bindings** (fn-85
  .11): `Hook`, `FaultLine`, `taskQueueRole` and `faultRuleId` are gone from `Realization`; the
  Caller Model's `workerStop` is an `ActionBinding` whose node is `Program.injectFault` on the
  handler task-queue role, and a silent step is folded into the next confirmed rule with a
  capability Known Gap, so the outage-order rule has to be derived by the Producer from two fault
  actions of one role on the path (nothing carries `faultRuleId` any more). `Template/Workflow.lean`
  is gone; the workflow realization's plan is written as `Realization.plan` entrypoint items, as
  `Realization/Nexus.lean` does. The `case` block takes a `Realization` value in its `as` clause.
  `register_case` has four lines in `model/Temporal/Tool/Testpilot.lean`; `Registry.register_case`
  goes with the last. The `HANDWRITTEN_INVENTORY.md` rows for `WorkerOutage`, `GetSystemInfo` and
  `CaseSupport` list every reader.

### Investigation targets
**Required:**
- `model/Temporal/Testpilot/WorkerOutage.lean:84-91,100-144,167` — the fault instructions, environment, the outage-order deadline prose
- `model/Temporal/Testpilot/GetSystemInfo.lean:39` — the RPC Case
- `tests/testcore/testpilot/worker_outage_artifact_test.go:22` — the rule ID, terminal state and Deadline pins; `artifact_test.go:168-236` — loads `get-system-info` by fixture name
- `model/Temporal/Case/Realization/Nexus.lean` (fn-85) and `model/Umpire/Case/Producer.lean:165-205,242-305` — the realization shape to follow for workflow and RPC; `Hook`, `FaultLine`, `ActionBinding`, `Realization`
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md` — the outage-order rule design this task carries

**Optional:**
- `common/testing/testpilot/temporal/worker/outage.go:57-108` — the fault kinds' Driver

### Key context
- fn-85 Query 6 already drives `workerStop`; reuse its fault binding shape and the handler task-queue role lesson (name the role explicitly).

## Acceptance
- [x] both Cases are produced from command Models through the workflow and RPC realizations; the hand-written modules and their `register_case` lines are gone; `register_case` itself is gone if nothing uses it
- [x] the outage-order rule is Producer-derived for fault-bearing paths, pinned by a Producer unit test and the artifact test's rule ID, terminal state and `rule_events` deadline
- [x] both live tests pass with the same Verdict shapes; fixture diffs listed in the receipt; the inventory rows are marked migrated
- [x] `make umpire-check-regression` exit 0; `make lint-model` green


## Done summary

Done 2026-09-20; self-review. Commits 8183773 (the outage-order rule) and e18b520 (the migration).

### The outage-order rule (first commit)

`Umpire.Case.Producer.outageOrderRules` derives, from the assembled Program alone, one
bounded-liveness rule per role whose injected faults stop the worker and later resume it: the
rule `worker-outage-order` (a second role's carries its ordinal), from `awaiting-stop` through
`stopped` to `resumed`, its two transitions reading the recorded fault's `role_id` and `kind` off
the `FAULT_INJECTED` Run Event payload, expired by the `rule_events` deadline the realization
states (`Realization.outageDeadline`, default 16: what lies between the stop and its resume on the
traced path, with slack, and never elapsed time). A stop with no resume, or a resume before any
stop, derives no rule: the Contract answers it at Run time rather than the Producer refusing the
path. `produce` attaches the rules to the correlated Property's definition beside the lowered
relations. `injectedFaults` and `faultKindName` (the exhaustive name table the hand-written Case
carried) live beside it. `Umpire/Case/Tests/Producer.lean` pins the rule's id, states,
transitions, filters and deadline, the two no-rule shapes, and the ordinal on a second role.

### The Models (second commit)

**Worker outage.** `Temporal/Feature/Workflow/Outage/Model.lean`: entity `workflow` (keyed by the
workflow task that completed it), actions `startWorkflow` (caller, creates, the start request's
schema), `workerStop` and `workerResume` (worker, naming no entity) and `awaitCompletion` (caller,
on the workflow); machine `workflowOutage` over `pending`, `started`, `completed`, where the two
faults keep the state and record nothing, the start records nothing either, and the wait records
`workflowExecutionCompleted`; `property completes` on `awaitCompletion`; `scenario outage` is
exactly stop, start, resume, wait; `limits four`; `query survived`; `set workerOutageTests`
(caller and worker `driven`); `case workerOutageCases … as Temporal.Case.Realization.workflowOutage`.
The Case is `temporal.case.workerOutageTests.survived`, fixture `workerOutageTests-survived-case.json`.

Why the start is silent: every history event of one operation must carry the same key, the
started event names the workflow by a run id (`first_execution_run_id`) that no later event
carries, and the completed event names it by the workflow task that completed it -- the one task
that can only have been dispatched once the worker was back, which is what the outage being
survived leaves behind. So the completed event is the one evidence kind, keyed by
`workflow_task_completed_event_id`, and it confirms the four steps at once (`projection
[("evidence.workflowExecutionCompleted", 4)]`); the three steps before it are capability Known
Gaps (`…startWorkflow.unobserved`, `…workerResume.unobserved`, `…workerStop.unobserved`). The
faults are not correlated evidence for the same reason: a `FAULT_INJECTED` payload names a role,
not the workflow, so keying it would split the operation. The outage-order rule reads them
instead.

**System info.** `Temporal/Feature/System/Info/Model.lean`: entity `server` (named by itself),
action `getSystemInfo` (caller, on the server, the `GetSystemInfoRequest` schema; the request's
message is admitted by a new `GetSystemInfo` root in `Temporal.Case.Schema`); machine `systemInfo`
over `unqueried`, `queried`, recording `systemInfoReturned`, whose evidence is the
`instructionCompleted` Run Event; `property infoReturned`; `scenario once`; `limits one`; `query
answered`; `set systemInfoTests`; `case systemInfoCases … as Temporal.Case.Realization.unaryRpc`.
The Case is `temporal.case.systemInfoTests.answered`, fixture `systemInfoTests-answered-case.json`.
A unary call leaves no history, so what confirms the step is the completion the runtime records
for the instruction that made it, keyed by its `protocol_code` (a Run makes the one call, so the
one call is the operation). The old `server-version-present` safety rule has no Model form: a
`present` relation admits only an optional field or a oneof member, and `server_version` is a
proto3 string, always present; the realization still reads it into the `server-version`
observation, and the Contract is the correlated capability alone.

### The realizations

`Temporal.Case.Realization.Workflow.plan` now takes the controller's items and the finish literal;
`workflowStart` is unchanged in what it emits (its fixture is byte-identical). `workflowOutage`
orders the controller stop, start, resume, wait, history -- whatever order the path performs the
classes in -- with `workerStopBinding`/`workerResumeBinding` (fault instructions on the Case's
task-queue role, `stop-worker`/`resume-worker`) and `awaitCompletionBinding` (the close-event read
as `await-close`), and `completedSource` as its one evidence kind. `Temporal.Case.Realization.Rpc`
is new: one endpoint role, the `server-version` and correlated observations, one controller item,
`getSystemInfoBinding` (`get-system-info`, timeout 5 s) and `completedSource` (the run-event
kind). `Temporal.Case.Support` gained `getSystemInfoMethod`; `Conformance.lean` reads it from there.

### Removed

`Temporal/Testpilot/WorkerOutage.lean` and `GetSystemInfo.lean`; the two `register_case` lines and
`register_case` itself (`Temporal.Case.Registry`; every checked-in Case is a `case` block's);
`Temporal.TestpilotTests` is a facade smoke check; `worker-outage-case.json` and
`get-system-info-case.json`.

### Go

`tests/testcore/testpilot/worker_outage_fixture.go` names the two fixtures, the rule id and the
task-queue role. `worker_outage_artifact_test.go` pins the rule id, the `rule_events` deadline and
the `expired` state on the new fixture, that the Contract is that rule beside the correlated
capability, and the two injected faults. `artifact_test.go`'s two-shapes test is over the
system-info Case (capability alone, derived Profile with the one `InvokeRPC` opcode, snapshotted
once, two mutations rejected) and the outage Case (rule beside capability).
`common/testing/testpilot/internal/verification/worker_outage_test.go` prepares the shipped
Contract's rule alone for its online/offline agreement (the capability needs the correlated
observation the live Program lifts). `tests/testpilot_worker_outage_case_test.go` runs
`workerOutageTests-survived`: the outage-order rule satisfied at `resumed` on the two fault events,
every correlated clause rule satisfied at `correlated.satisfied` on the history read's event, whose
correlated evidence is the completed kind keyed by the completing task's event id. The conformance
generator's fake registry names the new Cases.

### Fixture diff

`worker-outage-case.json` and `get-system-info-case.json` deleted; `workerOutageTests-survived-case.json`
and `systemInfoTests-answered-case.json` added; the seven `nexusCallerTests-*`, the pair and the
workflow-start fixtures byte-identical.

### Gates

`lake build` green (588 targets); `make umpire-gen-case-runtime-conformance` (the fixture diff
above) then `make umpire-check-case-runtime-conformance` exit 0; `make umpire-gen-goldens &&
make umpire-check-goldens` exit 0 with no golden changed; regression views, inventory, retired
vocabulary, protocol and authoring checks exit 0; `LEAN_NUM_THREADS=1 make lint-model` at the `.1`
baseline (40 warnings, none new; the same two generated `Proto.lean` errors);
`GOLANGCI_LINT_BASE_REV=f4bbd4b make lint-code-fast` clean; `go vet` and `go test` over
`tools/umpire`, `common/testing/testpilot` and `tests/testcore/testpilot` ok;
`TestTestpilotWorkerOutageCase` and `TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone` pass
live; `make umpire-check-regression` exit 0 with 29 passing live identities (29 before: the
worker-outage identities are the same two tests over the new fixture, and no live test names the
unary Case, as none named the old one).

## Evidence
- Commits: 8183773, e18b520
- Tests: `lake build`; `make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance`; `make umpire-gen-goldens && make umpire-check-goldens`; `make umpire-gen-regression-views && make umpire-check-regression-views`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `LEAN_NUM_THREADS=1 make lint-model`; `make lint-code-fast`; `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `go test -count=1 -tags test_dep,integration ./tests -run TestTestpilotWorkerOutageCase`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
