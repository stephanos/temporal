/*
 * # The Nexus caller-side Model
 *
 * One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
 * operation does, the protocol machine says how the server gets there and refines it, and the
 * functional set runs one Query per side effect that settles the operation, once per value of the
 * implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79) and
 * no concurrency-limit setup parameter.
 *
 * Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.
 *
 * Every declaration is a top-level `val`, so a step function naming an action that does not exist is
 * an unresolved reference at compile time. What the builders check when they run (a Property naming
 * an action its machine has no step for, the refinement, the finite table) surfaces the first time
 * this file's class is loaded, which in this sample is from `Pins.kt`.
 */
package temporal.feature.nexus.caller

import temporal.feature.worker.Worker
import umpire.Cover
import umpire.Finite
import umpire.Purpose
import umpire.Repeat
import umpire.Step
import umpire.action
import umpire.caller
import umpire.compose
import umpire.driven
import umpire.entity
import umpire.event
import umpire.handler
import umpire.limits
import umpire.machine
import umpire.network
import umpire.observation
import umpire.observed
import umpire.property
import umpire.query
import umpire.read
import umpire.restrict
import umpire.scenario
import umpire.set
import umpire.timer
import umpire.worker

// ---- Entities -------------------------------------------------------------------------------
//
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled event:
// every history event of the operation carries that event's id.

val workflow = entity("workflow")

val operation = entity("operation") {
    refer("caller", workflow)
    key = "scheduledEvent"
}

// ---- The input domains --------------------------------------------------------------------------
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: `HandlerError(retryable: Boolean)` is one constructor and two classes,
// which is the granularity an example is written at and what mirrors a protobuf oneof. Domains
// without fields are `enum class`; a domain with a field is a `sealed interface`, since Kotlin enum
// entries cannot carry per-entry parameters.

enum class Timeout { Unset, Expires }

sealed interface Reply {
    data object SyncSuccess : Reply
    data object Async : Reply
    data object OperationFailed : Reply
    data object OperationCanceled : Reply
    data class HandlerError(val retryable: Boolean) : Reply
}

enum class Resolution { Succeeded, Failed, Canceled }

enum class Delivery { Accepted, NotFound }

// ---- Actions --------------------------------------------------------------------------------------
//
// Parties are the framework's `caller`, `handler`, `network`, `worker`; the reserved party `system` is
// the server. A fault is an ordinary action of a declared party, and a timer is `system` behavior the
// machine owns, so neither is a separate kind.

val schedule = action<Timeout, Timeout, Timeout>("schedule") {
    party = caller
    creates = operation
    schema = "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"
    input("scheduleToClose", "scheduleToStart", "startToClose")
}

val handlerReply = action<Reply>("handlerReply") {
    party = handler
    on = operation
    schema = "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError"
    input("reply")
    examples {
        Reply.HandlerError(retryable = false) realizedAs "BadRequest"
        Reply.HandlerError(retryable = true) realizedAs "Internal"
    }
}

/**
 * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes are
 * names the realization interprets.
 */
val complete = action<Resolution>("complete") {
    party = handler
    on = operation
    input("resolution")
    results<Delivery>()
}

val transportFault = action("transportFault") {
    party = network
    on = operation
}

/**
 * The handler's worker stops polling. An action that names no entity is behavior no entity records:
 * the Run records the fault, but nothing recorded names the operation, so the machines keep their
 * state and record nothing at it.
 */
val workerStop = action("workerStop") { party = worker }

// ---- The derived observation ----------------------------------------------------------------------
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = observation("pendingAttempts") {
    on = operation
    read = "attempts"
}

// ---- The product machine --------------------------------------------------------------------------
//
// What an operation does, with no account of how. Every Property written against it is carried to the
// protocol machine by the refinement declared there.

enum class ProductPhase { Scheduled, Started, Succeeded, Failed, Canceled, TimedOut }

data class ProductState(val phase: ProductPhase)

