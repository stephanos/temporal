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

> Amended during fn-85 `.4`, 2026-09-19. Instances are declared where a Scenario runs over them:
> `instances: 2` on the `scenario`, with each action naming the instance that takes it
> (`awaitStart 2`). The Search runs over the product of that many copies of the machine, so the
> instances' steps interleave and every interleaving is a path; a Property written over one instance
> is read over the product as the same claim per instance, on the acting instance's own slot, which
> is a field of the product state. A Case follows each operation through one sequence, so every
> instance performs the same actions, and the Producer reads the first instance back with every
> instance's actions as the Program's path. The product's size is checked against the enumeration
> bound where the count is written, an instance count of zero rejects there, and a Search that
> cannot finish its interleavings within its Limits reports that the bound stopped it.

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
fields (`handlerError (retryable : Bool)`), which mirrors a protobuf oneof. Each **member** of that
domain is a class: a set of concrete values claimed to behave alike. A constructor that carries no
field is one class; one that carries fields contributes a class per assignment of them, so
`handlerError (retryable : Bool)` is one constructor and the two classes
`handlerError (retryable := false)` and `handlerError (retryable := true)`. A class covers values
this Model does not enumerate -- `handlerError (retryable := false)` covers BadRequest,
Unauthenticated, NotFound and more -- so writing an **example** for a class is what claims those
values behave alike; the example is the concrete value a functional test uses, and an exploratory run
tries the others (section 2.6).

> Amended during fn-85 `.2`, 2026-09-15. This paragraph read "each *constructor* is a class", and
> then, one sentence later, called `handlerError (retryable := false)` a class -- a fully applied
> constructor, which is a member. The `examples:` block below says the same thing again, writing one
> example for `(retryable := false)` and another for `(retryable := true)`. The old text was
> internally inconsistent rather than uniformly wrong, and the implementation follows the members
> reading. It also follows that nothing in a Model counts the realized values a class covers, so
> R8's abstraction claim is triggered by the presence of an example rather than by a member count --
> recorded on task `.7`.

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

> Amended during fn-85 `.6`, 2026-09-19. The `map:` above is written in the row grammar the user's
> 2026-09-12 decision replaced with ordinary Lean, so a machine names a function instead:
>
> ```lean
> def productOf (state : ProtocolState) : ProductState :=
>   { phase := match state.phase with
>     | .unscheduled | .scheduled | .backingOff => .scheduled
>     | .started => .started
>     | .succeeded => .succeeded
>     | .failed => .failed
>     | .canceled => .canceled
>     | .timedOut => .timedOut }
>
> machine nexusProtocol
>   for: operation
>   refines: nexusProduct
>   map: productOf
> ```
>
> A field the map does not read is hidden by not being read, and `unscheduled` reads as `scheduled`
> because the product machine begins there, which makes the schedule command a stutter. Outcomes
> and facts read as the product's value of the same name, a fact's constructor covering its members
> the way an `evidence:` line does; a fact the product does not name is one the product does not
> see, and an outcome it does not name rejects. A product step may record less than the protocol
> step it carries (a completion before the start records the Started event first), never more. The
> product machine gains one timer, `timeout`, because a deadline firing is neither a stutter nor a
> step a product without one could take. The derived step mapping is read back as
> `nexusProtocol.refinement`, the witness `nexusProtocol.refines` is decided by the kernel over the
> rows, and a Property on `nexusProduct` is read on `nexusProtocol` through a state field named
> `nexusProduct` that carries the product state each protocol state reads as. The simulation is
> `Umpire.ImplementationLink.Refinement`, a forward simulation that may stutter.

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

The caller side of one workflow-scheduled Nexus operation, as `model/Temporal/Feature/Nexus/Caller/Model.lean`
writes it since fn-85 `.10`: the Model file is the specimen, and the blocks below are its regions
(`-- authoring: <name>` markers) in the landed grammar. Cancellation is fn-79's deferred scope and
is not in the Model; the concurrency-limit rejection is not modeled either (the amendment at the end
of this section says why).

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

observation pendingAttempts
  on: operation
  read: attempts
```

A machine is a structure of finite fields and one step function per action: the function returns
every successor the Model permits from a state, and the empty list where the action is not
permitted. The `machine` command enumerates the functions over the structure into the finite table
that Search, the Behavior Fingerprint and Contract lowering read. The product machine says what an
operation does:

```lean
enum ProductPhase
  | scheduled
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

