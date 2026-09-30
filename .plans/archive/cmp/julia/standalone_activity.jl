# authoring: header
"""
# The standalone activity Model

A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded in
`chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred (like
cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled. Standalone
activities write no history events, so every fact is a status read through
`DescribeActivityExecution` or a result read through `PollActivityExecution`.

Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
"""
module StandaloneActivity

using Umpire
using ..Worker
using ..NexusCaller: Timeout, Delivery      # the same two domains; a class is a class

# authoring: entities

# ### Entities
#
# An activity is named by the id the caller chose for it.

@entity activity begin
    key = activityId
end

# authoring: domains

# ### The input domains
#
# `failed(retryable::Bool)` is one constructor and two classes, as `handlerError` is in the Nexus
# Model; the example table below is written at that granularity.

@data AttemptResult begin
    completed
    failed(retryable::Bool)
    canceled
end

@enumx Control pause unpause requestCancel terminate

# authoring: actions

# ### Actions
#
# Parties: `caller` starts and controls, `worker` attempts. There is no `handler` and no `network`: a
# dropped poll is a worker that never attempted, and the timers cover it.

@action start begin
    party = caller
    creates = activity
    schema = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    input = (
        scheduleToClose = Timeout.T,
        scheduleToStart = Timeout.T,
        startToClose = Timeout.T,
    )
end

# The worker's poll receives the task (`PollActivityTaskQueue`).
@action attemptStart begin
    party = worker
    on = activity
    schema = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"
end

@action attemptResult begin
    party = worker
    on = activity
    input = (result = AttemptResult.Type,)
    schema = "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest"
    examples = (
        failed(false) = "ApplicationFailure nonRetryable",
        failed(true) = "ApplicationFailure retryable",
    )
end

@action control begin
    party = caller
    on = activity
    input = (control = Control.T,)
    results = Delivery.T
    schema = "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest"
end

# `workerStop` is the `Worker` module's: party worker, no entity. This Model steps on it.

# authoring: observation

# ### The derived observation
#
# The attempt count, read from `DescribeActivityExecution`. A retryable failure raises it and
# records no status change the product machine could see.

@observation attemptCount begin
    on = activity
    read = attempt
end

# authoring: product

# ### The product machine
#
# What the caller sees through Describe. Every Property written against it is carried to the
# protocol machine by the refinement declared there. Unlike the Nexus product, this one sees a
# retry: a retryable failure puts the activity back to SCHEDULED in Describe, with a higher attempt
# count, so `attemptResult(failed(true))` is a visible row here rather than an empty one.

@enumx ProductPhase begin
    scheduled
    started
    paused
    cancelRequested
    completed
    failed
    canceled
    terminated
    timedOut
end

Base.@kwdef struct ProductState
    phase::ProductPhase.T
end

@enumx ProductOutcome accepted notFound

@enumx ProductFact begin
    statusScheduled
    statusStarted
    statusPaused
    statusCancelRequested
    statusCompleted
    statusFailed
    statusCanceled
    statusTerminated
    statusTimedOut
end

const PStep = Step{ProductState,ProductOutcome.T,ProductFact.T}

productStep(phase::ProductPhase.T, recorded::ProductFact.T) =
    [PStep(outcome = ProductOutcome.accepted, state = ProductState(phase = phase), facts = [recorded])]

"""The five phases the product machine ends on."""
productTerminal(state::ProductState) = state.phase in (ProductPhase.completed, ProductPhase.failed,
    ProductPhase.canceled, ProductPhase.terminated, ProductPhase.timedOut)

"""The worker's poll receives the task. Only a scheduled activity is dispatched."""
function attemptStartStep(state::ProductState)::Vector{PStep}
    state.phase == ProductPhase.scheduled || return PStep[]
    return productStep(ProductPhase.started, ProductFact.statusStarted)
end

