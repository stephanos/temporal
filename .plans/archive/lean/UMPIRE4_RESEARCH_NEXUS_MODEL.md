# Research: the Nexus Model before fn-85 `.10` and `.11`

Read-only spike, 2026-09-19, on branch `claude/umpire-ordered-flow-specs-p2a48h`. Nothing here is
a decision; each section ends with a recommendation and the cost of the alternatives. Citations are
`file:line` at the working tree of this session. Landed context: `.15` (predicate Properties),
`.4` (state fields, instances), `.6` (refinement, lifted Properties), `.5` (setup bindings,
switches), `.7` (`set`, derived Cases). Not landed: `.8`, `.9` (R10 typed instructions), `.10`,
`.11`, `.12`, `.13`. `.10` depends on `.9` and `.15` (`.10.json:6-9`).

## 1. `atConcurrencyLimit`

### Evidence

| Claim | Where |
| --- | --- |
| HSM has a per-workflow pending-operation limit, default 30 | `service/history/hsm/nexusoperations/config.go:36-43`, key `component.nexusoperations.limit.operation.concurrency` |
| HSM rejects the schedule command at the limit | `service/history/hsm/nexusoperations/workflow/commands.go:184-191`: `coll.Size() >= MaxConcurrentOperations` returns `FailWorkflowTaskError` with cause `PENDING_NEXUS_OPERATIONS_LIMIT_EXCEEDED` |
| CHASM has its own key, default 2000 | `chasm/lib/nexusoperation/config.go:89-94`, key `nexusoperation.limit.operation.concurrencyPerWorkflow.max` |
| CHASM rejects the same way | `chasm/lib/workflow/nexus_commands.go:177-182` |
| The cause is an API enum value | `WORKFLOW_TASK_FAILED_CAUSE_PENDING_NEXUS_OPERATIONS_LIMIT_EXCEEDED = 33`, `go.temporal.io/api@v1.63.5/enums/v1/failed_cause.pb.go:89` |
| How the rejection reaches history | `FailWorkflowTaskError` (`chasm/lib/workflow/registry.go:128-136`) is caught in `service/history/api/respondworkflowtaskcompleted/workflow_task_completed_handler.go:374`, goes through `failWorkflowTask` (`:1628`) to `historybuilder/history_builder.go:266` `AddWorkflowTaskFailedEvent`. The workflow task fails and is retried; **no `NexusOperationScheduled` event is written**, so no operation instance exists |
| Both keys are in the generated catalog | `model/Temporal/DynamicConfig/Settings.lean:238` and `:5368` |
| No upstream functional test exercises it | only `chasm/lib/workflow/nexus_commands_test.go:381` (a unit test); none of R11's seven tests |

So `.5`'s "no dynamic-config setting bounds pending Nexus operations" is wrong: there are two, one per
implementation switch value.

### What binding it would take

| Gap | Where it shows | Change |
| --- | --- | --- |
| The table does not vary with `setup:` | `model/Umpire/Command/Syntax.lean:2010-2018` ("a machine carries one setup"); `Machines.lean:176-178` says the reject row is absent for that reason; a step function is `State → inputs → List Step` (`Machines.lean:1001`), no setup argument | R5's "guard rows" half: step signature gains the setup, `Finite` enumerates the product, fingerprint moves. M |
| A binding is key-only | `Umpire/Case/Producer.lean:250-253` `SetupBinding {parameter, key}`; a `Bool` needs a value per member, and the key differs per switch value (HSM vs CHASM) | `SetupBinding` gains `values : List (String × String)` per switch value, or the switch carries it. S |
| The harness applies only switch flags | `tests/testcore/testpilot/switch.go:14-17` `SwitchSetting.Value bool`; `common/testing/testpilot/temporal/profile.go:22` `DynamicConfig map[string]string`; `tests/testcore/test_env.go:253-261` validates a typed value, so an int setting needs an int; live tests build environments only from switch values (`tests/testpilot_async_nexus_case_test.go:47-54`) | a setup-value path beside the switch path. S-M |
| Nothing keys the evidence to the operation | `workflowTaskFailed` is admitted by name (`model/Temporal/Case/EventKind.lean:31-41` reads the `HistoryEvent.attributes` oneof) but every source is keyed by `scheduled_event_id` (`Template/NexusOperation.lean:52-64`, `Testpilot/CaseSupport.lean:48-49`); a failed workflow task names no operation and none was created | an observation on the `workflow` entity, which the correlated projection cannot scope today. L, no clean design |
| Reachability | limit 0 rejects every schedule (one instance); limit 1 needs `instances: 2`, and the lift over instances is rejected (`.6`: "rejects a lift over several instances") | — |

