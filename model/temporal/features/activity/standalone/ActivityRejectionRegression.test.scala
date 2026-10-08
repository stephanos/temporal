package umpire

import temporal.capabilities.Closable
import temporal.features.activity.standalone.{client, product, system, worker}
import temporal.features.activity.standalone.product.ActivityProduct
import temporal.features.activity.standalone.product.ActivityProduct.phased
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.{Outcome, Rejection}

class ActivityRejectionRegression extends munit.FunSuite:
  private def take[S, F](
      m: Machine[S, Outcome, F],
      s: S,
      a: Action[EmptyTuple]
  ): List[Step[S, Outcome, F]] =
    m.bindings
      .find(_.decl == a.decl)
      .get
      .function
      .asInstanceOf[S => List[Step[S, Outcome, F]]](
        s
      ) // scalafix:ok DisableSyntax.asInstanceOf

  private def rejectsUnchanged[S, F](
      m: Machine[S, Outcome, F],
      s: S,
      a: Action[EmptyTuple],
      why: Rejection
  ): Unit =
    val steps = take(m, s, a)
    assertEquals(steps.size, 1, s"${m.name}: ${a.decl.name} in $s")
    assertEquals(steps.head.outcome, Outcome.rejected(why))
    assertEquals(steps.head.state, s)
    assertEquals(steps.head.facts, Nil)
    assert(steps.head.because.nonEmpty)

  test("forbidden live controls preserve every state field (model.go:232,251)") {
    for
      s <- summon[Finite[system.State]].values
      (action, forbidden) <- List(
        client.pause -> Set[system.Phase](system.Phase.paused, system.Phase.pauseRequested, system.Phase.cancelRequested),
        client.unpause -> Set[system.Phase](system.Phase.scheduled, system.Phase.started, system.Phase.cancelRequested)
      )
      if forbidden(s.phase)
    do rejectsUnchanged(ActivitySystem, s, action, Rejection.failedPrecondition)

    for
      s <- summon[Finite[product.State]].values
      (action, forbidden) <- List(
        client.pause -> Set[product.Phase](product.Phase.paused, product.Phase.cancelRequested),
        client.unpause -> Set[product.Phase](product.Phase.scheduled, product.Phase.started, product.Phase.cancelRequested)
      )
      if forbidden(s.phase)
    do rejectsUnchanged(ActivityProduct, s, action, Rejection.failedPrecondition)
  }

  test("Product started carries both concrete pause responses without losing acceptance") {
    val before = product.State(product.Phase.started)
    val rows = take(ActivityProduct, before, client.pause)
    assertEquals(rows.map(_.outcome), List(Outcome.accepted, Outcome.rejected(Rejection.failedPrecondition)))
    assertEquals(rows.head.state, product.State(product.Phase.paused))
    assertEquals(rows.head.facts, List(product.Fact.statusPaused))
    assertEquals(rows.last.state, before)
    assertEquals(rows.last.facts, Nil)
    for s <- summon[Finite[system.State]].values if s.phase == system.Phase.pauseRequested do
      val result = take(ActivitySystem, s, client.pause).head
      assertEquals(ActivitySystem.refinement.toProduct(s), before)
      assert(rows.exists(r =>
        r.outcome == result.outcome && r.state == ActivitySystem.refinement.toProduct(result.state)
      ))
  }

  test("a worker cannot answer canceled before cancellation is requested (model.go:171)") {
    for s <- summon[Finite[system.State]].values
        if s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested do
      rejectsUnchanged(ActivitySystem, s, worker.respondCanceled, Rejection.invalidArgument)
    rejectsUnchanged(ActivityProduct, product.State(product.Phase.started), worker.respondCanceled, Rejection.invalidArgument)
  }

  test("a repeated RequestCancel rejects without recording another request (model.go:201-202)") {
    val started = ActivitySystem.init.copy(phase = system.Phase.started, attempts = UpTo(1))
    val requested = take(ActivitySystem, started, client.requestCancel).head
    assertEquals(requested.outcome, Outcome.accepted)
    assertEquals(requested.facts, List(system.Fact.statusCancelRequested))
    assertEquals(requested.state.phase, system.Phase.cancelRequested)
    for s <- summon[Finite[system.State]].values if s.phase == system.Phase.cancelRequested do
      rejectsUnchanged(ActivitySystem, s, client.requestCancel, Rejection.failedPrecondition)
    rejectsUnchanged(ActivitySystem, requested.state, client.requestCancel, Rejection.failedPrecondition)
    rejectsUnchanged(ActivityProduct, product.State(product.Phase.cancelRequested), client.requestCancel, Rejection.failedPrecondition)
  }

  test("closed controls still satisfy closedIsRejectedUniformly with notFound") {
    val property = Closable.closedIsRejectedUniformly[product.State, product.Phase, Outcome](ActivityProduct)(Outcome.rejected(Rejection.notFound))
    val holds = property.decl.holds2.get.asInstanceOf[(product.State, Step[product.State, Outcome, product.Fact]) => Boolean] // scalafix:ok DisableSyntax.asInstanceOf
    val controls = List(client.pause, client.unpause, client.requestCancel, client.terminate)
    for
      s <- summon[Finite[product.State]].values if s.phase.in[Closed]
      action <- controls
    do
      val rows = take(ActivityProduct, s, action)
      assertEquals(rows, List(Step(Outcome.rejected(Rejection.notFound), s)))
      assert(holds(s, rows.head))
      assert(!holds(s, Step(Outcome.rejected(Rejection.failedPrecondition), s)))
    for
      s <- summon[Finite[system.State]].values if s.phase.in[Closed]
      action <- controls
    do assertEquals(take(ActivitySystem, s, action), List(Step(Outcome.rejected(Rejection.notFound), s)))
  }
