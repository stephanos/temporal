package umpire

import temporal.features.activity.{deadline, Timeout}
import temporal.features.activity.standalone.{client, system}
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.Outcome

class ActivityRetryRegression extends munit.FunSuite:
  test("the start declares a retained retry-policy input") {
    assertEquals(client.start.decl.domains.size, 5)
  }

  test("an unlimited start-to-close timeout retries without terminal timeout evidence") {
    val before = ActivitySystem.init.copy(
      phase = system.Phase.started,
      attempts = UpTo(1),
      startToClose = Timeout.expires
    )
    val after = ActivitySystem.bindings
      .find(_.decl == deadline.startToClose.decl)
      .get
      .function
      .asInstanceOf[system.State => List[Step[system.State, Outcome, system.Fact]]](before)
      .head // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(after.state, before.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff))
    assertEquals(after.facts, List(system.Fact.statusScheduled, system.Fact.attemptCount))
  }