### Options

| Option | Cost | Consequence |
| --- | --- | --- |
| A. Bind it: key per switch value, value `0` when `true` | all five rows above (L) | a Query nobody asked for; the rejection is a stutter at `unscheduled` under `productOf`, so no product Property can see it |
| B. Drop the parameter from the machine and from section 3's rewrite | zero code in `.10`; `.5`'s unbound-gap pin (`configuredLifecycle`, `Success/Tests.lean`) must move to `Caller/Tests.lean` or `Umpire` tests when `.11` deletes Success | the specimen keeps "rejections go first" only as prose |
| C. Keep it declared and unbound (Known Gap) | zero | every one of the seven Cases carries `…setup.nexusProtocol.atConcurrencyLimit.unbound` about a parameter no row reads, so the gap states nothing true; R12's canary rule rejects a Query with a white-box gap and it is unsettled whether an `input` gap counts |

**Recommendation: B.** `.10` writes `machine nexusProtocol` without `setup:` and rewrites section 3
without the reject row; it adds a dated amendment naming the two keys, the cause, and the three
reasons it is not modeled (table does not vary with setup; the value/key pair is per switch value;
the evidence names no operation). `.11`, when deleting `Success/Tests.lean`, moves the `.5` pin
(`probe: Bool` unbound gap) onto a throwaway machine in `Caller/Tests.lean`. A later task reopens
the row when a Query needs it, with R5's second half as its first step.

## 2. Stutter-invariance under the lift

### Mechanism

`refinedProperty` (`model/Umpire/Command/Refinement.lean:145-173`) reads a product Property on the
protocol machine by rewriting each group: a `.priorState` trigger becomes a `PropertyPattern` on
field `.priorState`, reference the `nexusProduct` state-field id, constraint `.equals <product key>`
(`:162`); a state clause becomes `.resultingState` on the same field; outcomes and facts are the
protocol values of the same name. The evaluator offers a step's field values beside its state
(`model/Umpire/Property/Evaluate.lean:613-622` `valuesInStep`), so every protocol step is checked:
on a stutter the prior and resulting field values are equal, and a `transitionContract` whose
postcondition fixes a product state different from the precondition's fails there.

Stutters of `productOf` (`Machines.lean:334-341`, count pinned at `:491`): `schedule` from
`unscheduled`; `handlerReply (handlerError true)` and `transportFault` from `scheduled`;
`backoff` from `backingOff`. All four have mapped prior state `scheduled`, outcome `accepted`,
and hidden or unrecorded facts. No stutter leaves a terminal phase.

The clause language cannot exclude a stutter by guard: `PropertyPredicateContext.allows`
(`model/Umpire/Property.lean:390-397`) lets a `.before` guard read only `priorState` and
`selectedAction`, and an `.after` expectation only the result, so "prior field equals resulting
field" is not a spelling of any `PropertyClause` (`Property.lean:571-595`).

### The Properties the seven Queries need

The Producer lowers only clauses triggered by `selectedAction`
(`model/Umpire/Case/Producer.lean:383-409`, `property.clause-shape` at `:395`), so every Property a
functional `find` Query names must be a same-step `when:` claim; a transition claim like
`terminalIsFinal` can be searched and verified but never becomes a Case.

| Property | Machine | Shape | Stutter-invariant | Why |
| --- | --- | --- | --- | --- |
| `terminalIsFinal` | product | transition, one group per terminal prior phase (`Machines.lean:518-524`) | yes | its groups trigger only at terminal prior states; no stutter has one (all four map to `scheduled`); steps out of terminal phases (`workerStop`, `complete → notFound`) keep the phase |
| Q1 `syncSucceeds`: `when: handlerReply (syncSuccess)` fixes `phase: succeeded`, fact `nexusOperationCompleted` | protocol | same-step | n/a (not lifted) | the trigger is a product step |
| Q2 `asyncStarts`: `when: handlerReply (async)` fixes `started`, `nexusOperationStarted` | protocol | same-step | n/a | |
| Q2 `completionSucceeds`: `when: complete (succeeded)` fixes fact `nexusOperationCompleted` (state not fixed: 3 running phases × timeouts) | protocol | same-step | n/a | the `notFound` steps out of terminal phases carry no fact, so the fact clause carries the predicate exactly |
| Q3 `completionFails`: `when: complete (failed)` fixes `nexusOperationFailed` | protocol | same-step | n/a | |
| Q4 `handlerErrorFails`: `when: handlerReply (handlerError false)` fixes `failed`, `nexusOperationFailed` | protocol | same-step | n/a | |
| Q5 `retryBacksOff`: `when: handlerReply (handlerError true)` fixes fact `pendingAttempts` | protocol only | same-step | n/a | on the product this trigger has no step (`Machines.lean:70`), so a product spelling is refused `noStep`; `attempts == 1` is not expressible: `Predicate.lean` emits only `equals` clauses, never `naturalAtLeast` |
| Q6/Q7 `timesOutOn<Timer>`: `when: scheduleToStart` / `startToClose` fixes `timedOut`, `nexusOperationTimedOut (timeoutType := …)` | protocol only | same-step | n/a | the product's `timeout` is not a protocol action (pinned rejection `Machines.lean:568-583`) |

