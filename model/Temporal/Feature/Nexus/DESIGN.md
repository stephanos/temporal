# Nexus: side effects in the behavioral model

Design specimen. Nothing here compiles, and no module imports it. The spec that implements it is
fn-85 ("Model side effects as typed actions and run query sets"). Server citations were read at
commit `7c42dec82c`; no test was run to produce them.

## 1. The idea

The Nexus success Model has two actions, `awaitStart` and `awaitSuccess`, and both only wait.
Everything that decides the outcome (the schedule command, the handler's reply, whether an error is
retryable, the timeouts, the completion callback) lives in a hand-written Program template outside
the Model. The Model cannot say that a retryable handler error leads to a retry while a
non-retryable one fails the operation, so no Property can check it.

This design moves those side effects into the Model while keeping Umpire free of Temporal:

```lean
-- today: a wait, with the real behavior hidden in a template
start: scheduled + awaitStart → started

-- with this design: the side effect, its classes, and what confirms each result
phase: scheduled + handlerReply (async)
  → phase: started, evidence: nexusOperationStarted
phase: scheduled + handlerReply (handlerError (retryable := false))
  → phase: failed, evidence: nexusOperationFailed
phase: scheduled + handlerReply (handlerError (retryable := true))
  → phase: backingOff, attempts: +1, evidence: pendingAttempts
```

Seven concepts carry it. Every one is Umpire-generic except the realization.

| Concept | One line | Section |
| --- | --- | --- |
| **Entity** | a thing with identity that other things refer to: an operation, a workflow | 2.1 |
| **Action** | a side effect a party performs, with typed input grouped into classes | 2.2 |
| **Machine** | the state a machine keeps per entity and the rows that change it | 2.3 |
| **Observation** | recorded data that confirms a row | 2.4 |
| **Refinement** | a detailed machine proven consistent with a simpler one | 2.5 |
| **Set** | which Queries run for which purpose, and who drives each party | 2.6 |
| **Realization** | the Temporal-owned binding from actions and observations to RPCs, instructions and events | 2.7 |

Property, Scenario, Limits and Query keep their current meaning; they now read machines.

## 2. Concepts

### 2.1 Entity

An entity is a kind of thing with identity. It declares what it refers to and the key recorded data
uses to name an instance. It declares no state: state belongs to the machines that track the entity
(section 2.3), so two machines can describe the same entity at different detail.

```lean
entity workflow

entity operation
  refer:
    caller: workflow
  key: scheduledEvent
```

A Model holds several instances of each entity, bounded by Limits. References are compared, never
interpreted; the realization binds them to runtime identifiers. Several instances are what
`TestNexusAsyncOperationWithMultipleCallers` needs (five operations sharing one handler workflow).

### 2.2 Action

An action is a side effect performed by a **party**. It declares the entity it acts on (or creates),
its input fields, its results, and optionally the protobuf message that types them.

```lean
enum Reply
  | syncSuccess
  | async
  | operationFailed
  | operationCanceled
  | handlerError (retryable : Bool)

action handlerReply
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: Reply
  examples:
    handlerError (retryable := false) → BadRequest
    handlerError (retryable := true) → Internal
```

**Parties.** A feature declares its parties by using them (`caller`, `handler`, `network`, `worker`). One party
is reserved: `system`, the server under test, which never performs a declared action; its steps are
the machine's rows.

**Input classes.** An input field's type is a finite `enum`, and an enum constructor may carry finite
fields (`handlerError (retryable : Bool)`), which mirrors a protobuf oneof. Each constructor is a
class: a set of concrete values claimed to behave alike. A class with exactly one concrete value
needs no evidence. A class with several (`handlerError (retryable := false)` covers BadRequest,
Unauthenticated, NotFound and more) is a claim; its **example** is the concrete value a functional
test uses, and an exploratory run tries the other members (section 2.6).

**Schema.** When present, class members and examples are checked against the named protobuf message
while the file compiles. An action whose payload has no protobuf message (the Nexus HTTP completion
callback) omits it, and its classes are names the realization interprets.

**Results.** An action that returns something declares a result enum (`results: Delivery`, with
`accepted` and `notFound`), and rows state which result each case produces.

Timers and faults need no separate concept. A timer is part of the `system` party's behavior and
appears in rows as `after (<timer>)` (section 2.3). A fault is an ordinary action of a declared party:

