# Nexus: connecting side effects to the behavioral model

Design specimen. Nothing here compiles, and no module imports it. The spec that implements it is
fn-85 ("Model side effects as typed interfaces and run query sets").

The Nexus success Model has two Actions, `awaitStart` and `awaitSuccess`, and both only wait. Every
real side effect (the workflow command that schedules the operation, the handler's reply, the
completion callback, the cancel request, the timers) lives in a hand-written Program template
outside the Model. This file proposes the abstractions that bring those side effects into Umpire
Models without teaching Umpire anything about Temporal, and checks them against two sources:

- the server's actual Nexus operation behavior, read at commit `7c42dec82c`;
- the Nexus functional tests under `tests/`, which a Model built on these abstractions must be able
  to express eventually.

No test was run to produce the citations.

## 1. What the functional tests exercise

About 13,000 lines across 16 files. Grouped by what they verify:

| Group | Files | What varies | Kind |
| --- | --- | --- | --- |
| Caller workflow lifecycle | `nexus_workflow_test.go` (31 tests) | handler reply form (sync, async, operation failure, handler error retryable or not, transport fault); completion path (callback before or after start, after reset, after caller close); cancel (before start, failed then retried); three timeouts; endpoint target (worker, external, system) | lifecycle |
| Handlers built from other APIs | `nexus_workflow_update_test.go` (16), `nexus_workflow_query_test.go` (1), activity-backed cases in `nexus_workflow_test.go:695,831` | the handler starts a workflow, sends an update, runs a query or starts an activity; attach semantics; target workflow closes, resets or continues as new | lifecycle across entities |
| External caller over HTTP | `nexus_api_test.go` (5), `nexus_api_validation_test.go` (8) | dispatch route (by endpoint, by namespace and task queue); reply outcome; auth and claims; size and token limits | lifecycle and input validation |
| Standalone operations | `nexus_standalone_test.go` (9) | conflict policy and request-ID idempotency; describe and poll; cancel; terminate; delete; visibility list and count | lifecycle, idempotency, eventual consistency |
| Completion callbacks | `callbacks_test.go` (6), `callbacks_migration_test.go` (4) | delivery retries after faults; carried over continue-as-new and retries; reset; HSM/CHASM flag flips | lifecycle across runs |
| Endpoint registry | `nexus_endpoint_test.go` (10) | create, update, delete, list; versions; long-poll on table version | registry CRUD |
| Matching dispatch | `nexus_matching_test.go` (2) | forwarding between partitions | infrastructure |
| Tracing and metrics | `nexus_otel_test.go` (4), metric assertions inside many tests | span and metric shapes | observability |
| Cross-cluster | `xdc/nexus_request_forwarding_test.go` (3), `xdc/nexus_state_replication_test.go` (5), `xdc/buffered_nexus_events_replication_test.go` (3) | forwarding from standby; replication; failover; conflict resolution with buffered events | topology |

Four properties of this set drive the design more than any single test:

1. **Many entities, related by identity.** Most tests have one caller, one operation and one
   handler. `TestNexusAsyncOperationWithMultipleCallers` (`nexus_workflow_test.go:2763`) has five
   operations sharing one handler workflow. `TestNexusOperationAsyncCompletionBeforeStart`
   (`:1358`) has two callers attached to one handler run. Update-backed handlers relate an operation
   to an update on another workflow.
2. **Three sources of decisions.** The test decides some steps (which reply the handler gives, when
   the caller cancels). The server decides others (writing `NexusOperationStarted`, retrying). The
   environment decides the rest (a timeout elapses, an HTTP call fails, two completions race).
3. **Steps without events.** A retryable attempt failure writes no history event; only
   `DescribeWorkflowExecution` shows it through the attempt count and `LastAttemptFailure`.
4. **Behavior depends on configuration.** The HSM or CHASM implementation, retry intervals, the
   callback URL template, concurrency limits and auth settings all change outcomes, and tests set
   them through dynamic config.

## 2. What the server does

