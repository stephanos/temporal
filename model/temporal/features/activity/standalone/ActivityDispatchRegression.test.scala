package umpire

import temporal.features.activity.{deadline, Failure, timers, Timeout}
import temporal.features.activity.standalone.{client, system, worker}
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.Outcome

class ActivityDispatchRegression extends munit.FunSuite:
  private def take(s: system.State, a: Action[EmptyTuple]): List[Step[system.State, Outcome, system.Fact]] =
    ActivitySystem.bindings.find(_.decl == a.decl).get.function
      .asInstanceOf[system.State => List[Step[system.State, Outcome, system.Fact]]](s) // scalafix:ok DisableSyntax.asInstanceOf

  private def backoff(s: system.State): system.State =
    val started = take(s, worker.poll).head.state
    ActivitySystem.bindings.find(_.decl == worker.respondFailed.decl).get.function
      .asInstanceOf[(system.State, Failure) => List[Step[system.State, Outcome, system.Fact]]](started, Failure.retryable).head.state // scalafix:ok DisableSyntax.asInstanceOf

  test("unpause retains retry backoff until its timer (model.go:239-244,369-376)") {
    val waiting = backoff(ActivitySystem.init.copy(phase = system.Phase.scheduled))
    val paused = take(waiting, client.pause).head.state
    val resumed = take(paused, client.unpause).head.state
    assertEquals(take(resumed, worker.poll), Nil)
    val ready = take(resumed, timers.backoff).head.state
    assertEquals(take(ready, worker.poll).head.state.phase, system.Phase.started)
  }

  test("schedule-to-start does not fire during retry backoff (model.go:312-322)") {
    val waiting = backoff(ActivitySystem.init.copy(
      phase = system.Phase.scheduled,
      scheduleToStart = Timeout.expires
    ))
    assertEquals(take(waiting, deadline.scheduleToStart), Nil)
    val ready = take(waiting, timers.backoff).head.state
    assertEquals(take(ready, deadline.scheduleToStart).head.state.phase, system.Phase.timedOut)
  }