Every product same-step claim a stutter could trigger (`when: handlerReply (handlerError true)`,
`when: transportFault`, `when: schedule`) is either refused `noStep` on the product or names an
action the product lacks, so no lifted same-step claim can be checked on a stutter in this pair.

### Options

| Option | Change | Cost |
| --- | --- | --- |
| Exclude stutters at enumeration | not expressible as a clause guard (above); `refinedProperty` would have to drop the `.priorState` groups whose fixed state equals the trigger state on rows the refinement report marks `none` -- but a group is per prior state, not per row, so it cannot drop only the stutter rows | not available without a new clause form |
| Restate: a transition claim on a refined machine must accept `(before, {accepted, before.state, []})` | a check in the `property` elaborator for machines something `refines:`; rejects with a located message; `#guard_msgs` pin | S, ~20 lines in `Umpire/Command/Syntax.lean`; only worth it when a non-invariant claim appears |
| Accept the limitation | a sentence in the section 2.5 amendment | zero |

**Recommendation: accept**, and write the invariance argument for `terminalIsFinal` into
`Caller/Tests.lean` as a `#guard` over `nexusProtocol.refinement.rows`: every row whose result is
`none` has a source phase in `{unscheduled, scheduled, backingOff}`. That pins the fact the argument
rests on, without a new mechanism. (`.6`'s receipt already states the Nexus Properties are
stutter-invariant; this is the evidence it did not pin.)

Note for `.10`: `Umpire/Command/Predicate.lean` gained a `readsBefore` refusal today (`:45`,
`:186`), so a transition predicate must depend on the step before only through `before.state`;
`terminalIsFinal` does.

## 3. Pre-plan for `.10` and `.11`

### 3.1 The Model file (draft, current grammar)

Namespace `Temporal.Feature.Nexus.Caller` gives family `temporal.nexus.caller`
(`Temporal/Case/Conventions.lean:11`, `Tests/SecondModel.lean:31-33`). Vocabulary is
`Tests/Commands.lean` moved verbatim (entities, `Timeout`, `Reply`, `Resolution`, `Delivery`, the
five actions, `pendingAttempts`); machines and step functions are `Tests/Machines.lean:26-373`
moved, minus `setup:` (section 1). Shown here: only what changes or is new.