"""
The worker reports the attempt. A retryable failure is visible: from `started` the activity reads as
scheduled again (`TransitionRescheduled`); while a cancel is requested it settles as canceled, since
the retry is not taken. A canceled result is honored only where a cancel was requested.
"""
function attemptResultStep(state::ProductState, result::AttemptResult.Type)::Vector{PStep}
    state.phase in (ProductPhase.started, ProductPhase.cancelRequested) || return PStep[]
    return @match result begin
        AttemptResult.completed     => productStep(ProductPhase.completed, ProductFact.statusCompleted)
        AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
        AttemptResult.failed(true)  => state.phase == ProductPhase.started ?
            productStep(ProductPhase.scheduled, ProductFact.statusScheduled) :
            productStep(ProductPhase.canceled, ProductFact.statusCanceled)
        AttemptResult.canceled      => state.phase == ProductPhase.cancelRequested ?
            productStep(ProductPhase.canceled, ProductFact.statusCanceled) : PStep[]
    end
end

"""
The caller's control call. After the activity is over, every control is not found and changes
nothing; a repeated cancel request is idempotent and re-reads the same status.
"""
function controlStep(state::ProductState, control::Control.T)::Vector{PStep}
    productTerminal(state) && return [PStep(outcome = ProductOutcome.notFound, state = state)]
    p = state.phase
    return @match control begin
        Control.pause => p in (ProductPhase.scheduled, ProductPhase.started) ?
            productStep(ProductPhase.paused, ProductFact.statusPaused) : PStep[]
        Control.unpause => p == ProductPhase.paused ?
            productStep(ProductPhase.scheduled, ProductFact.statusScheduled) : PStep[]
        Control.requestCancel => p in (ProductPhase.scheduled, ProductPhase.started, ProductPhase.paused,
                                       ProductPhase.cancelRequested) ?
            productStep(ProductPhase.cancelRequested, ProductFact.statusCancelRequested) : PStep[]
        Control.terminate => productStep(ProductPhase.terminated, ProductFact.statusTerminated)
    end
end

"""The worker stopping is invisible to the product, as in the Nexus Model."""
workerStopStep(::ProductState)::Vector{PStep} = PStep[]

"""One of the activity's deadlines firing, while it is not over."""
function timeoutStep(state::ProductState)::Vector{PStep}
    state.phase in (ProductPhase.scheduled, ProductPhase.started, ProductPhase.cancelRequested,
                    ProductPhase.paused) || return PStep[]
    return productStep(ProductPhase.timedOut, ProductFact.statusTimedOut)
end

@machine activityProduct begin
    var"for" = activity
    state = ProductState
    outcome = ProductOutcome.T
    fact = ProductFact.T
    starts = [scheduled]
    ends = [completed, failed, canceled, terminated, timedOut]
    timers = [timeout]
    evidence = (
        statusScheduled = statusScheduled,
        statusStarted = statusStarted,
        statusPaused = statusPaused,
        statusCancelRequested = statusCancelRequested,
        statusCompleted = statusCompleted,
        statusFailed = statusFailed,
        statusCanceled = statusCanceled,
        statusTerminated = statusTerminated,
        statusTimedOut = statusTimedOut,
    )
    steps = (
        attemptStart = attemptStartStep,
        attemptResult = attemptResultStep,
        control = controlStep,
        workerStop = workerStopStep,
        timeout = timeoutStep,
    )
end

# authoring: protocol

# ### The protocol machine
#
# How the server gets there: the retry and backoff the product cannot see, the pause requested of a
# running attempt, the three timers the start request sets, and the attempt count. The machine
# begins before the activity exists, so `unstarted` is the member the start request leaves.

@enumx Phase begin
    unstarted
    scheduled
    backingOff
    started
    paused
    pauseRequested
    cancelRequested
    completed
    failed
    canceled
    terminated
    timedOut
end

@enumx TimeoutType scheduleToClose scheduleToStart startToClose

const attemptBound = 2

Base.@kwdef struct ProtocolState
    phase::Phase.T
    attempts::Fin{attemptBound}
    scheduleToClose::Timeout.T
    scheduleToStart::Timeout.T
    startToClose::Timeout.T
end

@enumx ProtocolOutcome accepted notFound