enum class ProductOutcome { Accepted, NotFound }

enum class ProductFact {
    NexusOperationScheduled,
    NexusOperationStarted,
    NexusOperationCompleted,
    NexusOperationFailed,
    NexusOperationCanceled,
    NexusOperationTimedOut,
}

typealias ProductSteps = List<Step<ProductState, ProductOutcome, ProductFact>>

private fun productStep(phase: ProductPhase, recorded: ProductFact): ProductSteps =
    listOf(Step(ProductOutcome.Accepted, ProductState(phase), listOf(recorded)))

/**
 * The handler's reply to the server's start request. An operation that has not started yet is the
 * only one a reply can move.
 */
fun handlerReplyStep(state: ProductState, reply: Reply): ProductSteps {
    if (state.phase != ProductPhase.Scheduled) return emptyList()
    return when (reply) {
        Reply.SyncSuccess -> productStep(ProductPhase.Succeeded, ProductFact.NexusOperationCompleted)
        Reply.Async -> productStep(ProductPhase.Started, ProductFact.NexusOperationStarted)
        Reply.OperationFailed -> productStep(ProductPhase.Failed, ProductFact.NexusOperationFailed)
        Reply.OperationCanceled -> productStep(ProductPhase.Canceled, ProductFact.NexusOperationCanceled)
        // A retryable handler error leaves the operation where it is: the product machine does not know
        // about backing off, which is the whole of what the protocol machine adds.
        is Reply.HandlerError ->
            if (reply.retryable) emptyList()
            else productStep(ProductPhase.Failed, ProductFact.NexusOperationFailed)
    }
}

/** The four phases the product machine ends on. */
fun productTerminal(state: ProductState): Boolean = when (state.phase) {
    ProductPhase.Succeeded, ProductPhase.Failed, ProductPhase.Canceled, ProductPhase.TimedOut -> true
    ProductPhase.Scheduled, ProductPhase.Started -> false
}

/**
 * An asynchronous completion. A completion that arrives after the operation is over is not found, and
 * changes nothing.
 */
fun completeStep(state: ProductState, resolution: Resolution): ProductSteps =
    if (productTerminal(state)) {
        listOf(Step(ProductOutcome.NotFound, state, emptyList()))
    } else when (resolution) {
        Resolution.Succeeded -> productStep(ProductPhase.Succeeded, ProductFact.NexusOperationCompleted)
        Resolution.Failed -> productStep(ProductPhase.Failed, ProductFact.NexusOperationFailed)
        Resolution.Canceled -> productStep(ProductPhase.Canceled, ProductFact.NexusOperationCanceled)
    }

/**
 * A transport fault is an ordinary action of the network. The product machine cannot see one: whether
 * a delivery was retried is the protocol's account of how, not what.
 */
fun transportFaultStep(@Suppress("UNUSED_PARAMETER") state: ProductState): ProductSteps = emptyList()

/**
 * The handler's worker stopping is a fault the Run records and the operation does not feel. The
 * product machine cannot see it, like the transport fault: a step that kept the state and recorded
 * nothing would be indistinguishable from a stutter, and the refinement would read every stutter as
 * this step.
 */
fun workerStopStep(@Suppress("UNUSED_PARAMETER") state: ProductState): ProductSteps = emptyList()

/** Which deadline is the protocol's account of how, so the product machine has one timer. */
val timeout = timer("timeout")

/** One of the operation's deadlines firing, while the operation runs. */
fun timeoutStep(state: ProductState): ProductSteps =
    if (state.phase == ProductPhase.Scheduled || state.phase == ProductPhase.Started)
        productStep(ProductPhase.TimedOut, ProductFact.NexusOperationTimedOut)
    else emptyList()

