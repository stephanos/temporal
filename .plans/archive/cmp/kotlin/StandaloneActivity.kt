/*
 * # The standalone activity Model
 *
 * A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded in
 * `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred (like
 * cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled. Standalone
 * activities write no history events, so every fact is a status read through
 * `DescribeActivityExecution` or a result read through `PollActivityExecution`.
 *
 * Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.
 */
package temporal.feature.activity

import temporal.feature.nexus.caller.Timeout
import temporal.feature.worker.Worker
import umpire.Cover
import umpire.Finite
import umpire.Purpose
import umpire.Step
import umpire.action
import umpire.caller
import umpire.compose
import umpire.driven
import umpire.entity
import umpire.limits
import umpire.machine
import umpire.observation
import umpire.observed
import umpire.property
import umpire.query
import umpire.read
import umpire.restrict
import umpire.scenario
import umpire.set
import umpire.status
import umpire.timer
import umpire.worker

// ---- Entities -------------------------------------------------------------------------------
//
// An activity is named by its id; there is no workflow whose history would name it.

val activity = entity("activity") { key = "activityId" }

// ---- The input domains --------------------------------------------------------------------------
//
// `Timeout` is the Nexus Model's, imported. `Failed(retryable)` is one constructor and two classes,
// the granularity the two `ApplicationFailure` examples are written at.

sealed interface AttemptResult {
    data object Completed : AttemptResult
    data class Failed(val retryable: Boolean) : AttemptResult
    data object Canceled : AttemptResult
}

enum class Delivery { Accepted, NotFound }

enum class Control { Pause, Unpause, RequestCancel, Terminate }

// ---- Actions --------------------------------------------------------------------------------------
//
// The caller starts and controls the activity; the worker's poll receives the task and its response
// resolves the attempt. The worker stopping is a fault the Run records against no entity.

val start = action<Timeout, Timeout, Timeout>("start") {
    party = caller
    creates = activity
    schema = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    input("scheduleToClose", "scheduleToStart", "startToClose")
}

/** The worker's poll receives the task (`PollActivityTaskQueue`). */
val attemptStart = action("attemptStart") {
    party = worker
    on = activity
    schema = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"
}

val attemptResult = action<AttemptResult>("attemptResult") {
    party = worker
    on = activity
    schema = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest | " +
        "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest | " +
        "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest"
    input("result")
    examples {
        AttemptResult.Failed(retryable = false) realizedAs "ApplicationFailure nonRetryable"
        AttemptResult.Failed(retryable = true) realizedAs "ApplicationFailure retryable"
    }
}

val control = action<Control>("control") {
    party = caller
    on = activity
    schema = "temporal.api.workflowservice.v1.PauseActivityExecutionRequest | " +
        "temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest | " +
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest | " +
        "temporal.api.workflowservice.v1.TerminateActivityExecutionRequest"
    input("control")
    results<Delivery>()
}

val workerStop = action("workerStop") { party = worker }

// ---- The derived observation ----------------------------------------------------------------------
//
// A retryable attempt failure changes no status the caller can name, so the attempt number is read
// back through `DescribeActivityExecution`.

val attemptCount = observation("attemptCount") {
    on = activity
    read = "attempt"
}

// ---- The product machine --------------------------------------------------------------------------
//
// What the caller sees through Describe, with no account of how: no backoff, and no distinction between
// an attempt running and one whose pause is requested, since the worker still holds both.

enum class ProductPhase {
    Scheduled, Started, Paused, CancelRequested, Completed, Failed, Canceled, Terminated, TimedOut,
}

data class ProductState(val phase: ProductPhase)

enum class ProductOutcome { Accepted, NotFound }

enum class ProductFact {
    StatusScheduled,
    StatusStarted,
    StatusPaused,
    StatusCancelRequested,
    StatusCompleted,
    StatusFailed,
    StatusCanceled,
    StatusTerminated,
    StatusTimedOut,
}