```lean
action transportFault
  party: network
  on: operation

action workerStop
  party: worker
```

### 2.3 Machine

A machine is the transition relation the glossary calls a Machine. It names the entity it tracks,
the state it keeps per instance, the setup parameters it reads, its timers, and its rows.

```lean
machine nexusProtocol
  for: operation
  ends: [succeeded, failed, canceled, timedOut]
  state:
    phase: Phase
    attempts: count
    scheduleToStart: Timeout
  setup:
    recordCancelCompletion: Bool
  timers: [backoff, scheduleToStart]
  steps:
    none + schedule (scheduleToStart := t)
      → phase: scheduled, attempts: 0, scheduleToStart: t, evidence: nexusOperationScheduled
    phase: scheduled + handlerReply (async)
      → phase: started, evidence: nexusOperationStarted
    phase: backingOff + after (backoff)
      → phase: scheduled
```

A row reads: **guard** `+` **what happens** `→` **state changes**, **evidence**. `ends:` names the
`phase` values that end an instance.

| Part | Forms |
| --- | --- |
| guard | state fields and setup parameters (`phase: scheduled \| backingOff`, `recordCancelCompletion: true`); `none` before an instance exists; `terminal` and `not terminal` for the machine's `ends:` values; omitted fields match anything |
| what happens | an action with a class pattern (`handlerReply (async)`), alternatives across actions (`handlerReply (handlerError (retryable := true)) \| transportFault`), a timer (`after (backoff)`), or nothing: a `system` step the server takes on its own when the guard holds |
| state changes | field updates (`phase: started`, `attempts: +1`); a lowercase name bound in the pattern may be stored into a field of the same type (`scheduleToStart: t`); `reject` when the action changes nothing; `result: <value>` for an action with results |
| evidence | one or more observations (section 2.4), each optionally guarded by `when` over the state before the step; `unobservable` becomes a Known Gap in every Case whose path uses the row |

Rows are ordered and the first matching row applies, so rejections go first. A row no input can
reach is rejected as shadowed.

A row with no guard (`+ workerStop`) matches in every state; `faultInjected` is a Testpilot Run
Event in the observation catalog.

**Setup parameters** are configuration that changes behavior on purpose, named after the setting
(`recordCancelCompletion`, `atConcurrencyLimit`). The Profile binds them; a Case that needs a value its
environment cannot set carries a Known Gap. A rollout switch between two implementations of the same
behavior (HSM or CHASM Nexus operations) is not a setup parameter; a set repeats its Queries once per
switch value instead (section 2.6).

### 2.4 Observation

An observation is recorded data that confirms a row. Evidence names observations and needs no
declaration when the name comes from the realization's catalog: for Temporal, the generated history
event kinds (`nexusOperationStarted`), resolved the way `evidence` lines resolve today. Recorded data
finds its instance through the entity's `key`.

Only derived observations are declared, such as a value read through a call:

```lean
observation pendingAttempts
  on: operation
  read: attempts
```

This covers the steps that write no history event: a retryable attempt failure is visible only
through `DescribeWorkflowExecution`, as the pending operation's `attempt`. An observation may also
belong to another entity than the row's (the handler workflow's callback state in
`TestNexusCallbackAfterCallerComplete`).

### 2.5 Refinement

A feature starts with one machine. It adds a second, simpler **product machine** when a product
Property would otherwise mention protocol detail (attempts, backoff, cancel delivery), or when
several realizations share one product meaning. Nexus meets both: an operation is created by a
workflow command, by an external HTTP caller, or by the standalone API, and two implementations run
it. Both machines describe behavior observable at the API, so both belong to `Temporal.Feature`.

The detailed **protocol machine** declares `refines:` and a `map:` from its state to the product
machine's. Values with the same name map to each other, so the map lists only the differences, and a
field the product machine does not have maps to `hidden`.

```lean
machine nexusProtocol
  for: operation
  refines: nexusProduct
  map:
    phase: backingOff → scheduled
    attempts, cancel, scheduleToClose, scheduleToStart, startToClose → hidden
```

The checker walks every protocol row through the map. A row whose mapped before and after states are
a product step is allowed; a row whose mapped states are equal is a stutter (nothing visible
happened, such as a retry); any other row rejects the refinement. The step mapping is derived, never
written. A Property on the product machine then holds on every protocol-machine path, and so on
every Case built from one.