val nexusProduct = machine<ProductState, ProductOutcome, ProductFact>("nexusProduct") {
    entity = operation
    starts(ProductState(ProductPhase.Scheduled))
    ends { productTerminal(this) }
    timers(timeout)
    evidence { fact ->
        when (fact) {
            // No product step records the schedule: the command is the protocol's.
            ProductFact.NexusOperationScheduled -> null
            ProductFact.NexusOperationStarted -> event("nexusOperationStarted")
            ProductFact.NexusOperationCompleted -> event("nexusOperationCompleted")
            ProductFact.NexusOperationFailed -> event("nexusOperationFailed")
            ProductFact.NexusOperationCanceled -> event("nexusOperationCanceled")
            ProductFact.NexusOperationTimedOut -> event("nexusOperationTimedOut")
        }
    }
    steps {
        handlerReply runs ::handlerReplyStep
        complete runs ::completeStep
        transportFault runs ::transportFaultStep
        workerStop runs ::workerStopStep
        timeout runs ::timeoutStep
    }
}

// ---- The protocol machine -------------------------------------------------------------------------
//
// How the server gets there: the retry the product machine cannot see, the three timers the schedule
// command sets, and the attempt count a retryable failure raises. Written against the same actions, so
// a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state class has no "no instance yet" member, so
// `Unstarted` is that member, and it is what makes the three deadline fields reachable at anything but
// their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and the
// concurrency-limit rejection. The limit exists -- one dynamic-config key per implementation, and the
// schedule command fails the workflow task at it without writing a `NexusOperationScheduled` event --
// but a step function does not read the setup, the key and value differ per switch value, and the
// rejection names no operation, so it is not modeled until a Query needs it.

enum class Phase { Unscheduled, Scheduled, BackingOff, Started, Succeeded, Failed, Canceled, TimedOut }

/**
 * Which timer fired. The history event records it, so a Contract that did not check it would pass a
 * run that timed out on the wrong deadline.
 */
enum class TimeoutType { ScheduleToClose, ScheduleToStart, StartToClose }

/**
 * The attempt count is bounded by the Limits in the design; nothing wires the Limits into a machine's
 * state, so the bound is written here and the saturating successor keeps a retry inside it.
 */
const val attemptBound = 2

/** `0..attemptBound`, Kotlin's stand-in for `Fin (attemptBound + 1)`; the companion lists its values. */
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

/** Defaults are the values the operation begins with, so `ProtocolState(Phase.Unscheduled)` is the start. */
data class ProtocolState(
    val phase: Phase,
    val attempts: Attempts = Attempts(0),
    val scheduleToClose: Timeout = Timeout.Unset,
    val scheduleToStart: Timeout = Timeout.Unset,
    val startToClose: Timeout = Timeout.Unset,
)

enum class ProtocolOutcome { Accepted, NotFound }

sealed interface ProtocolFact {
    data object NexusOperationScheduled : ProtocolFact
    data object NexusOperationStarted : ProtocolFact
    data object NexusOperationCompleted : ProtocolFact
    data object NexusOperationFailed : ProtocolFact
    data object NexusOperationCanceled : ProtocolFact
    data class NexusOperationTimedOut(val timeoutType: TimeoutType) : ProtocolFact
    data object PendingAttempts : ProtocolFact
}

typealias ProtocolSteps = List<Step<ProtocolState, ProtocolOutcome, ProtocolFact>>

/** The four phases the design ends on. A completion that arrives after one of them is not found. */
fun terminalPhase(phase: Phase): Boolean = when (phase) {
    Phase.Succeeded, Phase.Failed, Phase.Canceled, Phase.TimedOut -> true
    Phase.Unscheduled, Phase.Scheduled, Phase.BackingOff, Phase.Started -> false
}

/** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
fun running(phase: Phase): Boolean =
    phase == Phase.Scheduled || phase == Phase.BackingOff || phase == Phase.Started

private fun moves(state: ProtocolState, phase: Phase, recorded: List<ProtocolFact>): ProtocolSteps =
    listOf(Step(ProtocolOutcome.Accepted, state.copy(phase = phase), recorded))

/**
 * The caller's schedule command. It names the operation's three deadlines, and every one of them is a
 * state field because whether a timer fires is a question about the operation and not about the
 * command that started it.
 */
