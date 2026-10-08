package umpire

import temporal.capabilities.Deadline
import temporal.features.activity.{deadline, Timeout, TimeoutType}
import temporal.features.activity.standalone.{
  client,
  heartbeat,
  product,
  system,
  worker,
  MaxAttempts
}
import temporal.features.activity.standalone.system.ActivitySystem
import temporal.features.activity.standalone.system.{
  HeartbeatCompletion,
  HeartbeatExhaustion,
  HeartbeatRetry
}
import temporal.features.activity.standalone.system.{
  ExhaustAfterHeartbeat,
  HeartbeatThenCompletion,
  RetryAfterHeartbeat
}
import umpire.realize.{
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
import umpire.outcomes.{Outcome, Rejection}

class ActivityHeartbeatRegression extends munit.FunSuite:
  private def take(s: system.State, c: Class): List[Step[system.State, Outcome, system.Fact]] =
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get
    effectOf(binding.decl, binding.function)(s, c.values)

  test("heartbeat is an independently declared held action with a separate armed deadline") {
    assertEquals(worker.heartbeat.decl.name, "recordHeartbeat")
    assertNotEquals(worker.heartbeat.decl, deadline.heartbeat.decl)
    assert(
      ActivitySystem.bindings.exists(_.decl == worker.heartbeat.decl),
      "missing held heartbeat action"
    )
    assertEquals(ActivitySystem.init.productElementNames.toList.count(_ == "heartbeat"), 1)
    assertEquals(
      summon[Finite[temporal.features.activity.standalone.system.State]].values.size,
      4752
    )
  }

  test("heartbeat keeps the held attempt and rejects unheld or closed requests without receipt") {
    for before <- summon[Finite[system.State]].values do
      val after = take(before, worker.heartbeat()).head
      assertEquals(after.state, before)
      if before.phase.in[Held] then
        assertEquals(after.outcome, Outcome.accepted)
        assertEquals(after.facts, List(system.Fact.heartbeatReceived))
      else
        assertEquals(after.outcome, Outcome.rejected(Rejection.notFound))
        assertEquals(after.facts, Nil)
  }

  test("heartbeat deadline requires an armed held attempt and preserves retry control priority") {
    for before <- summon[Finite[system.State]].values do
      val steps = take(before, deadline.heartbeat())
      val enabled = before.heartbeat == Timeout.expires && before.phase.in[Held]
      assertEquals(steps.nonEmpty, enabled)
      for after <- steps do
        val retry = before.phase != system.Phase.cancelRequested &&
          (before.maxAttempts == MaxAttempts.unlimited ||
            (before.maxAttempts == MaxAttempts.two && before.attempts < 2) ||
            (before.maxAttempts == MaxAttempts.one && before.attempts < 1))
        val landing = if !retry then system.Phase.timedOut
        else if before.phase == system.Phase.pauseRequested then system.Phase.paused
        else system.Phase.scheduled
        assertEquals(after.state.phase, landing)
        assertEquals(after.state.attempts, before.attempts)
        assertEquals(after.state.heartbeat, Timeout.expires)
        val facts = if !retry then
          List(system.Fact.statusTimedOut(TimeoutType.heartbeat), system.Fact.heartbeatTimedOut)
        else
          List(
            if landing == system.Phase.paused then system.Fact.statusPaused
            else system.Fact.statusScheduled,
            system.Fact.attemptCount,
            system.Fact.heartbeatTimedOut
          )
        assertEquals(after.facts, facts)
        if retry then assertEquals(after.state.dispatch, system.Dispatch.backoff)
  }

  test("every public heartbeat fact has a Product carrier and refinements keep closedness") {
    assertEquals(Refinement.unclosed(ActivitySystem.refinement)(_.phase, _.phase), IndexedSeq.empty)
    for fact <- List(system.Fact.heartbeatReceived, system.Fact.heartbeatTimedOut) do
      assert(ActivitySystem.refinement.visible(fact))
    val held = ActivitySystem.init.copy(
      phase = system.Phase.started,
      attempts = UpTo(1),
      heartbeat = Timeout.expires
    )
    for action <- List(worker.heartbeat(), deadline.heartbeat()) do
      val after = take(held, action).head
      val binding = product.ActivityProduct.bindings.find(_.decl == action.decl).get
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        ActivitySystem.refinement.toProduct(held),
        action.values
      )
      assert(
        carriers.exists(p =>
          p.state == ActivitySystem.refinement.toProduct(after.state) &&
            p.facts.exists(_.toString == after.facts.last.toString)
        )
      )
  }

  test("complete Deadline family rejects a heartbeat terminal fact on another timer's retry") {
    import ActivitySystem.given
    assertEquals(
      ActivitySystem.states.timeoutFacts.toList,
      List(
        system.Fact.statusTimedOut(TimeoutType.scheduleToClose),
        system.Fact.statusTimedOut(TimeoutType.scheduleToStart),
        system.Fact.statusTimedOut(TimeoutType.startToClose),
        system.Fact.statusTimedOut(TimeoutType.heartbeat)
      )
    )
    val property = Deadline.deadlineReturnsToWaiting(ActivitySystem)(
      deadline.startToClose,
      true,
      ActivitySystem.states.retriesRemaining,
      ActivitySystem.states.timeoutFacts,
      ActivitySystem.states.pendingPause,
      ActivitySystem.states.pendingCancel
    )
    val holds = property.decl.holds2.get
      .asInstanceOf[(system.State, Step[system.State, Outcome, system.Fact]) => Boolean]
    // scalafix:ok DisableSyntax.asInstanceOf
    val before = ActivitySystem.init.copy(
      phase = system.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val after = take(before, deadline.startToClose()).head
    assert(holds(before, after))
    assert(
      !holds(
        before,
        after.copy(facts = after.facts :+ system.Fact.statusTimedOut(TimeoutType.heartbeat))
      )
    )
    val exhausted = before.copy(heartbeat = Timeout.expires, maxAttempts = MaxAttempts.one)
    val terminal = take(exhausted, deadline.heartbeat()).head
    val typed = Deadline.deadlineTimesOut(ActivitySystem)(
      deadline.heartbeat,
      true,
      system.Fact.statusTimedOut(TimeoutType.heartbeat),
      ActivitySystem.states.retriesRemaining,
      ActivitySystem.states.pendingCancel
    )
    val timeoutHolds = typed.decl.holds2.get
      .asInstanceOf[(system.State, Step[system.State, Outcome, system.Fact]) => Boolean]
    // scalafix:ok DisableSyntax.asInstanceOf
    assert(timeoutHolds(exhausted, terminal))
    assert(
      !timeoutHolds(
        exhausted,
        terminal.copy(facts = List(system.Fact.statusTimedOut(TimeoutType.startToClose)))
      )
    )
  }

  test("the start retains heartbeat independently of the other timers") {
    val scheduled = take(ActivitySystem.init, client.start(heartbeat := Timeout.expires)).head.state
    assertEquals(scheduled.heartbeat, Timeout.expires)
    assertEquals(scheduled.startToClose, Timeout.unset)
    assertEquals(scheduled.scheduleToClose, Timeout.unset)
  }

  test("heartbeat witnesses are pinned ordinary finds with strict satisfied expectations") {
    for (query, length) <- List(
        HeartbeatCompletion.queries.heartbeatThenCompletes -> 4,
        HeartbeatRetry.queries.heartbeatTimeoutRetriesThenCompletes -> 7,
        HeartbeatExhaustion.queries.heartbeatTimeoutExhausts -> 4
      )
    do
      assertEquals(query.form, QueryForm.find)
      assert(!query.scenario.free)
      assertEquals(query.scenario.actions.size, length)
      assertEquals(query.total, Some(4752L * length))
      assertEquals(query.expectedRun, Some(temporal.realize.satisfied))
  }

  test("local heartbeat invocation confirms only delivery and keeps attempt and delivery roles") {
    for declarations <- List(
        HeartbeatThenCompletion.evidence,
        RetryAfterHeartbeat.evidence,
        ExhaustAfterHeartbeat.evidence
      )
    do
      val delivery = declarations.items(1).asInstanceOf[TypedEvidence[InstructionOutcome]]
      // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(delivery.evidence.confirms, Vector(Taking(worker.poll, 1)))
      assertEquals(
        delivery.fields.flatMap(_.role).toSet,
        Set(FieldRole.attempt, FieldRole.delivery)
      )
      val record = delivery.evidence.from.asInstanceOf[Recorded.TypedRunEvent[InstructionOutcome]]
      // scalafix:ok DisableSyntax.asInstanceOf
      val invoked = record.guard.get.children.last
      assertEquals(invoked.value.get.operand, Operand.Literal(ProtoValue.Flag(true)))
      val absent =
        InstructionOutcome(activityAttempt = Some(ActivityAttempt(heartbeatInvoked = false)))
      val present =
        absent.copy(activityAttempt = Some(absent.getActivityAttempt.copy(heartbeatInvoked = true)))
      assertEquals[Any, Any](invoked.field.get.select(absent), false)
      assertEquals[Any, Any](invoked.field.get.select(present), true)
      val receipt = declarations.items(2).asInstanceOf[EvidenceRef[?, ?]].evidence
      // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(receipt.records, system.Fact.heartbeatReceived)
      assert(!receipt.exhaustive)
      assert(receipt.confirms.isEmpty)
  }

  test("heartbeat realizers retain a second-delivery count kind without another first record") {
    for declarations <- List(
        HeartbeatThenCompletion.evidence,
        RetryAfterHeartbeat.evidence,
        ExhaustAfterHeartbeat.evidence
      )
    do
      val count = declarations.items.collect {
        case delivery: TypedEvidence[?] if delivery.evidence.records == system.Fact.attemptCount =>
          delivery.evidence
      }
      assertEquals(count.size, 1)
      assert(count.head.confirms.isEmpty)
      val record = count.head.from.asInstanceOf[Recorded.TypedRunEvent[InstructionOutcome]]
      // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(record.attempt.map(_.number), Some(2L))
      val first = declarations.items
        .collect { case delivery: TypedEvidence[?] =>
          delivery.evidence.from
        }
        .collect {
          case record: Recorded.TypedRunEvent[?] if record.attempt.exists(_.number == 1) => record
        }
      assertEquals(first.size, 1)
  }

  test("first heartbeat delivery requires the authored local disposition enum") {
    for (declarations, expected) <- List(
        HeartbeatThenCompletion.evidence -> ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED,
        RetryAfterHeartbeat.evidence -> ACTIVITY_ATTEMPT_RESPONSE_PENDING,
        ExhaustAfterHeartbeat.evidence -> ACTIVITY_ATTEMPT_RESPONSE_PENDING
      )
    do
      val delivery = declarations.items(1).asInstanceOf[TypedEvidence[InstructionOutcome]]
      // scalafix:ok DisableSyntax.asInstanceOf
      val record = delivery.evidence.from.asInstanceOf[Recorded.TypedRunEvent[InstructionOutcome]]
      // scalafix:ok DisableSyntax.asInstanceOf
      assertEquals(record.guard.get.children.size, 4)
      val disposition = record.guard.get.children(2)
      assertEquals(disposition.value.get.operand, Operand.enumValue(expected).operand)
      val outcome = InstructionOutcome(activityAttempt = Some(ActivityAttempt(response = expected)))
      assertEquals[Any, Any](disposition.field.get.select(outcome), expected)
      val wrong = if expected == ACTIVITY_ATTEMPT_RESPONSE_PENDING then
        ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
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
    for controller <- List(RetryAfterHeartbeat.controller, ExhaustAfterHeartbeat.controller) do
      val polls = controller.items.flatMap(_.command.map(_.instruction)).collect {
        case poll: Instruction.TypedPoll[?, ?] => poll
      }
      assertEquals(polls.size, 2)
      val timeout = polls.last.asInstanceOf[Instruction.TypedPoll[?, ActivityExecutionInfo]]
      // scalafix:ok DisableSyntax.asInstanceOf
      val failure = timeout.until.children.last
      assertEquals(failure.value.get.operand, Operand.enumValue(TIMEOUT_TYPE_HEARTBEAT).operand)
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
      assertEquals[Any, Any](failure.field.get.select(matching), TIMEOUT_TYPE_HEARTBEAT)
      assertEquals[Any, Any](failure.field.get.select(wrongTimer), TIMEOUT_TYPE_START_TO_CLOSE)
      val receipt = polls.head.asInstanceOf[Instruction.TypedPoll[?, ActivityExecutionInfo]]
      // scalafix:ok DisableSyntax.asInstanceOf
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
