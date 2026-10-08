package umpire

import temporal.features.activity.Timeout
import temporal.features.activity.standalone.{client, product, system, MaxAttempts}
import temporal.features.activity.standalone.product.ActivityProduct
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.{Outcome, Rejection}

class ActivityVisibilityRegression extends munit.FunSuite:
  private def take(a: Action[EmptyTuple], s: product.State) =
    ActivityProduct.bindings
      .find(_.decl == a.decl)
      .get
      .function
      .asInstanceOf[product.State => List[Step[product.State, Outcome, product.Fact]]](
        s
      ) // scalafix:ok DisableSyntax.asInstanceOf

  test("every public status is visible, including every timeout type; only attemptCount is hidden") {
    for fact <- summon[Finite[system.Fact]].values do
      assertEquals(
        ActivitySystem.refinement.visible(fact),
        fact != system.Fact.attemptCount,
        fact.toString
      )
  }

  test("creation, held pause and withdrawal have exact public fact carriers") {
    val scheduled = product.State(product.Phase.scheduled)
    val started = product.State(product.Phase.started)
    val start = ActivityProduct.bindings.find(_.decl == client.start.decl)
    assert(start.nonEmpty, "the Product must carry creation")
    val schedule = start.get.function.asInstanceOf[
      (product.State, Timeout, Timeout, Timeout, Timeout, MaxAttempts) => List[
        Step[product.State, Outcome, product.Fact]
      ]
    ] // scalafix:ok DisableSyntax.asInstanceOf
    for
      close <- Timeout.values
      waiting <- Timeout.values
      held <- Timeout.values
      delay <- Timeout.values
      policy <- MaxAttempts.values
    do
      assertEquals(
        schedule(scheduled, close, waiting, held, delay, policy),
        List(Step(Outcome.accepted, scheduled, List(product.Fact.statusScheduled)))
      )

    val pause = take(client.pause, started)
    assertEquals(
      pause.map(s => (s.outcome, s.state, s.facts)),
      List(
        (Outcome.accepted, product.State(product.Phase.paused), List(product.Fact.statusPaused)),
        (Outcome.accepted, started, List(product.Fact.statusPaused)),
        (Outcome.rejected(Rejection.failedPrecondition), started, Nil)
      )
    )
    val unpause = take(client.unpause, started)
    assertEquals(
      unpause.map(s => (s.outcome, s.state, s.facts)),
      List(
        (Outcome.accepted, started, List(product.Fact.statusStarted)),
        (Outcome.rejected(Rejection.failedPrecondition), started, Nil)
      )
    )
    assert(pause.last.because.nonEmpty)
    assert(unpause.last.because.nonEmpty)
  }