fun scheduleStep(
    state: ProtocolState,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
): ProtocolSteps {
    if (state.phase != Phase.Unscheduled) return emptyList()
    return listOf(
        Step(
            ProtocolOutcome.Accepted,
            ProtocolState(Phase.Scheduled, Attempts(0), scheduleToClose, scheduleToStart, startToClose),
            listOf(ProtocolFact.NexusOperationScheduled),
        ),
    )
}

/**
 * The handler's reply to the server's start request. What the product machine cannot see is the last
 * arm: a retryable failure backs the operation off and raises its attempt count, and the count is read
 * back through the `pendingAttempts` observation because no history event records it.
 */
fun protocolHandlerReplyStep(state: ProtocolState, reply: Reply): ProtocolSteps {
    if (state.phase != Phase.Scheduled) return emptyList()
    return when (reply) {
        Reply.SyncSuccess -> moves(state, Phase.Succeeded, listOf(ProtocolFact.NexusOperationCompleted))
        Reply.Async -> moves(state, Phase.Started, listOf(ProtocolFact.NexusOperationStarted))
        Reply.OperationFailed -> moves(state, Phase.Failed, listOf(ProtocolFact.NexusOperationFailed))
        Reply.OperationCanceled -> moves(state, Phase.Canceled, listOf(ProtocolFact.NexusOperationCanceled))
        is Reply.HandlerError ->
            if (!reply.retryable) moves(state, Phase.Failed, listOf(ProtocolFact.NexusOperationFailed))
            else listOf(
                Step(
                    ProtocolOutcome.Accepted,
                    state.copy(phase = Phase.BackingOff, attempts = state.attempts.saturatingSucc()),
                    listOf(ProtocolFact.PendingAttempts),
                ),
            )
    }
}

/** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
fun protocolTransportFaultStep(state: ProtocolState): ProtocolSteps {
    if (state.phase != Phase.Scheduled) return emptyList()
    return listOf(
        Step(
            ProtocolOutcome.Accepted,
            state.copy(phase = Phase.BackingOff, attempts = state.attempts.saturatingSucc()),
            listOf(ProtocolFact.PendingAttempts),
        ),
    )
}

/**
 * The handler's worker stopping is a fault the Run records and the operation does not feel, so the
 * step keeps the state and records nothing. On a path it is confirmed by the evidence of the step after
 * it, and the Case says so in a Known Gap.
 */
fun protocolWorkerStopStep(state: ProtocolState): ProtocolSteps =
    listOf(Step(ProtocolOutcome.Accepted, state, emptyList()))

/**
 * An asynchronous completion. Before a start, the server records a Started event first, which is why
 * the evidence is two facts and not one -- and why the product machine, which has no `BackingOff`
 * phase to have skipped, could write the completion alone.
 */
fun protocolCompleteStep(state: ProtocolState, resolution: Resolution): ProtocolSteps {
    if (terminalPhase(state.phase)) return listOf(Step(ProtocolOutcome.NotFound, state, emptyList()))
    if (state.phase == Phase.Unscheduled) return emptyList()
    val startedFirst = if (state.phase == Phase.Started) emptyList() else listOf(ProtocolFact.NexusOperationStarted)
    return when (resolution) {
        Resolution.Succeeded -> moves(state, Phase.Succeeded, startedFirst + ProtocolFact.NexusOperationCompleted)
        Resolution.Failed -> moves(state, Phase.Failed, startedFirst + ProtocolFact.NexusOperationFailed)
        Resolution.Canceled -> moves(state, Phase.Canceled, startedFirst + ProtocolFact.NexusOperationCanceled)
    }
}

val backoff = timer("backoff")
val scheduleToClose = timer("scheduleToClose")
val scheduleToStart = timer("scheduleToStart")
val startToClose = timer("startToClose")