typealias ProductSteps = List<Step<ProductState, ProductOutcome, ProductFact>>

private fun productStep(phase: ProductPhase, recorded: ProductFact): ProductSteps =
    listOf(Step(ProductOutcome.Accepted, ProductState(phase), listOf(recorded)))

/** The five phases the product machine ends on. */
fun productTerminal(state: ProductState): Boolean = when (state.phase) {
    ProductPhase.Completed, ProductPhase.Failed, ProductPhase.Canceled, ProductPhase.Terminated, ProductPhase.TimedOut -> true
    ProductPhase.Scheduled, ProductPhase.Started, ProductPhase.Paused, ProductPhase.CancelRequested -> false
}

/** A poll picks up a scheduled activity; nothing else has a task to hand out. */
fun attemptStartStep(state: ProductState): ProductSteps =
    if (state.phase == ProductPhase.Scheduled) productStep(ProductPhase.Started, ProductFact.StatusStarted)
    else emptyList()

/**
 * The worker's response to the attempt it holds. Unlike the Nexus Model, the retry is visible here:
 * Describe reads SCHEDULED again after a retryable failure (`TransitionRescheduled`), so the product
 * returns the activity to scheduled; a retryable failure while a cancel is requested lands in canceled.
 * What the product still cannot see is the backoff in between. A canceled response is honored only
 * when a cancel was requested.
 */
fun attemptResultStep(state: ProductState, result: AttemptResult): ProductSteps {
    if (state.phase != ProductPhase.Started && state.phase != ProductPhase.CancelRequested) return emptyList()
    return when (result) {
        AttemptResult.Completed -> productStep(ProductPhase.Completed, ProductFact.StatusCompleted)
        is AttemptResult.Failed -> when {
            !result.retryable -> productStep(ProductPhase.Failed, ProductFact.StatusFailed)
            state.phase == ProductPhase.CancelRequested -> productStep(ProductPhase.Canceled, ProductFact.StatusCanceled)
            else -> productStep(ProductPhase.Scheduled, ProductFact.StatusScheduled)
        }
        AttemptResult.Canceled ->
            if (state.phase == ProductPhase.CancelRequested) productStep(ProductPhase.Canceled, ProductFact.StatusCanceled)
            else emptyList()
    }
}

/**
 * The caller's control requests. Any of them against an activity that is over is not found and changes
 * nothing; a repeated cancel request is idempotent and reads the same status again.
 */
fun controlStep(state: ProductState, control: Control): ProductSteps {
    if (productTerminal(state)) return listOf(Step(ProductOutcome.NotFound, state, emptyList()))
    return when (control) {
        Control.Pause -> when (state.phase) {
            ProductPhase.Scheduled, ProductPhase.Started -> productStep(ProductPhase.Paused, ProductFact.StatusPaused)
            else -> emptyList()
        }
        Control.Unpause ->
            if (state.phase == ProductPhase.Paused) productStep(ProductPhase.Scheduled, ProductFact.StatusScheduled)
            else emptyList()
        Control.RequestCancel -> when (state.phase) {
            ProductPhase.Scheduled, ProductPhase.Started, ProductPhase.Paused, ProductPhase.CancelRequested ->
                productStep(ProductPhase.CancelRequested, ProductFact.StatusCancelRequested)
            else -> emptyList()
        }
        Control.Terminate -> productStep(ProductPhase.Terminated, ProductFact.StatusTerminated)
    }
}

/** The worker stopping is invisible to the product machine, as in the Nexus Model. */
fun workerStopStep(@Suppress("UNUSED_PARAMETER") state: ProductState): ProductSteps = emptyList()

/** Which deadline is the protocol's account of how, so the product machine has one timer. */
val timeout = timer("timeout")

