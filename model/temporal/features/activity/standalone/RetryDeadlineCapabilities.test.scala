package umpire

import temporal.capabilities.{Deadline, Retries}
import temporal.features.activity.{deadline as activityDeadline, Failure, TimeoutType}
import temporal.features.activity.standalone.{worker, MaxAttempts}
import temporal.features.activity.standalone.system as activity
import temporal.features.nexus.Reply
import temporal.features.nexus.workflow.{deadline as nexusDeadline, handler, network}
import temporal.features.nexus.workflow.system as nexus
import umpire.outcomes.Outcome

class CapabilityRetryDeadlineFeatures extends munit.FunSuite:
  private def fields(section: AnyRef, name: String): Map[String, Any] =
    val binding = section.getClass
      .getMethod(name)
      .invoke(section)
      .asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
    binding.productElementNames.zip(binding.productIterator).toMap

  test(
    "each declared failure and deadline binds its exact class classification and optional controls"
  ) {
    for (section, name, failure, retryable, controlled) <- Seq(
        (
          activity.ActivitySystem.capabilities,
          "retryableFailure",
          worker.respondFailed(Failure.retryable),
          true,
          true
        ),
        (
          activity.ActivitySystem.capabilities,
          "fatalFailure",
          worker.respondFailed(Failure.fatal),
          false,
          true
        ),
        (
          nexus.NexusSystem.capabilities,
          "retryableHandlerError",
          handler.reply(Reply.handlerError(true)),
          true,
          false
        ),
        (
          nexus.NexusSystem.capabilities,
          "fatalHandlerError",
          handler.reply(Reply.handlerError(false)),
          false,
          false
        ),
        (nexus.NexusSystem.capabilities, "networkFailure", network.fault, true, false)
      )
    do
      val binding = fields(section, name)
      assertEquals(binding("failure"), failure)
      assertEquals(binding("retryable"), retryable)
      assertEquals(binding("pendingPause") != None, controlled)
      assertEquals(binding("pendingCancel") != None, controlled)
    for (section, name, timer, timeout, retryable) <- Seq(
        (
          activity.ActivitySystem.capabilities,
          "scheduleToCloseDeadline",
          activityDeadline.scheduleToClose,
          activity.Fact.statusTimedOut(TimeoutType.scheduleToClose),
          false
        ),
        (
          activity.ActivitySystem.capabilities,
          "scheduleToStartDeadline",
          activityDeadline.scheduleToStart,
          activity.Fact.statusTimedOut(TimeoutType.scheduleToStart),
          false
        ),
        (
          activity.ActivitySystem.capabilities,
          "startToCloseDeadline",
          activityDeadline.startToClose,
          activity.Fact.statusTimedOut(TimeoutType.startToClose),
          true
        ),
        (
          activity.ActivitySystem.capabilities,
          "heartbeatDeadline",
          activityDeadline.heartbeat,
          activity.Fact.statusTimedOut(TimeoutType.heartbeat),
          true
        ),
        (
          nexus.NexusSystem.capabilities,
          "scheduleToCloseDeadline",
          nexusDeadline.scheduleToClose,
          nexus.Fact.nexusOperationTimedOut(nexus.TimeoutType.scheduleToClose),
          false
        ),
        (
          nexus.NexusSystem.capabilities,
          "scheduleToStartDeadline",
          nexusDeadline.scheduleToStart,
          nexus.Fact.nexusOperationTimedOut(nexus.TimeoutType.scheduleToStart),
          false
        ),
        (
          nexus.NexusSystem.capabilities,
          "startToCloseDeadline",
          nexusDeadline.startToClose,
          nexus.Fact.nexusOperationTimedOut(nexus.TimeoutType.startToClose),
          false
        )
      )
    do
      val binding = fields(section, name)
      assertEquals(binding("timer"), timer)
      assertEquals(binding("timeout"), timeout)
      assertEquals(binding("retryable"), retryable)
      assertEquals(binding("retriesRemaining") != None, retryable)
      assertEquals(binding("pendingPause") != None, retryable)
      assertEquals(binding("pendingCancel") != None, retryable)
  }

  private def take[S, F](
      machine: Machine[S, Outcome, F],
      before: S,
      action: Class
  ): List[Step[S, Outcome, F]] =
    val binding = machine.bindings.find(_.decl == action.decl).get
    effectOf(binding.decl, binding.function)(before, action.values)

  private def holds[S, O, F](
      properties: Seq[Property[S]],
      before: S,
      after: Step[S, O, F]
  ): Boolean =
    properties.forall { property =>
      property.decl.holds2 match
        case Some(predicate) =>
          predicate.asInstanceOf[(S, Step[S, O, F]) => Boolean](
            before,
            after
          ) // scalafix:ok DisableSyntax.asInstanceOf
        case None =>
          property.decl.holds.get
            .asInstanceOf[Step[S, O, F] => Boolean](after) // scalafix:ok DisableSyntax.asInstanceOf
    }

  test(
    "conditional companions exercise distinct fixture states and reject wrong landings and counts"
  ) {
    import RetryFixture.Record.given
    val finitePolicy =
      Retries.attemptCountIsWithinPolicy(RetryFixture.Record)(RetryFixture.count, RetryFixture.one)
    for (count, expected) <- Seq(1 -> true, 2 -> false) do
      val state = RetryFixture.Record.init.copy(attempts = UpTo[2](count))
      assertEquals(
        holds(Seq(finitePolicy), state, Step(RetryFixture.Answer.accepted, state)),
        expected
      )
    val retryProperties = Seq(
      Retries.failurePauses(RetryFixture.Record)(
        RetryFixture.failure,
        true,
        RetryFixture.remaining,
        RetryFixture.pendingPause,
        RetryFixture.pendingCancel
      ),
      Retries
        .failureCancels(RetryFixture.Record)(RetryFixture.failure, true, RetryFixture.pendingCancel)
    )
    for cancel <- Seq(false, true) do
      val before = RetryFixture.Record.init.copy(pause = true, cancel = cancel)
      val expected = if cancel then RetryFixture.Phase.canceled else RetryFixture.Phase.paused
      for landing <- summon[Finite[RetryFixture.Phase]].values do
        assertEquals(
          holds(
            retryProperties,
            before,
            Step(RetryFixture.Answer.accepted, before.copy(phase = landing))
          ),
          landing == expected
        )

    import DeadlineFixture.Record.given
    val deadlineProperties = Seq(
      Deadline.deadlineReturnsToWaiting(DeadlineFixture.Record)(
        DeadlineFixture.expired,
        true,
        DeadlineFixture.remaining,
        DeadlineFixture.timeoutFacts,
        DeadlineFixture.pendingPause,
        DeadlineFixture.pendingCancel
      ),
      Deadline.deadlinePauses(DeadlineFixture.Record)(
        DeadlineFixture.expired,
        true,
        DeadlineFixture.remaining,
        DeadlineFixture.pendingPause,
        DeadlineFixture.timeoutFacts,
        DeadlineFixture.pendingCancel
      )
    )
    for pause <- Seq(false, true) do
      val before = DeadlineFixture.Record.init.copy(pause = pause)
      val expected = if pause then DeadlineFixture.Phase.paused else DeadlineFixture.Phase.waiting
      for landing <- summon[Finite[DeadlineFixture.Phase]].values do
        assertEquals(
          holds(
            deadlineProperties,
            before,
            Step(DeadlineFixture.Answer.accepted, before.copy(phase = landing))
          ),
          landing == expected
        )
  }

  private def activityRetries(retryable: Boolean): Seq[Property[activity.State]] =
    import activity.ActivitySystem.given
    val machine = activity.ActivitySystem
    val failure = worker.respondFailed(if retryable then Failure.retryable else Failure.fatal)
    Seq(
      Retries.failureReturnsToWaiting(machine)(
        failure,
        retryable,
        machine.states.retriesRemaining,
        machine.states.pendingPause,
        machine.states.pendingCancel
      ),
      Retries.failureEndsFailed(machine)(
        failure,
        retryable,
        machine.states.retriesRemaining,
        machine.states.pendingCancel
      ),
      Retries.failurePauses(machine)(
        failure,
        retryable,
        machine.states.retriesRemaining,
        machine.states.pendingPause,
        machine.states.pendingCancel
      ),
      Retries.failureCancels(machine)(failure, retryable, machine.states.pendingCancel)
    )

  private def activityDeadlineSettlement(
      timer: ClassRef,
      timeout: activity.Fact,
      retryable: Boolean
  ): Seq[Property[activity.State]] =
    import activity.ActivitySystem.given
    val machine = activity.ActivitySystem
    val terminal = Deadline.deadlineTimesOut(machine)(
      timer,
      retryable,
      timeout,
      machine.states.retriesRemaining,
      machine.states.pendingCancel
    )
    if !retryable then Seq(terminal)
    else
      Seq(
        terminal,
        Deadline.deadlineReturnsToWaiting(machine)(
          timer,
          true,
          machine.states.retriesRemaining,
          machine.states.timeoutFacts,
          machine.states.pendingPause,
          machine.states.pendingCancel
        ),
        Deadline.deadlinePauses(machine)(
          timer,
          true,
          machine.states.retriesRemaining,
          machine.states.pendingPause,
          machine.states.timeoutFacts,
          machine.states.pendingCancel
        )
      )

  test(
    "Activity failure bindings preserve ordinary pause cancel fatal and exhaustion settlements"
  ) {
    for
      retryable <- Seq(false, true)
      policy <- MaxAttempts.values
      count <- 1 to MaxAttempts.bound
      phase <- Seq(
        activity.Phase.started,
        activity.Phase.pauseRequested,
        activity.Phase.cancelRequested
      )
    do
      val before = activity.ActivitySystem.init.copy(
        phase = phase,
        attempts = UpTo(count),
        maxAttempts = policy
      )
      val properties = activityRetries(retryable)
      val action = worker.respondFailed(if retryable then Failure.retryable else Failure.fatal)
      assert(properties.forall(_.decl.when.contains(action)))
      val after = take(activity.ActivitySystem, before, action).head
      assert(holds(properties, before, after), s"before=$before after=$after")
      for landing <- summon[Finite[activity.Phase]].values if landing != after.state.phase do
        assert(
          !holds(properties, before, after.copy(state = after.state.copy(phase = landing))),
          s"wrong landing before=$before landing=$landing"
        )
  }

  test("Nexus handler and network retry bindings remain unlimited at represented saturation") {
    import nexus.NexusSystem.given
    val machine = nexus.NexusSystem
    for (action, retryable) <- Seq(
        handler.reply(Reply.handlerError(true)) -> true,
        handler.reply(Reply.handlerError(false)) -> false,
        network.fault() -> true
      )
    do
      val properties = Seq(
        Retries
          .failureReturnsToWaiting(machine)(action, retryable, machine.states.retriesRemaining),
        Retries.failureEndsFailed(machine)(action, retryable, machine.states.retriesRemaining)
      )
      for count <- 0 to nexus.attemptBound do
        val before = machine.init.copy(phase = nexus.Phase.scheduled, attempts = count)
        val after = take(machine, before, action).head
        assert(holds(properties, before, after))
        assertEquals(
          after.state.attempts,
          if retryable then (count + 1).min(nexus.attemptBound) else count
        )
        for landing <- summon[Finite[nexus.Phase]].values do
          assertEquals(
            holds(properties, before, after.copy(state = after.state.copy(phase = landing))),
            if retryable then landing.in[Waiting] else landing.in[Failed]
          )
  }

  test("the finite maximum rejects count two under max one while unlimited accepts saturation") {
    val activityPolicy = Retries.attemptCountIsWithinPolicy(activity.ActivitySystem)(
      activity.ActivitySystem.states.attemptCount,
      activity.ActivitySystem.states.maximumAttempts
    )
    for (policy, expected) <- Seq(
        MaxAttempts.one -> false,
        MaxAttempts.two -> true,
        MaxAttempts.unlimited -> true
      )
    do
      val state = activity.ActivitySystem.init.copy(attempts = UpTo(2), maxAttempts = policy)
      assertEquals(holds(Seq(activityPolicy), state, Step(Outcome.accepted, state)), expected)
    val nexusPolicy = Retries.attemptCountIsWithinPolicy(nexus.NexusSystem)(
      nexus.NexusSystem.states.attemptCount,
      nexus.NexusSystem.states.maximumAttempts
    )
    val saturated = nexus.NexusSystem.init.copy(attempts = nexus.attemptBound)
    assert(holds(Seq(nexusPolicy), saturated, Step(Outcome.accepted, saturated)))
  }

  test("every native timer window rejects unarmed phases and Activity dispatch delays") {
    import activity.ActivitySystem.given
    val activityWindows = Seq(
      activityDeadline
        .scheduleToClose() -> Deadline.firesInWindow[activity.State, activity.Phase, Live](
        activity.ActivitySystem
      )(activityDeadline.scheduleToClose, activity.ActivitySystem.states.scheduleToCloseArmed),
      activityDeadline
        .scheduleToStart() -> Deadline.firesInWindow[activity.State, activity.Phase, Waiting](
        activity.ActivitySystem
      )(activityDeadline.scheduleToStart, activity.ActivitySystem.states.scheduleToStartArmed),
      activityDeadline
        .startToClose() -> Deadline.firesInWindow[activity.State, activity.Phase, Held](
        activity.ActivitySystem
      )(activityDeadline.startToClose, activity.ActivitySystem.states.startToCloseArmed),
      activityDeadline
        .heartbeat() -> Deadline.firesInWindow[activity.State, activity.Phase, Held](
        activity.ActivitySystem
      )(activityDeadline.heartbeat, activity.ActivitySystem.states.heartbeatArmed)
    )
    for before <- summon[Finite[activity.State]].values; (timer, property) <- activityWindows do
      val enabled = take(activity.ActivitySystem, before, timer).nonEmpty
      val unrelatedAfter = Step(Outcome.accepted, activity.ActivitySystem.init)
      assertEquals(holds(Seq(property), before, unrelatedAfter), enabled, s"$timer before=$before")

    import nexus.NexusSystem.given
    val nexusWindows = Seq(
      nexusDeadline.scheduleToClose() -> Deadline.firesInWindow[nexus.State, nexus.Phase, Live](
        nexus.NexusSystem
      )(nexusDeadline.scheduleToClose, nexus.NexusSystem.states.scheduleToCloseArmed),
      nexusDeadline.scheduleToStart() -> Deadline.firesInWindow[nexus.State, nexus.Phase, Waiting](
        nexus.NexusSystem
      )(nexusDeadline.scheduleToStart, nexus.NexusSystem.states.scheduleToStartArmed),
      nexusDeadline.startToClose() -> Deadline.firesInWindow[nexus.State, nexus.Phase, Held](
        nexus.NexusSystem
      )(nexusDeadline.startToClose, nexus.NexusSystem.states.startToCloseArmed)
    )
    for before <- summon[Finite[nexus.State]].values; (timer, property) <- nexusWindows do
      val enabled = take(nexus.NexusSystem, before, timer).nonEmpty
      assertEquals(
        holds(Seq(property), before, Step(Outcome.accepted, nexus.NexusSystem.init)),
        enabled,
        s"$timer before=$before"
      )
  }

  test("Activity deadlines reject extra retries wrong landing and terminal timeout-family facts") {
    val timers = Seq(
      activityDeadline.scheduleToClose() -> TimeoutType.scheduleToClose,
      activityDeadline.scheduleToStart() -> TimeoutType.scheduleToStart,
      activityDeadline.startToClose() -> TimeoutType.startToClose,
      activityDeadline.heartbeat() -> TimeoutType.heartbeat
    )
    for before <- summon[Finite[activity.State]].values; (timer, kind) <- timers do
      for after <- take(activity.ActivitySystem, before, timer) do
        val timeout = activity.Fact.statusTimedOut(kind)
        val properties =
          activityDeadlineSettlement(
            timer,
            timeout,
            kind == TimeoutType.startToClose || kind == TimeoutType.heartbeat
          )
        assert(holds(properties, before, after), s"$timer before=$before after=$after")
        for landing <- summon[Finite[activity.Phase]].values if landing != after.state.phase do
          assert(!holds(properties, before, after.copy(state = after.state.copy(phase = landing))))
        if after.state.phase == activity.Phase.timedOut then
          assert(!holds(properties, before, after.copy(facts = Nil)))
          for other <- activity.ActivitySystem.states.timeoutFacts if other != timeout do
            assert(!holds(properties, before, after.copy(facts = List(other))))
        else
          for terminalFact <- activity.ActivitySystem.states.timeoutFacts do
            assert(!holds(properties, before, after.copy(facts = after.facts :+ terminalFact)))
  }

  test("Nexus deadlines all settle terminally and require their own exact timeout fact") {
    import nexus.NexusSystem.given
    val timers = Seq(
      nexusDeadline.scheduleToClose() -> nexus.TimeoutType.scheduleToClose,
      nexusDeadline.scheduleToStart() -> nexus.TimeoutType.scheduleToStart,
      nexusDeadline.startToClose() -> nexus.TimeoutType.startToClose
    )
    for before <- summon[Finite[nexus.State]].values; (timer, kind) <- timers do
      for after <- take(nexus.NexusSystem, before, timer) do
        val timeout = nexus.Fact.nexusOperationTimedOut(kind)
        val property = Deadline.deadlineTimesOut(nexus.NexusSystem)(timer, false, timeout)
        assert(holds(Seq(property), before, after))
        for landing <- summon[Finite[nexus.Phase]].values if landing != nexus.Phase.timedOut do
          assert(
            !holds(Seq(property), before, after.copy(state = after.state.copy(phase = landing)))
          )
        assert(!holds(Seq(property), before, after.copy(facts = Nil)))
        for other <- nexus.NexusSystem.states.timeoutFacts if other != timeout do
          assert(!holds(Seq(property), before, after.copy(facts = List(other))))
  }