```lean
import Temporal.Case.Syntax
namespace Temporal.Feature.Nexus.Caller

-- … entities, enums, actions, observation as Tests/Commands.lean:35-190 …
-- … ProductPhase, ProductState, ProductOutcome, ProductFact, the five product step functions
--     as Tests/Machines.lean:26-103 …

machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [timeout]
  evidence:
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
    faultInjected: faultInjected
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    timeout: timeoutStep

-- … Phase, TimeoutType, attemptBound, ProtocolState, ProtocolOutcome, ProtocolFact and the nine
--     protocol step functions as Tests/Machines.lean:184-331, productOf as :334 …

machine nexusProtocol
  for: operation
  state: ProtocolState
  refines: nexusProduct
  map: productOf
  starts: [unscheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  evidence:
    nexusOperationScheduled: nexusOperationScheduled
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
    pendingAttempts: pendingAttempts
    faultInjected: faultInjected
  steps:
    schedule: scheduleStep
    handlerReply: protocolHandlerReplyStep
    complete: protocolCompleteStep
    transportFault: protocolTransportFaultStep
    workerStop: protocolWorkerStopStep
    backoff: backoffStep
    scheduleToClose: scheduleToCloseStep
    scheduleToStart: scheduleToStartStep
    startToClose: startToCloseStep

/- The product claim, verified on every protocol path (section 2); not in the set. -/
property terminalIsFinal
  machine: nexusProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase

/- Queries 1 to 4: same-step claims on the protocol machine, one per side effect that settles. -/
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step => step.state.phase == .succeeded &&
    step.facts.contains .nexusOperationCompleted

property asyncStarts
  machine: nexusProtocol
  when: handlerReply (async)
  holds: fun step => step.state.phase == .started && step.facts.contains .nexusOperationStarted

property completionSucceeds
  machine: nexusProtocol
  when: complete (succeeded)
  holds: fun step => step.outcome == .accepted && step.facts.contains .nexusOperationCompleted

property completionFails
  machine: nexusProtocol
  when: complete (failed)
  holds: fun step => step.outcome == .accepted && step.facts.contains .nexusOperationFailed

property handlerErrorFails
  machine: nexusProtocol
  when: handlerReply (handlerError false)
  holds: fun step => step.state.phase == .failed && step.facts.contains .nexusOperationFailed

scenario syncCompletion
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (syncSuccess)]

scenario asyncThenSucceeded
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (async), complete (succeeded)]

scenario asyncThenFailed
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (async), complete (failed)]

scenario nonRetryableError
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (handlerError false)]

limits two
  steps: 2
  actions: 2
  search: 128        -- 9 enabled at unscheduled × 11 at scheduled = 99 candidates

limits three
  steps: 3
  actions: 3
  search: 2048       -- 9 × 11 × 11 = 1089

query syncCompletionQuery
  find: syncSucceeds
  in: syncCompletion
  limits: two

query asyncCompletionQuery
  find: completionSucceeds
  in: asyncThenSucceeded
  limits: three

query asyncFailureQuery
  find: completionFails
  in: asyncThenFailed
  limits: three

query handlerErrorQuery
  find: handlerErrorFails
  in: nonRetryableError
  limits: two

query asyncCompletionHolds      -- the product claim, outside the set (a verify Query rejects in one)
  verify: terminalIsFinal
  in: asyncThenSucceeded
  limits: three

set nexusCallerTests
  purpose: functional
  bind:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  repeat: implementation
  queries: [syncCompletionQuery, asyncCompletionQuery, asyncFailureQuery, handlerErrorQuery]

case nexusCallerCases
  realizes nexusCallerTests
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    schedule ← history nexusOperationScheduled          -- see B5: the key is `schedule-unset-unset-unset`
    handlerReply ← history nexusOperationStarted        -- see B5: one line per class, not per action
    complete ← history nexusOperationCompleted          -- see B5: `complete-succeeded` / `complete-failed`
```

### 3.2 Hand check of each Scenario against the step functions

| Scenario | Step | Leaves | Timers enabled after | Facts |
| --- | --- | --- | --- | --- |
| all | `schedule (unset, unset, unset)` from `unscheduled-0-unset-unset-unset` (`scheduleStep`, `:245`) | `scheduled-0-unset-unset-unset` | none (`scheduleToClose/Start` need `expires`, `:307-321`); `backoff` off | `nexusOperationScheduled` |
| Q1 | `handlerReply (syncSuccess)` (`:256-260`) | `succeeded-0-…` | none | `nexusOperationCompleted` |
| Q2/Q3 | `handlerReply (async)` (`:258`) | `started-0-…` | none (`startToClose` unset, `:323-327`) | `nexusOperationStarted` |
| Q2 | `complete (succeeded)` from `started` (`:286-299`) | `succeeded-0-…`, outcome `accepted` | none | `[nexusOperationCompleted]` only (`startedFirst` empty at `started`) |
| Q3 | `complete (failed)` from `started` | `failed-0-…`, `accepted` | none | `[nexusOperationFailed]` |
| Q4 | `handlerReply (handlerError false)` (`:262`) | `failed-0-…` | none | `nexusOperationFailed` |

Every path ends in an `ends:` phase with no timer enabled, so no `unobservable` timer and no
Known Gap is on any of the four paths. The set admits: no `verify` Query, no `observed` party's
action on a path (`network` performs none), `repeat: implementation` is registered
(`Case/Syntax.lean:29`).

### 3.3 Blockers, each with the file that changes

| # | Blocker | Evidence | Fix | Size |
| --- | --- | --- | --- | --- |
| B1 | The realization binds ids a machine never emits. Machine action ids are `<family>.action.<machine>.<memberKey>`: `Authoring.lean:319` `ownedId "action" ownerKey`, fixture `temporal.nexus.success.action.lifecycle.awaitStart`. The realization hard-codes `temporal.nexus.caller.action.schedule` (`Realization/Nexus.lean:47-54`). `produceCase` translates nothing (`Authoring.lean:644-657`), so every action on a path rejects `realization.action-unbound` (`Producer.lean:503`) | `ActionBinding.action` is a `DefinitionId` (`Producer.lean:242-245`) | bind by member key and resolve at the `case` block against the vocabulary, the way `EvidenceMapping` uses `vocabulary.namedAction` (`Case/Syntax.lean:146-148`); `Temporal.Case` cannot import the Model (the Model imports `Temporal.Case.Syntax`) | `Producer.lean`, `Realization/Nexus.lean`, `Case/Syntax.lean`; M |
| B2 | One binding per class is needed but the realization has one per action. `handlerReply-syncSuccess` needs `NEXUS_RESPONSE_KIND_SYNCHRONOUS` (`instruction.proto:121`), `-async` the asynchronous kind, `-handlerError-false` `_ERROR` (`:123`); `ActionBinding.node` receives only identity and id | `Realization/Nexus.lean:65-88` binds async only, `:50-51` says sync is `.10`'s | with B1's keys, three `handlerReply-*` bindings and two `complete-*` | `Realization/Nexus.lean`; S once B1 lands |
| B3 | One plan serves every Query of a set, and the async plan's controller waits for a completion authority (`Realization/Nexus.lean:107` `await-completion-authority`) that a sync or error path never publishes: Queries 1 and 4 hang, and `case … realizes <set> as <template>` allows one template per set (`Case/Syntax.lean:185`, `:220`) | `EntrypointItem` is `fixed` or `actions` (`Producer.lean:215-219`) | add an item variant emitted only when a named class is on the path (`.whenOnPath classes node`), or select the plan per path in the realization | `Producer.lean` (`EntrypointItem`, `assembleProgram :513-538`), `Realization/Nexus.lean`; M |
| B4 | Admitted evidence sources are `nexusOperationStarted` and `nexusOperationCompleted` only (`Template/NexusOperation.lean:66-70`, `:232`); Queries 3/4 need `nexusOperationFailed`; every selected action needs a line (`Producer.lean:458-463` `evidence.action-unmapped`), so `schedule` needs `nexusOperationScheduled`, whose operation key is its own `event_id`, not a `scheduled_event_id` attribute (`:52-64`) | `EvidenceSource.operationKeyPath` (`Producer.lean:171-177`) | add `failedSource`; add a source keyed by the event's own id, which needs the projection to accept an event-id key path (`Umpire/Case/Projection`) | `Template/NexusOperation.lean` or `Realization/Nexus.lean`, possibly `Umpire/Case/Projection/*`; M |
| B5 | `case` evidence lines are `ident ← history ident` (`Case/Syntax.lean:47`) matched against Scenario spellings that are member keys (`handlerReply-async`, `schedule-unset-unset-unset`; `Case/Syntax.lean:128-130`, keys from `Syntax.lean:566-570`); a hyphenated key is not an identifier, so no line can name a classed action | `unselectedActionMessage` | the line takes a `scenarioAction` (`handlerReply (async) ← history …`) resolved through `actionKeyOf`; better, derive the mapping from the machine's own `evidence:` block (`Registry.lean:136-137`) along the witness so the `case` lines go away, which is what `.11` wants anyway | `Case/Syntax.lean`; S for the grammar, M for the derivation |
| B6 | `terminalIsFinal` cannot be a functional Query (section 2, `Producer.lean:395`) although `DESIGN.md` section 3 and `Machines.lean:544-547` write `find: terminalIsFinal` | | write the four same-step Properties above; keep `terminalIsFinal` as `verify` Queries; section 3's `query asyncCompletion` is rewritten | `Caller/Model.lean`, `DESIGN.md`; S |
| B7 | The lifted-Property production path is unexercised: `patternHolds` (`Producer.lean:358-372`) compares `pattern.reference` to the state's id, never a field's, so the early-response vacuity check is skipped for any lifted clause; no Case has been produced from a lifted Property | | not needed if B6 is followed; record as a Known Gap in `COVERAGE.md` if a product Property is ever realized | — |
| B8 | The Driver's error reply is fixed at `Internal`/`NonRetryable` (`common/testing/testpilot/temporal/worker/interpreter.go:220-225`), so Query 4's class example `BadRequest` (`Commands.lean:117`) is not what runs; the abstraction claim row would name an example the Run did not perform | R10 (`.8`/`.9`, both `todo`) carries the `HandlerError` message | `.10` is ordered after `.9` (`.10.json:7`); if `.10` runs first, Query 4 must record a `claim` Known Gap or the example must read `Internal` | `Realization/Nexus.lean` or R10; depends on order |
| B9 | Query 3's `complete (failed)` needs a failure completion; `completeNexusOperation` carries one `result : Expression` (`Testpilot/Authoring.lean:367-369`); whether the Driver sends a Nexus failure from it is R10's "completion payload or failure" | | same as B8: after `.9`, or a Known Gap | depends on order |
| B10 | Elaboration time: the protocol machine costs ~40 s, the refinement ~30 s, and two Queries at `search: 8192` took the module to ~150 s (`.6` receipt). Four `find` Queries plus one `verify` at the budgets above (`128`, `2048`) should stay under that, but the file is on every `lake build` | | keep the exact-sequence Scenarios at depth ≤ 3, budgets as drafted, `terminalIsFinal` verified on one Scenario only | `Caller/Model.lean`; risk, not blocker |
| B11 (`.11`) | Unbound driven actions are dropped silently: `actionNodes` skips a class no item names (`Producer.lean:488-505`) and the final check covers bound actions only (`:533-535`); R9's "reject naming it" is not implemented. Query 6's `workerStop` needs a binding to `injectFault` (`Testpilot/Authoring.lean:390-391`) on the controller with the handler's task-queue role | `FaultLine`/`Hook` are the template mechanism (`Producer.lean:159-199`) | a `workerStop` `ActionBinding`; a rejection for a driven action on the path with no binding | `Producer.lean`, `Realization/Nexus.lean`; S |
| B12 (`.11`) | Query 5 cannot map its evidence: `backoff` is `unobservable` (no event to name; `evidence.action-unmapped`), and the retry step's fact is the read observation `pendingAttempts`, not an event kind (`evidence.kind-unknown`, `Producer.lean:452`); the machine's own `evidence:` block already says both | | B5's derivation: read the machine's `evidence:` along the witness, emit a Known Gap for an unobservable timer, and a read-observation source (`DescribeWorkflowExecution` … `attempt`, R10 `.9`) | `Case/Syntax.lean`, `Producer.lean`, `Realization/Nexus.lean`; M |
| B13 (`.11`) | Query 6/7 need `schedule (unset, expires, unset)` / `(unset, unset, expires)`: the timeout inputs must reach the command attributes; today `scheduleBinding` builds `startNexusOperation` with no timeouts (`Realization/Nexus.lean:65-70`) and a per-class binding (B1) must carry a duration from the realization's `timers:` | | eight `schedule-*` bindings or a binding function of the key; the duration table (2 s in the design) | `Realization/Nexus.lean`; S after B1 |

