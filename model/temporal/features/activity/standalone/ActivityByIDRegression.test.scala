package framework

import framework.Finite.given

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{
  activity,
  activityWorker,
  client,
  completionClient,
  product,
  protocol
}
import temporal.features.activity.standalone.protocol.ActivityProtocol
import framework.outcomes.{Outcome, Rejection}

class ActivityByIDRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  test(
    "the completion client declares three independent by-ID answer actions"
  ) {
    val answers =
      ActivityProtocol.bindings.filter(
        _.decl.actor.name == "completionClient"
      )
    assertEquals(
      answers.map(_.decl),
      List(
        completionClient.respondCanceledByID.decl,
        completionClient.respondCompletedByID.decl,
        completionClient.respondFailedByID.decl
      )
    )
    for answer <- answers do assertEquals(answer.decl.on, Some(activity))
    assertEquals(
      completionClient.respondFailedByID.decl.tokens,
      List(Some(failure))
    )
    assertEquals(
      completionClient.respondFailedByID.decl.domains.map(_.values.toList),
      List(Failure.values.toList)
    )
    assertNotEquals(
      completionClient.respondCompletedByID.decl,
      activityWorker.respondCompleted.decl
    )
    assertNotEquals(
      completionClient.respondFailedByID.decl,
      activityWorker.respondFailed.decl
    )
    assertNotEquals(
      completionClient.respondCanceledByID.decl,
      activityWorker.respondCanceled.decl
    )
  }

  private def take(
      s: protocol.State,
      action: ClassRef
  ): Step[protocol.State, Outcome, protocol.Fact] =
    val c = action match
      case c: Class              => c
      case a: Action[EmptyTuple] => a()
    val binding = ActivityProtocol.bindings.find(_.decl == c.decl).get
    effectOf[protocol.State, Outcome, protocol.Fact](
      binding.decl,
      binding.function
    )(
      s,
      c.values
    ).head

  test(
    "every completion client result is carried by Product without hiding a public status"
  ) {
    val classes = List(
      completionClient.respondCompletedByID(),
      completionClient.respondFailedByID(Failure.fatal),
      completionClient.respondFailedByID(Failure.retryable),
      completionClient.respondCanceledByID()
    )
    for
      before <- summon[Finite[protocol.State]].values
      c <- classes
    do
      val after = take(before, c)
      val visible =
        after.facts.filter(ActivityProtocol.refinement.visible).map(_.toString)
      val source = ActivityProtocol.refinement.toProduct(before)
      val target = ActivityProtocol.refinement.toProduct(after.state)
      val binding =
        product.ActivityProduct.bindings.find(_.decl == c.decl).get
      val carriers = effectOf[product.State, Outcome, product.Fact](
        binding.decl,
        binding.function
      )(
        source,
        c.values
      )
      assert(
        carriers.exists(p =>
          p.outcome == after.outcome && p.state == target && p.facts.map(
            _.toString
          ) == visible
        ) ||
          (before.phase == protocol.Phase.unstarted && source == target && visible.isEmpty),
        s"completion client refinement $before -> $after"
      )
      // A failure answer applies a pending reset, which restarts the count; nothing else moves it.
      val restarts = ActivityProtocol.states.pendingReset(before) &&
        c.decl == completionClient.respondFailedByID.decl
      assertEquals(
        after.state.attempts: Int,
        if restarts then 0 else before.attempts
      )
      assertEquals(after.state.scheduleToClose, before.scheduleToClose)
      assertEquals(after.state.scheduleToStart, before.scheduleToStart)
      assertEquals(after.state.startToClose, before.startToClose)
      assertEquals(after.state.heartbeat, before.heartbeat)
      assertEquals(after.state.maxAttempts, before.maxAttempts)
  }

  test("cancellation without held work settles before any by-ID disposition") {
    for
      before <- summon[Finite[protocol.State]].values
      if before.phase == protocol.Phase.scheduled || before.phase == protocol.Phase.paused
    do
      val after = take(before, client.requestCancel)
      assertEquals(after.state, before.copy(phase = protocol.Phase.canceled))
      assertEquals(after.facts, List(protocol.Fact.statusCanceled))
      assertEquals(after.outcome, Outcome.accepted)
      val binding = product.ActivityProduct.bindings
        .find(_.decl == client.requestCancel.decl)
        .get
      val carriers = effectOf[product.State, Outcome, product.Fact](
        binding.decl,
        binding.function
      )(
        ActivityProtocol.refinement.toProduct(before),
        Nil
      )
      assertEquals(
        carriers.map(_.state),
        List(product.State(product.Phase.canceled))
      )
      assertEquals(carriers.map(_.outcome), List(Outcome.accepted))
      assertEquals(
        carriers.map(_.facts),
        List(List(product.Fact.statusCanceled))
      )
      for answer <- List(
          completionClient.respondFailedByID(Failure.fatal),
          completionClient.respondCanceledByID()
        )
      do
        val refused = take(after.state, answer)
        assertEquals(refused.state, after.state)
        assertEquals(refused.facts, Nil)
        assertEquals(refused.outcome, Outcome.rejected(Rejection.notFound))
  }

  test(
    "the preserved cancel-request claim uses its complete held completion client witness"
  ) {
    val query = ActivityProtocol.cancellation.queries.cancelIsRequested
    assertEquals(query.name, "activityProtocol.cancelIsRequested")
    assert(
      query.property eq ActivityProtocol.cancellation.properties.safety.cancelRequested.decl
    )
    assertEquals(query.property.machine, ActivityProtocol)
    assert(query.property.when.contains(client.requestCancel))
    assertEquals(query.scenario.machine, ActivityProtocol)
    assertEquals(query.form, QueryForm.find)
    assertEquals(
      query.scenario.actions,
      ActivityProtocol.cancellation.queries.heldCanceledByID.scenario.actions
    )
    assertEquals(query.total, Some(22464L))
    assertEquals(
      query.expectedRun.map(_.copy(because = "")),
      Some(
        temporal.realize.inconclusive(
          framework.realize.Reason.explanationsDisagree,
          because = ""
        )
      )
    )
    val held = take(
      take(ActivityProtocol.init, client.start()).state,
      activityWorker.poll
    ).state
    val requested = take(held, client.requestCancel)
    assertEquals(requested.state.phase, protocol.Phase.cancelRequested)
    val holds =
      query.property.holds.get
        .asInstanceOf[Step[ // scalafix:ok DisableSyntax.asInstanceOf
          protocol.State,
          Outcome,
          protocol.Fact
        ] => Boolean]
    assert(holds(requested))
    assert(!holds(requested.copy(facts = Nil)))
    assert(
      !holds(
        requested.copy(state = requested.state.copy(phase = protocol.Phase.canceled))
      )
    )
    assertEquals(
      take(requested.state, completionClient.respondCanceledByID).state.phase,
      protocol.Phase.canceled
    )
  }

  test(
    "three completion client witnesses pin no-poll completion, held fatal failure and requested cancellation"
  ) {
    val witnesses = List(
      (
        ActivityProtocol.completion.queries.scheduledCompletedByID,
        Vector[ClassRef](client.start(), completionClient.respondCompletedByID),
        protocol.Phase.completed
      ),
      (
        ActivityProtocol.retry.queries.heldFailedByID,
        Vector[ClassRef](
          client.start(),
          activityWorker.poll,
          completionClient.respondFailedByID(Failure.fatal)
        ),
        protocol.Phase.failed
      ),
      (
        ActivityProtocol.cancellation.queries.heldCanceledByID,
        Vector[ClassRef](
          client.start(),
          activityWorker.poll,
          client.requestCancel,
          completionClient.respondCanceledByID
        ),
        protocol.Phase.canceled
      )
    )
    for (query, actions, phase) <- witnesses do
      assertEquals(query.form, QueryForm.find)
      assert(!query.scenario.free)
      assertEquals(query.scenario.actions, actions)
      assertEquals(query.total, Some(5616L * actions.size))
      assertEquals(
        query.expectedRun.map(_.copy(because = "")),
        Some(
          temporal.realize.inconclusive(
            framework.realize.Reason.explanationsDisagree,
            because = ""
          )
        )
      )
      val before =
        actions.init.foldLeft(ActivityProtocol.init)((s, a) => take(s, a).state)
      val after = take(before, actions.last)
      assertEquals(after.state.phase, phase)
      val holds =
        query.property.holds.get
          .asInstanceOf[Step[ // scalafix:ok DisableSyntax.asInstanceOf
            protocol.State,
            Outcome,
            protocol.Fact
          ] => Boolean]
      assert(holds(after))
      assert(!holds(after.copy(facts = Nil)))
      assert(
        !holds(
          after.copy(state = after.state.copy(phase = protocol.Phase.scheduled))
        )
      )
    assertEquals(
      witnesses.head._1.scenario.actions.count {
        case a: Action[EmptyTuple] =>
          a.decl.actor eq temporal.actors.worker.worker
        case c: Class    => c.decl.actor eq temporal.actors.worker.worker
        case _: Composed => false
      },
      0
    )
  }