/**
 * The backoff timer. It is what makes `BackingOff` a phase the operation leaves rather than a state it
 * is stuck in, and it records nothing: a retry writes no history event.
 */
fun backoffStep(state: ProtocolState): ProtocolSteps =
    if (state.phase != Phase.BackingOff) emptyList() else moves(state, Phase.Scheduled, emptyList())

/**
 * The schedule-to-close deadline covers the whole operation, so it fires in every running phase -- and
 * only when the schedule command set it.
 */
fun scheduleToCloseStep(state: ProtocolState): ProtocolSteps =
    if (running(state.phase) && state.scheduleToClose == Timeout.Expires)
        moves(state, Phase.TimedOut, listOf(ProtocolFact.NexusOperationTimedOut(TimeoutType.ScheduleToClose)))
    else emptyList()

/** The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the start. */
fun scheduleToStartStep(state: ProtocolState): ProtocolSteps =
    if ((state.phase == Phase.Scheduled || state.phase == Phase.BackingOff) && state.scheduleToStart == Timeout.Expires)
        moves(state, Phase.TimedOut, listOf(ProtocolFact.NexusOperationTimedOut(TimeoutType.ScheduleToStart)))
    else emptyList()

/** The start-to-close deadline covers the handler's own work, so it begins at the start. */
fun startToCloseStep(state: ProtocolState): ProtocolSteps =
    if (state.phase == Phase.Started && state.startToClose == Timeout.Expires)
        moves(state, Phase.TimedOut, listOf(ProtocolFact.NexusOperationTimedOut(TimeoutType.StartToClose)))
    else emptyList()

/**
 * How a protocol state reads as a product state. A phase of the same name is that phase; backing off
 * is still scheduled, because the product machine cannot see a retry; and an operation not yet
 * scheduled reads as scheduled, because the product machine begins there. Every other field is hidden,
 * which is what a map that does not read it says.
 */
fun productOf(state: ProtocolState): ProductState = ProductState(
    when (state.phase) {
        Phase.Unscheduled, Phase.Scheduled, Phase.BackingOff -> ProductPhase.Scheduled
        Phase.Started -> ProductPhase.Started
        Phase.Succeeded -> ProductPhase.Succeeded
        Phase.Failed -> ProductPhase.Failed
        Phase.Canceled -> ProductPhase.Canceled
        Phase.TimedOut -> ProductPhase.TimedOut
    },
)

val nexusProtocol = machine<ProtocolState, ProtocolOutcome, ProtocolFact>("nexusProtocol") {
    entity = operation
    refines(nexusProduct) via ::productOf
    starts(ProtocolState(Phase.Unscheduled))
    ends { terminalPhase(phase) }
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence { fact ->
        when (fact) {
            ProtocolFact.NexusOperationScheduled -> event("nexusOperationScheduled")
            ProtocolFact.NexusOperationStarted -> event("nexusOperationStarted")
            ProtocolFact.NexusOperationCompleted -> event("nexusOperationCompleted")
            ProtocolFact.NexusOperationFailed -> event("nexusOperationFailed")
            ProtocolFact.NexusOperationCanceled -> event("nexusOperationCanceled")
            is ProtocolFact.NexusOperationTimedOut -> event("nexusOperationTimedOut")
            ProtocolFact.PendingAttempts -> read(pendingAttempts)
        }
    }
    steps {
        schedule runs ::scheduleStep
        handlerReply runs ::protocolHandlerReplyStep
        complete runs ::protocolCompleteStep
        transportFault runs ::protocolTransportFaultStep
        workerStop runs ::protocolWorkerStopStep
        backoff runs ::backoffStep
        scheduleToClose runs ::scheduleToCloseStep
        scheduleToStart runs ::scheduleToStartStep
        startToClose runs ::startToCloseStep
    }
}

