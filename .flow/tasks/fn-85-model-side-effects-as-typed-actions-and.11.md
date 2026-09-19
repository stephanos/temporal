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
- [x] Queries 5 to 7 have generated fixtures and live tests passing under both switch values; Query 5's Case reads the attempt count through `pendingAttempts`; Query 6's fault stops the handler worker
- [x] no whole-Program template, no `case` command and no `Success` Model remain; their names are in the retired-vocabulary gate; `umpire-case --list` prints the seven Caller Queries and nothing from a `case` block (as reworded in the Approach: the set-realizing `case` block is the one Case-producing command, and the `Success` Model file stays as the command-surface specimen its tests pin, with no set and no case; see the Done summary)
- [x] `COVERAGE.md` maps every assertion of all seven upstream tests; timer stability over three runs recorded
- [x] `make umpire-check-regression` exit 0 with the identity count recorded; `make lint-model` green


## Done summary

Done 2026-09-19; self-review. Commit 251f4cd. The receipt's shape follows the Approach's
2026-09-19 adjustment: the set-realizing `case` block stays as the one Case-producing command,
the `fixture`-named form, the whole-Program templates and the `caseTemplate` grammar go.

### Queries 5 to 7

`Caller/Model.lean` adds three Scenarios, three Properties and three Queries to the set
`nexusCallerTests` (seven Queries, `limits four`, 4/4/32768): `retriedThenSucceeded`
(`schedule (unset, unset, unset)`, `handlerReply (handlerError true)`, `backoff`,
`handlerReply (syncSuccess)`) with `retrySucceeds`, which fixes the whole state `succeededOnRetry`
(`phase := .succeeded`, `attempts := 1`, no timeouts) and fact `nexusOperationCompleted`;
`scheduleToStartExpires` (`schedule (unset, expires, unset)`, `workerStop`, `scheduleToStart`)
with `scheduleToStartFires`; `startToCloseExpires` (`schedule (unset, unset, expires)`,
`handlerReply (async)`, `startToClose`) with `startToCloseFires`. A `holds:` predicate must fix
one state or none, so the retry Property's state is a separate `def`, not a structure literal in
the body. `faultInjected` is gone from both machines' facts and evidence lines: the product
machine's `workerStopStep` returns `[]`, because a product self-loop with no facts is
indistinguishable from a stutter in the refinement derivation, and the protocol machine's keeps
its state with no facts. `Tests.lean` pins the step results, the stutter count (456), the
refinement rows ending in `-workerStop`, the three Queries' found outcomes, the instruction ids per
entrypoint, the declared kinds, the known gaps and the confirmed steps per Case.

### The Producer: timers, silent steps, gaps

`Producer.lean` loses the template-era `HookPlacement`, `Hook`, `FaultKind`, `FaultLine`,
`taskQueueRole`, `faultRuleId` and `hooks`, and gains `TimerBinding` (name, milliseconds) with
`Realization.timers` and `timer?`. `resolveEvidence` walks the Scenario's actions and folds each
silent step (a step whose action has no evidence line: `backoff`, `workerStop`) into the next
confirmed rule as `.confirmed none (silent ++ [(action, step)])`, so the Contract's rule confirms
the reply or the timed-out event with the silent step's transition ahead of it; each silent action
is also a capability Known Gap `<actionId>.unobserved` on the Case (`retry` carries
`backoff.unobserved`, `scheduleToStartTimeout` carries `workerStop.unobserved`). A trailing
silent step with no rule after it rejects `evidence.action-unmapped`, a repeated mapped action
`evidence.action-repeated`.

### The realization