/** One of the activity's deadlines firing, in any phase that is neither over nor unseen. */
fun timeoutStep(state: ProductState): ProductSteps = when (state.phase) {
    ProductPhase.Scheduled, ProductPhase.Started, ProductPhase.CancelRequested, ProductPhase.Paused ->
        productStep(ProductPhase.TimedOut, ProductFact.StatusTimedOut)
    else -> emptyList()
}

val activityProduct = machine<ProductState, ProductOutcome, ProductFact>("activityProduct") {
    entity = activity
    starts(ProductState(ProductPhase.Scheduled))
    ends { productTerminal(this) }
    timers(timeout)
    // Every fact is a status value of `DescribeActivityExecution`, resolved against the realization's
    // catalog like a history event would be: there is no history to name an event in.
    evidence { fact ->
        when (fact) {
            ProductFact.StatusScheduled -> status("statusScheduled")
            ProductFact.StatusStarted -> status("statusStarted")
            ProductFact.StatusPaused -> status("statusPaused")
            ProductFact.StatusCancelRequested -> status("statusCancelRequested")
            ProductFact.StatusCompleted -> status("statusCompleted")
            ProductFact.StatusFailed -> status("statusFailed")
            ProductFact.StatusCanceled -> status("statusCanceled")
            ProductFact.StatusTerminated -> status("statusTerminated")
            ProductFact.StatusTimedOut -> status("statusTimedOut")
        }
    }
    steps {
        attemptStart runs ::attemptStartStep
        attemptResult runs ::attemptResultStep
        control runs ::controlStep
        workerStop runs ::workerStopStep
        timeout runs ::timeoutStep
    }
}

// ---- The protocol machine -------------------------------------------------------------------------
//
// How the server gets there: the backoff the product cannot see, the three timers the start request sets,
// the attempt count a dispatch raises, and the `PauseRequested` phase Describe reports as paused while
// the worker still holds the attempt. The machine begins before the activity exists, so `Unstarted` is
// the phase the start request leaves and what makes the deadline fields reachable.

enum class Phase {
    Unstarted, Scheduled, BackingOff, Started, Paused, PauseRequested, CancelRequested,
    Completed, Failed, Canceled, Terminated, TimedOut,
}

/** Which timer fired; the timed-out status carries it, so a Contract reads it. */
enum class TimeoutType { ScheduleToClose, ScheduleToStart, StartToClose }

const val attemptBound = 2

/** `0..attemptBound` with a saturating successor; declared here as the Lean file declares its own. */
@JvmInline
value class Attempts(val count: Int) {
    init {
        require(count in 0..attemptBound) { "attempts $count outside 0..$attemptBound" }
    }

    fun saturatingSucc() = Attempts(minOf(count + 1, attemptBound))

    companion object : Finite<Attempts> {
        override val values = (0..attemptBound).map(::Attempts)
    }
}

data class ProtocolState(
    val phase: Phase,
    val attempts: Attempts = Attempts(0),
    val scheduleToClose: Timeout = Timeout.Unset,
    val scheduleToStart: Timeout = Timeout.Unset,
    val startToClose: Timeout = Timeout.Unset,
)

enum class ProtocolOutcome { Accepted, NotFound }

/** As the product's, with the timed-out status carrying which deadline, plus the attempt count read. */
sealed interface ProtocolFact {
    data object StatusScheduled : ProtocolFact
    data object StatusStarted : ProtocolFact
    data object StatusPaused : ProtocolFact
    data object StatusCancelRequested : ProtocolFact
    data object StatusCompleted : ProtocolFact
    data object StatusFailed : ProtocolFact
    data object StatusCanceled : ProtocolFact
    data object StatusTerminated : ProtocolFact
    data class StatusTimedOut(val timeoutType: TimeoutType) : ProtocolFact
    data object AttemptCount : ProtocolFact
}

typealias ProtocolSteps = List<Step<ProtocolState, ProtocolOutcome, ProtocolFact>>