// ---- What the machines promise --------------------------------------------------------------------
//
// A same-step claim names the action it is about (`handlerReply(SyncSuccess) holds { ... }`) and holds
// of the step that action produces; a transition claim (`holds { before, after -> ... }`) holds of the
// step before and the step after. A functional Query realizes a same-step claim, because the Case's
// Contract is the claim's clause triggered by the action the Case performs; a transition claim is
// searched and verified, never realized. The machine is a parameter rather than a line in the block so
// the `step` in the lambda is typed by it.

/**
 * Once an operation is over, no step changes its phase. Declared on the product machine and read on
 * the protocol machine through the map.
 */
val terminalIsFinal = property("terminalIsFinal", nexusProduct) {
    holds { before, after -> !productTerminal(before.state) || after.state.phase == before.state.phase }
}

/** A synchronous reply settles the operation as succeeded, and the completed event records it. */
val syncSucceeds = property("syncSucceeds", nexusProtocol) {
    handlerReply(Reply.SyncSuccess) holds { step ->
        step.state.phase == Phase.Succeeded && ProtocolFact.NexusOperationCompleted in step.facts
    }
}

/** An asynchronous reply starts the operation, and the started event records it. */
val asyncStarts = property("asyncStarts", nexusProtocol) {
    handlerReply(Reply.Async) holds { step ->
        step.state.phase == Phase.Started && ProtocolFact.NexusOperationStarted in step.facts
    }
}

/**
 * A successful completion is recorded by the completed event. Neither the phase nor the outcome is
 * fixed: a completion resolves any running phase, and `Accepted` is every earlier step's outcome too,
 * so a clause fixing it would be answered before the completion.
 */
val completionSucceeds = property("completionSucceeds", nexusProtocol) {
    complete(Resolution.Succeeded) holds { step -> ProtocolFact.NexusOperationCompleted in step.facts }
}

/** A failed completion is recorded by the failed event. */
val completionFails = property("completionFails", nexusProtocol) {
    complete(Resolution.Failed) holds { step -> ProtocolFact.NexusOperationFailed in step.facts }
}

/** A non-retryable handler error settles the operation as failed, and the failed event records it. */
val handlerErrorFails = property("handlerErrorFails", nexusProtocol) {
    handlerReply(Reply.HandlerError(retryable = false)) holds { step ->
        step.state.phase == Phase.Failed && ProtocolFact.NexusOperationFailed in step.facts
    }
}

/**
 * Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state, so
 * every field is named.
 */
val succeededOnRetry = ProtocolState(
    phase = Phase.Succeeded,
    attempts = Attempts(1),
    scheduleToClose = Timeout.Unset,
    scheduleToStart = Timeout.Unset,
    startToClose = Timeout.Unset,
)

/**
 * A synchronous reply to the retried attempt settles the operation as succeeded on its second attempt:
 * the count the retryable failure raised is still one, and the completed event records the reply.
 */
val retrySucceeds = property("retrySucceeds", nexusProtocol) {
    handlerReply(Reply.SyncSuccess) holds { step ->
        step.state == succeededOnRetry && ProtocolFact.NexusOperationCompleted in step.facts
    }
}

/**
 * The schedule-to-start deadline settles an operation no handler started as timed out, and the
 * timed-out event records which deadline it was.
 */
val scheduleToStartFires = property("scheduleToStartFires", nexusProtocol) {
    scheduleToStart holds { step ->
        step.state.phase == Phase.TimedOut &&
            ProtocolFact.NexusOperationTimedOut(TimeoutType.ScheduleToStart) in step.facts
    }
}

/** The start-to-close deadline settles a started operation no handler completed as timed out. */
val startToCloseFires = property("startToCloseFires", nexusProtocol) {
    startToClose holds { step ->
        step.state.phase == Phase.TimedOut &&
            ProtocolFact.NexusOperationTimedOut(TimeoutType.StartToClose) in step.facts
    }
}