`Realization/Nexus.lean` no longer builds on a template: the shared roles (five, the handler's
own task queue among them, `Support.handlerTaskQueueRole` bound by
`temporal.handler-task-queue.resource`), the history sources (scheduled keyed by `event_id`,
started, completed, failed, canceled, timedOut) and the node builders live in it. Timers:
`scheduleToStart` and `startToClose` at 2000 ms, passed to `scheduleNexusOperation` as its
`scheduleToStart` and `startToClose` durations by the three `schedule` bindings; the timed-out
event is read by the fixed `await-close` and `history` reads under the Case's own deadline, so
**no wait-for-duration instruction was needed** (fn-87's boundary line holds). `workerStop` is an
`ActionBinding` (key `workerStop`, instruction `stop-handler-worker`, `Program.injectFault` of
`FAULT_KIND_WORKER_STOP` on the handler's task-queue role), placed first on the controller so the
handler's worker is stopped before the workflow schedules. The retry path places
`pending-attempts`, a `pendingAttempts` read polled until `attempt == 1`, on the
`handlerReply-handlerError-true` path. Every controller path opens with `await-scheduled`, a
`ReadEvidence` poll of the history until an event carries
`nexus_operation_scheduled_event_attributes` (`ReadKind.scheduledEvent`, kind
`evidence.scheduled`): the runtime processes a read's evidence when its instruction commits, and
without it the `pendingAttempts` read of Query 5 was verified before the scheduled transition
(`unauthorized operation transition`). Fields of the reads are `[]`: the runtime types correlated
fields as text, boolean and unsigned only, and `attempt` is signed (Known Gap `attempts-field`).

### The runtime and the Driver

`execution/values.go` `chainEvidence`: lifted evidence names the operation's previously lifted
evidence as its parent when that came from another source, so a read's evidence and the history's
are comparable (`incomparable operation transitions` otherwise); the execution README says so.
`scheduler.go` `publishCompletion` admits a CANCELED reservation of an entrypoint that performs
nothing (`performsNothing`, from the source Program's entrypoints), which is Query 6's handler:
its reservation is never consumed because the worker is stopped, and cancelling it at parent
terminal was closing the Run `activation_failed`. `worker/interpreter.go`: a retryable handler
error keeps the activation open (`nexusResult.retryable`, `nexusActivationOutcome` returns
`open`) and the retried start, which reuses the operation request id and is admitted as a replay,
resumes the entrypoint at the next instruction (`executeNexus` with a resume);
`TestSessionAnswersTheRetriedStartWithTheNextReply` pins it. `profile.go` `HandlerTaskQueue` and
`HandlerTaskQueueBindingID`; `provision.go` `Resources.NexusTaskQueue` targets the endpoint at
the handler's queue; `umpire-run` takes `--handler-task-queue` (default derived from the task
queue).

### Removal

Deleted: `Case/Template.lean`, `Template/NexusOperation.lean`, `Template/Workflow.lean`,
`Case/Tests/Template.lean`, `Case/Tests/ProofPoint.lean`, the `fixture`-named `case` form and
the `caseTemplate` grammar (`Syntax.lean` parses `as <term>` for a `Realization` value), the
`Success` Model's set and `case` block and `nexusSuccessTests-completion-case.json`.
`Success/Model.lean` stays as the command specimen `Success/Tests.lean` and `RaceSyntaxTests.lean`
pin the command surface against, with a local identity and realization for the one produced
specimen; fn-86 R3 retires what remains of it. Retired tokens: `Temporal.Case.Template`,
`Case.Template.NexusOperation`, `Case.Template.Workflow`, `Case/Template`, `Case.Tests.ProofPoint`,
`Case/Tests/ProofPoint`, `Case.Tests.Template`, `Case/Tests/Template`, `caseTemplate`,
`nexusSuccessSet`, `HookPlacement`, `faultRuleId`; the async-Nexus allowlist entry is gone.
`umpire-case --list` prints the seven Caller Queries beside `get-system-info`, `typed-nexus`,
`typed-unary` and `worker-outage`. README, ARCHITECTURE.md, the worker and execution READMEs and
DESIGN.md (a `.11` amendment before the realization sketch) follow.

### Coverage, live and gates

`COVERAGE.md` maps every assertion of the three upstream tests (`tests/nexus_workflow_test.go:579,
3061,3155`) with new Known Gaps `transport-fault`, `attempts-field`, `pending-timeouts`,
`timeout-type` and the two `.unobserved` gaps the Cases carry. Fixtures:
`nexusCallerTests-{retry,scheduleToStartTimeout,startToCloseTimeout}-case.json` new, the four
`.10` fixtures regenerated with `await-scheduled` and the handler task-queue binding.
`tests/testpilot_nexus_caller_case_test.go` runs seven Queries under both switch values, reads
the read evidence (`requireReadEvidence`: kind and the scheduled event's key), the timed-out
event's type and the fault events; `TestTestpilotNexusCallerCaseMissingRemoteEndpoint` now times
out at `await-scheduled`, because a missing endpoint rejects the schedule command at the workflow
task. Timer stability: Queries 6 and 7 passed three consecutive runs under both values
(`-count=3`, 25.9 s in total); across the two full live runs each took 4.2 to 4.8 s (HSM 1.3 to
2.3 s, CHASM 2.2 to 3.4 s) against the 2 s timers. `make umpire-check-live-tests`: failure
identities match the empty expected set across **29 passing identities** (20 before: each new
Query is one test identity and two subtests, one per switch value). `make umpire-check-regression` exit 0. `LEAN_NUM_THREADS=1 make
lint-model` reports the same baseline as `.10` (generated `Proto.lean`, `Tests/Commands.lean`
and the two `enum` binders in `Caller/Model.lean`); `lint-code-fast` 0 issues after one revive
error-return order fix.

## Evidence
- Commits: 251f4cd
- Tests: `lake build`; `make umpire-gen-case-runtime-conformance`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-case-runtime-conformance`; `make umpire-check-goldens`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `LEAN_NUM_THREADS=1 make lint-model`; `GOLANGCI_LINT_BASE_REV=e8aae3f make lint-code-fast`; `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `go vet -tags 'test_dep integration' ./tests/`; `go test -count=3 -tags test_dep,integration ./tests -run 'TestTestpilotNexusCaller(ScheduleToStart|StartToClose)Timeout'`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-live-tests` (29 passing identities); `make umpire-check-regression` (exit 0)
- PRs:
