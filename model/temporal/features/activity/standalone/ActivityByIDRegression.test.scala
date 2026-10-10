package framework

import framework.Finite.given

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{
  activity,
  activityWorker,
  client,
  completionClient,
  product,
  system
}
import temporal.features.activity.standalone.system.{
  ActivitySystem,
  ByIDCancellation,
  ByIDCompletion,
  ByIDFailure
}
import framework.outcomes.{Outcome, Rejection}

class ActivityByIDRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[system.Phase, system.Phase]]
  test("the completion client declares three independent by-ID answer actions") {
    val answers =
      ActivitySystem.bindings.filter(_.decl.actor.name == "completionClient") // scalafix:ok DisableSyntax.==
    assertEquals(
      answers.map(_.decl),
      List(
        completionClient.respondCompletedByID.decl,
        completionClient.respondFailedByID.decl,
        completionClient.respondCanceledByID.decl
      )
    )
    for answer <- answers do assertEquals(answer.decl.on, Some(activity))
    assertEquals(completionClient.respondFailedByID.decl.tokens, List(Some(failure)))
    assertEquals(
      completionClient.respondFailedByID.decl.domains.map(_.values.toList),
      List(Failure.values.toList)
    )
    assertNotEquals(
      completionClient.respondCompletedByID.decl,
      activityWorker.respondCompleted.decl
    )
    assertNotEquals(completionClient.respondFailedByID.decl, activityWorker.respondFailed.decl)
    assertNotEquals(completionClient.respondCanceledByID.decl, activityWorker.respondCanceled.decl)
  }

  private def take(s: system.State, action: ClassRef): Step[system.State, Outcome, system.Fact] =
    val c = action match
      case c: Class              => c
      case a: Action[EmptyTuple] => a()
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
    effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)(s, c.values).head

  test("every completion client result is carried by Product without hiding a public status") {
    val classes = List(
      completionClient.respondCompletedByID(),
      completionClient.respondFailedByID(Failure.fatal),
      completionClient.respondFailedByID(Failure.retryable),
      completionClient.respondCanceledByID()
    )
    for
      before <- summon[Finite[system.State]].values
      c <- classes
    do
      val after = take(before, c)
      val visible = after.facts.filter(ActivitySystem.refinement.visible).map(_.toString)
      val source = ActivitySystem.refinement.toProduct(before)
      val target = ActivitySystem.refinement.toProduct(after.state)
      val binding =
        product.ActivityProduct.bindings.find(_.decl == c.decl).get // scalafix:ok DisableSyntax.==
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        source,
        c.values
      )
      assert(
        carriers.exists(p =>
          p.outcome == after.outcome && p.state == target && p.facts.map(
            _.toString
          ) == visible
        ) ||
          (before.phase == system.Phase.unstarted && source == target && visible.isEmpty), // scalafix:ok DisableSyntax.==
        s"completion client refinement $before -> $after"
      )
      // A failure answer applies a pending reset, which restarts the count; nothing else moves it.
      val restarts = ActivitySystem.states.pendingReset(before) &&
        c.decl == completionClient.respondFailedByID.decl // scalafix:ok DisableSyntax.==
      assertEquals(after.state.attempts: Int, if restarts then 0 else before.attempts)
      assertEquals(after.state.scheduleToClose, before.scheduleToClose)
      assertEquals(after.state.scheduleToStart, before.scheduleToStart)
      assertEquals(after.state.startToClose, before.startToClose)
      assertEquals(after.state.heartbeat, before.heartbeat)
      assertEquals(after.state.maxAttempts, before.maxAttempts)
  }

  test("cancellation without held work settles before any by-ID disposition") {
    for
      before <- summon[Finite[system.State]].values
      if before.phase == system.Phase.scheduled || before.phase == system.Phase.paused // scalafix:ok DisableSyntax.==
    do
      val after = take(before, client.requestCancel)
      assertEquals(after.state, before.copy(phase = system.Phase.canceled))
      assertEquals(after.facts, List(system.Fact.statusCanceled))
      assertEquals(after.outcome, Outcome.accepted)
      val binding = product.ActivityProduct.bindings
        .find(_.decl == client.requestCancel.decl)
        .get // scalafix:ok DisableSyntax.==
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        ActivitySystem.refinement.toProduct(before),
        Nil
      )
      assertEquals(carriers.map(_.state), List(product.State(product.Phase.canceled)))
      assertEquals(carriers.map(_.outcome), List(Outcome.accepted))
      assertEquals(carriers.map(_.facts), List(List(product.Fact.statusCanceled)))
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

  test("the preserved cancel-request claim uses its complete held completion client witness") {
    val query = ByIDCancellation.queries.cancelIsRequested
    assertEquals(query.name, "activitySystem.cancelIsRequested")
    assertEquals(query.property.name, "activitySystem.cancelIsRequested")
    assertEquals(query.property.machine, ByIDCancellation)
    assert(query.property.when.contains(client.requestCancel))
    assertEquals(query.scenario.machine, ByIDCancellation)
    assertEquals(query.form, QueryForm.find)
    assertEquals(query.scenario.actions, ByIDCancellation.queries.heldCanceledByID.scenario.actions)
    assertEquals(query.total, Some(22464L))
    assertEquals(
      query.expectedRun,
      Some(temporal.realize.inconclusive(framework.realize.Reason.explanationsDisagree))
    )
    val held = take(take(ActivitySystem.init, client.start()).state, activityWorker.poll).state
    val requested = take(held, client.requestCancel)
    assertEquals(requested.state.phase, system.Phase.cancelRequested)
    val holds =
      query.property.holds.get.asInstanceOf[Step[ // scalafix:ok DisableSyntax.asInstanceOf
        system.State,
        Outcome,
        system.Fact
      ] => Boolean]
    assert(holds(requested))
    assert(!holds(requested.copy(facts = Nil)))
    assert(holds(requested.copy(state = requested.state.copy(phase = system.Phase.canceled))))
    assertEquals(
      take(requested.state, completionClient.respondCanceledByID).state.phase,
      system.Phase.canceled
    )
  }

  test(
    "three completion client witnesses pin no-poll completion, held fatal failure and requested cancellation"
  ) {
    val witnesses = List(
      (
        ByIDCompletion.queries.scheduledCompletedByID,
        Vector[ClassRef](client.start(), completionClient.respondCompletedByID),
        system.Phase.completed
      ),
      (
        ByIDFailure.queries.heldFailedByID,
        Vector[ClassRef](
          client.start(),
          activityWorker.poll,
          completionClient.respondFailedByID(Failure.fatal)
        ),
        system.Phase.failed
      ),
      (
        ByIDCancellation.queries.heldCanceledByID,
        Vector[ClassRef](
          client.start(),
          activityWorker.poll,
          client.requestCancel,
          completionClient.respondCanceledByID
        ),
        system.Phase.canceled
      )
    )
    for (query, actions, phase) <- witnesses do
      assertEquals(query.form, QueryForm.find)
      assert(!query.scenario.free)
      assertEquals(query.scenario.actions, actions)
      assertEquals(query.total, Some(5616L * actions.size))
      assertEquals(
        query.expectedRun,
        Some(temporal.realize.inconclusive(framework.realize.Reason.explanationsDisagree))
      )
      val before = actions.init.foldLeft(ActivitySystem.init)((s, a) => take(s, a).state)
      val after = take(before, actions.last)
      assertEquals(after.state.phase, phase)
      val holds =
        query.property.holds.get.asInstanceOf[Step[ // scalafix:ok DisableSyntax.asInstanceOf
          system.State,
          Outcome,
          system.Fact
        ] => Boolean]
      assert(holds(after))
      assert(!holds(after.copy(facts = Nil)))
      assert(!holds(after.copy(state = after.state.copy(phase = system.Phase.scheduled))))
    assertEquals(
      witnesses.head._1.scenario.actions.count {
        case a: Action[EmptyTuple] => a.decl.actor eq temporal.actors.worker.worker
        case c: Class              => c.decl.actor eq temporal.actors.worker.worker
        case _: Composed           => false
      },
      0
    )
  }
