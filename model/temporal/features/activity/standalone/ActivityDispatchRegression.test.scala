package umpire

import temporal.features.activity.{deadline, timers, Failure, Timeout}
import temporal.features.activity.standalone.{client, system, worker, MaxAttempts}
import temporal.features.activity.standalone.system.ActivitySystem
import umpire.outcomes.Outcome

class ActivityDispatchRegression extends munit.FunSuite:
  private def take(
      s: system.State,
      a: Action[EmptyTuple]
  ): List[Step[system.State, Outcome, system.Fact]] =
    ActivitySystem.bindings
      .find(_.decl == a.decl)
      .get
      .function
      .asInstanceOf[system.State => List[Step[system.State, Outcome, system.Fact]]](
        s
      ) // scalafix:ok DisableSyntax.asInstanceOf

  private def backoff(s: system.State): system.State =
    val started = take(s, worker.poll).head.state
    failAttempt(started)

  private def failAttempt(s: system.State): system.State =
    ActivitySystem.bindings
      .find(_.decl == worker.respondFailed.decl)
      .get
      .function
      .asInstanceOf[(system.State, Failure) => List[Step[system.State, Outcome, system.Fact]]](
        s,
        Failure.retryable
      )
      .head
      .state // scalafix:ok DisableSyntax.asInstanceOf

  private def delayedStart: system.State =
    ActivitySystem.bindings
      .find(_.decl == client.start.decl)
      .get
      .function
      .asInstanceOf[
        (system.State, Timeout, Timeout, Timeout, Timeout, Timeout, MaxAttempts) => List[
          Step[system.State, Outcome, system.Fact]
        ]
      ](
        ActivitySystem.init,
        Timeout.expires,
        Timeout.expires,
        Timeout.unset,
        Timeout.unset,
        Timeout.expires,
        MaxAttempts.unlimited
      )
      .head
      .state // scalafix:ok DisableSyntax.asInstanceOf

  test("unpause retains retry backoff until its timer (model.go:239-244,369-376)") {
    val waiting = backoff(ActivitySystem.init.copy(phase = system.Phase.scheduled))
    val paused = take(waiting, client.pause).head.state
    val resumed = take(paused, client.unpause).head.state
    assertEquals(take(resumed, worker.poll), Nil)
    val ready = take(resumed, timers.backoff).head.state
    assertEquals(take(ready, worker.poll).head.state.phase, system.Phase.started)
  }

  test("schedule-to-start does not fire during retry backoff (model.go:312-322)") {
    val waiting = backoff(
      ActivitySystem.init.copy(
        phase = system.Phase.scheduled,
        scheduleToStart = Timeout.expires
      )
    )
    assertEquals(take(waiting, deadline.scheduleToStart), Nil)
    val ready = take(waiting, timers.backoff).head.state
    assertEquals(take(ready, deadline.scheduleToStart).head.state.phase, system.Phase.timedOut)
  }

  test("start delay defers dispatch and both scheduling deadlines (model.go:312-335,355-365)") {
    val waiting = delayedStart
    assertEquals(waiting.dispatch, system.Dispatch.startDelay)
    assertEquals(take(waiting, worker.poll), Nil)
    assertEquals(take(waiting, deadline.scheduleToStart), Nil)
    assertEquals(take(waiting, deadline.scheduleToClose), Nil)
    val ready = take(waiting, timers.startDelay).head.state
    assertEquals(ready.dispatch, system.Dispatch.now)
    assertEquals(take(ready, timers.startDelay), Nil)
    assertEquals(take(ready, worker.poll).head.state.phase, system.Phase.started)
    for timer <- List(deadline.scheduleToStart, deadline.scheduleToClose) do
      assertEquals(take(ready, timer).head.state.phase, system.Phase.timedOut)
  }

  test("both dispatch delays can expire while paused (model.go:355-376)") {
    val retries = backoff(ActivitySystem.init.copy(phase = system.Phase.scheduled))
    for (waiting, timer) <- List(delayedStart -> timers.startDelay, retries -> timers.backoff) do
      val paused = take(waiting, client.pause).head.state
      assertEquals(paused.dispatch, waiting.dispatch)
      assertEquals(take(paused, client.unpause).head.state.dispatch, waiting.dispatch)
      val elapsed = take(paused, timer).head.state
      assertEquals(elapsed.phase, system.Phase.paused)
      assertEquals(take(elapsed, worker.poll), Nil)
      val resumed = take(elapsed, client.unpause).head.state
      assertEquals(take(resumed, worker.poll).head.state.phase, system.Phase.started)
      assertEquals(take(resumed, timer), Nil)
  }

  test("a pause requested during an attempt retains its retry's backoff (model.go:130-137)") {
    val started =
      take(ActivitySystem.init.copy(phase = system.Phase.scheduled), worker.poll).head.state
    val requested = take(started, client.pause).head.state
    val paused = failAttempt(requested)
    assertEquals(paused.phase, system.Phase.paused)
    assertEquals(paused.dispatch, system.Dispatch.backoff)
    assertEquals(take(take(paused, client.unpause).head.state, worker.poll), Nil)
  }

  test("every waiting dispatch maps to the Product's scheduled state") {
    assertEquals(summon[Finite[system.State]].values.size, 4752)
    for dispatch <- system.Dispatch.values do
      val waiting = ActivitySystem.init.copy(phase = system.Phase.scheduled, dispatch = dispatch)
      assertEquals(
        ActivitySystem.refinement.toProduct(waiting).phase,
        temporal.features.activity.standalone.product.Phase.scheduled
      )
  }
