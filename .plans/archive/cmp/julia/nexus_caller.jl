# authoring: header
"""
# The Nexus caller-side Model

One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
operation does, the protocol machine says how the server gets there and refines it, and the
functional set runs one Query per side effect that settles the operation, once per value of the
implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79) and
no concurrency-limit setup parameter.

The regions `AUTHORING.md` quotes are marked `# authoring: <name>`; a region runs to the next
marker. The drift test reads the markers, so a quoted block and the Model cannot part.

Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
"""
module NexusCaller

using Umpire
using ..Worker

# authoring: entities

# ### Entities
#
# An operation is scheduled by a caller workflow, and recorded data names one by its scheduled event:
# every history event of the operation carries that event's id.

@entity workflow

@entity operation begin
    refer = (caller = workflow,)
    key = scheduledEvent
end

# authoring: domains

# ### The input domains
#
# A class is one member of a domain, and a constructor that carries finite fields contributes one
# class per assignment of them: `handlerError(retryable::Bool)` is one constructor and two classes,
# which is the granularity an example is written at and what mirrors a protobuf oneof.
#
# Flat domains are EnumX enums (`Timeout.unset`, type `Timeout.T`); a domain with a fielded member
# is a Moshi sum type (`Reply.handlerError(true)`, type `Reply.Type`). Both spell a member the way
# Lean spells `.unset`, module-qualified, so two domains may share a member name.

@enumx Timeout unset expires

@data Reply begin
    syncSuccess
    async
    operationFailed
    operationCanceled
    handlerError(retryable::Bool)
end

@enumx Resolution succeeded failed canceled

@enumx Delivery accepted notFound

# authoring: actions

# ### Actions
#
# Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
# The reserved party `system` is the server. A fault is an ordinary action of a declared party, and a
# timer is `system` behavior the machine owns, so neither is a separate kind.

@action schedule begin
    party = caller
    creates = operation
    schema = "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"
    input = (
        scheduleToClose = Timeout.T,
        scheduleToStart = Timeout.T,
        startToClose = Timeout.T,
    )
end

@action handlerReply begin
    party = handler
    on = operation
    schema = "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError"
    input = (reply = Reply.Type,)
    examples = (
        handlerError(false) = BadRequest,
        handlerError(true) = Internal,
    )
end

# The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
# are names the realization interprets.
@action complete begin
    party = handler
    on = operation
    input = (resolution = Resolution.T,)
    results = Delivery.T
end

@action transportFault begin
    party = network
    on = operation
end

# The handler's worker stops polling. An action that names no entity is behavior no entity records:
# the Run records the fault, but nothing recorded names the operation, so the machines keep their
# state and record nothing at it. (Declared by `Worker`; this Model steps on it.)

# authoring: observation

# ### The derived observation
#
# A retryable attempt failure writes no history event, so the attempt count is read back through
# `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's catalog,
# which is why only a derived observation is declared.

@observation pendingAttempts begin
    on = operation
    read = attempts
end

# authoring: product

# ### The product machine
#
# What an operation does, with no account of how. Every Property written against it is carried to
# the protocol machine by the refinement declared there.

@enumx ProductPhase scheduled started succeeded failed canceled timedOut

Base.@kwdef struct ProductState
    phase::ProductPhase.T
end

@enumx ProductOutcome accepted notFound

@enumx ProductFact begin
    nexusOperationScheduled
    nexusOperationStarted
    nexusOperationCompleted
    nexusOperationFailed
    nexusOperationCanceled
    nexusOperationTimedOut
end

const PStep = Step{ProductState,ProductOutcome.T,ProductFact.T}

productStep(phase::ProductPhase.T, recorded::ProductFact.T) =
    [PStep(outcome = ProductOutcome.accepted, state = ProductState(phase = phase), facts = [recorded])]

"""
The handler's reply to the server's start request. An operation that has not started yet is the
only one a reply can move.
"""
function handlerReplyStep(state::ProductState, reply::Reply.Type)::Vector{PStep}
    state.phase == ProductPhase.scheduled || return PStep[]
    return @match reply begin
        Reply.syncSuccess       => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
        Reply.async             => productStep(ProductPhase.started, ProductFact.nexusOperationStarted)
        Reply.operationFailed   => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
        Reply.operationCanceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
        # A retryable handler error leaves the operation where it is: the product machine does not
        # know about backing off, which is the whole of what the protocol machine adds.
        Reply.handlerError(true)  => PStep[]
        Reply.handlerError(false) => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
    end
