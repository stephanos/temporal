package umpire

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{activity, client, product, service, system, worker}
import temporal.features.activity.standalone.system.{
  ActivitySystem,
  ByIDCancellation,
  ByIDCompletion,
  ByIDFailure
}
import umpire.outcomes.{Outcome, Rejection}

class ActivityByIDRegression extends munit.FunSuite:
  test("the service declares three independent by-ID answer actions") {
    val answers = ActivitySystem.bindings.filter(_.decl.actor.name == "service")
    assertEquals(
      answers.map(_.decl),
      List(
        service.respondCompletedByID.decl,
        service.respondFailedByID.decl,
        service.respondCanceledByID.decl
      )
    )
    for answer <- answers do assertEquals(answer.decl.on, Some(activity))
    assertEquals(service.respondFailedByID.decl.tokens, List(Some(failure)))
    assertEquals(
      service.respondFailedByID.decl.domains.map(_.values.toList),
      List(Failure.values.toList)
    )
    assertNotEquals(service.respondCompletedByID.decl, worker.respondCompleted.decl)
    assertNotEquals(service.respondFailedByID.decl, worker.respondFailed.decl)
    assertNotEquals(service.respondCanceledByID.decl, worker.respondCanceled.decl)
  }

  private def take(s: system.State, action: ClassRef): Step[system.State, Outcome, system.Fact] =
    val c = action match
      case c: Class              => c
      case a: Action[EmptyTuple] => a()
    val binding = ActivitySystem.bindings.find(_.decl == c.decl).get
    effectOf[system.State, Outcome, system.Fact](binding.decl, binding.function)(s, c.values).head

  test("every service result is carried by Product without hiding a public status") {
    val classes = List(
      service.respondCompletedByID(),
      service.respondFailedByID(Failure.fatal),
      service.respondFailedByID(Failure.retryable),
      service.respondCanceledByID()
    )
    for
      before <- summon[Finite[system.State]].values
      c <- classes
    do
      val after = take(before, c)
      val visible = after.facts.filter(ActivitySystem.refinement.visible).map(_.toString)
      val source = ActivitySystem.refinement.toProduct(before)
      val target = ActivitySystem.refinement.toProduct(after.state)
      val binding = product.ActivityProduct.bindings.find(_.decl == c.decl).get
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        source,
        c.values
      )
      assert(
        carriers.exists(p =>
          p.outcome == after.outcome && p.state == target && p.facts.map(_.toString) == visible
        ) ||
          (before.phase == system.Phase.unstarted && source == target && visible.isEmpty),
        s"service refinement $before -> $after"
      )
      assertEquals(after.state.attempts, before.attempts)
      assertEquals(after.state.scheduleToClose, before.scheduleToClose)
      assertEquals(after.state.scheduleToStart, before.scheduleToStart)
      assertEquals(after.state.startToClose, before.startToClose)
      assertEquals(after.state.heartbeat, before.heartbeat)
      assertEquals(after.state.maxAttempts, before.maxAttempts)
  }

  test("cancellation without held work settles before any by-ID disposition") {
    for
      before <- summon[Finite[system.State]].values
      if before.phase == system.Phase.scheduled || before.phase == system.Phase.paused
    do
      val after = take(before, client.requestCancel)
      assertEquals(after.state, before.copy(phase = system.Phase.canceled))
      assertEquals(after.facts, List(system.Fact.statusCanceled))
      assertEquals(after.outcome, Outcome.accepted)
      val binding = product.ActivityProduct.bindings.find(_.decl == client.requestCancel.decl).get
      val carriers = effectOf[product.State, Outcome, product.Fact](binding.decl, binding.function)(
        ActivitySystem.refinement.toProduct(before),
        Nil
      )
      assertEquals(carriers.map(_.state), List(product.State(product.Phase.canceled)))
      assertEquals(carriers.map(_.outcome), List(Outcome.accepted))
      assertEquals(carriers.map(_.facts), List(List(product.Fact.statusCanceled)))
      for answer <- List(service.respondFailedByID(Failure.fatal), service.respondCanceledByID())
      do
        val refused = take(after.state, answer)
        assertEquals(refused.state, after.state)
        assertEquals(refused.facts, Nil)
        assertEquals(refused.outcome, Outcome.rejected(Rejection.notFound))
  }

  test("the preserved cancel-request claim uses its complete held service witness") {
    val query = ByIDCancellation.queries.cancelIsRequested
    assertEquals(query.name, "activitySystem.cancelIsRequested")
    assertEquals(query.property.name, "activitySystem.cancelIsRequested")
    assertEquals(query.property.machine, ByIDCancellation)
    assert(query.property.when.contains(client.requestCancel))
    assertEquals(query.scenario.machine, ByIDCancellation)
    assertEquals(query.form, QueryForm.find)
    assertEquals(query.scenario.actions, ByIDCancellation.queries.heldCanceledByID.scenario.actions)
    assertEquals(query.total, Some(19008L))
    assertEquals(
      query.expectedRun,
      Some(temporal.realize.inconclusive(umpire.realize.Reason.explanationsDisagree))
    )
    val held = take(take(ActivitySystem.init, client.start()).state, worker.poll).state
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
      take(requested.state, service.respondCanceledByID).state.phase,
      system.Phase.canceled
    )
  }

  test(
    "three service witnesses pin no-poll completion, held fatal failure and requested cancellation"
  ) {
    val witnesses = List(
      (
        ByIDCompletion.queries.scheduledCompletedByID,
        Vector[ClassRef](client.start(), service.respondCompletedByID),
        system.Phase.completed
      ),
      (
        ByIDFailure.queries.heldFailedByID,
        Vector[ClassRef](client.start(), worker.poll, service.respondFailedByID(Failure.fatal)),
        system.Phase.failed
      ),
      (
        ByIDCancellation.queries.heldCanceledByID,
        Vector[ClassRef](
          client.start(),
          worker.poll,
          client.requestCancel,
          service.respondCanceledByID
        ),
        system.Phase.canceled
      )
    )
    for (query, actions, phase) <- witnesses do
      assertEquals(query.form, QueryForm.find)
      assert(!query.scenario.free)
      assertEquals(query.scenario.actions, actions)
      assertEquals(query.total, Some(4752L * actions.size))
      assertEquals(query.expectedRun, Some(temporal.realize.satisfied))
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
        case a: Action[EmptyTuple] => a.decl.actor == temporal.shared.worker.worker
        case c: Class              => c.decl.actor == temporal.shared.worker.worker
        case _: Composed           => false
      },
      0
    )
  }