// ---- The paths the Queries run --------------------------------------------------------------------
//
// A protocol Scenario names its classed actions with their inputs and its start by its phase (the other
// fields default to their first value). Each path below is one upstream functional test's shape: the
// schedule command with no deadline set, then the side effects that settle the operation.

private val unset = Timeout.Unset
private val expires = Timeout.Expires

val syncReplied = scenario("syncReplied", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, unset), handlerReply(Reply.SyncSuccess))
}

val asyncThenSucceeded = scenario("asyncThenSucceeded", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, unset), handlerReply(Reply.Async), complete(Resolution.Succeeded))
}

val asyncThenFailed = scenario("asyncThenFailed", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, unset), handlerReply(Reply.Async), complete(Resolution.Failed))
}

val nonRetryableError = scenario("nonRetryableError", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, unset), handlerReply(Reply.HandlerError(retryable = false)))
}

/**
 * The retryable error backs the operation off; the backoff timer fires and records nothing; the
 * retried attempt is answered synchronously.
 */
val retriedThenSucceeded = scenario("retriedThenSucceeded", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(
        schedule(unset, unset, unset),
        handlerReply(Reply.HandlerError(retryable = true)),
        backoff,
        handlerReply(Reply.SyncSuccess),
    )
}

/**
 * The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
 * answers the start request; the deadline fires. The worker stops after the schedule in the operation's
 * order, where the stop changes nothing; the realization stops it before the workflow starts, where the
 * stop cannot race the dispatch.
 */
val scheduleToStartExpires = scenario("scheduleToStartExpires", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, expires, unset), workerStop, scheduleToStart)
}

/**
 * The schedule command sets the start-to-close deadline; the handler accepts asynchronously and never
 * completes; the deadline fires.
 */
val startToCloseExpires = scenario("startToCloseExpires", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, expires), handlerReply(Reply.Async), startToClose)
}

/**
 * Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
 * sequence of two is found among ninety-nine candidates, one of three among about a thousand and one of
 * four among about ten thousand.
 */
val two = limits("two", steps = 2, actions = 2, search = 512)
val three = limits("three", steps = 3, actions = 3, search = 4096)
val four = limits("four", steps = 4, actions = 4, search = 32768)

// ---- The Queries ----------------------------------------------------------------------------------
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one backoff,
// schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after an
// asynchronous reply. Each finds its same-step claim on its path and is realized by the set below. The
// product claim is verified over every trace of one path, outside the set, because a `verify` Query
// realizes nothing.

val syncCompletion = query("syncCompletion") { find(syncSucceeds) on syncReplied within two }
val asyncCompletion = query("asyncCompletion") { find(completionSucceeds) on asyncThenSucceeded within three }
val asyncFailure = query("asyncFailure") { find(completionFails) on asyncThenFailed within three }
val handlerError = query("handlerError") { find(handlerErrorFails) on nonRetryableError within two }
val retry = query("retry") { find(retrySucceeds) on retriedThenSucceeded within four }
val scheduleToStartTimeout = query("scheduleToStartTimeout") { find(scheduleToStartFires) on scheduleToStartExpires within three }
val startToCloseTimeout = query("startToCloseTimeout") { find(startToCloseFires) on startToCloseExpires within three }
val terminalHolds = query("terminalHolds") { verify(terminalIsFinal) on asyncThenSucceeded within three }

// ---- The functional set ---------------------------------------------------------------------------
//
// Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
// observes the network. The set repeats over the implementation switch, so each Query's Case runs once
// under HSM and once under CHASM.

val nexusCallerTests = set("nexusCallerTests") {
    purpose = Purpose.Functional
    bind(caller to driven, handler to driven, network to observed, worker to driven)
    repeat = Repeat.Implementation
    queries(
        syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
        scheduleToStartTimeout, startToCloseTimeout,
    )
}

