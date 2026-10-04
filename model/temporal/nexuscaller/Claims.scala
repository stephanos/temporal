package temporal
package nexuscaller

import umpire.*
import umpire.realize.{Alternative, Conformance, Exploration, Outcome, RunExpectation, Variation}
import worker.Phase as WorkerPhase

// ### What the machines promise
//
// A same-step claim names the action it is about under `when` and holds of the step that action
// produces; a transition claim holds of the state before and the step after. A functional Query
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.

/**
 * Once an operation is over, no step changes its phase. Declared on the product machine and read on
 * the protocol machine through the map.
 */
val terminalIsFinal: Property[ProductState] = nexusProduct.property("terminalIsFinal") holdsAcross {
  (before, after) =>
    !Product.productTerminal(before) || after.state.phase == before.phase
}

/** A synchronous reply settles the operation as succeeded, and the completed event records it. */
val syncSucceeds: Property[ProtocolState] =
  nexusProtocol.property("syncSucceeds") when handlerReply(Reply.syncSuccess) holds { s =>
    s.state.phase == Phase.succeeded && s.facts.contains(ProtocolFact.nexusOperationCompleted)
  }

/** An asynchronous reply starts the operation, and the started event records it. */
val asyncStarts: Property[ProtocolState] =
  nexusProtocol.property("asyncStarts") when handlerReply(Reply.async) holds { s =>
    s.state.phase == Phase.started && s.facts.contains(ProtocolFact.nexusOperationStarted)
  }

/**
 * A successful completion is recorded by the completed event. Neither the phase nor the outcome is
 * fixed: a completion resolves any running phase, and accepted is every earlier step's outcome
 * too, so a clause fixing it would be answered before the completion.
 */
val completionSucceeds: Property[ProtocolState] =
  nexusProtocol.property("completionSucceeds") when complete(Resolution.succeeded) holds
    (_.facts.contains(ProtocolFact.nexusOperationCompleted))

/** A failed completion is recorded by the failed event. */
val completionFails: Property[ProtocolState] =
  nexusProtocol.property("completionFails") when complete(Resolution.failed) holds
    (_.facts.contains(ProtocolFact.nexusOperationFailed))

/** A non-retryable handler error settles the operation as failed, and the failed event records it. */
val handlerErrorFails: Property[ProtocolState] =
  nexusProtocol.property("handlerErrorFails") when handlerReply(Reply.handlerError(false)) holds {
    s =>
      s.state.phase == Phase.failed && s.facts.contains(ProtocolFact.nexusOperationFailed)
  }

/**
 * Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
 * so every field is named.
 */
val succeededOnRetry: ProtocolState =
  ProtocolState(Phase.succeeded, 1, Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * A synchronous reply to the retried attempt settles the operation as succeeded on its second
 * attempt: the count the retryable failure raised is still one, and the completed event records the
 * reply.
 */
val retrySucceeds: Property[ProtocolState] =
  nexusProtocol.property("retrySucceeds") when handlerReply(Reply.syncSuccess) holds { s =>
    s.state == succeededOnRetry && s.facts.contains(ProtocolFact.nexusOperationCompleted)
  }

/**
 * The schedule-to-start deadline settles an operation no handler started as timed out, and the
 * timed-out event records which deadline it was.
 */
val scheduleToStartFires: Property[ProtocolState] =
  nexusProtocol.property("scheduleToStartFires") when scheduleToStart holds { s =>
    s.state.phase == Phase.timedOut && s.facts.contains(
      ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)
    )
  }

/** The start-to-close deadline settles a started operation no handler completed as timed out. */
val startToCloseFires: Property[ProtocolState] =
  nexusProtocol.property("startToCloseFires") when startToClose holds { s =>
    s.state.phase == Phase.timedOut && s.facts.contains(
      ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose)
    )
  }

// ### The paths the Queries run
//
// A protocol Scenario names its classed actions with their inputs and its start. Each path below is
// one upstream functional test's shape: the schedule command with no deadline set, then the side
// effects that settle the operation.

import Timeout.{expires, unset}

val syncReplied: Scenario[ProtocolState] = nexusProtocol
  .scenario("syncReplied")
  .starts(unscheduled)
  .actions(schedule(unset, unset, unset), handlerReply(Reply.syncSuccess))

val asyncThenSucceeded: Scenario[ProtocolState] = nexusProtocol
  .scenario("asyncThenSucceeded")
  .starts(unscheduled)
  .actions(schedule(unset, unset, unset), handlerReply(Reply.async), complete(Resolution.succeeded))

val asyncThenFailed: Scenario[ProtocolState] = nexusProtocol
  .scenario("asyncThenFailed")
  .starts(unscheduled)
  .actions(schedule(unset, unset, unset), handlerReply(Reply.async), complete(Resolution.failed))

val nonRetryableError: Scenario[ProtocolState] = nexusProtocol
  .scenario("nonRetryableError")
  .starts(unscheduled)
  .actions(schedule(unset, unset, unset), handlerReply(Reply.handlerError(false)))

/**
 * The retryable error backs the operation off; the backoff timer fires and records nothing; the
 * retried attempt is answered synchronously.
 */