# The product's untyped `statusTimedOut` is replaced by one that says which deadline, and the
# attempt-count observation is added.
@data ProtocolFact begin
    statusScheduled
    statusStarted
    statusPaused
    statusCancelRequested
    statusCompleted
    statusFailed
    statusCanceled
    statusTerminated
    statusTimedOut(timeoutType::TimeoutType.T)
    attemptCount
end

const TStep = Step{ProtocolState,ProtocolOutcome.T,ProtocolFact.Type}

"""The five phases the design ends on."""
terminalPhase(phase::Phase.T) =
    phase in (Phase.completed, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)

"""Started and not yet over: every phase the schedule-to-close deadline covers."""
running(phase::Phase.T) = phase in (Phase.scheduled, Phase.backingOff, Phase.started, Phase.paused,
                                    Phase.pauseRequested, Phase.cancelRequested)

moves(state::ProtocolState, phase::Phase.T, recorded) =
    [TStep(outcome = ProtocolOutcome.accepted, state = with(state; phase), facts = recorded)]

"""The caller's start request. Sets the three deadlines; the activity is scheduled at attempt zero."""
function startStep(state::ProtocolState, scheduleToClose::Timeout.T, scheduleToStart::Timeout.T,
                   startToClose::Timeout.T)::Vector{TStep}
    state.phase == Phase.unstarted || return TStep[]
    return [TStep(outcome = ProtocolOutcome.accepted,
                  state = ProtocolState(; phase = Phase.scheduled, attempts = 0,
                                        scheduleToClose, scheduleToStart, startToClose),
                  facts = [ProtocolFact.statusScheduled])]
end

"""The poll dispatches a scheduled activity; the attempt count is what the dispatch raises."""
function protocolAttemptStartStep(state::ProtocolState)::Vector{TStep}
    state.phase == Phase.scheduled || return TStep[]
    return [TStep(outcome = ProtocolOutcome.accepted,
                  state = with(state; phase = Phase.started, attempts = saturatingSucc(state.attempts)),
                  facts = [ProtocolFact.statusStarted, ProtocolFact.attemptCount])]
end

"""
The worker reports the attempt. From `started`, a retryable failure backs off. From
`cancelRequested`, a retryable failure does not retry: `statemachine.go` names CANCEL_REQUESTED as
a source of Canceled, so it settles as canceled. From `pauseRequested`, a retryable failure lands in
`paused` (`TransitionAttemptFailedWhilePauseRequested`). A canceled result is honored only where a
cancel was requested.
"""
function protocolAttemptResultStep(state::ProtocolState, result::AttemptResult.Type)::Vector{TStep}
    p = state.phase
    p in (Phase.started, Phase.cancelRequested, Phase.pauseRequested) || return TStep[]
    return @match result begin
        AttemptResult.completed     => moves(state, Phase.completed, [ProtocolFact.statusCompleted])
        AttemptResult.failed(false) => moves(state, Phase.failed, [ProtocolFact.statusFailed])
        AttemptResult.failed(true)  => @match p begin
            Phase.started         => moves(state, Phase.backingOff, [ProtocolFact.attemptCount])
            Phase.cancelRequested => moves(state, Phase.canceled, [ProtocolFact.statusCanceled])
            Phase.pauseRequested  => moves(state, Phase.paused, [ProtocolFact.statusPaused])
        end
        AttemptResult.canceled => p == Phase.cancelRequested ?
            moves(state, Phase.canceled, [ProtocolFact.statusCanceled]) : TStep[]
    end
end

