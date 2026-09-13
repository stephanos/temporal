---
satisfies: [R11, R9]
---
# fn-85-model-side-effects-as-typed-actions-and.11 The Nexus Model, part two: retry, timeouts and the worker fault; templates and the case command removed

## Description
Finish R11 with Queries 5 to 7: a retryable handler error then sync success completes after one backoff with the attempt count observed through `pendingAttempts`; a schedule-to-start timeout after a driven `workerStop` of the `worker` party realized by the existing worker-stop fault; an async reply with start-to-close timeout. Timers realize as concrete durations from the realization, observed through the `nexusOperationTimedOut` history event with a Contract deadline bounding the wait. With every Case now produced from a set, delete the whole-Program templates and the `case` command (R9) and the `Success` Model they served.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Caller/Model.lean` and `COVERAGE.md` and `Tests.lean`, `model/Temporal/Case/Realization/Nexus.lean` (timer durations; the `workerStop` fault binding names the handler task-queue role; `transportFault` bound `observed` only), `tests/testcore/testpilot/testdata/<set>-<query>-case.json` (three new fixtures), `tests/testpilot_nexus_caller_case_test.go`, `model/Temporal/Case/Template.lean`, `Template/NexusOperation.lean`, `Template/Workflow.lean`, `model/Temporal/Case/Syntax.lean` (the `case` command) and `Tests/Template.lean` (deleted), `model/Temporal/Feature/Nexus/Success/Model.lean` and the `case`-dependent parts of `Success/Tests.lean` (deleted or moved to the Caller tests), `model/Umpire/Case/Producer.lean` (`Hook`, `FaultLine`, `EvidenceMapping` from the template era removed), `tools/umpire/internal/retiredvocabulary/check.go` (the `case` command and template names retired)
**Touches:** [model/Temporal/Feature/Nexus/**, model/Temporal/Case/**, model/Umpire/Case/Producer.lean, tests/testcore/testpilot/**, tests/testpilot_*_test.go, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Query 6's `workerStop` must stop the handler's worker: the fault instruction's role is the handler task-queue role the realization binds, not the caller's; the outage Driver (`temporal/worker/outage.go`) already realizes stop and resume.
- Timers: the realization sets schedule-to-start and start-to-close to a short duration (2 s in the design); the Program waits for `nexusOperationTimedOut` with the Case's elapsed deadline as the bound; the live test's wall-clock tolerance is a realization parameter. Measure three runs of Queries 6 and 7 and record the stability (the spec's parked unknown); a Query whose duration the Profile's limits reject fails at preparation.
- Query 5's backoff: the handler returns a retryable `HandlerError` once, then sync success; `pendingAttempts` reads `attempt == 1` between them (task .9's read observation); the transport-fault assertion of the upstream test is a Known Gap in `COVERAGE.md`.
- Removal order: the worker-outage and get-system-info Cases and the typed examples use `register_case`, not the templates, so the templates and `case` can go once the Success Model is gone; fn-86 retires the rest.

### Investigation targets
**Required:**
- `tests/nexus_workflow_test.go:579,3061,3155` — the three upstream tests
- `common/testing/testpilot/temporal/worker/outage.go:57-108` — the fault kinds' realization
- `model/Temporal/Case/Template/NexusOperation.lean:165-170` — which worker the handler activation binds today
- `model/Temporal/Case/Syntax.lean:93-151` — the `case` command to delete
- `model/Umpire/Case/Producer.lean:128-168` — `Hook`, `FaultLine`

**Optional:**
- `service/history/hsm/nexusoperations/config.go:123-129` — retry intervals a setup parameter may need

### Key context
- fn-87's boundary lists a wait-for-duration instruction as fn-85's if the timer realization needs one; this task's default is no new instruction, and the receipt states whether that held.

## Acceptance
- [ ] Queries 5 to 7 have generated fixtures and live tests passing under both switch values; Query 5's Case reads the attempt count through `pendingAttempts`; Query 6's fault stops the handler worker
- [ ] no whole-Program template, no `case` command and no `Success` Model remain; their names are in the retired-vocabulary gate; `umpire-case --list` prints the seven Caller Queries and nothing from a `case` block
- [ ] `COVERAGE.md` maps every assertion of all seven upstream tests; timer stability over three runs recorded
- [ ] `make umpire-check-regression` exit 0 with the identity count recorded; `make lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
