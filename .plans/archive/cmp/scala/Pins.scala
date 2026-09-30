package temporal.feature.pins

/* What the two Models say
 *
 * The Lean `#guard` pins, as munit tests plus a few compile-time checks. The test module is a
 * separate sbt project downstream of the Model module, which is what lets `Query.pinned` run the
 * bounded search while this file compiles: a macro can evaluate code only from a module that is
 * already compiled. A pin that fails at compile time reports on its own line; the rest fail as
 * tests. */

import munit.FunSuite
import umpire.*

// ---------------------------------------------------------------------------------------------
// Compile-time pins: decided by the inliner and the macro, before any test runs
// ---------------------------------------------------------------------------------------------

object CompileTimePins:
  import temporal.feature.nexus.caller as Nexus
  import temporal.feature.activity.standalone as Activity

  /* State counts. `Finite.sizeOf` folds to a literal, so a state field added without a pin
   * update is a compile error on the pin's line: "state count is 384, pin says 192". */
  pinStates[Nexus.ProductState](6)
  pinStates[Nexus.ProtocolState](8 * (Nexus.attemptBound + 1) * 2 * 2 * 2)
  pinStates[Activity.ProductState](9)
  pinStates[Activity.ProtocolState](12 * (Activity.attemptBound + 1) * 2 * 2 * 2)

  /* The searches, run during compilation of this module. A Query whose path does not reach its
   * claim is an error at the `Query.pinned(...)` expression, and the message names the Property,
   * the Scenario, the Limits and the traces searched. */
  val syncCompletion = Query.pinned(Nexus.syncCompletion)
  val asyncCompletion = Query.pinned(Nexus.asyncCompletion)
  val asyncFailure = Query.pinned(Nexus.asyncFailure)
  val handlerError = Query.pinned(Nexus.handlerError)
  val retry = Query.pinned(Nexus.retry)
  val scheduleToStartTimeout = Query.pinned(Nexus.scheduleToStartTimeout)
  val startToCloseTimeout = Query.pinned(Nexus.startToCloseTimeout)
  val terminalHolds = Query.pinned(Nexus.terminalHolds)
  val stoppedWorkerRepliesNothing = Query.pinned(Nexus.stoppedWorkerRepliesNothing)

  val completion = Query.pinned(Activity.completion)
  val nonRetryableFailure = Query.pinned(Activity.nonRetryableFailure)
  val activityRetry = Query.pinned(Activity.retry)
  val cancel = Query.pinned(Activity.cancel)
  val terminate = Query.pinned(Activity.terminate)
  val pauseResume = Query.pinned(Activity.pauseResume)
  val activityScheduleToStartTimeout = Query.pinned(Activity.scheduleToStartTimeout)
  val activityStartToCloseTimeout = Query.pinned(Activity.startToCloseTimeout)
  val activityTerminalHolds = Query.pinned(Activity.terminalHolds)
  val pauseHolds = Query.pinned(Activity.pauseHolds)
  val stoppedWorkerStartsNothing = Query.pinned(Activity.stoppedWorkerStartsNothing)

// ---------------------------------------------------------------------------------------------
// The Nexus caller
// ---------------------------------------------------------------------------------------------