The check reuses the forward simulation inside `Umpire.ImplementationLink`. A refinement is not an
Implementation Link: SEM-08 reserves that name for connecting `Temporal.Feature` to `Temporal.System`.

### 2.6 Set

A set names a purpose, the Queries it runs (or, for exploration, what it must cover), and how each
party other than `system` is **bound**:

| Binding | Who performs the party's actions | What the verifier does |
| --- | --- | --- |
| `driven` | the Case's own Program: the controller, a pre-programmed workflow or handler | chooses the class the Query's path needs, using the class's example |
| `observed` | a real deployment or the world | lets it happen, reads which class occurred, and checks the machine allows it |

The machine is the same for every set; only the bindings differ. A Scenario lists the actions of
non-`system` parties in order: under `driven` the Case performs them, under `observed` the verifier
expects them. `system` rows follow from the machine.

| Purpose | Contains | Produces |
| --- | --- | --- |
| `functional` | a list of `find` Queries | one checked-in Case per Query, run once per `repeat` switch value |
| `canary` | a list of `find` Queries; a Query whose Case would carry a white-box Known Gap rejects | Cases for fn-70 and fn-29 to run against a deployment |
| `exploratory` | a coverage goal (rows, result values, members of claimed classes) and a budget | coverage targets for fn-33; two members of one class with different verdicts are a counterexample that splits the class and becomes a Regression through Promotion |

### 2.7 Realization

The realization is the only Temporal-owned part. It binds each action to what performs it, each
observation to where it is recorded, each timer to a concrete duration, each setup parameter and
switch to dynamic config, and each reference to a runtime identifier. The Producer assembles a
Case's Program and Contract from a Query's path and the realization, so no Program template is
written per Case.

It lives beside the templates it replaces in `Temporal.Case`, not in `Temporal.System`: MOD-10
forbids `Temporal.System` from importing the Feature machines a realization must name. MOD-02 lists
Evidence mappings under `Temporal.System`, so this placement needs a rule amendment (section 6).

## 3. Specimen

The caller side of one workflow-scheduled Nexus operation on the HSM implementation. Cancellation is
included to show a second state field; delivering it is fn-79's deferred scope, not fn-85's.