end

"""The four phases the product machine ends on."""
productTerminal(state::ProductState) = state.phase in
    (ProductPhase.succeeded, ProductPhase.failed, ProductPhase.canceled, ProductPhase.timedOut)

"""
An asynchronous completion. A completion that arrives after the operation is over is not found,
and changes nothing.
"""
function completeStep(state::ProductState, resolution::Resolution.T)::Vector{PStep}
    productTerminal(state) && return [PStep(outcome = ProductOutcome.notFound, state = state)]
    return @match resolution begin
        Resolution.succeeded => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
        Resolution.failed    => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
        Resolution.canceled  => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
    end
end

"""
A transport fault is an ordinary action of the network. The product machine cannot see one:
whether a delivery was retried is the protocol's account of how, not what.
"""
transportFaultStep(::ProductState)::Vector{PStep} = PStep[]

"""
The handler's worker stopping is a fault the Run records and the operation does not feel. The
product machine cannot see it, like the transport fault: a step that kept the state and recorded
nothing would be indistinguishable from a stutter, and the refinement would read every stutter as
this step.
"""
workerStopStep(::ProductState)::Vector{PStep} = PStep[]

"""
One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
product machine has one timer, and it fires while the operation runs.
"""
function timeoutStep(state::ProductState)::Vector{PStep}
    state.phase in (ProductPhase.scheduled, ProductPhase.started) || return PStep[]
    return productStep(ProductPhase.timedOut, ProductFact.nexusOperationTimedOut)
end

@machine nexusProduct begin
    var"for" = operation
    state = ProductState
    outcome = ProductOutcome.T
    fact = ProductFact.T
    starts = [scheduled]
    ends = [succeeded, failed, canceled, timedOut]
    timers = [timeout]
    evidence = (
        nexusOperationStarted = nexusOperationStarted,
        nexusOperationCompleted = nexusOperationCompleted,
        nexusOperationFailed = nexusOperationFailed,
        nexusOperationCanceled = nexusOperationCanceled,
        nexusOperationTimedOut = nexusOperationTimedOut,
    )
    steps = (
        handlerReply = handlerReplyStep,
        complete = completeStep,
        transportFault = transportFaultStep,
        workerStop = workerStopStep,
        timeout = timeoutStep,
    )
end

# authoring: protocol

# ### The protocol machine
#
# How the server gets there: the retry the product machine cannot see, the three timers the schedule
# command sets, and the attempt count a retryable failure raises. Written against the same actions,
# so a Property proved on the product machine is carried here by the refinement.
#
# The machine begins before the operation exists: a state structure has no "no instance yet" member,
# so `unscheduled` is that member, and it is what makes the three deadline fields reachable at
# anything but their first value -- the schedule command is what sets them.
#
# Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and the
# concurrency-limit rejection. The limit exists, but a step function does not read the setup, the
# key and value differ per switch value, and the rejection names no operation, so it is not modeled
# until a Query needs it.

@enumx Phase unscheduled scheduled backingOff started succeeded failed canceled timedOut

# Which timer fired. The history event records it, so a Contract that did not check it would pass
# a run that timed out on the wrong deadline.
@enumx TimeoutType scheduleToClose scheduleToStart startToClose

# The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
# machine's state, so the bound is written here and the saturating successor keeps a retry inside it.
const attemptBound = 2

Base.@kwdef struct ProtocolState
    phase::Phase.T
    attempts::Fin{attemptBound}
    scheduleToClose::Timeout.T
    scheduleToStart::Timeout.T
    startToClose::Timeout.T
end

@enumx ProtocolOutcome accepted notFound

@data ProtocolFact begin
    nexusOperationScheduled
    nexusOperationStarted
    nexusOperationCompleted
    nexusOperationFailed
    nexusOperationCanceled
    nexusOperationTimedOut(timeoutType::TimeoutType.T)
    pendingAttempts
end

const TStep = Step{ProtocolState,ProtocolOutcome.T,ProtocolFact.Type}