structure ProductState where
  phase : ProductPhase
  deriving BEq, DecidableEq, Repr, Finite

enum ProductOutcome
  | accepted
  | notFound

enum ProductFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut
  | faultInjected

def handlerReplyStep (state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => productStep .succeeded .nexusOperationCompleted
  | .async => productStep .started .nexusOperationStarted
  | .operationFailed => productStep .failed .nexusOperationFailed
  | .operationCanceled => productStep .canceled .nexusOperationCanceled
  | .handlerError true => []                      -- the product cannot see a retry
  | .handlerError false => productStep .failed .nexusOperationFailed

def completeStep (state : ProductState) (resolution : Resolution) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if productTerminal state then [{ outcome := .notFound, state, facts := [] }]
  else match resolution with
    | .succeeded => productStep .succeeded .nexusOperationCompleted
    | .failed => productStep .failed .nexusOperationFailed
    | .canceled => productStep .canceled .nexusOperationCanceled

-- transportFaultStep returns []; workerStopStep records faultInjected and keeps the state;
-- timeoutStep moves scheduled or started to timedOut and records nexusOperationTimedOut.

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
```

The protocol machine says how the server gets there and refines the product machine through a map
from its state to the product's. It begins before the operation exists (`unscheduled`), because a
state structure has no "no instance yet" member and the schedule command is what sets the three
deadline fields:

```lean
enum Phase
  | unscheduled
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

enum TimeoutType
  | scheduleToClose
  | scheduleToStart
  | startToClose

abbrev attemptBound : Nat := 2

structure ProtocolState where
  phase : Phase
  attempts : Fin (attemptBound + 1)
  scheduleToClose : Timeout
  scheduleToStart : Timeout
  startToClose : Timeout
  deriving BEq, DecidableEq, Repr, Finite

enum ProtocolFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut (timeoutType : TimeoutType)
  | pendingAttempts
  | faultInjected

def scheduleStep (state : ProtocolState)
    (scheduleToClose scheduleToStart startToClose : Timeout) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .unscheduled then [] else
  [{ outcome := .accepted
     state := { phase := .scheduled, attempts := 0, scheduleToClose, scheduleToStart, startToClose }
     facts := [.nexusOperationScheduled] }]

def protocolHandlerReplyStep (state : ProtocolState) (reply : Reply) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => moves state .succeeded [.nexusOperationCompleted]
  | .async => moves state .started [.nexusOperationStarted]
  | .operationFailed => moves state .failed [.nexusOperationFailed]
  | .operationCanceled => moves state .canceled [.nexusOperationCanceled]
  | .handlerError false => moves state .failed [.nexusOperationFailed]
  | .handlerError true =>                          -- retry: no history event, the count is read back
      [{ outcome := .accepted
         state := { state with phase := .backingOff, attempts := saturatingSucc state.attempts }
         facts := [.pendingAttempts] }]

-- protocolTransportFaultStep is the retryable arm arriving as a dropped delivery;
-- protocolWorkerStopStep records faultInjected and keeps the state.

def protocolCompleteStep (state : ProtocolState) (resolution : Resolution) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if terminalPhase state.phase then [{ outcome := .notFound, state, facts := [] }]
  else if state.phase == .unscheduled then []
  else
    -- before a start, the server records a Started event first
    let startedFirst : List ProtocolFact :=
      if state.phase == .started then [] else [.nexusOperationStarted]
    match resolution with
    | .succeeded => moves state .succeeded (startedFirst ++ [.nexusOperationCompleted])
    | .failed => moves state .failed (startedFirst ++ [.nexusOperationFailed])
    | .canceled => moves state .canceled (startedFirst ++ [.nexusOperationCanceled])

-- backoffStep moves backingOff to scheduled and records nothing.
-- Each deadline fires only when the schedule command set it, over its own span:
-- scheduleToClose in every running phase, scheduleToStart until the start, startToClose after it.

def productOf (state : ProtocolState) : ProductState :=
  { phase := match state.phase with
    | .unscheduled | .scheduled | .backingOff => .scheduled
    | .started => .started
    | .succeeded => .succeeded
    | .failed => .failed
    | .canceled => .canceled
    | .timedOut => .timedOut }

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
```

A Property is a predicate over the step an action produces (`Step → Bool` under `when:`) or over
the step before and the step after (`Step → Step → Bool`, no `when:`); the command enumerates it
over the machine's table into the clauses Search and the Case read. A functional Query realizes a
same-step claim, because the Case's Contract is that claim's clause triggered by the action the
Case performs; a transition claim is searched and verified, never realized (section 2.5).

```lean
-- A product Property, carried to every protocol path by the refinement.
property terminalIsFinal
  machine: nexusProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase

-- Queries 1 to 4: one same-step claim per side effect that settles the operation.
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

property completionSucceeds                        -- neither phase nor outcome fixed: a completion
  machine: nexusProtocol                            -- resolves any running phase, and `accepted` is
  when: complete (succeeded)                        -- every earlier step's outcome too
  holds: fun step => step.facts.contains .nexusOperationCompleted

property completionFails
  machine: nexusProtocol
  when: complete (failed)
  holds: fun step => step.facts.contains .nexusOperationFailed

property handlerErrorFails
  machine: nexusProtocol
  when: handlerReply (handlerError false)
  holds: fun step => step.state.phase == .failed && step.facts.contains .nexusOperationFailed

scenario asyncThenSucceeded
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (async), complete (succeeded)]