```lean
entity workflow

entity operation
  refer:
    caller: workflow
  key: scheduledEvent

enum Timeout
  | unset
  | expires

enum Reply
  | syncSuccess
  | async
  | operationFailed
  | operationCanceled
  | handlerError (retryable : Bool)

enum Resolution
  | succeeded
  | failed
  | canceled

enum Delivery
  | accepted
  | notFound

enum CancelReply
  | delivered
  | handlerError (retryable : Bool)

-- Parties: caller, handler, network, worker. The reserved party `system` is the server.
action schedule
  party: caller
  creates: operation
  schema: temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

action handlerReply
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: Reply
  examples:
    handlerError (retryable := false) → BadRequest
    handlerError (retryable := true) → Internal

action complete                                   -- no protobuf message: a Nexus HTTP completion
  party: handler
  on: operation
  input:
    resolution: Resolution
  results: Delivery

action transportFault
  party: network
  on: operation

action workerStop                                 -- the handler's worker stops polling
  party: worker

action requestCancel                              -- fn-79
  party: caller
  on: operation
  schema: temporal.api.command.v1.RequestCancelNexusOperationCommandAttributes

action cancelReply                                -- fn-79
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.CancelOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: CancelReply

observation pendingAttempts
  on: operation
  read: attempts

enum ProductPhase
  | scheduled
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

machine nexusProduct
  for: operation
  ends: [succeeded, failed, canceled, timedOut]
  state:
    phase: ProductPhase
  steps:
    none → phase: scheduled
    phase: scheduled → phase: started | succeeded | failed | canceled | timedOut
    phase: started → phase: succeeded | failed | canceled | timedOut

enum Phase
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

enum CancelPhase
  | notRequested
  | requested
  | delivering
  | delivered
  | rejected

machine nexusProtocol
  for: operation
  refines: nexusProduct
  map:
    phase: backingOff → scheduled
    attempts, cancel, scheduleToClose, scheduleToStart, startToClose → hidden
  ends: [succeeded, failed, canceled, timedOut]
  state:
    phase: Phase
    cancel: CancelPhase
    attempts: count
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout
  setup:
    atConcurrencyLimit: Bool
    recordCancelCompletion: Bool
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  steps:
    -- rejection first: the first matching row applies
    atConcurrencyLimit: true + schedule
      → reject, evidence: workflowTaskFailed
    none + schedule (scheduleToClose := c, scheduleToStart := s, startToClose := t)
      → phase: scheduled, cancel: notRequested, attempts: 0,
        scheduleToClose: c, scheduleToStart: s, startToClose: t,
        evidence: nexusOperationScheduled

    -- the handler's reply to the server's start request
    phase: scheduled + handlerReply (syncSuccess)
      → phase: succeeded, evidence: nexusOperationCompleted
    phase: scheduled + handlerReply (async)
      → phase: started, evidence: nexusOperationStarted
    phase: scheduled + handlerReply (operationFailed | handlerError (retryable := false))
      → phase: failed, evidence: nexusOperationFailed
    phase: scheduled + handlerReply (operationCanceled)
      → phase: canceled, evidence: nexusOperationCanceled
    phase: scheduled + handlerReply (handlerError (retryable := true)) | transportFault
      → phase: backingOff, attempts: +1, evidence: pendingAttempts
    phase: backingOff + after (backoff)
      → phase: scheduled
    + workerStop
      → evidence: faultInjected

    -- an async completion; before a start, the server records a Started event first
    phase: scheduled | backingOff | started + complete (succeeded)
      → phase: succeeded, result: accepted,
        evidence: nexusOperationStarted when phase: scheduled | backingOff, nexusOperationCompleted
    phase: scheduled | backingOff | started + complete (failed)
      → phase: failed, result: accepted,
        evidence: nexusOperationStarted when phase: scheduled | backingOff, nexusOperationFailed
    phase: scheduled | backingOff | started + complete (canceled)
      → phase: canceled, result: accepted,
        evidence: nexusOperationStarted when phase: scheduled | backingOff, nexusOperationCanceled
    terminal + complete
      → result: notFound

    -- timers fire only when the schedule command set them
    phase: scheduled | backingOff | started, scheduleToClose: expires + after (scheduleToClose)
      → phase: timedOut, evidence: nexusOperationTimedOut (timeoutType := scheduleToClose)
    phase: scheduled | backingOff, scheduleToStart: expires + after (scheduleToStart)
      → phase: timedOut, evidence: nexusOperationTimedOut (timeoutType := scheduleToStart)
    phase: started, startToClose: expires + after (startToClose)
      → phase: timedOut, evidence: nexusOperationTimedOut (timeoutType := startToClose)

    -- cancellation (fn-79)
    not terminal, cancel: notRequested + requestCancel
      → cancel: requested, evidence: nexusOperationCancelRequested
    phase: started, cancel: requested
      → cancel: delivering
    cancel: delivering + cancelReply (delivered)
      → cancel: delivered,
        evidence: nexusOperationCancelRequestCompleted when recordCancelCompletion: true
    cancel: delivering + cancelReply (handlerError (retryable := false))
      → cancel: rejected,
        evidence: nexusOperationCancelRequestFailed when recordCancelCompletion: true

-- A product Property, carried to every protocol path by the refinement.
property terminalIsFinal
  machine: nexusProduct
  when: terminal
  require: phase unchanged

-- A bounded-progress Property (SEM-09): the bound is the timer the schedule command set.
property endsBySchedulingDeadline
  machine: nexusProtocol
  when: schedule (scheduleToClose := expires)
  require: terminal
  within: after (scheduleToClose)

scenario asyncThenSucceeded
  machine: nexusProtocol
  actions: [schedule, handlerReply (async), complete (succeeded)]

limits short
  steps: 4
  actions: 3
  search: 32

query asyncCompletion
  find: terminalIsFinal
  in: asyncThenSucceeded
  limits: short

set nexusCallerTests
  purpose: functional
  bind:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  repeat: implementation
  queries: [syncCompletion, asyncCompletion, retryAfterHandlerError, scheduleToStartTimeout]

set nexusCallerCanary
  purpose: canary
  bind:
    caller: driven
    handler: observed
    network: observed
    worker: observed
  queries: [syncCompletion, asyncCompletion]
```

The realization, in sketch syntax (fn-85 makes it a Lean value in `Temporal.Case`). A worker
instruction carries the Temporal API message the action's `schema:` names, so a class example fills
that message and the Driver maps it to the SDK call that produces it. Each observation is declared
once in the Case and read by both the Program and the Contract.

