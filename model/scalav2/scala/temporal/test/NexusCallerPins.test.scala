package temporal
package nexuscaller
// What the Caller Model says, pinned the way model/lean/Temporal/Feature/Nexus/Caller/Tests.lean
// pins it and model/go/nexuscaller/pins_test.go translates it. Each assertion cites the Lean pin.
// Pins about Lean internals with no Scala counterpart are listed at the end, with the reason; the
// Case pins live with the Case producer.

import umpire.*

class NexusCallerPins extends munit.FunSuite:
  def table(m: Model): Table = m.table.fold(e => fail(e.toString), identity)
  def answer(q: Query): Answer = q.answer.fold(e => fail(e.toString), identity)

  /**
   * A state written the way a reader names one: the phase, and whichever fields are not at the
   * value the operation begins with (Tests.lean `at'`).
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
    // Six phases, and the four the design ends on. (Tests.lean:23-24)
    assertEquals((t.states.size, t.ends.size), (6, 4))
    // Six replies, three resolutions, the two faults it cannot see, and the one timer. (:28)
    assertEquals(t.actions.size, 12)
    // A retryable handler error is invisible here: it is the protocol machine that backs off. (:31)
    assertEquals(
      Product.handlerReplyStep(ProductState(ProductPhase.scheduled), Reply.handlerError(true)),
      Nil
    )
    // What the Model actually reaches: every phase. (:34-36)
    assertEquals(
      t.reachable,
      Vector("scheduled", "canceled", "failed", "succeeded", "started", "timedOut")
    )
    // (:40)
    assertEquals(t.stuck, None)
  }

  test("the protocol machine") {
    val t = table(nexusProtocol)
    val n = Protocol.attemptBound + 1
    // Eight phases, three attempt counts and three deadlines, and the four ending phases. (:52-53)
    assertEquals((t.states.size, t.ends.size), (8 * n * 8, 4 * n * 8))
    // Eight schedule commands, six replies, three resolutions, the two faults and the four timers.
    // The catalog is in canonical order, so it opens on the backoff timer. (:58-59)
    assertEquals(t.actions.size, 8 + 6 + 3 + 1 + 1 + 4)
    assertEquals(t.actions.take(2), Vector("backoff", "complete-canceled"))
    // The machine begins before the operation exists, with every deadline at its first value. (:62)
    assertEquals(t.starts, Vector(Keys.of(at(Phase.unscheduled))))
    // A retryable handler error backs the operation off and raises the attempt count. (:69-70)
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
    // The count saturates rather than wrapping. (:73-74)
    assertEquals(
      Protocol
        .handlerReplyStep(at(Phase.scheduled, attempts = 2), Reply.handlerError(true))
        .map(_.state.attempts),
      List(2)
    )
    // A completion before the start records the Started event first, one after it does not. (:78-81)
    assertEquals(
      Protocol.completeStep(at(Phase.backingOff, attempts = 1), Resolution.succeeded).map(_.facts),
      List(List(ProtocolFact.nexusOperationStarted, ProtocolFact.nexusOperationCompleted))
    )
    assertEquals(
      Protocol.completeStep(at(Phase.started), Resolution.succeeded).map(_.facts),
      List(List(ProtocolFact.nexusOperationCompleted))
    )
    // A completion after the operation is over is not found and changes nothing. (:84-85)
    assertEquals(
      Protocol.completeStep(at(Phase.timedOut), Resolution.succeeded),
      List(Step(Outcome.notFound, at(Phase.timedOut), Nil))
    )
    // A timer fires only when the schedule command set it, and each covers its own span. (:88-90)
    assertEquals(Protocol.startToCloseStep(at(Phase.scheduled, stc = Timeout.expires)), Nil)
    assertEquals(
      Protocol.startToCloseStep(at(Phase.started, stc = Timeout.expires)).map(_.state.phase),
      List(Phase.timedOut)
    )
    assertEquals(Protocol.scheduleToCloseStep(at(Phase.started)), Nil)
    // Which timer fired is recorded. (:93-94)
    assertEquals(
      Protocol.scheduleToStartStep(at(Phase.scheduled, sts = Timeout.expires)).map(_.facts),
      List(List(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)))
    )
    // The worker stopping keeps the state and records nothing; the product machine does not see it. (:98-100)
    assertEquals(
      Protocol.workerStopStep(at(Phase.scheduled, sts = Timeout.expires)),
      List(Step(Outcome.accepted, at(Phase.scheduled, sts = Timeout.expires), Nil))
    )
    assertEquals(Product.workerStopStep(ProductState(ProductPhase.scheduled)), Nil)
    // Nothing is stuck. (:103)
    assertEquals(t.stuck, None)
    // Not every state is reachable. The Behavior Fingerprint reads the table, so this number is part
    // of the Model's identity. (:108)
    assertEquals(t.reachable.size, 158)
  }

  test("the refinement") {
    val ref = nexusProtocol.refinementCheck.fold(e => fail(e.toString), identity) // (:118)
    assertEquals(ref.rows.size, table(nexusProtocol).rows.size) // (:119)
    val lookup = ref.rows.map(r => r.key -> r.product).toMap
    // A reply the product machine sees is that reply's step. A retry it cannot see is a stutter, and
    // so are the schedule command and the backoff timer. (:123-129)
    assertEquals(
      lookup("scheduled-0-unset-unset-unset-handlerReply-async"),
      Some("handlerReply-async")
    )
    assertEquals(lookup("scheduled-0-unset-unset-unset-handlerReply-handlerError-true"), None)
    assertEquals(lookup("unscheduled-0-unset-unset-unset-schedule-unset-unset-expires"), None)
    assertEquals(lookup("backingOff-1-unset-unset-unset-backoff"), None)
    // A deadline firing is the product's one timer, whichever deadline it was. (:132-133)
    assertEquals(lookup("started-0-unset-unset-expires-startToClose"), Some("timeout"))
    // A completion before the start records the Started event first and still carries the step. (:136-137)
    assertEquals(
      lookup("backingOff-1-unset-unset-unset-complete-succeeded"),
      Some("complete-succeeded")
    )
    // Every schedule command, every retry, every backoff and every worker stop. (:141)
    val stutters = ref.rows.filter(_.product.isEmpty)
    assertEquals(stutters.size, 24 * 8 + 24 + 24 + 24 + 192)
    // Stutter invariance: every stutter leaves a phase that reads as scheduled, or is a worker stop. (:153-155)
    for r <- stutters do
      assert(
        Set("unscheduled", "scheduled", "backingOff")(Keys.actionName(r.key)) || r.key.endsWith(
          "-workerStop"
        ),
        r.key
      )
    // The product state a protocol state reads as is a field named after the product machine. (:159-160)
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
    // A protocol Scenario names its classed actions with their inputs, and its start. (:170-172)
    assertEquals(asyncThenSucceeded.start, "unscheduled-0-unset-unset-unset")
    assertEquals(
      asyncThenSucceeded.actions,
      Vector("schedule-unset-unset-unset", "handlerReply-async", "complete-succeeded")
    )
    // Each functional Query finds its claim on its path. (:176-196)
    for q <- functionalQueries do assertEquals(answer(q).outcome, Verdict.found, q.name)
    // A timer is named like any action, and a Scenario lists it where it fires. (:199-203)
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
    // identity: it is the product Property and no other. (:207-211)
    assertEquals(answer(terminalHolds).outcome, Verdict.verifiedWithinLimits)
    assertEquals(
      (terminalHolds.property.name, terminalHolds.property.machine.name),
      ("terminalIsFinal", "nexusProduct")
    )
  }

  // Both searches verify it; they count different product states. Lean lowers terminalIsFinal into
  // four clause groups and keeps a fired bit per group, so Veil visits 171 states; Go and Scala keep
  // one fired bit per Property and visit 111. The count is a search statistic. (:228-230)
  test("the product Property over every trace within four") {
    val everywhere = nexusProtocol.scenario("everywhere").starts(unscheduled).free
    val a =
      answer(query("terminalHoldsEverywhere") verify terminalIsFinal in everywhere limits four)
    assertEquals((a.outcome, a.explored), (Verdict.verifiedWithinLimits, 111))
  }

  // A product Property about an action the protocol machine does not have cannot be read there. (:233-245)
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

  test("the sets") {
    // (:254-256)
    assertEquals(nexusCallerCanary.purpose, Purpose.canary)
    assertEquals(
      nexusCallerCanary.bindings,
      Map(
        caller -> Binding.driven,
        handler -> Binding.observed,
        network -> Binding.observed,
        worker.party -> Binding.driven
      )
    )
    assertEquals(check(nexusCallerCanary), Nil)
    // A Query whose path takes a silent step carries a gap no deployment closes. (:262-277)
    val canaryRetry = nexusCallerCanary.copy(name = "canaryRetry", queries = Vector(retry))
    assertEquals(
      check(canaryRetry).map(_.toString),
      List("set canaryRetry: retry takes the silent step backoff, a gap no deployment closes")
    )
    // The exploration covers the protocol machine under four. (:282-306)
    assertEquals(nexusCallerExploration.machine.map(_.name), Some("nexusProtocol"))
    assertEquals(table(nexusProtocol).rows.size, 1152)
    val targets = nexusCallerExploration.targets.fold(e => fail(e.toString), identity)
    assertEquals(targets.size, 885 + 2 + 2)
    assertEquals(targets.map(_.kind).distinct, Vector("row", "result", "classMember"))
    assertEquals(targets.count(_.kind == "row"), 885)
    assertEquals(
      targets.filter(_.kind == "result").map(_.outcome),
      Vector(
        "temporal.nexus.caller.outcome.nexusProtocol.accepted",
        "temporal.nexus.caller.outcome.nexusProtocol.notFound"
      )
    )
    assertEquals(
      targets(targets.size - 2),
      CoverageTarget(
        "classMember",
        member = "temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-false",
        action = "temporal.nexus.caller.action.handlerReply",
        field = "reply",
        className = "handlerError (retryable := false)",
        example = "BadRequest"
      )
    )
  }

  test("the composition") {
    assertEquals(table(handlerWorker).actions, Vector("serve", "workerStop")) // (:435)
    val t = table(nexusCaller)
    // Every reachable protocol state under both worker phases. (:437-439)
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
    ) // (:440-450)
    // A reply has a row only where the worker polls. (:453-456)
    val replies = t.rows.filter(_.key.contains("-handlerReply"))
    assertEquals(replies.size, 144)
    assert(replies.forall(!_.key.contains("_stopped-")))
    // Verified over the path: the one reply comes before the stop. (:470-472)
    assertEquals(answer(stoppedWorkerRepliesNothing).outcome, Verdict.verifiedWithinLimits)
  }

  test("every declaration passes the checks the Lean elaborator runs") {
    assertEquals(
      check(
        nexusProduct,
        nexusProtocol,
        handlerWorker,
        nexusCaller,
        nexusCallerTests,
        nexusCallerCanary,
        nexusCallerExploration,
        terminalHolds,
        stoppedWorkerRepliesNothing
      ),
      Nil
    )
  }

// Lean pins with no Scala counterpart, and why (the same list as the Go pins):
//
//   - assert_axioms (:38, :110, :162, :458-459): kernel axiom inventories. The Stainless lemmas in
//     proofs/temporal/NexusLemmas.scala are the nearest counterpart, and they cover every state rather than
//     the enumerated table.
//   - terminalIsFinal.names.groups.length == 4 (:166) and repliedByPollingWorker.names.groups
//     (:463-467): Scala keeps the predicate as a function, so there are no groups to count.
//   - The reference backend's path counts (:223-230): Scala has one backend, the breadth-first
//     product search; its product-state counts are pinned above.
//   - The Case production pins (:257, :310-360): CaseBytes.test.scala checks every produced Case byte
//     for byte against the checked-in fixture.