class NexusCallerPins extends FunSuite:
  import temporal.feature.nexus.caller.*

  /** A state written the way a reader names one: the phase, and whichever fields are not at the
    * value the operation begins with. */
  private def at(
      phase: Phase,
      attempts: Attempts = Attempts(0),
      scheduleToClose: Timeout = unset,
      scheduleToStart: Timeout = unset,
      startToClose: Timeout = unset,
  ): ProtocolState = ProtocolState(phase, attempts, scheduleToClose, scheduleToStart, startToClose)

  // The product machine

  test("product: six phases, the four the design ends on, twelve action classes") {
    assertEquals(nexusProduct.table.states.size, 6)
    assertEquals(nexusProduct.ends.size, 4)
    // Six replies, three resolutions, the two faults it cannot see, and the one timer.
    assertEquals(nexusProduct.actionKeys.size, 12)
  }

  test("product: a retryable handler error is invisible; the protocol machine backs off") {
    assertEquals(handlerReplyStep(ProductState(ProductPhase.scheduled), Reply.handlerError(true)), Nil)
  }

  test("product: every phase is reached from the start") {
    assertEquals(nexusProduct.reachable.toSet, summon[Finite[ProductState]].values.toSet)
  }

  test("product: nothing is stuck") {
    assertEquals(nexusProduct.stuck, None)
  }

  // The protocol machine

  test("protocol: eight phases, three counts, three deadlines; the four ends") {
    assertEquals(nexusProtocol.table.states.size, 8 * (attemptBound + 1) * 2 * 2 * 2)
    assertEquals(nexusProtocol.ends.size, 4 * (attemptBound + 1) * 2 * 2 * 2)
  }

  test("protocol: eight schedule commands, six replies, three resolutions, two faults, four timers") {
    assertEquals(nexusProtocol.actionKeys.size, 8 + 6 + 3 + 1 + 1 + 4)
  }

  test("protocol: the machine begins before the operation exists, at every deadline assignment") {
    assertEquals(nexusProtocol.starts.map(_.phase).distinct, List(Phase.unscheduled))
    assertEquals(nexusProtocol.starts.size, (attemptBound + 1) * 2 * 2 * 2)
    assert(nexusProtocol.starts.contains(at(Phase.unscheduled)))
  }

  test("protocol: a retryable handler error backs off and raises the count, read as pendingAttempts") {
    assertEquals(
      protocolHandlerReplyStep(at(Phase.scheduled), Reply.handlerError(true)),
      List(Step(ProtocolOutcome.accepted, at(Phase.backingOff, Attempts(1)), List(ProtocolFact.pendingAttempts))),
    )
  }

  test("protocol: the count saturates rather than wrapping") {
    val fromLast = protocolHandlerReplyStep(at(Phase.scheduled, Attempts(attemptBound)), Reply.handlerError(true))
    assertEquals(fromLast.map(_.state.attempts), List(Attempts(attemptBound)))
  }

  test("protocol: a completion before the start records Started first; after it, not") {
    assertEquals(
      protocolCompleteStep(at(Phase.backingOff, Attempts(1)), Resolution.succeeded).flatMap(_.facts),
      List(ProtocolFact.nexusOperationStarted, ProtocolFact.nexusOperationCompleted),
    )
    assertEquals(
      protocolCompleteStep(at(Phase.started), Resolution.succeeded).flatMap(_.facts),
      List(ProtocolFact.nexusOperationCompleted),
    )
  }

  test("protocol: a completion after the operation is over is not found and changes nothing") {
    assertEquals(
      protocolCompleteStep(at(Phase.timedOut), Resolution.succeeded),
      List(Step(ProtocolOutcome.notFound, at(Phase.timedOut), Nil)),
    )
  }

  test("protocol: each timer covers its own span and only fires when set") {
    assertEquals(startToCloseStep(at(Phase.scheduled, startToClose = expires)), Nil)
    assertEquals(startToCloseStep(at(Phase.started, startToClose = expires)).map(_.state.phase), List(Phase.timedOut))
    assertEquals(scheduleToCloseStep(at(Phase.started)), Nil)
    assertEquals(
      scheduleToStartStep(at(Phase.scheduled, scheduleToStart = expires)).flatMap(_.facts),
      List(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)),
    )
  }

  test("protocol: the worker stop keeps the state and records nothing; the product cannot see it") {
    val s = at(Phase.scheduled, scheduleToStart = expires)
    assertEquals(protocolWorkerStopStep(s), List(Step(ProtocolOutcome.accepted, s, Nil)))
    assertEquals(workerStopStep(ProductState(ProductPhase.scheduled)), Nil)
  }

  test("protocol: nothing is stuck") {
    assertEquals(nexusProtocol.stuck, None)
  }

  // The refinement

  test("refinement: every protocol row is a product step or a product stutter") {
    val r = nexusProtocol.refinement.getOrElse(fail("nexusProtocol declares no refinement"))
    assertEquals(r.rejected, None)
    assertEquals(r.rows.size, nexusProtocol.transitions.size)
  }

  // The Queries: the same searches `CompileTimePins` ran, as tests, so a run without the macro
  // module (say, a plain `scala-cli test`) still checks them.

  test("every find-Query's Scenario reaches its Property") {
    for q <- nexusCallerTests.queries do
      q.run match
        case Result.Found(_) => ()
        case other => fail(s"${q.name}: $other")
  }

  test("terminalIsFinal verifies on the async-then-succeeded path") {
    assert(terminalHolds.run.isInstanceOf[Result.Verified], terminalHolds.run.toString)
  }

  test("no handler replies while its worker is stopped") {
    assert(stoppedWorkerRepliesNothing.run.isInstanceOf[Result.Verified])
  }