"""The four phases the design ends on. A completion that arrives after one of them is not found."""
terminalPhase(phase::Phase.T) = phase in (Phase.succeeded, Phase.failed, Phase.canceled, Phase.timedOut)

"""Scheduled and not yet over: the phases a completion resolves and a timer can fire in."""
running(phase::Phase.T) = phase in (Phase.scheduled, Phase.backingOff, Phase.started)

moves(state::ProtocolState, phase::Phase.T, recorded) =
    [TStep(outcome = ProtocolOutcome.accepted, state = with(state; phase), facts = recorded)]

"""
The caller's schedule command. It names the operation's three deadlines, and every one of them is a
state field because whether a timer fires is a question about the operation and not about the
command that started it.
"""
function scheduleStep(state::ProtocolState, scheduleToClose::Timeout.T, scheduleToStart::Timeout.T,
                      startToClose::Timeout.T)::Vector{TStep}
    state.phase == Phase.unscheduled || return TStep[]
    return [TStep(outcome = ProtocolOutcome.accepted,
                  state = ProtocolState(; phase = Phase.scheduled, attempts = 0,
                                        scheduleToClose, scheduleToStart, startToClose),
                  facts = [ProtocolFact.nexusOperationScheduled])]
end

"""
The handler's reply to the server's start request. What the product machine cannot see is the last
arm: a retryable failure backs the operation off and raises its attempt count, and the count is
read back through the `pendingAttempts` observation because no history event records it.
"""
function protocolHandlerReplyStep(state::ProtocolState, reply::Reply.Type)::Vector{TStep}
    state.phase == Phase.scheduled || return TStep[]
    return @match reply begin
        Reply.syncSuccess         => moves(state, Phase.succeeded, [ProtocolFact.nexusOperationCompleted])
        Reply.async               => moves(state, Phase.started, [ProtocolFact.nexusOperationStarted])
        Reply.operationFailed     => moves(state, Phase.failed, [ProtocolFact.nexusOperationFailed])
        Reply.operationCanceled   => moves(state, Phase.canceled, [ProtocolFact.nexusOperationCanceled])
        Reply.handlerError(false) => moves(state, Phase.failed, [ProtocolFact.nexusOperationFailed])
        Reply.handlerError(true)  =>
            [TStep(outcome = ProtocolOutcome.accepted,
                   state = with(state; phase = Phase.backingOff, attempts = saturatingSucc(state.attempts)),
                   facts = [ProtocolFact.pendingAttempts])]
    end
end

"""A transport fault is the same failure arriving as a dropped delivery rather than as a reply."""
function protocolTransportFaultStep(state::ProtocolState)::Vector{TStep}
    state.phase == Phase.scheduled || return TStep[]
    return [TStep(outcome = ProtocolOutcome.accepted,
                  state = with(state; phase = Phase.backingOff, attempts = saturatingSucc(state.attempts)),
                  facts = [ProtocolFact.pendingAttempts])]
end

"""
The handler's worker stopping is a fault the Run records and the operation does not feel, so the
step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
after it, and the Case says so in a Known Gap.
"""
protocolWorkerStopStep(state::ProtocolState)::Vector{TStep} =
    [TStep(outcome = ProtocolOutcome.accepted, state = state)]

"""
An asynchronous completion. Before a start, the server records a Started event first, which is why
the evidence is two facts and not one -- and why the product machine, which has no `backingOff`
phase to have skipped, could write the completion alone.
"""
function protocolCompleteStep(state::ProtocolState, resolution::Resolution.T)::Vector{TStep}
    terminalPhase(state.phase) && return [TStep(outcome = ProtocolOutcome.notFound, state = state)]
    state.phase == Phase.unscheduled && return TStep[]
    startedFirst = state.phase == Phase.started ? ProtocolFact.Type[] : [ProtocolFact.nexusOperationStarted]
    return @match resolution begin
        Resolution.succeeded => moves(state, Phase.succeeded, [startedFirst; ProtocolFact.nexusOperationCompleted])
        Resolution.failed    => moves(state, Phase.failed, [startedFirst; ProtocolFact.nexusOperationFailed])
        Resolution.canceled  => moves(state, Phase.canceled, [startedFirst; ProtocolFact.nexusOperationCanceled])
    end