### 3.4 What `.10` writes, in order

1. `Caller/Model.lean` as in 3.1 with `setup:` gone (section 1) and the five same-step Properties;
   `Caller/Tests.lean` with the stutter pin (section 2) and the `.5` gap pin moved from Success.
2. B1, B2, B3, B5 (grammar half) so a Case is produced; B4's `failedSource` and a keyed
   `scheduled` source. B6 is a design rewrite (section 3 amendment), not code.
3. If `.9` has not landed: Queries 3 and 4 carry `claim` Known Gaps (B8, B9) named in
   `COVERAGE.md`; the fixtures still regenerate.
4. `COVERAGE.md`: the four upstream tests' assertions (`tests/nexus_workflow_test.go:512`,
   `:1024`, `:1617`, `:2242`) map as: result value and completed-event link (Q1) → Known Gap
   (links and `DescribeMutableState` are not observations); `NexusOperationStarted` then
   `Completed`/`Failed` events → the same-step Properties; error message and type rehydration
   (`:2242`) → Known Gap until R10 exposes failure fields; metric assertions → Known Gap (spec
   Boundaries).

### 3.5 What `.11` adds

B11, B12, B13; the `evidence:`-derived mapping replaces the `case` lines, which is what lets the
`case` command and the templates go (R9); Query 5's Property is `retryBacksOff` fixing
`pendingAttempts`, and its attempt-count assertion is a Known Gap until a `naturalAtLeast` clause
can be enumerated from a predicate (`Predicate.lean` emits `equals` only).