val retriedThenSucceeded: Scenario[ProtocolState] = nexusProtocol
  .scenario("retriedThenSucceeded")
  .starts(unscheduled)
  .actions(
    schedule(unset, unset, unset),
    handlerReply(Reply.handlerError(true)),
    backoff,
    handlerReply(Reply.syncSuccess)
  )

/**
 * The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
 * answers the start request; the deadline fires. The worker stops after the schedule in the
 * operation's order, where the stop changes nothing; the realization stops it before the workflow
 * starts, where the stop cannot race the dispatch.
 */
val scheduleToStartExpires: Scenario[ProtocolState] = nexusProtocol
  .scenario("scheduleToStartExpires")
  .starts(unscheduled)
  .actions(schedule(unset, expires, unset), workerStop, scheduleToStart)

/**
 * The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
 * never completes; the deadline fires.
 */
val startToCloseExpires: Scenario[ProtocolState] = nexusProtocol
  .scenario("startToCloseExpires")
  .starts(unscheduled)
  .actions(schedule(unset, unset, expires), handlerReply(Reply.async), startToClose)

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and one
// of four among about ten thousand.
val two: Limits = Limits("two", steps = 2, actions = 2, search = 512)
val three: Limits = Limits("three", steps = 3, actions = 3, search = 4096)
val four: Limits = Limits("four", steps = 4, actions = 4, search = 32768)

// ### The Queries
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one
// backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after
// an asynchronous reply. Each finds its same-step claim on its path and is realized as a Case.
// The product claim is verified over every trace of one path, outside `functionalQueries`, because
// a verify Query realizes nothing.

val syncCompletion: Query = (query("syncCompletion") find syncSucceeds in syncReplied limits two)
  .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
  .explore(
    Exploration(
      "nexusDeadlines",
      Vector(
        Variation(
          0,
          Vector(
            Alternative("startDeadline", 30, Vector(schedule(unset, unset, expires))),
            Alternative("scheduleDeadline", 20, Vector(schedule(unset, expires, unset))),
            Alternative("unbounded", 10, Vector(schedule(unset, unset, unset)))
          )
        )
      ),
      runs = 1,
      edits = 1,
      dropPrefix = true
    )
  )
val asyncCompletion: Query =
  (query("asyncCompletion") find completionSucceeds in asyncThenSucceeded limits three).expect(
    RunExpectation(
      Conformance.conformant,
      Outcome.inconclusive,
      "the executions that explain the evidence disagree"
    )
  )
val asyncFailure: Query =
  (query("asyncFailure") find completionFails in asyncThenFailed limits three).expect(
    RunExpectation(
      Conformance.conformant,
      Outcome.inconclusive,
      "the executions that explain the evidence disagree"
    )
  )
val handlerError: Query =
  (query("handlerError") find handlerErrorFails in nonRetryableError limits two).expect(
    RunExpectation(
      Conformance.conformant,
      Outcome.inconclusive,
      "an execution that explains the evidence never reaches the claim's evaluation point"
    )
  )
val retry: Query = (query("retry") find retrySucceeds in retriedThenSucceeded limits four)
  .expect(
    RunExpectation(
      Conformance.conformant,
      Outcome.inconclusive,
      "the executions that explain the evidence disagree"
    )
  )
val scheduleToStartTimeout: Query =
  (query("scheduleToStartTimeout") find scheduleToStartFires in scheduleToStartExpires limits three)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "an execution that explains the evidence never reaches the claim's evaluation point"
      )
    )
val startToCloseTimeout: Query =
  (query("startToCloseTimeout") find startToCloseFires in startToCloseExpires limits three).expect(
    RunExpectation(
      Conformance.conformant,
      Outcome.inconclusive,
      "an execution that explains the evidence never reaches the claim's evaluation point"
    )
  )

/**
 * A product claim on a protocol path, read through the refinement the protocol machine declares.
 */
val terminalHolds: Query =
  query("terminalHolds") verify terminalIsFinal in asyncThenSucceeded limits three

/** The functional Queries in declaration order. */
val functionalQueries: Vector[Query] =
  Vector(
    syncCompletion,
    asyncCompletion,
    asyncFailure,
    handlerError,
    retry,
    scheduleToStartTimeout,
    startToCloseTimeout
  )

// ### The cross-entity claim

/**
 * Every reply, of any class, leaves the handler's worker polling: no handler replies while its
 * worker is stopped.
 */
val repliedByPollingWorker: Property[NexusCallerState] =
  nexusCaller
    .property("repliedByPollingWorker")
    .whenAction("handlerReply") holds (_.state.worker.phase == WorkerPhase.polling)

/**
 * A retryable reply backs the operation off; the handler's worker then stops, so the retried
 * attempt is never answered and the schedule-to-start deadline fires.
 */
val repliedThenStopped: Scenario[NexusCallerState] = nexusCaller
  .scenario("repliedThenStopped")
  .starts(NexusCallerState(unscheduled, pollingWorker))
  .actionKeys(
    "operation_schedule-unset-expires-unset",
    "handlerReply-handlerError-true",
    "workerStop",
    "operation_scheduleToStart"
  )

/** The cross-entity claim, verified over that path. */
val stoppedWorkerRepliesNothing: Query =
  query(
    "stoppedWorkerRepliesNothing"
  ) verify repliedByPollingWorker in repliedThenStopped limits four
