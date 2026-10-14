package framework

import temporal.features.activity.standalone.realization.*
import framework.Finite.given

import temporal.capabilities.system.Expirable
import temporal.features.activity.{deadline, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  activityWorker,
  client,
  heartbeat,
  product,
  protocol,
  MaxAttempts
}
import temporal.features.activity.standalone.protocol.ActivityProtocol

import framework.realize.{
  EvidenceRef,
  FieldRole,
  Instruction,
  Operand,
  ProtoValue,
  Recorded,
  Taking,
  TypedEvidence
}
import io.temporal.api.activity.v1.ActivityExecutionInfo
import io.temporal.api.failure.v1.{Failure as ApiFailure, TimeoutFailureInfo}
import io.temporal.api.enums.v1.ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED
import io.temporal.api.enums.v1.TimeoutType.{TIMEOUT_TYPE_HEARTBEAT, TIMEOUT_TYPE_START_TO_CLOSE}
import temporal.server.api.testpilot.v1.{ActivityAttempt, InstructionOutcome}
import temporal.server.api.testpilot.v1.ActivityAttemptResponse.{
  ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED,
  ACTIVITY_ATTEMPT_RESPONSE_PENDING
}
import framework.outcomes.{Outcome, Rejection}

class ActivityHeartbeatRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private def heartbeatTimeout(records: framework.realize.Fact): Boolean =
    records match
      case fact: protocol.Fact => fact == protocol.Fact.heartbeatTimedOut
      case _                   => false
  private def take(
      s: protocol.State,
      c: Class
  ): List[Step[protocol.State, Outcome, protocol.Fact]] =
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get
    effectOf(binding.decl, binding.function)(s, c.values)

  test(
    "heartbeat is an independently declared held action with a separate armed deadline"
  ) {
    assertEquals(activityWorker.heartbeat.decl.name, "recordHeartbeat")
    assertNotEquals(activityWorker.heartbeat.decl, deadline.heartbeat.decl)
    assert(
      ActivityProtocol.bindings.exists(
        _.decl == activityWorker.heartbeat.decl
      ),
      "missing held heartbeat action"
    )
    assertEquals(
      ActivityProtocol.init.productElementNames.toList.count(_ == "heartbeat"),
      1
    )
    assertEquals(
      summon[
        Finite[temporal.features.activity.standalone.protocol.State]
      ].values.size,
      5616
    )
  }

  test(
    "heartbeat keeps the held attempt and rejects unheld or closed requests without receipt"
  ) {
    for before <- summon[Finite[protocol.State]].values do
      val after = take(before, activityWorker.heartbeat()).head
      assertEquals(after.state, before)
      if before.phase.in[Held] then
        assertEquals(after.outcome, Outcome.accepted)
        assertEquals(after.facts, List(protocol.Fact.heartbeatReceived))
      else
        assertEquals(after.outcome, Outcome.rejected(Rejection.notFound))
        assertEquals(after.facts, Nil)
  }

  test(
    "heartbeat deadline requires an armed held attempt and preserves retry control priority"
  ) {
    for before <- summon[Finite[protocol.State]].values do
      val steps = take(before, deadline.heartbeat())
      val enabled =
        before.heartbeat == Timeout.expires && before.phase.in[Held]
      assertEquals(steps.nonEmpty, enabled)
      for after <- steps if ActivityProtocol.states.pendingReset(before) do
        val landing: protocol.Phase =
          if before.phase == protocol.Phase.resetKeepingPause then protocol.Phase.paused
          else protocol.Phase.scheduled
        assertEquals(
          after.state,
          before.copy(
            phase = landing,
            attempts = UpTo(0),
            dispatch = protocol.Dispatch.now
          )
        )
        assertEquals(
          after.facts,
          List(
            if landing == protocol.Phase.paused then protocol.Fact.statusPaused
            else protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount,
            protocol.Fact.heartbeatTimedOut
          )
        )
      for after <- steps if !ActivityProtocol.states.pendingReset(before) do
        val retry = before.phase != protocol.Phase.cancelRequested &&
          (before.maxAttempts == MaxAttempts.unlimited ||
            (before.maxAttempts == MaxAttempts.two && before.attempts < 2) ||
            (before.maxAttempts == MaxAttempts.one && before.attempts < 1))
        val landing = if !retry then protocol.Phase.timedOut
        else if before.phase == protocol.Phase.pauseRequested then protocol.Phase.paused
        else protocol.Phase.scheduled
        assertEquals(after.state.phase, landing)
        assertEquals(after.state.attempts, before.attempts)
        assertEquals(after.state.heartbeat, Timeout.expires)
        val facts = if !retry then
          List(
            protocol.Fact.statusTimedOut(TimeoutType.heartbeat),
            protocol.Fact.heartbeatTimedOut
          )
        else
          List(
            if landing == protocol.Phase.paused then protocol.Fact.statusPaused
            else protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount,
            protocol.Fact.heartbeatTimedOut
          )
        assertEquals(after.facts, facts)
        if retry then assertEquals(after.state.dispatch, protocol.Dispatch.backoff)
  }

  test(
    "every public heartbeat fact has a Product carrier and refinements keep closedness"
  ) {
    assertEquals(
      Refinement.unclosed(ActivityProtocol.refinement)(_.phase, _.phase),
      IndexedSeq.empty
    )
    for fact <- List(
        protocol.Fact.heartbeatReceived,
        protocol.Fact.heartbeatTimedOut
      )
    do assert(ActivityProtocol.refinement.visible(fact))
    val held = ActivityProtocol.init.copy(
      phase = protocol.Phase.started,
      attempts = UpTo(1),
      heartbeat = Timeout.expires
    )
    for action <- List(activityWorker.heartbeat(), deadline.heartbeat()) do
      val after = take(held, action).head
      val binding = product.ActivityProduct.bindings
        .find(_.decl == action.decl)
        .get
      val carriers = effectOf[product.State, Outcome, product.Fact](
        binding.decl,
        binding.function
      )(
        ActivityProtocol.refinement.toProduct(held),
        action.values
      )
      assert(
        carriers.exists(p =>
          p.state == ActivityProtocol.refinement.toProduct(
            after.state
          ) &&
            p.facts.exists(_.toString == after.facts.last.toString)
        )
      )
  }

  test(
    "complete Expirable family rejects a heartbeat terminal fact on another timer's retry"
  ) {
    import ActivityProtocol.given
    assertEquals(
      ActivityProtocol.states.timeoutFacts.toList,
      List(
        protocol.Fact.statusTimedOut(TimeoutType.scheduleToClose),
        protocol.Fact.statusTimedOut(TimeoutType.scheduleToStart),
        protocol.Fact.statusTimedOut(TimeoutType.startToClose),
        protocol.Fact.statusTimedOut(TimeoutType.heartbeat)
      )
    )
    val property = Expirable.deadlineReturnsToWaiting(ActivityProtocol)(
      deadline.startToClose,
      true,
      ActivityProtocol.states.retriesRemaining,
      ActivityProtocol.states.timeoutFacts,
      ActivityProtocol.states.pendingPause,
      ActivityProtocol.states.pendingCancel
    )
    val holds = property.decl.holds2.get
      .asInstanceOf[
        (
            protocol.State,
            Step[protocol.State, Outcome, protocol.Fact]
        ) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    val before = ActivityProtocol.init.copy(
      phase = protocol.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val after = take(before, deadline.startToClose()).head
    assert(holds(before, after))
    assert(
      !holds(
        before,
        after.copy(facts = after.facts :+ protocol.Fact.statusTimedOut(TimeoutType.heartbeat))
      )
    )
    val exhausted =
      before.copy(heartbeat = Timeout.expires, maxAttempts = MaxAttempts.one)
    val terminal = take(exhausted, deadline.heartbeat()).head
    val typed = Expirable.deadlineTimesOut(ActivityProtocol)(
      deadline.heartbeat,
      true,
      protocol.Fact.statusTimedOut(TimeoutType.heartbeat),
      ActivityProtocol.states.retriesRemaining,
      ActivityProtocol.states.pendingCancel
    )
    val timeoutHolds = typed.decl.holds2.get
      .asInstanceOf[
        (
            protocol.State,
            Step[protocol.State, Outcome, protocol.Fact]
        ) => Boolean
      ] // scalafix:ok DisableSyntax.asInstanceOf
    assert(timeoutHolds(exhausted, terminal))
    assert(
      !timeoutHolds(
        exhausted,
        terminal.copy(facts = List(protocol.Fact.statusTimedOut(TimeoutType.startToClose)))
      )
    )
  }

  test("the start retains heartbeat independently of the other timers") {
    val scheduled =
      take(
        ActivityProtocol.init,
        client.start(heartbeat := Timeout.expires)
      ).head.state
    assertEquals(scheduled.heartbeat, Timeout.expires)
    assertEquals(scheduled.startToClose, Timeout.unset)
    assertEquals(scheduled.scheduleToClose, Timeout.unset)
  }

  test(
    "heartbeat witnesses are pinned ordinary finds with strict satisfied expectations"
  ) {
    for (query, length) <- List(
        ActivityProtocol.heartbeat.queries.heartbeatThenCompletes -> 4,
        ActivityProtocol.heartbeat.queries.heartbeatTimeoutRetriesThenCompletes -> 7,
        ActivityProtocol.heartbeat.queries.heartbeatTimeoutExhausts -> 4
      )
    do
      assertEquals(query.form, QueryForm.find)
      assert(!query.scenario.free)
      assertEquals(query.scenario.actions.size, length)
      assertEquals(query.total, Some(5616L * length))
      assertEquals(query.expectedRun, Some(temporal.realize.satisfied))
  }

  test(
    "heartbeat exhaustion claim covers exhausted policies without contradicting eligible retries"
  ) {
    val declared =
      ActivityProtocol.heartbeat.queries.heartbeatTimeoutExhausts.property
    assert(
      declared.holds2.isEmpty,
      "the pinned FIND must retain a same-step claim"
    )
    val holds = declared.holds2 match
      case Some(predicate) =>
        predicate.asInstanceOf[
          (
              protocol.State,
              Step[protocol.State, Outcome, protocol.Fact]
          ) => Boolean
        ] // scalafix:ok DisableSyntax.asInstanceOf
      case None =>
        val predicate = declared.holds.get
          .asInstanceOf[Step[
            protocol.State,
            Outcome,
            protocol.Fact
          ] => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
        (
            _: protocol.State,
            after: Step[protocol.State, Outcome, protocol.Fact]
        ) => predicate(after)
    val rows = summon[Finite[protocol.State]].values.flatMap(before =>
      take(before, deadline.heartbeat()).map(after => before -> after)
    )
    // A pending reset applies as the deadline ends the attempt, whatever the policy left: the
    // frame restarts the count, so eligibility holds after the step even where it did not before.
    val resets =
      rows.filter((before, _) => ActivityProtocol.states.pendingReset(before))
    for (before, after) <- resets do
      assertEquals(after.state.attempts: Int, 0)
      assertEquals(after.state.maxAttempts, before.maxAttempts)
      assert(ActivityProtocol.states.retriesRemaining(after.state))
      assert(holds(before, after), s"legitimate reset row: $before -> $after")
      assert(after.records(protocol.Fact.heartbeatTimedOut))
      assert(
        !holds(
          before,
          after.copy(facts = after.facts.filterNot(_ == protocol.Fact.heartbeatTimedOut))
        )
      )
    assertEquals(resets.size, 432)
    assertEquals(
      resets.count((before, _) => !ActivityProtocol.states.retriesRemaining(before)),
      144
    )
    for (before, after) <- rows if !ActivityProtocol.states.pendingReset(before)
    do
      assertEquals(after.state.attempts, before.attempts)
      assertEquals(after.state.maxAttempts, before.maxAttempts)
      assertEquals(
        ActivityProtocol.states.retriesRemaining(after.state),
        ActivityProtocol.states.retriesRemaining(before)
      )
      assert(
        holds(before, after),
        s"legitimate heartbeat row: $before -> $after"
      )
      assert(after.records(protocol.Fact.heartbeatTimedOut))
      assert(
        !holds(
          before,
          after.copy(facts = after.facts.filterNot(_ == protocol.Fact.heartbeatTimedOut))
        ),
        "heartbeat evidence is mandatory for eligible and exhausted policies"
      )
      if !ActivityProtocol.states.retriesRemaining(before) then
        assertEquals(after.state.phase, protocol.Phase.timedOut)
        assert(
          !holds(
            before,
            after.copy(state = after.state.copy(phase = protocol.Phase.scheduled))
          )
        )
        assert(
          !holds(
            before,
            after.copy(facts =
              List(
                protocol.Fact.heartbeatTimedOut,
                protocol.Fact.statusTimedOut(TimeoutType.startToClose)
              )
            )
          )
        )
      else if before.phase != protocol.Phase.cancelRequested then
        assert(
          !holds(
            before.copy(maxAttempts = MaxAttempts.one, attempts = UpTo(1)),
            after.copy(state =
              after.state
                .copy(maxAttempts = MaxAttempts.one, attempts = UpTo(1))
            )
          ),
          "an exhausted policy cannot authorize this retry landing"
        )
    val ordinary = rows.filterNot((before, _) => ActivityProtocol.states.pendingReset(before))
    assertEquals(
      ordinary.count((before, _) => !ActivityProtocol.states.retriesRemaining(before)),
      216
    )
    assertEquals(ordinary.size, 648)
    assertEquals(rows.size, 1080)
    assertEquals(
      ordinary.count((before, _) =>
        ActivityProtocol.states.retriesRemaining(
          before
        ) && before.phase != protocol.Phase.cancelRequested
      ),
      288
    )
    assertEquals(
      rows.count((before, _) =>
        ActivityProtocol.states.retriesRemaining(
          before
        ) && before.phase == protocol.Phase.cancelRequested
      ),
      144
    )
  }

  test(
    "local heartbeat invocation confirms only delivery and keeps attempt and delivery roles"
  ) {
    for declarations <- List(
        HeartbeatThenCompletionRun.evidence,
        RetryAfterHeartbeatRun.evidence,
        ExhaustAfterHeartbeatRun.evidence
      )
    do
      val delivery = declarations
        .items(1)
        .asInstanceOf[TypedEvidence[
          InstructionOutcome
        ]] // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(
        delivery.evidence.confirms,
        Vector(Taking(activityWorker.poll, 1))
      )
      assertEquals(
        delivery.fields.flatMap(_.role).toSet,
        Set(FieldRole.attempt, FieldRole.delivery)
      )
      val record = delivery.evidence.from.asInstanceOf[Recorded.TypedRunEvent[
        InstructionOutcome
      ]] // scalafix:ok DisableSyntax.asInstanceOf
      val invoked = record.guard.get.children.last
      assertEquals(
        invoked.value.get.operand,
        Operand.Literal(ProtoValue.Flag(true))
      )
      val absent =
        InstructionOutcome(activityAttempt = Some(ActivityAttempt(heartbeatInvoked = false)))
      val present =
        absent.copy(activityAttempt = Some(absent.getActivityAttempt.copy(heartbeatInvoked = true)))
      assertEquals[Any, Any](invoked.field.get.select(absent), false)
      assertEquals[Any, Any](invoked.field.get.select(present), true)
      val receipt = declarations
        .items(2)
        .asInstanceOf[EvidenceRef[?, ?]]
        .evidence // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(receipt.records, protocol.Fact.heartbeatReceived)
      assert(!receipt.exhaustive)
      assert(receipt.confirms.isEmpty)
  }

  test(
    "heartbeat realizers retain a second-delivery count kind without another first record"
  ) {
    for declarations <- List(
        HeartbeatThenCompletionRun.evidence,
        RetryAfterHeartbeatRun.evidence,
        ExhaustAfterHeartbeatRun.evidence
      )
    do
      val count = declarations.items.collect {
        case delivery: TypedEvidence[?]
            if delivery.evidence.id == temporal.realize.evidenceId(
              protocol.Fact.attemptCount
            ) =>
          delivery.evidence
      }
      assertEquals(count.size, 1)
      assert(count.head.confirms.isEmpty)
      val record = count.head.from.asInstanceOf[Recorded.TypedRunEvent[
        InstructionOutcome
      ]] // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(record.attempt.map(_.number), Some(2L))
      val first = declarations.items
        .collect { case delivery: TypedEvidence[?] =>
          delivery.evidence.from
        }
        .collect {
          case record: Recorded.TypedRunEvent[?] if record.attempt.exists(_.number == 1) =>
            record
        }
      assertEquals(first.size, 1)
  }

  test(
    "heartbeat retry confirms both timer and second poll with their common attempt fact"
  ) {
    val selected = RetryAfterHeartbeatRun.evidence.items.collect {
      case evidence: TypedEvidence[?]
          if evidence.evidence.confirms.nonEmpty &&
            evidence.evidence.id == temporal.realize.evidenceId(
              protocol.Fact.heartbeatTimedOut
            ) =>
        evidence.evidence
    }.head
    assertEquals(selected.records, protocol.Fact.attemptCount)
    assertEquals(
      selected.confirms,
      Vector(Taking(deadline.heartbeat, 1), Taking(activityWorker.poll, 2))
    )
    val started = ActivityProtocol.init.copy(
      phase = protocol.Phase.started,
      attempts = UpTo(1),
      heartbeat = Timeout.expires,
      maxAttempts = MaxAttempts.two
    )
    val expired = take(started, deadline.heartbeat()).head
    val second =
      take(
        expired.state.copy(dispatch = protocol.Dispatch.now),
        activityWorker.poll()
      ).head
    assert(expired.facts.contains(selected.records))
    assert(second.facts.contains(selected.records))
    val unusedKind = RetryAfterHeartbeatRun.evidence.items.collect {
      case evidence: TypedEvidence[?] if heartbeatTimeout(evidence.evidence.records) =>
        evidence.evidence
    }
    assertEquals(unusedKind.size, 1)
    assert(unusedKind.head.confirms.isEmpty)
    assertNotEquals(unusedKind.head.id, selected.id)
  }

  test(
    "heartbeat exhaustion declares the complete timeout kind but selects its typed timer read"
  ) {
    val reads = ExhaustAfterHeartbeatRun.evidence.items.collect {
      case evidence: EvidenceRef[?, ?] =>
        evidence.evidence
    }
    val timeout = reads.find(
      _.records eq temporal.realize.everyValue(protocol.Fact.statusTimedOut)
    )
    assert(
      timeout.nonEmpty,
      "missing generic statusTimedOut kind for the terminal heartbeat row"
    )
    assert(timeout.get.confirms.isEmpty)
    val heartbeat = reads.find(read => heartbeatTimeout(read.records)).get
    assertEquals(heartbeat.confirms, Vector(Taking(deadline.heartbeat, 1)))
  }

  test(
    "first heartbeat delivery requires the authored local disposition enum"
  ) {
    for (declarations, expected) <- List(
        HeartbeatThenCompletionRun.evidence -> ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED,
        RetryAfterHeartbeatRun.evidence -> ACTIVITY_ATTEMPT_RESPONSE_PENDING,
        ExhaustAfterHeartbeatRun.evidence -> ACTIVITY_ATTEMPT_RESPONSE_PENDING
      )
    do
      val delivery = declarations
        .items(1)
        .asInstanceOf[TypedEvidence[
          InstructionOutcome
        ]] // scalafix:ok DisableSyntax.asInstanceOf
      val record = delivery.evidence.from.asInstanceOf[Recorded.TypedRunEvent[
        InstructionOutcome
      ]] // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(record.guard.get.children.size, 4)
      val disposition = record.guard.get.children(2)
      assertEquals(
        disposition.value.get.operand,
        Operand.enumValue(expected).operand
      )
      val outcome = InstructionOutcome(activityAttempt = Some(ActivityAttempt(response = expected)))
      assertEquals[Any, Any](disposition.field.get.select(outcome), expected)
      val wrong =
        if expected.value == ACTIVITY_ATTEMPT_RESPONSE_PENDING.value
        then ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
        else ACTIVITY_ATTEMPT_RESPONSE_PENDING
      assertEquals[Any, Any](
        disposition.field.get.select(
          outcome.copy(activityAttempt = Some(outcome.getActivityAttempt.copy(response = wrong)))
        ),
        wrong
      )
  }

  test(
    "heartbeat timeout reads require actual typed HEARTBEAT failure rather than retry inference"
  ) {
    for controller <- List(
        RetryAfterHeartbeatRun.controller,
        ExhaustAfterHeartbeatRun.controller
      )
    do
      val polls =
        controller.items.flatMap(_.command.map(_.instruction)).collect {
          case poll: Instruction.TypedPoll[?, ?] => poll
        }
      assertEquals(polls.size, 2)
      val timeout = polls.last.asInstanceOf[
        Instruction.TypedPoll[?, ActivityExecutionInfo]
      ] // scalafix:ok DisableSyntax.asInstanceOf
      val failure = timeout.until.children.last
      assertEquals(
        failure.value.get.operand,
        Operand.enumValue(TIMEOUT_TYPE_HEARTBEAT).operand
      )
      val matching = ActivityExecutionInfo(
        status = ACTIVITY_EXECUTION_STATUS_COMPLETED,
        lastFailure = Some(
          ApiFailure().withTimeoutFailureInfo(
            TimeoutFailureInfo(timeoutType = TIMEOUT_TYPE_HEARTBEAT)
          )
        )
      )
      val wrongTimer = matching.copy(lastFailure =
        Some(
          ApiFailure().withTimeoutFailureInfo(
            TimeoutFailureInfo(timeoutType = TIMEOUT_TYPE_START_TO_CLOSE)
          )
        )
      )
      assertEquals[Any, Any](
        failure.field.get.select(matching),
        TIMEOUT_TYPE_HEARTBEAT
      )
      assertEquals[Any, Any](
        failure.field.get.select(wrongTimer),
        TIMEOUT_TYPE_START_TO_CLOSE
      )
      val receipt = polls.head.asInstanceOf[
        Instruction.TypedPoll[?, ActivityExecutionInfo]
      ] // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(receipt.until.children.size, 2)
      assertEquals(
        receipt.until.children.head.value.get.operand,
        Operand.Literal(ProtoValue.Number(0))
      )
      assertEquals[Any, Any](
        receipt.until.children.last.field.get.select(ActivityExecutionInfo()),
        None
      )
  }