/** The five phases the design ends on. A control request after one of them is not found. */
fun terminalPhase(phase: Phase): Boolean = when (phase) {
    Phase.Completed, Phase.Failed, Phase.Canceled, Phase.Terminated, Phase.TimedOut -> true
    Phase.Unstarted, Phase.Scheduled, Phase.BackingOff, Phase.Started, Phase.Paused,
    Phase.PauseRequested, Phase.CancelRequested,
    -> false
}

/** Started and not yet over: the phases the schedule-to-close deadline covers. */
fun running(phase: Phase): Boolean = when (phase) {
    Phase.Scheduled, Phase.BackingOff, Phase.Started, Phase.Paused, Phase.PauseRequested, Phase.CancelRequested -> true
    else -> false
}

private fun moves(state: ProtocolState, phase: Phase, recorded: List<ProtocolFact>): ProtocolSteps =
    listOf(Step(ProtocolOutcome.Accepted, state.copy(phase = phase), recorded))

private fun notFound(state: ProtocolState): ProtocolSteps =
    listOf(Step(ProtocolOutcome.NotFound, state, emptyList()))

/** The caller's start request names the three deadlines, each a state field for the same reason as in the Nexus Model. */
fun startStep(
    state: ProtocolState,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
): ProtocolSteps {
    if (state.phase != Phase.Unstarted) return emptyList()
    return listOf(
        Step(
            ProtocolOutcome.Accepted,
            ProtocolState(Phase.Scheduled, Attempts(0), scheduleToClose, scheduleToStart, startToClose),
            listOf(ProtocolFact.StatusScheduled),
        ),
    )
}

/** A dispatch is an attempt: the count rises with it and is read back, since no status names it. */
fun protocolAttemptStartStep(state: ProtocolState): ProtocolSteps {
    if (state.phase != Phase.Scheduled) return emptyList()
    return listOf(
        Step(
            ProtocolOutcome.Accepted,
            state.copy(phase = Phase.Started, attempts = state.attempts.saturatingSucc()),
            listOf(ProtocolFact.StatusStarted, ProtocolFact.AttemptCount),
        ),
    )
}

/**
 * The worker's response, by the phase the attempt is in. From `Started`, a retryable failure backs the
 * activity off and a canceled response is refused, since no cancel was requested. From
 * `CancelRequested`, statemachine.go makes it a source of Completed, Failed, Canceled, TimedOut and
 * Terminated, and a retryable failure lands in Canceled rather than rescheduling. From `PauseRequested`,
 * a retryable failure lands in Paused (`TransitionAttemptFailedWhilePauseRequested`).
 */
fun protocolAttemptResultStep(state: ProtocolState, result: AttemptResult): ProtocolSteps = when (state.phase) {
    Phase.Started -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.BackingOff, listOf(ProtocolFact.AttemptCount))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> emptyList()
    }
    Phase.CancelRequested -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.Canceled, listOf(ProtocolFact.StatusCanceled))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> moves(state, Phase.Canceled, listOf(ProtocolFact.StatusCanceled))
    }
    Phase.PauseRequested -> when (result) {
        AttemptResult.Completed -> moves(state, Phase.Completed, listOf(ProtocolFact.StatusCompleted))
        is AttemptResult.Failed ->
            if (result.retryable) moves(state, Phase.Paused, listOf(ProtocolFact.StatusPaused))
            else moves(state, Phase.Failed, listOf(ProtocolFact.StatusFailed))
        AttemptResult.Canceled -> emptyList()
    }
    else -> emptyList()
}

/**
 * The caller's control requests. A pause of a running attempt is only requested (`TransitionPauseRequested`);
 * Describe reads it as paused, so the fact is `StatusPaused` for both, as the product cannot tell them
 * apart. A cancel request is honored from every live phase and is idempotent; a terminate ends any
 * activity that exists.
 */