"""
The caller's control call. A pause of a running attempt is only requested (the Describe status
reads PAUSE_REQUESTED; the fact is `statusPaused` for both, as the product cannot tell them apart).
Cancel and terminate of an activity that is over are not found; nothing is enabled before a start.
"""
function protocolControlStep(state::ProtocolState, control::Control.T)::Vector{TStep}
    p = state.phase
    notFound = [TStep(outcome = ProtocolOutcome.notFound, state = state)]
    return @match control begin
        Control.pause => @match p begin
            Phase.scheduled || Phase.backingOff => moves(state, Phase.paused, [ProtocolFact.statusPaused])
            Phase.started                       => moves(state, Phase.pauseRequested, [ProtocolFact.statusPaused])
            _                                   => TStep[]
        end
        Control.unpause => @match p begin
            Phase.paused         => moves(state, Phase.scheduled, [ProtocolFact.statusScheduled])
            Phase.pauseRequested => moves(state, Phase.started, [ProtocolFact.statusStarted])
            _                    => TStep[]
        end
        Control.requestCancel =>
            terminalPhase(p)          ? notFound :
            p == Phase.unstarted      ? TStep[] :
            moves(state, Phase.cancelRequested, [ProtocolFact.statusCancelRequested])
        Control.terminate =>
            terminalPhase(p)          ? notFound :
            p == Phase.unstarted      ? TStep[] :
            moves(state, Phase.terminated, [ProtocolFact.statusTerminated])
    end
end

"""The worker stopping: a stutter the Run records and the activity does not feel."""
protocolWorkerStopStep(state::ProtocolState)::Vector{TStep} =
    [TStep(outcome = ProtocolOutcome.accepted, state = state)]

"""The backoff timer returns a backed-off activity to the queue and records nothing."""
function backoffStep(state::ProtocolState)::Vector{TStep}
    state.phase == Phase.backingOff || return TStep[]
    return moves(state, Phase.scheduled, ProtocolFact.Type[])
end

"""The schedule-to-close deadline covers the whole activity, paused or not."""
function scheduleToCloseStep(state::ProtocolState)::Vector{TStep}
    running(state.phase) && state.scheduleToClose == Timeout.expires || return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose)])
end

"""The schedule-to-start deadline covers the wait for a poll, so it stops at the dispatch."""
function scheduleToStartStep(state::ProtocolState)::Vector{TStep}
    state.phase in (Phase.scheduled, Phase.backingOff) && state.scheduleToStart == Timeout.expires ||
        return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart)])
end

"""The start-to-close deadline covers one attempt, whatever the caller asked of it meanwhile."""
function startToCloseStep(state::ProtocolState)::Vector{TStep}
    state.phase in (Phase.started, Phase.pauseRequested, Phase.cancelRequested) &&
        state.startToClose == Timeout.expires || return TStep[]
    return moves(state, Phase.timedOut, [ProtocolFact.statusTimedOut(TimeoutType.startToClose)])
end

"""
How a protocol state reads as a product state: before a start and while backing off it reads as
scheduled; a requested pause reads as started, because the worker still holds the attempt, so every
answer it can give is a product row from started and the request itself is a stutter; every other
phase is the one of the same name.
"""
function productOf(state::ProtocolState)::ProductState
    phase = @match state.phase begin
        Phase.unstarted || Phase.scheduled || Phase.backingOff => ProductPhase.scheduled
        Phase.started || Phase.pauseRequested => ProductPhase.started
        Phase.paused          => ProductPhase.paused
        Phase.cancelRequested => ProductPhase.cancelRequested
        Phase.completed       => ProductPhase.completed
        Phase.failed          => ProductPhase.failed
        Phase.canceled        => ProductPhase.canceled
        Phase.terminated      => ProductPhase.terminated
        Phase.timedOut        => ProductPhase.timedOut
    end
    return ProductState(phase = phase)
end

@machine activityProtocol begin
    var"for" = activity
    state = ProtocolState
    outcome = ProtocolOutcome.T
    fact = ProtocolFact.Type
    refines = activityProduct
    map = productOf
    starts = [unstarted]
    ends = [completed, failed, canceled, terminated, timedOut]
    timers = [backoff, scheduleToClose, scheduleToStart, startToClose]
    unobservable = [backoff]
    evidence = (
        statusScheduled = statusScheduled,
        statusStarted = statusStarted,
        statusPaused = statusPaused,
        statusCancelRequested = statusCancelRequested,
        statusCompleted = statusCompleted,
        statusFailed = statusFailed,
        statusCanceled = statusCanceled,
        statusTerminated = statusTerminated,
        statusTimedOut = statusTimedOut,
        attemptCount = attemptCount,
    )
    steps = (
        start = startStep,
        attemptStart = protocolAttemptStartStep,
        attemptResult = protocolAttemptResultStep,
        control = protocolControlStep,
        workerStop = protocolWorkerStopStep,
        backoff = backoffStep,
        scheduleToClose = scheduleToCloseStep,
        scheduleToStart = scheduleToStartStep,
        startToClose = startToCloseStep,
    )