```lean
realization nexusCaller
  machine: nexusProtocol
  actions:
    schedule → workflow command ScheduleNexusOperationCommandAttributes
    handlerReply (syncSuccess) → handler reply StartOperationResponse (sync_success)
    handlerReply (async) → handler reply StartOperationResponse (async_success)
    handlerReply (operationFailed) → handler reply StartOperationResponse (operation_error)
    handlerReply (handlerError (retryable := r)) → handler reply HandlerError (retry_behavior: r)
    complete (succeeded) → controller completion Payload
    complete (failed) → controller completion Failure
    transportFault → none                                  -- cannot be driven; Known Gap when bound driven
    workerStop → controller instruction InjectFault (kind: workerStop, role: handler task queue)
  observations:
    catalog: history events from GetWorkflowExecutionHistory, key scheduled_event_id
    pendingAttempts: DescribeWorkflowExecution pending_nexus_operations.attempt
  timers:
    scheduleToStart: expires → 2s
    startToClose: expires → 2s
  setup:
    recordCancelCompletion → component.nexusoperations.recordCancelRequestCompletionEvents
  switches:
    implementation: hsm → nexusoperation.enableChasmWorkflowOperations = false,
                    chasm → nexusoperation.enableChasmWorkflowOperations = true
```

## 4. Extension points

How the Nexus tests outside the specimen, and features beyond Nexus, fit without new concepts or
with one named addition.

| Need | Fits as | New concept? |
| --- | --- | --- |
| an external HTTP caller (`nexus_api_test.go`) | a second action that `creates: operation`, performed by a declared `externalCaller` party | no |
| standalone operations (`nexus_standalone_test.go`) | the same `operation` entity with no `caller` reference; terminate and delete as actions; describe and poll as observations | no |
| several callers or operations sharing a handler workflow | several entity instances and references; the handler's workflow start as an action whose conflict policy is an input class | no |
| update-, query- or activity-backed handlers | bind the `handler` party to another machine instead of `driven` or `observed`, so the reply is that machine's result | yes: a machine as a binding target |
| completion callbacks with their own retries | a `callback` entity with a machine of its own; a shared retry fragment would avoid repeating attempt and backoff rows | maybe: reusable row fragments |
| visibility list and count | an observation with an explicit consistency bound | yes: bounded eventual observation |
| metrics and spans | observations from a second catalog | no |
| reset and its reapply rules | a `system` action that rewrites recorded history | yes: history-rewriting steps |
| cross-cluster failover | setup parameters for topology and a failover action | no |
| HSM/CHASM storage layout | not modeled; a white-box Known Gap | no |

Appendix B maps each functional test file to these rows.

## 5. What Umpire needs

| # | Need | Today |
| --- | --- | --- |
| 1 | Entities with references and a key, several instances bounded by Limits | one role, one instance |
| 2 | Actions with a party, input fields over finite enums whose constructors may carry finite fields, optional schema, results and examples | `Umpire.Operation.Declaration` with kinds `unaryRpc`, `sdkCommand`, `event` and exact request lists; abstraction rejected |
| 3 | Machines with per-instance state fields, setup parameters, timers, and ordered rows with guards, class patterns, alternatives, bound names, `system` rows without an action, and guarded evidence | one flat state enum, one row per state and action pair |
| 4 | Observations from a catalog, declared read observations, and observations on another entity | event projection only, keys inside templates |
| 5 | Refinement with a name-default state map and derived steps | the forward simulation inside `Umpire.ImplementationLink`, expert Lean only |
| 6 | Sets with a purpose, `driven`/`observed` bindings, `repeat` switches, and class claims with their examples recorded in Provenance | one `case` block per Query |
| 7 | A realization binding actions, observations, timers, setup parameters, switches and references | whole-Program templates |
| 8 | Worker instructions that carry Temporal API messages, and one observation declaration per Case read by Program and Contract | bespoke instruction fields (`StartNexusOperation`, `RespondNexus` kinds); observations named separately by Program and Contract |

Needs 1 to 8 cover the specimen; need 8 is Testpilot's, on top of fn-87's protocol. Section 4's three "yes" rows are later additions.

## 6. Decisions

Recorded 2026-09-10 with the user; items marked *revised* changed in the simplification pass and are
not yet reflected in fn-85.

