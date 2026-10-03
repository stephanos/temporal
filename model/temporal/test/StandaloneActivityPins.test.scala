package temporal
package standaloneactivity
// What the standalone activity Model says, pinned: the sizes, rows and answers the native evaluator
// derives from it.

import umpire.*

class StandaloneActivityPins extends munit.FunSuite:
  def table(m: Model): Table = m.table.fold(e => fail(e.toString), identity)
  def answer(q: Query): Answer = q.answer.fold(e => fail(e.toString), identity)

  /**
   * A state written the way a reader names one: the phase, the attempt count, and whichever
   * deadlines are not at the value the activity begins with.
   */
  def at(
      phase: Phase,
      attempts: Int,
      sts: Timeout = Timeout.unset,
      stc: Timeout = Timeout.unset
  ): ProtocolState =
    ProtocolState(phase, attempts, Timeout.unset, sts, stc)

  def phases(steps: List[ProtocolStep]): List[Phase] = steps.map(_.state.phase)

  test("the product machine") {
    val t = table(activityProduct)
    // Nine phases, and the five the design ends on.
    assertEquals((t.states.size, t.ends.size), (9, 5))
    // A canceled answer settles only an activity whose cancellation was requested.
    assertEquals(attemptResultStep(ProductState(ProductPhase.started), AttemptResult.canceled), Nil)
    assertEquals(
      attemptResultStep(ProductState(ProductPhase.cancelRequested), AttemptResult.canceled)
        .map(_.state.phase),
      List(ProductPhase.canceled)
    )
    // Unlike the Nexus product, a retry is visible: the caller reads scheduled again.
    assertEquals(
      attemptResultStep(ProductState(ProductPhase.started), AttemptResult.failed(true))
        .map(_.state.phase),
      List(ProductPhase.scheduled)
    )
    assertEquals(
      attemptResultStep(ProductState(ProductPhase.cancelRequested), AttemptResult.failed(true))
        .map(_.state.phase),
      List(ProductPhase.canceled)
    )
    // A paused activity is dispatched to no worker: no product row leaves paused for started.
    assertEquals(
      t.rows.filter(_.source == "paused").flatMap(_.results).filter(_.state == "started"),
      Vector.empty
    )
    assertEquals(t.stuck, None)
  }

  test("the protocol machine") {
    val t = table(activityProtocol)
    val n = attemptBound + 1
    // Twelve phases, three attempt counts and three deadlines: 288 states.
    assertEquals((t.states.size, t.ends.size), (12 * n * 8, 5 * n * 8))
    // Eight start requests, one attempt start, four answers, four controls, the fault and four
    // timers.
    assertEquals(t.actions.size, 8 + 1 + 4 + 4 + 1 + 4)
    assertEquals(t.starts, Vector(Keys.of(at(Phase.unstarted, 0))))
    // A retryable failure backs the attempt off and is read as scheduled again with the count
    // raised.
    assertEquals(
      protocolAttemptResultStep(at(Phase.started, 1), AttemptResult.failed(true)),
      List(
        Step(
          Outcome.accepted,
          at(Phase.backingOff, 1),
          List(ProtocolFact.statusScheduled, ProtocolFact.attemptCount),
          "a retryable failure backs off; the caller reads scheduled again"
        )
      )
    )
    // Under a cancel request the same failure settles the activity as canceled; under a pause
    // request it lands in paused.
    assertEquals(
      phases(protocolAttemptResultStep(at(Phase.cancelRequested, 1), AttemptResult.failed(true))),
      List(Phase.canceled)
    )
    assertEquals(
      phases(protocolAttemptResultStep(at(Phase.pauseRequested, 1), AttemptResult.failed(true))),
      List(Phase.paused)
    )
    // A pause of a held attempt is a request; of a scheduled one it takes effect at once.
    assertEquals(
      phases(protocolControlStep(at(Phase.started, 1), Control.pause)),
      List(Phase.pauseRequested)
    )
    assertEquals(
      phases(protocolControlStep(at(Phase.scheduled, 0), Control.pause)),
      List(Phase.paused)
    )
    // A control on an activity that is over is not found.
    assertEquals(
      protocolControlStep(at(Phase.completed, 1), Control.terminate),
      List(Step(Outcome.notFound, at(Phase.completed, 1)))
    )
    // Each deadline covers its own span.
    assertEquals(startToCloseStep(at(Phase.scheduled, 0, stc = Timeout.expires)), Nil)
    assertEquals(
      phases(startToCloseStep(at(Phase.pauseRequested, 1, stc = Timeout.expires))),
      List(Phase.timedOut)
    )
    assertEquals(
      scheduleToStartStep(at(Phase.backingOff, 1, sts = Timeout.expires)).map(_.facts),
      List(List(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart)))
    )
    assertEquals(t.stuck, None)
  }

  test("the refinement") {
    val ref = activityProtocol.refinementCheck.fold(e => fail(e.toString), identity)
    assertEquals(ref.rows.size, table(activityProtocol).rows.map(_.results.size).sum)
    val lookup = ref.rows.map(r => r.key -> r.product).toMap
    // The visible retry is the product's retryable-failure row; the pause request is a stutter; the
    // unpause of a requested pause is a stutter too.
    assertEquals(
      lookup("started-1-unset-unset-unset-attemptResult-failed-true"),
      Some("attemptResult-failed-true")
    )
    assertEquals(lookup("started-1-unset-unset-unset-control-pause"), None)
    assertEquals(lookup("pauseRequested-1-unset-unset-unset-control-unpause"), None)
    // A retryable failure under a pause request is the product's pause.
    assertEquals(
      lookup("pauseRequested-1-unset-unset-unset-attemptResult-failed-true"),
      Some("control-pause")
    )
  }

  test("the Queries") {
    for q <- functionalQueries do assertEquals(answer(q).outcome, Verdict.found, q.name)
    assertEquals(answer(terminalHolds).outcome, Verdict.verifiedWithinLimits)
    assertEquals(answer(pauseHolds).outcome, Verdict.verifiedWithinLimits)
    // Not vacuous: the scenario performs an attempt start while the worker polls, so the claim is
    // exercised, not merely never contradicted.
    val stopped = answer(stoppedWorkerStartsNothing)
    assertEquals((stopped.outcome, stopped.exercised), (Verdict.verifiedWithinLimits, true))
  }

  test("every declaration passes the framework's semantic checks") {
    assertEquals(
      check(
        activityProduct,
        activityProtocol,
        activityWorker,
        standaloneActivity,
        terminalHolds,
        pauseHolds,
        stoppedWorkerStartsNothing
      ),
      Nil
    )
  }

  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = loseAdmissionAnswer(responseLossInitial)
    assertEquals(choices.map(_.state.record), admitted(scheduledIdle).map(_.state))
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    assertEquals(choices.flatMap(s => loseAdmissionAnswer(s.state)), Nil)
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