// ---- The canary set -------------------------------------------------------------------------------
//
// A canary runs a Query against a deployment that performs the handler's part itself: the handler is
// `observed`, so the verifier reads which reply occurred and checks the machine allows it. What admits
// a canary is that a deployment can close every gap its Case carries, and every step of the sync and
// async completion paths records evidence; a path with a silent step -- the backoff, the worker stop --
// is a capability gap no deployment closes, so a canary naming it is rejected.

val nexusCallerCanary = set("nexusCallerCanary") {
    purpose = Purpose.Canary
    bind(caller to driven, handler to observed, network to observed, worker to driven)
    queries(syncCompletion, asyncCompletion)
}

// ---- The exploratory set --------------------------------------------------------------------------
//
// An exploration covers the protocol machine rather than listing Queries. Its targets are the rows an
// exploration within the budget's steps of a start can take, the results those rows reach and the
// members of the classes their actions claim, each in the machine's catalog order and cut at the
// budget's search count, so the enumeration is the same on every reading.

val nexusCallerExploration = set("nexusCallerExploration") {
    purpose = Purpose.Exploratory
    bind(caller to driven, handler to driven, network to observed, worker to driven)
    machine = nexusProtocol
    cover(Cover.Rows, Cover.Results, Cover.ClassMembers)
    budget = four
}

// ---- The Cases ------------------------------------------------------------------------------------
//
// Realization is out of this comparison's scope (SPEC.md): the Lean file's `case` blocks, which
// produce one protobuf Case per Query of a set, have no counterpart here.

// ---- The operation and the handler's worker -------------------------------------------------------
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's worker,
// so the schedule-to-start Scenario orders the stop before the request by convention. Composed with the
// worker of the handler's task queue, the stop is the worker's own phase change and every reply is the
// worker serving, so a reply has a row only while the worker polls. No set names the composition; it is
// what the cross-entity claim is verified over.

/**
 * The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
 * action no `syncs` line names would stay executable on its own and admit a stop, a resume and then a
 * reply; the operation's timers settle every state a stop leaves.
 */
val handlerWorker = Worker.polling.restrict("handlerWorker", Worker.workerStop, Worker.serve)

data class NexusCallerState(val operation: ProtocolState, val worker: Worker.WorkerState)

val nexusCaller = compose<NexusCallerState>("nexusCaller") {
    val operation = member(NexusCallerState::operation, nexusProtocol)
    val worker = member(NexusCallerState::worker, handlerWorker)
    workerStop syncs operation[workerStop] with worker[Worker.workerStop]
    handlerReply syncs operation[handlerReply] with worker[Worker.serve]
    starts(NexusCallerState(ProtocolState(Phase.Unscheduled), Worker.WorkerState(Worker.Phase.Polling)))
    // `this.`: the member handle `operation` above shadows the state's field inside the receiver lambda.
    ends { terminalPhase(this.operation.phase) }
}

/**
 * Every reply, of any class, leaves the handler's worker polling: no handler replies while its worker
 * is stopped.
 */
val repliedByPollingWorker = property("repliedByPollingWorker", nexusCaller) {
    handlerReply holds { step -> step.state.worker.phase == Worker.Phase.Polling }
}

/**
 * A retryable reply backs the operation off; the handler's worker then stops, so the retried attempt
 * is never answered and the schedule-to-start deadline fires. Member actions no `syncs` line names are
 * reached through the member: `operation[schedule]`, `operation[scheduleToStart]`.
 */
val repliedThenStopped = scenario("repliedThenStopped", nexusCaller) {
    val operation = nexusCaller.member(NexusCallerState::operation)
    starts(NexusCallerState(ProtocolState(Phase.Unscheduled), Worker.WorkerState(Worker.Phase.Polling)))
    actions(
        operation[schedule](unset, expires, unset),
        handlerReply(Reply.HandlerError(retryable = true)),
        workerStop,
        operation[scheduleToStart],
    )
}

val stoppedWorkerRepliesNothing = query("stoppedWorkerRepliesNothing") {
    verify(repliedByPollingWorker) on repliedThenStopped within four
}