Workflow-scheduled operations run on the HSM implementation by default:
`nexusoperation.enableChasmWorkflowOperations` defaults to false
(`chasm/lib/nexusoperation/config.go:38-43`). The two implementations read different dynamic
config families (`component.nexusoperations.*` and `nexusoperation.*`) with different defaults,
for example a concurrency limit of 30 (`service/history/hsm/nexusoperations/config.go:37-40`)
against 2000 (`chasm/lib/nexusoperation/config.go:90-91`).

Below, `NX` is `service/history/hsm/nexusoperations/`.

**Operation states and events.**

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

Retries have no attempt limit and no expiration; only a timeout ends them
(`NX/config.go:189-197`).

**Cancellation** is a separate sub-state of the operation. The cancel command always writes
`NexusOperationCancelRequested`. The request reaches the handler only once the operation has
started, because it needs the operation token; a cancel requested earlier waits for the start
(`NX/statemachine.go:425-450`). Delivery retries like a start. A delivered cancel writes
`NexusOperationCancelRequestCompleted` and does not end the operation; the final state still comes
from the handler's completion (`NX/executors.go:902-904`). A search of the operation's task
executors found no path that cancels pending operations when the caller workflow closes; every task
checks that the caller is still running and stops otherwise.

**Configuration that changes outcomes.** Scheduling rejects a missing endpoint, oversized names or
headers, and exceeding the concurrency limit by failing the workflow task. An external endpoint
needs `component.nexusoperations.callback.endpoint.template`; while it is `unset` every attempt
fails with an Internal error and no event (`NX/executors.go:145-147`).

## 3. The abstractions

All of these live in Umpire and name nothing Temporal-specific. The last section, realization, is
the only Temporal-owned part.

### 3.1 Entity

A kind of thing with identity and its own state: an operation, a workflow, an endpoint, a callback,
an update. A Model's state is a finite collection of entity instances, each with structured state
fields. Entities refer to each other (an operation has a caller workflow and an endpoint; a callback
belongs to an operation), and references are compared, never interpreted.

The current Model has one role with one instance and a flat state enum. The five-operations test
and the two-callers test cannot be written without this.

### 3.2 Interface, party and binding

A named side effect with a typed input, typed results, a **kind** and a **party**.

The kind is structural: *call* (request with a result), *command* (issued inside an entity,
answered later by events), *reply* (an answer to a call another party made), or *observation*
(recorded data with no request). Which RPC, workflow command, handler response or history event
realizes each one is decided in the realization.

The party names the side that performs the interface: `caller`, `handler`, `system` or
`environment` for Nexus. The interface fixes its party. A **set** binds each party other than
`system` to a source of decisions:

| Bound to | Who performs it | What the verifier does |
| --- | --- | --- |
| test | the Case's own Program (the controller, a pre-programmed workflow or handler) | chooses it from the Scenario |
| environment | a real deployment, time, faults, concurrency | observes or explores the choice and checks the Model allows it |
| (`system`, never bound) | the system under test | checks that the step it takes is one the Model allows |

A functional set binds `handler` to test: the Case's handler replies what the Scenario selects. A
canary set binds `handler` to environment: a real deployed handler replies, and the verifier checks
the observed reply class against the Model. The Model is the same for both sets; only the
verifier's treatment of the choice differs.

Today `awaitStart` is written as if the test performs it. It is really an observation of a
`system` step. With parties declared, a Scenario selects only choices whose party its set binds to
test, and `system` steps follow from the Model.

### 3.3 Abstract input

An interface instance is its interface plus an abstract argument:

- **dimensions**: finite partitions of input fields, where each class is claimed to behave alike
  (`reply: [syncSuccess, async, operationFailed, operationCanceled, handlerError]`);
- **references** to entities (`operation`, `caller`);
- **defaults** for fields the Model claims do not matter;
- **input rules**: normalization and rejection that apply before any state is read (the scheduling
  checks in section 2 fail the workflow task regardless of lifecycle state).

`Umpire.Operation.ParameterDomain` today lists whole concrete requests and rejects abstraction
(`ParameterCoverage.abstracted` throws `unsupportedAbstraction`). A dimension is the missing
abstraction.

