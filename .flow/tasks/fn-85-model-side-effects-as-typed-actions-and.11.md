---
satisfies: [R11, R9]
---
# fn-85-model-side-effects-as-typed-actions-and.11 The Nexus Model, part two: retry, timeouts and the worker fault; templates and the case command removed

## Description
Finish R11 with Queries 5 to 7: a retryable handler error then sync success completes after one backoff with the attempt count observed through `pendingAttempts`; a schedule-to-start timeout after a driven `workerStop` of the `worker` party realized by the existing worker-stop fault; an async reply with start-to-close timeout. Timers realize as concrete durations from the realization, observed through the `nexusOperationTimedOut` history event with a Contract deadline bounding the wait. With every Case now produced from a set, delete the whole-Program templates and the `case` command (R9) and the `Success` Model they served.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Caller/Model.lean` and `COVERAGE.md` and `Tests.lean`, `model/Temporal/Case/Realization/Nexus.lean` (timer durations; the `workerStop` fault binding names the handler task-queue role; `transportFault` bound `observed` only), `tests/testcore/testpilot/testdata/<set>-<query>-case.json` (three new fixtures), `tests/testpilot_nexus_caller_case_test.go`, `model/Temporal/Case/Template.lean`, `Template/NexusOperation.lean`, `Template/Workflow.lean`, `model/Temporal/Case/Syntax.lean` (the `fixture`-named `case` form at `:110-181` and the `caseTemplate` grammar deleted; the `case … realizes <set>` form at `:183-260` stays as the Case-producing command) and `Tests/Template.lean`, `Tests/ProofPoint.lean` (deleted or re-pinned; both read the template), `model/Temporal/Feature/Nexus/Success/Model.lean`, `Success/RaceSyntaxTests.lean` (imports the Success Model) and the `case`-dependent parts of `Success/Tests.lean` including the `set` and abstraction-claim specimens .7 added there (deleted or moved to the Caller tests), `tests/testcore/testpilot/testdata/{async-nexus,nexusSuccessTests-completion}-case.json` (gone), `model/Umpire/Case/Producer.lean` (`Hook`, `FaultLine` from the template era removed; `EvidenceMapping` only if .9 dropped the `case` block's evidence lines), `tools/umpire/internal/retiredvocabulary/check.go` (the template names and the `fixture` form retired)
**Touches:** [model/Temporal/Feature/Nexus/**, model/Temporal/Case/**, model/Umpire/Case/Producer.lean, tests/testcore/testpilot/**, tests/testpilot_*_test.go, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- Query 6's `workerStop` must stop the handler's worker: the fault instruction's role is the handler task-queue role the realization binds, not the caller's; the outage Driver (`temporal/worker/outage.go`) already realizes stop and resume.
- Timers: the realization sets schedule-to-start and start-to-close to a short duration (2 s in the design); the Program waits for `nexusOperationTimedOut` with the Case's elapsed deadline as the bound; the live test's wall-clock tolerance is a realization parameter. Measure three runs of Queries 6 and 7 and record the stability (the spec's parked unknown); a Query whose duration the Profile's limits reject fails at preparation.
- Query 5's backoff: the handler returns a retryable `HandlerError` once, then sync success; `pendingAttempts` reads `attempt == 1` between them (task .9's read observation); the transport-fault assertion of the upstream test is a Known Gap in `COVERAGE.md`.
- Removal order: the worker-outage and get-system-info Cases and the typed examples use `register_case`, not the templates, so the templates and `case` can go once the Success Model is gone; fn-86 retires the rest.
- Adjusted 2026-09-19 after .6, .5 and .7 landed. **The `case` command does not go away
  entirely.** .7 decided that a set is Umpire's and which Cases it produces is the platform's, and
  delivered that as a second `case` form, `case <name> realizes <set> as <template> evidence
  <lines>` (`Temporal/Case/Syntax.lean:183-260`), which is what produces
  `temporal.case.<set>.<query>`; what this task removes is the `fixture`-named form (`:110-181`),
  the whole-Program templates (`Template.lean`, `Template/NexusOperation.lean`,
  `Template/Workflow.lean`) and the `caseTemplate` grammar the set form still parses its `as`
  clause with, replacing that clause with a reference to a `Realization` value. The second
  acceptance line's "no `case` command" is therefore no longer possible as written; proposed
  wording: "no whole-Program template and no `fixture`-named `case` block remain; the set-realizing
  `case` block is the only Case-producing command; the template names and the `fixture` form are
  in the retired-vocabulary gate (bare `case` is a bare word SEM-20 keeps out of it); `umpire-case
  --list` prints the seven Caller Queries beside the four explicitly registered Cases that stay
  until fn-86 R3 and nothing from a `fixture` block". Consequences: `Temporal.Case.Realization.Nexus`
  is built `{ template with … }` and imports `Template.NexusOperation` for `sharedRoles`,
  `sharedObservations`, `Support` and the node builders, so those move into the realization
  (the module header says so); `Temporal/Case/Syntax.lean:29` registers the realization's
  `implementationSwitch`, which must survive. **Timers** have no binding on `Realization` yet
  (`plan`, `actions`, `setup`, `switches`, .5); add one (name to duration) for `scheduleToStart`,
  `startToClose` and `backoff`. **Faults:** `Hook` and `FaultLine` are the template-era `fault`
  lines; the design's `workerStop` is an ordinary action of the `worker` party, so it is an
  `ActionBinding` whose node is an `InjectFault` of `FAULT_KIND_WORKER_STOP`, and `Realization.taskQueueRole`
  and `faultRuleId` go with the lines unless fn-86 .6 needs them (it derives the outage-order rule
  from fault actions on the path; coordinate the shape). The template activates the handler with the
  same `workerRole` and `taskQueueRole` as the caller workflow (`Template/NexusOperation.lean:114,172,196`),
  so the realization gives the handler its own worker and task-queue roles before Query 6 stops
  it, or the outage stops the caller too. **Known Gap:** every Case over `nexusProtocol` carries
  `…atConcurrencyLimit.unbound` (an `input` gap, .5) until a config key exists; `COVERAGE.md`
  records it. **Evidence lines:** the set form's `evidence` lines duplicate the machine's
  `evidence:` once .9 emits declarations from the Model; drop them here if .9 did not. Live
  identity baseline is eleven since .5.

### Investigation targets
**Required:**
- `tests/nexus_workflow_test.go:579,3061,3155` — the three upstream tests
- `common/testing/testpilot/temporal/worker/outage.go:57-108` — the fault kinds' realization
- `model/Temporal/Case/Template/NexusOperation.lean:114,139-141,172,196` — the handler activation binds the caller's `workerRole` and `taskQueueRole` today
- `model/Temporal/Case/Syntax.lean:39-47,110-181` — the `caseTemplate` grammar and the `fixture` form to delete; `:183-260` the set form that stays
- `model/Temporal/Case/Realization/Nexus.lean` — built on the template; the node builders land here
- `model/Umpire/Case/Producer.lean:165-205,270-305` — `Hook`, `EvidenceMapping`, `FaultLine`; `Realization` and its `setup`/`switches`/`taskQueueRole`/`faultRuleId`/`hooks` fields

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