end

"""
The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a state it
is stuck in, and it records nothing: a retry writes no history event.
"""
function backoffStep(state::ProtocolState)::Vector{TStep}
    state.phase == Phase.backingOff || return TStep[]
    return moves(state, Phase.scheduled, ProtocolFact.Type[])
end

"""
The schedule-to-close deadline covers the whole operation, so it fires in every running phase -- and
only when the schedule command set it.
"""
function scheduleToCloseStep(state::ProtocolState)::Vector{TStep}
    running(state.phase) && state.scheduleToClose == Timeout.expires || return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToClose)])
end

"""The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the start."""
function scheduleToStartStep(state::ProtocolState)::Vector{TStep}
    state.phase in (Phase.scheduled, Phase.backingOff) && state.scheduleToStart == Timeout.expires ||
        return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)])
end

"""The start-to-close deadline covers the handler's own work, so it begins at the start."""
function startToCloseStep(state::ProtocolState)::Vector{TStep}
    state.phase == Phase.started && state.startToClose == Timeout.expires || return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose)])
end

"""
How a protocol state reads as a product state. A phase of the same name is that phase; backing off
is still scheduled, because the product machine cannot see a retry; and an operation not yet
scheduled reads as scheduled, because the product machine begins there. Every other field is
hidden, which is what a map that does not read it says.
"""
function productOf(state::ProtocolState)::ProductState
    phase = @match state.phase begin
        Phase.unscheduled || Phase.scheduled || Phase.backingOff => ProductPhase.scheduled
        Phase.started   => ProductPhase.started
        Phase.succeeded => ProductPhase.succeeded
        Phase.failed    => ProductPhase.failed
        Phase.canceled  => ProductPhase.canceled
        Phase.timedOut  => ProductPhase.timedOut
    end
    return ProductState(phase = phase)
end

@machine nexusProtocol begin
    var"for" = operation
    state = ProtocolState
    outcome = ProtocolOutcome.T
    fact = ProtocolFact.Type
    refines = nexusProduct
    map = productOf
    starts = [unscheduled]
    ends = [succeeded, failed, canceled, timedOut]
    timers = [backoff, scheduleToClose, scheduleToStart, startToClose]
    unobservable = [backoff]
    evidence = (
        nexusOperationScheduled = nexusOperationScheduled,
        nexusOperationStarted = nexusOperationStarted,
        nexusOperationCompleted = nexusOperationCompleted,
        nexusOperationFailed = nexusOperationFailed,
        nexusOperationCanceled = nexusOperationCanceled,
        nexusOperationTimedOut = nexusOperationTimedOut,
        pendingAttempts = pendingAttempts,
    )
    steps = (
        schedule = scheduleStep,
        handlerReply = protocolHandlerReplyStep,
        complete = protocolCompleteStep,
        transportFault = protocolTransportFaultStep,
        workerStop = protocolWorkerStopStep,
        backoff = backoffStep,
        scheduleToClose = scheduleToCloseStep,
        scheduleToStart = scheduleToStartStep,
        startToClose = startToCloseStep,
    )
end

# authoring: properties

# ### What the machines promise
#
# A same-step claim names the action it is about under `when` and holds of the step that action
# produces; a transition claim holds of the step before and the step after. A functional Query
# realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
# action the Case performs; a transition claim is searched and verified, never realized.

# Once an operation is over, no step changes its phase. Declared on the product machine and read on
# the protocol machine through the map.
@property terminalIsFinal begin
    machine = nexusProduct
    holds = (before, after) ->
        !productTerminal(before.state) || after.state.phase == before.state.phase
end

# A synchronous reply settles the operation as succeeded, and the completed event records it.
@property syncSucceeds begin
    machine = nexusProtocol
    when = handlerReply(syncSuccess)
    holds = step -> step.state.phase == Phase.succeeded &&
        ProtocolFact.nexusOperationCompleted in step.facts
end

# An asynchronous reply starts the operation, and the started event records it.
@property asyncStarts begin
    machine = nexusProtocol
    when = handlerReply(async)
    holds = step -> step.state.phase == Phase.started && ProtocolFact.nexusOperationStarted in step.facts
end

