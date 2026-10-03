package temporal
package nexuscaller
// What the Caller Model says, pinned: the sizes, rows and answers the native evaluator derives from
// it. The Case pins live with the Case producer.

import umpire.*

class NexusCallerPins extends munit.FunSuite:
  def table(m: Model): Table = m.table.fold(e => fail(e.toString), identity)
  def answer(q: Query): Answer = q.answer.fold(e => fail(e.toString), identity)

  /**
   * A state written the way a reader names one: the phase, and whichever fields are not at the
   * value the operation begins with.
   */
  def at(
      phase: Phase,
      attempts: Int = 0,
      sts: Timeout = Timeout.unset,
      stc: Timeout = Timeout.unset
  ): ProtocolState =
    ProtocolState(phase, attempts, Timeout.unset, sts, stc)

  test("the product machine") {
    val t = table(nexusProduct)
    // Six phases, and the four the design ends on.
    assertEquals((t.states.size, t.ends.size), (6, 4))
    // Six replies, three resolutions, the two faults it cannot see, and the one timer.
    assertEquals(t.actions.size, 12)
    // A retryable handler error is invisible here: it is the protocol machine that backs off.
    assertEquals(
      Product.handlerReplyStep(ProductState(ProductPhase.scheduled), Reply.handlerError(true)),
      Nil
    )
    // What the Model actually reaches: every phase.
    assertEquals(
      t.reachable,
      Vector("scheduled", "canceled", "failed", "succeeded", "started", "timedOut")
    )
    assertEquals(t.stuck, None)
  }

  test("the protocol machine") {
    val t = table(nexusProtocol)
    val n = Protocol.attemptBound + 1
    // Eight phases, three attempt counts and three deadlines, and the four ending phases.
    assertEquals((t.states.size, t.ends.size), (8 * n * 8, 4 * n * 8))
    // Eight schedule commands, six replies, three resolutions, the two faults and the four timers.
    // The catalog is in canonical order, so it opens on the backoff timer.
    assertEquals(t.actions.size, 8 + 6 + 3 + 1 + 1 + 4)
    assertEquals(t.actions.take(2), Vector("backoff", "complete-canceled"))
    // The machine begins before the operation exists, with every deadline at its first value.
    assertEquals(t.starts, Vector(Keys.of(at(Phase.unscheduled))))
    // A retryable handler error backs the operation off and raises the attempt count.
    assertEquals(
      Protocol.handlerReplyStep(at(Phase.scheduled), Reply.handlerError(true)),
      List(
        Step(
          Outcome.accepted,
          at(Phase.backingOff, attempts = 1),
          List(ProtocolFact.pendingAttempts)
        )
      )
    )
    // The count saturates rather than wrapping.
    assertEquals(
      Protocol
        .handlerReplyStep(at(Phase.scheduled, attempts = 2), Reply.handlerError(true))
        .map(_.state.attempts),
      List(2)
    )
    // A completion before the start records the Started event first, one after it does not.
    assertEquals(
      Protocol.completeStep(at(Phase.backingOff, attempts = 1), Resolution.succeeded).map(_.facts),
      List(List(ProtocolFact.nexusOperationStarted, ProtocolFact.nexusOperationCompleted))
    )
    assertEquals(
      Protocol.completeStep(at(Phase.started), Resolution.succeeded).map(_.facts),
      List(List(ProtocolFact.nexusOperationCompleted))
    )
    // A completion after the operation is over is not found and changes nothing.
    assertEquals(
      Protocol.completeStep(at(Phase.timedOut), Resolution.succeeded),
      List(Step(Outcome.notFound, at(Phase.timedOut), Nil))
    )
    // A timer fires only when the schedule command set it, and each covers its own span.
    assertEquals(Protocol.startToCloseStep(at(Phase.scheduled, stc = Timeout.expires)), Nil)
    assertEquals(
      Protocol.startToCloseStep(at(Phase.started, stc = Timeout.expires)).map(_.state.phase),
      List(Phase.timedOut)
    )
    assertEquals(Protocol.scheduleToCloseStep(at(Phase.started)), Nil)
    // Which timer fired is recorded.
    assertEquals(
      Protocol.scheduleToStartStep(at(Phase.scheduled, sts = Timeout.expires)).map(_.facts),
      List(List(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)))
    )
    // The worker stopping keeps the state and records nothing; the product machine does not see it.
    assertEquals(
      Protocol.workerStopStep(at(Phase.scheduled, sts = Timeout.expires)),
      List(Step(Outcome.accepted, at(Phase.scheduled, sts = Timeout.expires), Nil))
    )
    assertEquals(Product.workerStopStep(ProductState(ProductPhase.scheduled)), Nil)
    // Nothing is stuck.
    assertEquals(t.stuck, None)
    // Not every state is reachable. The Behavior Fingerprint reads the table, so this number is part
    // of the Model's identity.
    assertEquals(t.reachable.size, 158)
  }

  test("the refinement") {
    val ref = nexusProtocol.refinementCheck.fold(e => fail(e.toString), identity)
    assertEquals(ref.rows.size, table(nexusProtocol).rows.size)
    val lookup = ref.rows.map(r => r.key -> r.product).toMap
    // A reply the product machine sees is that reply's step. A retry it cannot see is a stutter, and
    // so are the schedule command and the backoff timer.
    assertEquals(
      lookup("scheduled-0-unset-unset-unset-handlerReply-async"),
      Some("handlerReply-async")
    )
    assertEquals(lookup("scheduled-0-unset-unset-unset-handlerReply-handlerError-true"), None)
    assertEquals(lookup("unscheduled-0-unset-unset-unset-schedule-unset-unset-expires"), None)
    assertEquals(lookup("backingOff-1-unset-unset-unset-backoff"), None)
    // A deadline firing is the product's one timer, whichever deadline it was.
    assertEquals(lookup("started-0-unset-unset-expires-startToClose"), Some("timeout"))
    // A completion before the start records the Started event first and still carries the step.
    assertEquals(
      lookup("backingOff-1-unset-unset-unset-complete-succeeded"),
      Some("complete-succeeded")
    )
    // Every schedule command, every retry, every backoff and every worker stop.
    val stutters = ref.rows.filter(_.product.isEmpty)
    assertEquals(stutters.size, 24 * 8 + 24 + 24 + 24 + 192)
    // Stutter invariance: every stutter leaves a phase that reads as scheduled, or is a worker stop.
    for r <- stutters do
      assert(
        Set("unscheduled", "scheduled", "backingOff")(Keys.actionName(r.key)) || r.key.endsWith(
          "-workerStop"
        ),
        r.key
      )
    // The product state a protocol state reads as is a field named after the product machine.
    assertEquals(
      table(nexusProtocol).stateFields,
      Vector(
        "phase",
        "attempts",
        "scheduleToClose",
        "scheduleToStart",
        "startToClose",
        "nexusProduct"
      )
    )
  }

  test("the Properties and the Queries") {
    // A protocol Scenario names its classed actions with their inputs, and its start.
    assertEquals(asyncThenSucceeded.start, "unscheduled-0-unset-unset-unset")
    assertEquals(
      asyncThenSucceeded.actions,
      Vector("schedule-unset-unset-unset", "handlerReply-async", "complete-succeeded")
    )
    // Each functional Query finds its claim on its path.
    for q <- functionalQueries do assertEquals(answer(q).outcome, Verdict.found, q.name)
    // A timer is named like any action, and a Scenario lists it where it fires.
    assertEquals(
      retriedThenSucceeded.actions,
      Vector(
        "schedule-unset-unset-unset",
        "handlerReply-handlerError-true",
        "backoff",
        "handlerReply-syncSuccess"
      )
    )
    assertEquals(
      scheduleToStartExpires.actions,
      Vector("schedule-unset-expires-unset", "workerStop", "scheduleToStart")
    )
    // The product claim is verified over every trace of the asynchronous path, and keeps its own
    // identity: it is the product Property and no other.
    assertEquals(answer(terminalHolds).outcome, Verdict.verifiedWithinLimits)
    assertEquals(
      (terminalHolds.property.name, terminalHolds.property.machine.name),
      ("terminalIsFinal", "nexusProduct")
    )
  }

  // The search keeps one fired bit per Property and visits 111 product states. The count is a
  // search statistic.
  test("the product Property over every trace within four") {
    val everywhere = nexusProtocol.scenario("everywhere").starts(unscheduled).free
    val a =
      answer(query("terminalHoldsEverywhere") verify terminalIsFinal in everywhere limits four)
    assertEquals((a.outcome, a.explored), (Verdict.verifiedWithinLimits, 111))
  }

  // A product Property about an action the protocol machine does not have cannot be read there.
  test("a product Property on a missing action") {
    val timesOut =
      nexusProduct.property("timesOut") when timeout holds (_.state.phase == ProductPhase.timedOut)
    val q = query("timesOutOnProtocol") verify timesOut in asyncThenSucceeded limits three
    assertEquals(
      q.answer.left.map(_.toString),
      Left(
        "query timesOutOnProtocol: the Property names the action 'timeout' of " +
          "'nexusProduct', and 'nexusProtocol' has no action of that name; a Property on the refined machine is read on " +
          "the refining one through the values of the same name, and a state through its map"
      )
    )
  }

  test("the composition") {
    assertEquals(table(handlerWorker).actions, Vector("serve", "workerStop"))
    val t = table(nexusCaller)
    // Every reachable protocol state under both worker phases.
    assertEquals(t.states.size, 316)
    assertEquals(t.states.count(_.endsWith("_stopped")), 158)
    assertEquals(t.rows.size, 1468)
    assertEquals(
      t.actions,
      Vector(
        "handlerReply-async",
        "handlerReply-handlerError-false",
        "handlerReply-handlerError-true",
        "handlerReply-operationCanceled",
        "handlerReply-operationFailed",
        "handlerReply-syncSuccess",
        "operation_backoff",
        "operation_complete-canceled",
        "operation_complete-failed",
        "operation_complete-succeeded",
        "operation_schedule-expires-expires-expires",
        "operation_schedule-expires-expires-unset",
        "operation_schedule-expires-unset-expires",
        "operation_schedule-expires-unset-unset",
        "operation_schedule-unset-expires-expires",
        "operation_schedule-unset-expires-unset",
        "operation_schedule-unset-unset-expires",
        "operation_schedule-unset-unset-unset",
        "operation_scheduleToClose",
        "operation_scheduleToStart",
        "operation_startToClose",
        "operation_transportFault",
        "workerStop"
      )
    )
    // A reply has a row only where the worker polls.
    val replies = t.rows.filter(_.key.contains("-handlerReply"))
    assertEquals(replies.size, 144)
    assert(replies.forall(!_.key.contains("_stopped-")))
    // Verified over the path: the one reply comes before the stop.
    assertEquals(answer(stoppedWorkerRepliesNothing).outcome, Verdict.verifiedWithinLimits)
  }

  test("every declaration passes the framework's semantic checks") {
    assertEquals(
      check(
        nexusProduct,
        nexusProtocol,
        handlerWorker,
        nexusCaller,
        terminalHolds,
        stoppedWorkerRepliesNothing
      ),
      Nil
    )
  }