fun protocolControlStep(state: ProtocolState, control: Control): ProtocolSteps = when (control) {
    Control.Pause -> when (state.phase) {
        Phase.Scheduled, Phase.BackingOff -> moves(state, Phase.Paused, listOf(ProtocolFact.StatusPaused))
        Phase.Started -> moves(state, Phase.PauseRequested, listOf(ProtocolFact.StatusPaused))
        else -> emptyList()
    }
    Control.Unpause -> when (state.phase) {
        Phase.Paused -> moves(state, Phase.Scheduled, listOf(ProtocolFact.StatusScheduled))
        Phase.PauseRequested -> moves(state, Phase.Started, listOf(ProtocolFact.StatusStarted))
        else -> emptyList()
    }
    Control.RequestCancel -> when {
        terminalPhase(state.phase) -> notFound(state)
        state.phase == Phase.Unstarted -> emptyList()
        else -> moves(state, Phase.CancelRequested, listOf(ProtocolFact.StatusCancelRequested))
    }
    Control.Terminate -> when {
        terminalPhase(state.phase) -> notFound(state)
        state.phase == Phase.Unstarted -> emptyList()
        else -> moves(state, Phase.Terminated, listOf(ProtocolFact.StatusTerminated))
    }
}

/** The worker stopping keeps the state and records nothing; the step after it confirms it. */
fun protocolWorkerStopStep(state: ProtocolState): ProtocolSteps =
    listOf(Step(ProtocolOutcome.Accepted, state, emptyList()))

val backoff = timer("backoff")
val scheduleToClose = timer("scheduleToClose")
val scheduleToStart = timer("scheduleToStart")
val startToClose = timer("startToClose")

/** The backoff timer returns the activity to the queue and records nothing. */
fun backoffStep(state: ProtocolState): ProtocolSteps =
    if (state.phase != Phase.BackingOff) emptyList() else moves(state, Phase.Scheduled, emptyList())

/** The schedule-to-close deadline covers the whole activity, paused phases included. */
fun scheduleToCloseStep(state: ProtocolState): ProtocolSteps =
    if (running(state.phase) && state.scheduleToClose == Timeout.Expires)
        moves(state, Phase.TimedOut, listOf(ProtocolFact.StatusTimedOut(TimeoutType.ScheduleToClose)))
    else emptyList()

/** The schedule-to-start deadline covers the wait for a poll. */
fun scheduleToStartStep(state: ProtocolState): ProtocolSteps =
    if ((state.phase == Phase.Scheduled || state.phase == Phase.BackingOff) && state.scheduleToStart == Timeout.Expires)
        moves(state, Phase.TimedOut, listOf(ProtocolFact.StatusTimedOut(TimeoutType.ScheduleToStart)))
    else emptyList()

/** The start-to-close deadline covers the attempt the worker holds, whatever the caller requested of it. */
fun startToCloseStep(state: ProtocolState): ProtocolSteps = when (state.phase) {
    Phase.Started, Phase.PauseRequested, Phase.CancelRequested ->
        if (state.startToClose == Timeout.Expires)
            moves(state, Phase.TimedOut, listOf(ProtocolFact.StatusTimedOut(TimeoutType.StartToClose)))
        else emptyList()
    else -> emptyList()
}

/**
 * How a protocol state reads as a product state. Not yet started and backing off read as scheduled; a
 * pause requested reads as started, because the worker still holds the attempt and its result lands
 * where a started attempt's would; every other phase by its name.
 */
fun productOf(state: ProtocolState): ProductState = ProductState(
    when (state.phase) {
        Phase.Unstarted, Phase.Scheduled, Phase.BackingOff -> ProductPhase.Scheduled
        Phase.Started, Phase.PauseRequested -> ProductPhase.Started
        Phase.Paused -> ProductPhase.Paused
        Phase.CancelRequested -> ProductPhase.CancelRequested
        Phase.Completed -> ProductPhase.Completed
        Phase.Failed -> ProductPhase.Failed
        Phase.Canceled -> ProductPhase.Canceled
        Phase.Terminated -> ProductPhase.Terminated
        Phase.TimedOut -> ProductPhase.TimedOut
    },
)