limits three
  steps: 3
  actions: 3
  search: 4096

query asyncCompletion
  find: completionSucceeds
  in: asyncThenSucceeded
  limits: three

query terminalHolds                               -- the product claim, outside the set
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
  queries: [syncCompletion, asyncCompletion, asyncFailure, handlerError]

case nexusCallerCases
  realizes nexusCallerTests
  as nexusOperation service "umpire.case.service" operation "complete" realized by asyncNexus
```

The `case` block writes no evidence lines: each fact a witness step records is confirmed by the
observation the machine's `evidence:` line maps it to, so the mapping is read off the witness at
production, and the Case declares each observation once (fn-85 `.9`). Queries 5 to 7 (a retryable
handler error then success, the two timeouts) are `.11`'s; the canary set of section 2.6 is `.12`'s.

> Amended during fn-85 `.15`, 2026-09-19. Properties are written with an ordinary Lean predicate
> by the same rule that replaced rows with step functions; the keyed `require:` block of the earlier
> specimen is not built. A same-step claim is `Step → Bool` under `when:`; a transition claim such
> as `terminalIsFinal` is `Step → Step → Bool` over the step before and the step after, with no
> `when:`. The command enumerates the predicate over the machine's table into the state, outcome and
> facts it fixes, so what Search and the Case read is the clause language the keyed block wrote by
> hand, with the same fingerprint. A bounded-progress claim (`within:`) is not in this slice.

> Amended during fn-85 `.10`, 2026-09-19. The earlier specimen's `setup: atConcurrencyLimit: Bool`
> and its `atConcurrencyLimit: true + schedule → reject` row are not modeled, and the Model declares
> no setup parameter. The limit exists: HSM bounds pending operations per workflow through
> `component.nexusoperations.limit.operation.concurrency` and CHASM through
> `nexusoperation.limit.operation.concurrencyPerWorkflow.max`, and at the limit the schedule command
> fails the workflow task with cause `PENDING_NEXUS_OPERATIONS_LIMIT_EXCEEDED` and writes no
> `NexusOperationScheduled` event. Three things keep it out: a step function does not read the
> setup, so the table cannot vary with it (R5's guard-row half); the key and the value differ per
> switch value, which a key-only setup binding cannot say; and the rejection names no operation, so
> no source keyed by the scheduled event can lift it. A Query that needs the row reopens it with R5's
> second half as its first step (`UMPIRE4_RESEARCH_NEXUS_MODEL.md` section 1). The earlier
> specimen's `find: terminalIsFinal` Query is rewritten too: a transition claim cannot be a
> functional Query (section 2.5), so the four Queries find same-step claims and the product claim is
> verified beside them.

The realization, in sketch syntax (`Temporal.Case.Realization.asyncNexus` is the Lean value). A
worker instruction carries the Temporal API message the action's `schema:` names, so a class example
fills that message and the Driver maps it to the SDK call that produces it. Each binding names its
class by the member key a path spells it by. Each observation is declared once in the Case and read
by both the Program and the Contract.

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