1. **A party is fixed on the action; a set binds it.** *Revised:* the bindings are `driven` and
   `observed` (previously `test` and `environment`, which collided with a party named
   `environment`), and `system` is the only reserved party. Timers are `system` behavior written as
   `after(<timer>)` in rows; faults are actions of a declared party.
2. **The author claims classes; exploration owns the evidence.** A class is an enum constructor.
   Single-value classes need no evidence; a multi-value class is a claim recorded in Provenance with
   its example (*revised* word, previously "representative"), and exploration tries its other
   members.
3. **Protobuf descriptors stay Umpire's only schema.** *Revised:* actions have no `kind`. Whether an
   action is a call, a command or a reply changed nothing the verifier does, so the distinction stays
   in the realization, where `Umpire.Operation`'s `unaryRpc` and `sdkCommand` live on. `event`
   becomes the observation catalog. The schema line is optional.
4. **One machine by default; a product machine when needed.** A protocol machine `refines:` a
   product machine through a state `map:` that defaults by name; steps are derived. It is a
   refinement, not an Implementation Link.
5. **One Model for HSM and CHASM, run under both.** Deliberate config-controlled differences are
   setup parameters; the rollout switch is a set's `repeat`; storage layout is a white-box Known Gap.
6. **Words.** `action`, `observation`, `machine`, `refines`, `examples`, `driven`, `observed`.
   `interface` was rejected as a method-set contract type duplicating Action; `model` and
   `statemachine` because Model is the checked behavior and Machine the glossary's transition
   relation; "mechanism machine" because MOD-02 gives implementation mechanisms to
   `Temporal.System`; a top-level `link` because SEM-08 reserves Implementation Link.
7. *New:* **State belongs to machines, not entities.** An entity declares identity, references and a
   key; each machine declares the state it keeps, so the product and protocol machines can track the
   same entity differently.
8. *New:* **Event observations are not declared.** Evidence names resolve against the realization's
   catalog; only derived observations such as reads are declared.
9. *New:* **Validation is rows, not a separate `rules:` block.** Rejections are the first rows of a
   machine, using the existing first-match order. Constant entity traits (`vary:`) are setup
   parameters. A machine names its `ends:`, which rows and Properties read as `terminal`.
10. *New, needs a rule amendment:* **The realization lives in `Temporal.Case`**, beside the templates
    it replaces, because MOD-10 forbids `Temporal.System` from importing Feature machines; MOD-02
    lists Evidence mappings under `Temporal.System` and needs amending to allow it.
11. *New:* **Worker instructions carry API messages.** A workflow command carries its
    `temporal.api.command.v1` attributes and a handler reply its `temporal.api.nexus.v1` message, so
    the action's schema and its instruction are the same type and no Testpilot field is added per
    server option. SDK-only options go in an extension field beside the message.

## Appendix A. What the server does

Workflow-scheduled operations run on the HSM implementation by default:
`nexusoperation.enableChasmWorkflowOperations` defaults to false
(`chasm/lib/nexusoperation/config.go:38-43`). The two implementations read different dynamic config
families (`component.nexusoperations.*` and `nexusoperation.*`) with different defaults, for example
a concurrency limit of 30 (`service/history/hsm/nexusoperations/config.go:37-40`) against 2000
(`chasm/lib/nexusoperation/config.go:90-91`).

Below, `NX` is `service/history/hsm/nexusoperations/`.

| Step | From | To | History event |
| --- | --- | --- | --- |
| schedule command | none | scheduled | `NexusOperationScheduled` |
| retryable start failure (handler error marked retryable, transport error, timeout of one call) | scheduled | backing off | **none**; attempt and last failure recorded (`NX/statemachine.go:276-289`) |
| backoff timer fires | backing off | scheduled | none |
| async reply with token | scheduled | started | `NexusOperationStarted` |
| sync success reply | scheduled | succeeded | `NexusOperationCompleted` |
| operation failed, non-retryable handler error | scheduled | failed | `NexusOperationFailed` |
| operation canceled reply | scheduled | canceled | `NexusOperationCanceled` |
| completion callback | scheduled, backing off, started | succeeded, failed, canceled | the completion event, preceded by a fabricated `NexusOperationStarted` when the operation had not started (`NX/completion.go:122-140`) |
| schedule-to-close timer | scheduled, backing off, started | timed out | `NexusOperationTimedOut`, type SCHEDULE_TO_CLOSE |
| schedule-to-start timer | scheduled, backing off | timed out | `NexusOperationTimedOut`, type SCHEDULE_TO_START |
| start-to-close timer | started | timed out | `NexusOperationTimedOut`, type START_TO_CLOSE |