val activityProtocol = machine<ProtocolState, ProtocolOutcome, ProtocolFact>("activityProtocol") {
    entity = activity
    refines(activityProduct) via ::productOf
    starts(ProtocolState(Phase.Unstarted))
    ends { terminalPhase(phase) }
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence { fact ->
        when (fact) {
            ProtocolFact.StatusScheduled -> status("statusScheduled")
            ProtocolFact.StatusStarted -> status("statusStarted")
            ProtocolFact.StatusPaused -> status("statusPaused")
            ProtocolFact.StatusCancelRequested -> status("statusCancelRequested")
            ProtocolFact.StatusCompleted -> status("statusCompleted")
            ProtocolFact.StatusFailed -> status("statusFailed")
            ProtocolFact.StatusCanceled -> status("statusCanceled")
            ProtocolFact.StatusTerminated -> status("statusTerminated")
            is ProtocolFact.StatusTimedOut -> status("statusTimedOut")
            ProtocolFact.AttemptCount -> read(attemptCount)
        }
    }
    steps {
        start runs ::startStep
        attemptStart runs ::protocolAttemptStartStep
        attemptResult runs ::protocolAttemptResultStep
        control runs ::protocolControlStep
        workerStop runs ::protocolWorkerStopStep
        backoff runs ::backoffStep
        scheduleToClose runs ::scheduleToCloseStep
        scheduleToStart runs ::scheduleToStartStep
        startToClose runs ::startToCloseStep
    }
}

// ---- What the machines promise --------------------------------------------------------------------

/** Once an activity is over, no step changes its phase. On the product machine, read on the protocol through the map. */
val terminalIsFinal = property("terminalIsFinal", activityProduct) {
    holds { before, after -> !productTerminal(before.state) || after.state.phase == before.state.phase }
}

/** A completed response settles the activity as completed, and Describe reads it. */
val completes = property("completes", activityProtocol) {
    attemptResult(AttemptResult.Completed) holds { step ->
        step.state.phase == Phase.Completed && ProtocolFact.StatusCompleted in step.facts
    }
}

/** A non-retryable failure settles the activity as failed. */
val nonRetryableFails = property("nonRetryableFails", activityProtocol) {
    attemptResult(AttemptResult.Failed(retryable = false)) holds { step ->
        step.state.phase == Phase.Failed && ProtocolFact.StatusFailed in step.facts
    }
}

/** Completed on the second attempt of an activity with no deadline set; a claim fixes one state, so every field is named. */
val completedOnRetry = ProtocolState(
    phase = Phase.Completed,
    attempts = Attempts(2),
    scheduleToClose = Timeout.Unset,
    scheduleToStart = Timeout.Unset,
    startToClose = Timeout.Unset,
)

/** The retried attempt completes: the count two dispatches raised is two. */
val retryCompletes = property("retryCompletes", activityProtocol) {
    attemptResult(AttemptResult.Completed) holds { step ->
        step.state == completedOnRetry && ProtocolFact.StatusCompleted in step.facts
    }
}

/** A cancel request lands in `CancelRequested` and Describe reads it. */
val cancelRequestedWhileStarted = property("cancelRequestedWhileStarted", activityProtocol) {
    control(Control.RequestCancel) holds { step ->
        step.state.phase == Phase.CancelRequested && ProtocolFact.StatusCancelRequested in step.facts
    }
}

/** The worker honoring the cancel settles the activity as canceled. */
val canceledByWorker = property("canceledByWorker", activityProtocol) {
    attemptResult(AttemptResult.Canceled) holds { step ->
        step.state.phase == Phase.Canceled && ProtocolFact.StatusCanceled in step.facts
    }
}