end

# authoring: properties

# ### What the machines promise

# Once an activity is over, no step changes its phase.
@property terminalIsFinal begin
    machine = activityProduct
    holds = (before, after) ->
        !productTerminal(before.state) || after.state.phase == before.state.phase
end

# A completed attempt settles the activity as completed, and the status read records it.
@property completes begin
    machine = activityProtocol
    when = attemptResult(completed)
    holds = step -> step.state.phase == Phase.completed && ProtocolFact.statusCompleted in step.facts
end

# A non-retryable failure settles the activity as failed.
@property nonRetryableFails begin
    machine = activityProtocol
    when = attemptResult(failed(false))
    holds = step -> step.state.phase == Phase.failed && ProtocolFact.statusFailed in step.facts
end

# Completed on the second attempt of an activity with no deadline set: two dispatches, so two.
const completedOnRetry = ProtocolState(phase = Phase.completed, attempts = 2,
    scheduleToClose = Timeout.unset, scheduleToStart = Timeout.unset, startToClose = Timeout.unset)

@property retryCompletes begin
    machine = activityProtocol
    when = attemptResult(completed)
    holds = step -> step.state == completedOnRetry && ProtocolFact.statusCompleted in step.facts
end

# A cancel request of a running attempt is recorded and waits for the worker.
@property cancelRequestedWhileStarted begin
    machine = activityProtocol
    when = control(requestCancel)
    holds = step -> step.state.phase == Phase.cancelRequested &&
        ProtocolFact.statusCancelRequested in step.facts
end

# The worker honors the cancel request, and the status read records it.
@property canceledByWorker begin
    machine = activityProtocol
    when = attemptResult(canceled)
    holds = step -> step.state.phase == Phase.canceled && ProtocolFact.statusCanceled in step.facts
end

# A terminate settles the activity as terminated whatever it was doing.
@property terminated begin
    machine = activityProtocol
    when = control(terminate)
    holds = step -> step.state.phase == Phase.terminated && ProtocolFact.statusTerminated in step.facts
end

# A paused activity is not dispatched: no step takes it from paused straight to started.
@property pausedIsNotDispatched begin
    machine = activityProduct
    holds = (before, after) ->
        before.state.phase != ProductPhase.paused || after.state.phase != ProductPhase.started
end

@property scheduleToStartFires begin
    machine = activityProtocol
    when = scheduleToStart
    holds = step -> step.state.phase == Phase.timedOut &&
        ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart) in step.facts
end

@property startToCloseFires begin
    machine = activityProtocol
    when = startToClose
    holds = step -> step.state.phase == Phase.timedOut &&
        ProtocolFact.statusTimedOut(TimeoutType.startToClose) in step.facts
end

# authoring: scenarios

# ### The paths the Queries run
#
# Each is one upstream functional test's shape: the start request with no deadline set unless the
# path is about one, then the side effects that settle the activity.

@scenario completed begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), attemptStart, attemptResult(completed)]
end

@scenario nonRetryable begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), attemptStart, attemptResult(failed(false))]
end

# The retryable failure backs the activity off; the backoff timer returns it to the queue and
# records nothing; the second dispatch completes.
@scenario retriedThenCompleted begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), attemptStart, attemptResult(failed(true)), backoff,
               attemptStart, attemptResult(completed)]
end

@scenario cancelRequestedThenCanceled begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), attemptStart, control(requestCancel), attemptResult(canceled)]
end