# A successful completion is recorded by the completed event. Neither the phase nor the outcome is
# fixed: a completion resolves any running phase, and `accepted` is every earlier step's outcome
# too, so a clause fixing it would be answered before the completion.
@property completionSucceeds begin
    machine = nexusProtocol
    when = complete(succeeded)
    holds = step -> ProtocolFact.nexusOperationCompleted in step.facts
end

# A failed completion is recorded by the failed event.
@property completionFails begin
    machine = nexusProtocol
    when = complete(failed)
    holds = step -> ProtocolFact.nexusOperationFailed in step.facts
end

# A non-retryable handler error settles the operation as failed, and the failed event records it.
@property handlerErrorFails begin
    machine = nexusProtocol
    when = handlerReply(handlerError(false))
    holds = step -> step.state.phase == Phase.failed && ProtocolFact.nexusOperationFailed in step.facts
end

# Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state, so
# every field is named.
const succeededOnRetry = ProtocolState(phase = Phase.succeeded, attempts = 1,
    scheduleToClose = Timeout.unset, scheduleToStart = Timeout.unset, startToClose = Timeout.unset)

# A synchronous reply to the retried attempt settles the operation as succeeded on its second
# attempt: the count the retryable failure raised is still one, and the completed event records the
# reply.
@property retrySucceeds begin
    machine = nexusProtocol
    when = handlerReply(syncSuccess)
    holds = step -> step.state == succeededOnRetry && ProtocolFact.nexusOperationCompleted in step.facts
end

# The schedule-to-start deadline settles an operation no handler started as timed out, and the
# timed-out event records which deadline it was.
@property scheduleToStartFires begin
    machine = nexusProtocol
    when = scheduleToStart
    holds = step -> step.state.phase == Phase.timedOut &&
        ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart) in step.facts
end

# The start-to-close deadline settles a started operation no handler completed as timed out.
@property startToCloseFires begin
    machine = nexusProtocol
    when = startToClose
    holds = step -> step.state.phase == Phase.timedOut &&
        ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose) in step.facts
end

# authoring: scenarios

# ### The paths the Queries run
#
# A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
# path below is one upstream functional test's shape: the schedule command with no deadline set,
# then the side effects that settle the operation.

@scenario syncReplied begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(syncSuccess)]
end

@scenario asyncThenSucceeded begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(async), complete(succeeded)]
end

@scenario asyncThenFailed begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(async), complete(failed)]
end

@scenario nonRetryableError begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(handlerError(false))]
end

# The retryable error backs the operation off; the backoff timer fires and records nothing; the
# retried attempt is answered synchronously.
@scenario retriedThenSucceeded begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, unset), handlerReply(handlerError(true)), backoff,
               handlerReply(syncSuccess)]
end

# The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
# answers the start request; the deadline fires. The worker stops after the schedule in the
# operation's order, where the stop changes nothing; the realization stops it before the workflow
# starts, where the stop cannot race the dispatch.
@scenario scheduleToStartExpires begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, expires, unset), workerStop, scheduleToStart]
end

# The schedule command sets the start-to-close deadline; the handler accepts asynchronously and never
# completes; the deadline fires.
@scenario startToCloseExpires begin
    model = nexusProtocol
    starts = unscheduled
    actions = [schedule(unset, unset, expires), handlerReply(async), startToClose]
end

# Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
# sequence of two is found among ninety-nine candidates, one of three among about a thousand and one
# of four among about ten thousand.
@limits two begin
    steps = 2
    actions = 2
    search = 512
end

@limits three begin
    steps = 3
    actions = 3
    search = 4096
end

@limits four begin
    steps = 4
    actions = 4
    search = 32768
end

# authoring: queries

# ### The Queries
#
# The design's seven: sync success, async reply then succeeded callback, async reply then failed
# callback, non-retryable handler error, retryable handler error then sync success after one backoff,
# schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after an
# asynchronous reply. Each finds its same-step claim on its path and is realized by the set below.
# The product claim is verified over every trace of one path, outside the set, because a `verify`
# Query realizes nothing.

@query syncCompletion begin
    find = syncSucceeds
    var"in" = syncReplied
    limits = two
end

@query asyncCompletion begin
    find = completionSucceeds
    var"in" = asyncThenSucceeded
    limits = three