/** A terminate settles the activity as terminated. */
val terminated = property("terminated", activityProtocol) {
    control(Control.Terminate) holds { step ->
        step.state.phase == Phase.Terminated && ProtocolFact.StatusTerminated in step.facts
    }
}

/** A paused activity is not dispatched: no step takes it from paused straight to started. */
val pausedIsNotDispatched = property("pausedIsNotDispatched", activityProduct) {
    holds { before, after -> before.state.phase != ProductPhase.Paused || after.state.phase != ProductPhase.Started }
}

/** The schedule-to-start deadline settles an activity no worker polled as timed out, naming the deadline. */
val scheduleToStartFires = property("scheduleToStartFires", activityProtocol) {
    scheduleToStart holds { step ->
        step.state.phase == Phase.TimedOut && ProtocolFact.StatusTimedOut(TimeoutType.ScheduleToStart) in step.facts
    }
}

/** The start-to-close deadline settles an attempt no worker resolved as timed out. */
val startToCloseFires = property("startToCloseFires", activityProtocol) {
    startToClose holds { step ->
        step.state.phase == Phase.TimedOut && ProtocolFact.StatusTimedOut(TimeoutType.StartToClose) in step.facts
    }
}

// ---- The paths the Queries run --------------------------------------------------------------------
//
// Each starts unstarted with `start(unset, unset, unset)` unless the path is about a deadline.

private val unset = Timeout.Unset
private val expires = Timeout.Expires
private val unstarted = ProtocolState(Phase.Unstarted)

val completed = scenario("completed", activityProtocol) {
    starts(unstarted)
    actions(start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.Completed))
}

val nonRetryable = scenario("nonRetryable", activityProtocol) {
    starts(unstarted)
    actions(start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.Failed(retryable = false)))
}

/** The retryable failure backs the activity off; the backoff fires silently; the second dispatch completes. */
val retriedThenCompleted = scenario("retriedThenCompleted", activityProtocol) {
    starts(unstarted)
    actions(
        start(unset, unset, unset),
        attemptStart,
        attemptResult(AttemptResult.Failed(retryable = true)),
        backoff,
        attemptStart,
        attemptResult(AttemptResult.Completed),
    )
}

val cancelRequestedThenCanceled = scenario("cancelRequestedThenCanceled", activityProtocol) {
    starts(unstarted)
    actions(start(unset, unset, unset), attemptStart, control(Control.RequestCancel), attemptResult(AttemptResult.Canceled))
}

/** The worker stops before it polls, so the terminate lands on a scheduled activity. */
val terminatedWhileScheduled = scenario("terminatedWhileScheduled", activityProtocol) {
    starts(unstarted)
    actions(start(unset, unset, unset), workerStop, control(Control.Terminate))
}

val pausedThenCompleted = scenario("pausedThenCompleted", activityProtocol) {
    starts(unstarted)
    actions(
        start(unset, unset, unset),
        control(Control.Pause),
        control(Control.Unpause),
        attemptStart,
        attemptResult(AttemptResult.Completed),
    )
}

val scheduleToStartExpires = scenario("scheduleToStartExpires", activityProtocol) {
    starts(unstarted)
    actions(start(unset, expires, unset), workerStop, scheduleToStart)
}

val startToCloseExpires = scenario("startToCloseExpires", activityProtocol) {
    starts(unstarted)
    actions(start(unset, unset, expires), attemptStart, startToClose)
}

val three = limits("three", steps = 3, actions = 3, search = 4096)
val four = limits("four", steps = 4, actions = 4, search = 32768)
val six = limits("six", steps = 6, actions = 6, search = 262144)

// ---- The Queries ----------------------------------------------------------------------------------