// ---------------------------------------------------------------------------------------------
// The standalone activity
// ---------------------------------------------------------------------------------------------

class StandaloneActivityPins extends FunSuite:
  import temporal.feature.activity.standalone.*
  import temporal.feature.nexus.caller.{Timeout, unset, expires}

  private def at(
      phase: Phase,
      attempts: Attempts = Attempts(0),
      scheduleToClose: Timeout = unset,
      scheduleToStart: Timeout = unset,
      startToClose: Timeout = unset,
  ): ProtocolState = ProtocolState(phase, attempts, scheduleToClose, scheduleToStart, startToClose)

  test("product: nine phases, the five the design ends on") {
    assertEquals(activityProduct.table.states.size, 9)
    assertEquals(activityProduct.ends.size, 5)
  }

  test("protocol: twelve phases, three counts, three deadlines; the five ends") {
    assertEquals(activityProtocol.table.states.size, 12 * (attemptBound + 1) * 2 * 2 * 2)
    assertEquals(activityProtocol.ends.size, 5 * (attemptBound + 1) * 2 * 2 * 2)
  }

  test("protocol: a cancel result with no cancel requested is not a row") {
    assertEquals(protocolAttemptResultStep(at(Phase.started, Attempts(1)), AttemptResult.canceled), Nil)
  }

  test("protocol: a pause of a running attempt is only requested, and still reads as started") {
    val rows = protocolControlStep(at(Phase.started, Attempts(1)), Control.pause)
    assertEquals(rows.map(_.state.phase), List(Phase.pauseRequested))
    assertEquals(rows.flatMap(_.facts), List(ProtocolFact.statusPaused))
    assertEquals(productOf(at(Phase.pauseRequested, Attempts(1))), ProductState(ProductPhase.started))
  }

  test("product: a retryable failure is visible, as a return to scheduled or as the requested cancel") {
    assertEquals(
      attemptResultStep(ProductState(ProductPhase.started), AttemptResult.failed(true)).map(_.state.phase),
      List(ProductPhase.scheduled),
    )
    assertEquals(
      attemptResultStep(ProductState(ProductPhase.cancelRequested), AttemptResult.failed(true)).map(_.state.phase),
      List(ProductPhase.canceled),
    )
  }

  test("protocol: a retryable failure under a cancel request settles as canceled") {
    assertEquals(
      protocolAttemptResultStep(at(Phase.cancelRequested, Attempts(1)), AttemptResult.failed(true)).map(_.state.phase),
      List(Phase.canceled),
    )
  }

  test("protocol: a control on a finished activity is not found and changes nothing") {
    assertEquals(
      protocolControlStep(at(Phase.completed, Attempts(1)), Control.terminate),
      List(Step(ProtocolOutcome.notFound, at(Phase.completed, Attempts(1)), Nil)),
    )
  }

  test("refinement: every protocol row is a product step or a product stutter") {
    val r = activityProtocol.refinement.getOrElse(fail("activityProtocol declares no refinement"))
    assertEquals(r.rejected, None)
  }

  test("every find-Query's Scenario reaches its Property") {
    for q <- standaloneActivityTests.queries do
      q.run match
        case Result.Found(_) => ()
        case other => fail(s"${q.name}: $other")
  }

  test("terminalIsFinal and pausedIsNotDispatched verify") {
    assert(terminalHolds.run.isInstanceOf[Result.Verified], terminalHolds.run.toString)
    assert(pauseHolds.run.isInstanceOf[Result.Verified], pauseHolds.run.toString)
  }

  test("no worker picks a task up while stopped") {
    assert(stoppedWorkerStartsNothing.run.isInstanceOf[Result.Verified])
  }