Retries have no attempt limit and no expiration; only a timeout ends them (`NX/config.go:189-197`).

**Cancellation** is a separate sub-state of the operation. The cancel command always writes
`NexusOperationCancelRequested`. The request reaches the handler only once the operation has started,
because it needs the operation token; a cancel requested earlier waits for the start
(`NX/statemachine.go:425-450`). Delivery retries like a start. A delivered cancel writes
`NexusOperationCancelRequestCompleted` and does not end the operation; the final state still comes
from the handler's completion (`NX/executors.go:902-904`). A search of the operation's task executors
found no path that cancels pending operations when the caller workflow closes; every task checks
that the caller is still running and stops otherwise.

**Configuration that changes outcomes.** Scheduling rejects a missing endpoint, oversized names or
headers, and exceeding the concurrency limit by failing the workflow task. An external endpoint needs
`component.nexusoperations.callback.endpoint.template`; while it is `unset` every attempt fails with
an Internal error and no event (`NX/executors.go:145-147`). A pending operation's attempt count is
visible as `pending_nexus_operations[].attempt` in `DescribeWorkflowExecution`.

## Appendix B. What the functional tests exercise

About 13,000 lines across 16 files.

| Group | Files | What varies | Fits as (section 4) |
| --- | --- | --- | --- |
| Caller workflow lifecycle | `nexus_workflow_test.go` (31 tests) | reply form (sync, async, operation failure, handler error retryable or not, transport fault); completion before or after start, after reset, after caller close; cancel before start, failed then retried; three timeouts; endpoint target | the specimen, plus reset and callback rows |
| Handlers built from other APIs | `nexus_workflow_update_test.go` (16), `nexus_workflow_query_test.go` (1), activity-backed cases in `nexus_workflow_test.go:695,831` | the handler starts a workflow, sends an update, runs a query or starts an activity; attach semantics; the target workflow closes, resets or continues as new | a machine as a binding target |
| External caller over HTTP | `nexus_api_test.go` (5), `nexus_api_validation_test.go` (8) | dispatch route; reply outcome; auth and claims; size and token limits | a second creating action; rejection rows |
| Standalone operations | `nexus_standalone_test.go` (9) | conflict policy and request-ID idempotency; describe and poll; cancel; terminate; delete; visibility list and count | entity without a caller; bounded eventual observation |
| Completion callbacks | `callbacks_test.go` (6), `callbacks_migration_test.go` (4) | delivery retries after faults; carried over continue-as-new and retries; reset; HSM/CHASM flag flips | a callback entity and machine; reset |
| Endpoint registry | `nexus_endpoint_test.go` (10) | create, update, delete, list; versions; long-poll on table version | an endpoint entity with its own machine |
| Matching dispatch | `nexus_matching_test.go` (2) | forwarding between partitions | topology setup parameters |
| Tracing and metrics | `nexus_otel_test.go` (4), metric assertions inside many tests | span and metric shapes | observations from a second catalog |
| Cross-cluster | `xdc/nexus_request_forwarding_test.go` (3), `xdc/nexus_state_replication_test.go` (5), `xdc/buffered_nexus_events_replication_test.go` (3) | forwarding from standby; replication; failover; conflict resolution with buffered events | topology setup and a failover action |

Four properties of this set drove the design:

1. **Many entities, related by identity.** `TestNexusAsyncOperationWithMultipleCallers`
   (`nexus_workflow_test.go:2763`) has five operations sharing one handler workflow;
   `TestNexusOperationAsyncCompletionBeforeStart` (`:1358`) has two callers attached to one handler
   run.
2. **Three sources of decisions.** The test decides some steps (the handler's reply, when the caller
   cancels), the server decides others (writing `NexusOperationStarted`, retrying), and the world
   decides the rest (a timeout elapses, an HTTP call fails, two completions race).
3. **Steps without events.** A retryable attempt failure writes no history event; only
   `DescribeWorkflowExecution` shows it.
4. **Behavior depends on configuration.** The implementation switch, retry intervals, the callback
   URL template, concurrency limits and auth settings all change outcomes.