val completion = query("completion") { find(completes) on completed within three }
val nonRetryableFailure = query("nonRetryableFailure") { find(nonRetryableFails) on nonRetryable within three }
val retry = query("retry") { find(retryCompletes) on retriedThenCompleted within six }
val cancel = query("cancel") { find(canceledByWorker) on cancelRequestedThenCanceled within four }
val terminate = query("terminate") { find(terminated) on terminatedWhileScheduled within three }
val pauseResume = query("pauseResume") { find(completes) on pausedThenCompleted within six }
val scheduleToStartTimeout = query("scheduleToStartTimeout") { find(scheduleToStartFires) on scheduleToStartExpires within three }
val startToCloseTimeout = query("startToCloseTimeout") { find(startToCloseFires) on startToCloseExpires within three }
val terminalHolds = query("terminalHolds") { verify(terminalIsFinal) on completed within three }
val pauseHolds = query("pauseHolds") { verify(pausedIsNotDispatched) on pausedThenCompleted within six }

// ---- The sets -------------------------------------------------------------------------------------
//
// No `repeat`: standalone activities are CHASM only, so there is no implementation switch to run over.

val standaloneActivityTests = set("standaloneActivityTests") {
    purpose = Purpose.Functional
    bind(caller to driven, worker to driven)
    queries(
        completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
        scheduleToStartTimeout, startToCloseTimeout,
    )
}

/** The worker is observed: a deployment performs the activity itself and the verifier reads what it did. */
val standaloneActivityCanary = set("standaloneActivityCanary") {
    purpose = Purpose.Canary
    bind(caller to driven, worker to observed)
    queries(completion, cancel)
}

val standaloneActivityExploration = set("standaloneActivityExploration") {
    purpose = Purpose.Exploratory
    bind(caller to driven, worker to driven)
    machine = activityProtocol
    cover(Cover.Rows, Cover.Results, Cover.ClassMembers)
    budget = four
}

// ---- The activity and its worker ------------------------------------------------------------------
//
// Composed with the worker of the activity's task queue, the stop is the worker's own phase change and
// every dispatch is the worker serving, so an attempt starts only while the worker polls.

val activityWorker = Worker.polling.restrict("activityWorker", Worker.workerStop, Worker.serve)

data class StandaloneActivityState(val activity: ProtocolState, val worker: Worker.WorkerState)

val standaloneActivity = compose<StandaloneActivityState>("standaloneActivity") {
    val activity = member(StandaloneActivityState::activity, activityProtocol)
    val worker = member(StandaloneActivityState::worker, activityWorker)
    workerStop syncs activity[workerStop] with worker[Worker.workerStop]
    attemptStart syncs activity[attemptStart] with worker[Worker.serve]
    starts(StandaloneActivityState(unstarted, Worker.WorkerState(Worker.Phase.Polling)))
    ends { terminalPhase(this.activity.phase) }
}

/** Every dispatch leaves the worker polling: no attempt starts while the worker is stopped. */
val startedByPollingWorker = property("startedByPollingWorker", standaloneActivity) {
    attemptStart holds { step -> step.state.worker.phase == Worker.Phase.Polling }
}

/**
 * The first attempt is dispatched and fails retryably; the worker stops during the backoff, so the
 * retry is never dispatched and the schedule-to-start deadline fires. The path performs `attemptStart`
 * once, so the claim below is exercised on it rather than holding vacuously.
 */
val stoppedBeforeRetry = scenario("stoppedBeforeRetry", standaloneActivity) {
    val activity = standaloneActivity.member(StandaloneActivityState::activity)
    starts(StandaloneActivityState(unstarted, Worker.WorkerState(Worker.Phase.Polling)))
    actions(
        activity[start](unset, expires, unset),
        attemptStart,
        activity[attemptResult](AttemptResult.Failed(retryable = true)),
        activity[backoff],
        workerStop,
        activity[scheduleToStart],
    )
}

val stoppedWorkerStartsNothing = query("stoppedWorkerStartsNothing") {
    verify(startedByPollingWorker) on stoppedBeforeRetry within six
}