end

@query asyncFailure begin
    find = completionFails
    var"in" = asyncThenFailed
    limits = three
end

@query handlerError begin
    find = handlerErrorFails
    var"in" = nonRetryableError
    limits = two
end

@query retry begin
    find = retrySucceeds
    var"in" = retriedThenSucceeded
    limits = four
end

@query scheduleToStartTimeout begin
    find = scheduleToStartFires
    var"in" = scheduleToStartExpires
    limits = three
end

@query startToCloseTimeout begin
    find = startToCloseFires
    var"in" = startToCloseExpires
    limits = three
end

@query terminalHolds begin
    verify = terminalIsFinal
    var"in" = asyncThenSucceeded
    limits = three
end

# authoring: set

# ### The functional set
#
# Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
# observes the network. The set repeats over the implementation switch, so each Query's Case runs
# once under HSM and once under CHASM.

@set nexusCallerTests begin
    purpose = functional
    bind = (
        caller = driven,
        handler = driven,
        network = observed,
        worker = driven,
    )
    repeat = implementation
    queries = [syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
               scheduleToStartTimeout, startToCloseTimeout]
end

# ### The canary set
#
# A canary runs a Query against a deployment that performs the handler's part itself: the handler is
# `observed`, so the verifier reads which reply occurred and checks the machine allows it. Every step
# of the sync and async completion paths records evidence; a path with a silent step -- the backoff,
# the worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected.

@set nexusCallerCanary begin
    purpose = canary
    bind = (
        caller = driven,
        handler = observed,
        network = observed,
        worker = driven,
    )
    queries = [syncCompletion, asyncCompletion]
end

# ### The exploratory set
#
# An exploration covers the protocol machine rather than listing Queries. Its targets are the rows an
# exploration within the budget's steps of a start can take, the results those rows reach and the
# members of the classes their actions claim, each in the machine's catalog order and cut at the
# budget's search count, so the enumeration is the same on every reading.

@set nexusCallerExploration begin
    purpose = exploratory
    bind = (
        caller = driven,
        handler = driven,
        network = observed,
        worker = driven,
    )
    machine = nexusProtocol
    cover = rows | results | classMembers
    budget = four
end

# authoring: composition

# ### The operation and the handler's worker
#
# The protocol machine's worker stop is a stutter row: the operation cannot see its handler's worker,
# so the schedule-to-start Scenario orders the stop before the request by convention. Composed with
# the worker of the handler's task queue, the stop is the worker's own phase change and every reply
# is the worker serving, so a reply has a row only while the worker polls. No set names the
# composition; it is what the cross-entity claim is verified over.

# The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
# action no `sync` line names would stay executable on its own and admit a stop, a resume and then a
# reply; the operation's timers settle every state a stop leaves.
@machine handlerWorker begin
    from = Worker.polling
    restrict = [workerStop, serve]
end

Base.@kwdef struct NexusCallerState
    operation::ProtocolState
    worker::Worker.WorkerState
end

@compose nexusCaller begin
    var"for" = [operation, Worker.worker]
    state = NexusCallerState
    members = (
        operation = nexusProtocol,
        worker = handlerWorker,
    )
    sync = (
        workerStop = operation.workerStop ∥ worker.workerStop,
        handlerReply = operation.handlerReply ∥ worker.serve,
    )
    starts = [operation.unscheduled, worker.polling]
    ends = [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]
end

# Every reply, of any class, leaves the handler's worker polling: no handler replies while its
# worker is stopped.
@property repliedByPollingWorker begin
    machine = nexusCaller
    when = handlerReply
    holds = step -> step.state.worker.phase == Worker.Phase.polling
end

# A retryable reply backs the operation off; the handler's worker then stops, so the retried attempt
# is never answered and the schedule-to-start deadline fires.
@scenario repliedThenStopped begin
    model = nexusCaller
    starts = operation.unscheduled
    actions = [operation.schedule(unset, expires, unset), handlerReply(handlerError(true)),
               workerStop, operation.scheduleToStart]
end

@query stoppedWorkerRepliesNothing begin
    verify = repliedByPollingWorker
    var"in" = repliedThenStopped
    limits = four
end

# authoring: end

end # module NexusCaller