A class is a claim, and exploration owns its evidence:

- a class with exactly one concrete member (one generated enum value, one literal) needs no
  evidence;
- a class with several members (`handlerError, retryable: no` covers BadRequest, Unauthenticated,
  NotFound and more) is recorded in Provenance as an unverified abstraction claim;
- a functional set realizes one declared representative per class, so its fixtures stay
  deterministic;
- an exploratory set's coverage goal includes the members of each claimed class; two members with
  different verdicts are a counterexample, the class is split, and Promotion keeps the
  counterexample as a Regression.

### 3.4 Result classification

The results of an interface are classes of its typed result: the handler error types the Nexus SDK
treats as non-retryable (BadRequest, Unauthenticated, Unauthorized, NotFound, NotImplemented,
Conflict, unless an explicit retry behavior overrides them) form one class, the rest another. The
classification function belongs to the interface's realization; the Model only sees classes. This
is how `TestNexusSyncOperationErrorRehydration` (`nexus_workflow_test.go:2242`) becomes five rows
over two result classes instead of five hand-written Programs.

### 3.5 Observation and correlation

An observation kind carries typed data and a key that names the entity it belongs to (the scheduled
event ID for a Nexus operation, a request ID for an attached start). An observation confirms a
Model step. Two extensions are needed:

- **observation by query**: steps that write no event are observed through a read interface
  (describe the operation's attempt count and last failure);
- **observation on another entity**: a step on the operation can confirm through an event on a
  different entity (the handler workflow's callback state in
  `TestNexusCallbackAfterCallerComplete`, `nexus_workflow_test.go:2582`).

`Umpire.Case.Projection` already maps events to steps generically; the Nexus event kinds and keys
currently sit inside the realization template.

### 3.6 Environment steps

Time, faults and interleavings are environment steps with logical timing: a timer is enabled when
its guard holds and may fire as a step; a fault replaces a call's result with a fault class; two
enabled steps may happen in either order. Limits bound how many environment steps a trace may take.
Wall-clock durations stay in the realization.

### 3.7 Configuration as setup

Dynamic config values that change behavior on purpose are setup parameters of the Model, named
after the setting rather than after an implementation (`concurrencyLimit: [atLimit, belowLimit]`,
`recordCancelCompletion: [yes, no]`). Rows may guard on them. The Profile binds them to concrete
values, and a Case that needs a value its environment cannot set carries a Known Gap.

A rollout switch between two implementations of the same behavior (HSM or CHASM Nexus operations)
is not a setup parameter. The Model states the behavior both must satisfy, and a set runs its
Queries once per switch value; a verdict that differs between the runs is a divergence to explain.
Incidental differences (attempt numbering in log tags) stay outside the Model, and internal storage
layout read through `DescribeMutableState` is a white-box Known Gap.

### 3.8 Two levels and a link

A **product model** says what Nexus means to a user: an operation is scheduled, may start, and ends
succeeded, failed, canceled or timed out. An **interface model** spells out attempts, backoff, reply
forms, callbacks, cancel delivery and timers. `Umpire.ImplementationLink` already relates two
independently checked Models by value mappings and a forward simulation, and
`Temporal/System/Nexus/ImplementationLink.lean` uses it for a small Nexus pair. Properties go on the
level where they are natural, and the link carries product properties to every interface-level
trace.

A feature starts with one level: its interface model is its product model. It adds a product model
when a product property would otherwise mention mechanism (attempts, backoff, cancel delivery), or
when several realizations share one product meaning. Nexus meets both: an operation is created by
a workflow command, by an external HTTP caller, or by the standalone API, and two implementations
run it. Both levels and the link live in one file.

### 3.9 Realization

The only Temporal-owned layer. It binds each interface to its concrete transport (an RPC method, a
workflow command, a handler reply instruction, a history event field), each result class to its
classification function, each observation key to a field path, each setup parameter to a dynamic
config key, and each entity reference to runtime identifiers.

## 4. Specimen

Sketch syntax in the respelled command style. It covers one workflow-scheduled operation on the
HSM implementation, including retries, callbacks, cancel and timeouts.

```lean
entity workflow
  state:
    closed: bool

entity endpoint
  vary:
    target: [worker, external, system]

entity operation
  refer:
    caller: workflow
    endpoint: endpoint
  key: scheduledEventId
  state:
    phase: Phase
    cancel: CancelPhase
    attempts: count

enum Phase
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

enum CancelPhase
  | none
  | waitingForStart
  | delivering
  | delivered
  | rejected

-- caller and handler parties: a set binds each to test or environment
interface schedule
  kind: command
  party: caller
  creates: operation
  input:
    scheduleToClose: [none, expires]
    scheduleToStart: [none, expires]
    startToClose: [none, expires]
  rules:
    endpoint missing → reject: workflowTaskFailed
    concurrencyLimit: atLimit → reject: workflowTaskFailed

interface handlerReply
  kind: reply
  party: handler
  on: operation
  input:
    reply: [syncSuccess, async, operationFailed, operationCanceled, handlerError]
    retryable: [yes, no]                       -- read only when reply is handlerError
  representatives:
    handlerError, retryable: no → BadRequest   -- one of several members; an unverified claim
    handlerError, retryable: yes → Internal

interface completeCallback
  kind: call
  party: handler
  on: operation
  input:
    result: [succeeded, failed, canceled]
  results:
    accepted
    notFound

interface requestCancel
  kind: command
  party: caller
  on: operation

interface cancelReply
  kind: reply
  party: handler
  on: operation
  input:
    reply: [delivered, handlerError]
    retryable: [yes, no]

-- environment party: explored within Limits
environment
  fault: transport on handlerReply
  timer: backoff, scheduleToClose, scheduleToStart, startToClose

-- system steps, checked against observations
model nexusOperation
  entity: operation
  setup:
    concurrencyLimit: [atLimit, belowLimit]
    recordCancelCompletion: [yes, no]
  steps:
    none + schedule
      → phase: scheduled, cancel: none, attempts: 0,
        evidence: nexusOperationScheduled

    phase: scheduled + handlerReply(reply: syncSuccess)
      → phase: succeeded, evidence: nexusOperationCompleted
    phase: scheduled + handlerReply(reply: async)
      → phase: started, evidence: nexusOperationStarted
    phase: scheduled + handlerReply(reply: operationFailed)
      → phase: failed, evidence: nexusOperationFailed
    phase: scheduled + handlerReply(reply: handlerError, retryable: no)
      → phase: failed, evidence: nexusOperationFailed
    phase: scheduled + handlerReply(reply: operationCanceled)
      → phase: canceled, evidence: nexusOperationCanceled
    phase: scheduled + handlerReply(reply: handlerError, retryable: yes)
      → phase: backingOff, attempts: +1, evidence: query attempts
    phase: scheduled + fault(transport)
      → phase: backingOff, attempts: +1, evidence: query attempts
    phase: backingOff + timer(backoff)
      → phase: scheduled

    phase: scheduled | backingOff + completeCallback(result: r)
      → phase: r, evidence: nexusOperationStarted then completion(r)
    phase: started + completeCallback(result: r)
      → phase: r, evidence: completion(r)
    phase: terminal + completeCallback
      → unchanged, result: notFound

    phase: scheduled | backingOff | started + timer(scheduleToClose)
      → phase: timedOut, evidence: nexusOperationTimedOut(scheduleToClose)
    phase: scheduled | backingOff + timer(scheduleToStart)
      → phase: timedOut, evidence: nexusOperationTimedOut(scheduleToStart)
    phase: started + timer(startToClose)
      → phase: timedOut, evidence: nexusOperationTimedOut(startToClose)

    phase: not terminal, cancel: none + requestCancel
      → cancel: waitingForStart, evidence: nexusOperationCancelRequested
    phase: started, cancel: waitingForStart
      → cancel: delivering
    cancel: delivering + cancelReply(reply: delivered)
      → cancel: delivered,
        evidence: nexusOperationCancelRequestCompleted when recordCancelCompletion: yes
    cancel: delivering + cancelReply(reply: handlerError, retryable: no)
      → cancel: rejected,
        evidence: nexusOperationCancelRequestFailed when recordCancelCompletion: yes

model nexusProduct
  entity: operation
  state:
    phase: [scheduled, started, succeeded, failed, canceled, timedOut]
  steps:
    none → scheduled
    scheduled → started | succeeded | failed | canceled | timedOut
    started → succeeded | failed | canceled | timedOut

link nexusOperation refines nexusProduct
  state:
    phase: backingOff → scheduled
    cancel: any → hidden
  steps:
    handlerReply(reply: handlerError, retryable: yes) → stutter
    fault(transport) → stutter
    timer(backoff) → stutter

property operationEnds on nexusProduct
  for: operation
  require: eventually phase in [succeeded, failed, canceled, timedOut]
    when scheduleToClose: expires

property retryWritesNoEvent on nexusOperation
  when: handlerReply(reply: handlerError, retryable: yes)
  require:
    no history event on operation
    query attempts = previous attempts + 1

property deliveredCancelDoesNotEnd on nexusOperation
  when: cancelReply(reply: delivered)
  require:
    phase = previous phase

-- One Model, two sets. The functional set runs every Query once per implementation switch value.
set functional nexusCaller
  bind:
    caller: test
    handler: test
  repeat: implementation [hsm, chasm]
  queries: [syncCompletion, asyncCompletion, retryAfterFault, cancelAfterStart]

set canary nexusCaller
  bind:
    caller: test
    handler: environment
  queries: [syncCompletion, asyncCompletion]
```

## 5. How the tests map onto the abstractions

| Test | Abstractions it needs | Status |
| --- | --- | --- |
| `TestNexusOperationSyncCompletion` (`nexus_workflow_test.go:512`), `TestNexusOperationAsyncCompletion` (`:1024`), `TestNexusOperationAsyncFailure` (`:1617`) | interfaces, reply dimension, observations | specimen |
| `TestNexusSyncOperationErrorRehydration` (`:2242`), `TestNexusOperationSyncNexusFailure` (`:2667`) | result classification over handler error types | specimen |
| `TestNexusOperationRetriesAfterHTTPFault` (`:579`) | environment fault, observation by query | specimen |
| `TestNexusOperationScheduleToCloseTimeout` (`:2988`), `...ScheduleToStartTimeout` (`:3061`), `...StartToCloseTimeout` (`:3155`) | environment timers | specimen |
| `TestNexusOperationCancelation` (`:89`), `TestNexusOperationCancelBeforeStarted_CancelationEventuallyDelivered` (`:2006`) | cancel sub-state, interleaving | specimen |
| `TestNexusOperationAsyncCompletionBeforeStart` (`:1358`), `TestNexusAsyncOperationWithMultipleCallers` (`:2763`) | several entity instances with references; the handler's workflow start as its own interface with a conflict policy dimension | needs entities and composition |
| update-, query- and activity-backed handlers (`nexus_workflow_update_test.go`, `nexus_workflow_query_test.go:20`, `nexus_workflow_test.go:695,831`) | a handler reply composed from another entity's interfaces; observation on another entity | needs composition |
| `TestNexusCallbackAfterCallerComplete` (`:2582`), `TestWorkflowNexusCallbacks_CarriedOver` (`callbacks_test.go:323`) | callback entity reusing the same attempt and backoff structure; caller close as a step | needs entities and a reusable retry structure |
| `TestNexusOperationAsyncCompletionErrors` (`:1680`), `nexus_api_validation_test.go` | input rules on the callback and start interfaces; token validity as a dimension | needs input rules |
| `TestNexusStartOperation_Outcomes` (`nexus_api_test.go:64`), `TestNexusCancelOperation_Outcomes` (`:461`) | the same operation from an external caller: a different creating interface, same entity | specimen plus a second creator |
| `nexus_standalone_test.go` | operation entity without a caller workflow; conflict policy and request-ID dimensions; describe and poll as query observations; list and count as eventually consistent queries | needs eventual observation |
| `TestNexusOperationCallerMetrics` (`:637`), `nexus_otel_test.go`, metric assertions elsewhere | metrics and spans as observations | later, or Known Gap |
| `TestNexusOperationAsyncCompletionAfterReset` (`:2078`), reset cases in `callbacks_test.go` and `nexus_workflow_update_test.go` | an action that rewrites history, with reapply rules | later |
| `TestNexusOperationCancellationCrossTree` (`:290`), `callbacks_migration_test.go`, `TestNexusOperationChasmReplicatedWithMixedFlag` (`xdc/nexus_state_replication_test.go:723`) | configuration changing mid-trace; internal storage layout read through `DescribeMutableState` | white-box Known Gap for the storage check; config change as an environment step |
| `nexus_endpoint_test.go` | endpoint entity with versioned CRUD interfaces | separate registry model |
| `nexus_matching_test.go`, `xdc/*` | partition and cluster topology as environment | later |

## 6. What Umpire needs

| # | Need | Today |
| --- | --- | --- |
| 1 | Entities with structured state, several instances, and references | one role, one instance, flat state enum |
| 2 | Interfaces with a kind (`call`, `command`, `reply`, `observation`) and a party; sets that bind parties to test or environment | `Umpire.Operation.Declaration` has kinds `unaryRpc`, `sdkCommand`, `event`, no party, and a protobuf-descriptor schema |
| 3 | Abstract inputs: dimensions, references, defaults, input rules, representatives, and class claims recorded in Provenance | exact request lists; abstraction rejected |
| 4 | Result classification | Response and Failure types on the declaration, no classes |
| 5 | Observations by event, by query, on another entity, and eventually consistent | event projection only, keys inside templates |
| 6 | Environment steps for timers, faults and interleavings, bounded by Limits | none in Models; faults exist only as Program instructions |
| 7 | Setup parameters bound by the Profile | none |
| 8 | Guards, alternatives and wildcards in rows, with precedence or a disjointness check | exact state and Action pairs, one row each |
| 9 | Refinement between a product and an interface model with an authoring surface | `Umpire.ImplementationLink`, expert Lean only |
| 10 | Composition: a reply or step built from another entity's interfaces; reusable sub-structures such as attempt and backoff | none |
| 11 | History-rewriting actions (reset) with reapply rules | none; later |

Needs 1 to 8 are required before the specimen's own test rows can run. Need 9 decides whether the
product and interface models stay separate. Needs 10 and 11 cover the remaining groups.

## 7. Decisions

Recorded 2026-09-10 with the user.

1. **A reply's party is fixed; a set binds it.** An interface declares its party (`caller`,
   `handler`, `system`, `environment`). A set binds each party except `system` to test or
   environment. The Model is the same for every set (section 3.2).
2. **The author claims classes; exploration owns the evidence.** Single-member classes need none.
   Multi-member classes are recorded in Provenance as unverified claims, functional sets realize one
   declared representative, and an exploratory set that finds members with different verdicts
   splits the class and promotes the counterexample (section 3.3).
3. **Protobuf descriptors stay Umpire's only schema.** Testpilot's wire format and `Umpire.Value`
   are built on them, and SCP-01 admits no capability without a use case. The Temporal-flavored
   kind names are renamed: `unaryRpc`, `sdkCommand` and `event` become `call`, `command` and
   `observation`, and `reply` is added. A schema interface waits for a second schema source.
4. **One level by default; Nexus uses two.** A product model is added when product properties would
   mention mechanism, or when several realizations share one product meaning. Nexus meets both
   (section 3.8).
5. **One Model for HSM and CHASM, run under both.** The implementation switch is a rollout flag,
   not behavior. Deliberate config-controlled differences are setup parameters named after the
   setting; incidental differences stay outside the Model; storage layout is a white-box Known Gap.
   A set repeats its Queries per switch value, and a differing verdict is a divergence
   (section 3.7).