# The worker stops, so nothing is dispatched; the caller terminates the scheduled activity.
@scenario terminatedWhileScheduled begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), workerStop, control(terminate)]
end

@scenario pausedThenCompleted begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), control(pause), control(unpause), attemptStart,
               attemptResult(completed)]
end

@scenario scheduleToStartExpires begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, expires, unset), workerStop, scheduleToStart]
end

@scenario startToCloseExpires begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, expires), attemptStart, startToClose]
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

@limits six begin
    steps = 6
    actions = 6
    search = 262144
end

# authoring: queries

# ### The Queries

@query completion begin
    find = completes
    var"in" = completed
    limits = three
end

@query nonRetryableFailure begin
    find = nonRetryableFails
    var"in" = nonRetryable
    limits = three
end

@query retry begin
    find = retryCompletes
    var"in" = retriedThenCompleted
    limits = six
end

@query cancel begin
    find = canceledByWorker
    var"in" = cancelRequestedThenCanceled
    limits = four
end

@query terminate begin
    find = terminated
    var"in" = terminatedWhileScheduled
    limits = three
end

@query pauseResume begin
    find = completes
    var"in" = pausedThenCompleted
    limits = six
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
    var"in" = completed
    limits = three
end

@query pauseHolds begin
    verify = pausedIsNotDispatched
    var"in" = pausedThenCompleted
    limits = six
end

# authoring: set

# ### The functional set
#
# Two parties, both driven. No `repeat`: standalone activities are CHASM only.

@set standaloneActivityTests begin
    purpose = functional
    bind = (
        caller = driven,
        worker = driven,
    )
    queries = [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
               scheduleToStartTimeout, startToCloseTimeout]
end

# ### The canary set
#
# The worker is `observed`: the deployment's own worker attempts, and the verifier checks the
# machine allows what it reported.

@set standaloneActivityCanary begin
    purpose = canary
    bind = (
        caller = driven,
        worker = observed,
    )
    queries = [completion, cancel]
end

# ### The exploratory set

@set standaloneActivityExploration begin
    purpose = exploratory
    bind = (
        caller = driven,
        worker = driven,
    )
    machine = activityProtocol
    cover = rows | results | classMembers
    budget = four
end

# authoring: composition

# ### The activity and its worker
#
# Composed with the worker of the task queue, a dispatch is the worker serving, so an attempt has a
# row only while the worker polls.

@machine activityWorker begin
    from = Worker.polling
    restrict = [workerStop, serve]
end

Base.@kwdef struct StandaloneActivityState
    activity::ProtocolState
    worker::Worker.WorkerState
end

@compose standaloneActivity begin
    var"for" = [activity, Worker.worker]
    state = StandaloneActivityState
    members = (
        activity = activityProtocol,
        worker = activityWorker,
    )
    sync = (
        workerStop = activity.workerStop ∥ worker.workerStop,
        attemptStart = activity.attemptStart ∥ worker.serve,
    )
    starts = [activity.unstarted, worker.polling]
    ends = [activity.completed, activity.failed, activity.canceled, activity.terminated,
            activity.timedOut]
end

# Every dispatch leaves the worker polling: a stopped worker starts nothing.
@property startedByPollingWorker begin
    machine = standaloneActivity
    when = attemptStart
    holds = step -> step.state.worker.phase == Worker.Phase.polling
end

# The start request sets the schedule-to-start deadline; the worker serves one attempt, which fails
# retryably; the worker stops during the backoff, so the retry is never dispatched; the deadline
# fires. The path performs an `attemptStart`, so the claim below is exercised rather than vacuous.
@scenario stoppedBeforeRetry begin
    model = standaloneActivity
    starts = activity.unstarted
    actions = [activity.start(unset, expires, unset), attemptStart, activity.attemptResult(failed(true)),
               activity.backoff, workerStop, activity.scheduleToStart]
end

@query stoppedWorkerStartsNothing begin
    verify = startedByPollingWorker
    var"in" = stoppedBeforeRetry
    limits = six
end

# authoring: end

end # module StandaloneActivity
