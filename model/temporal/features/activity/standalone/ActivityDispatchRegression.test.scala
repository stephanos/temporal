package framework

import framework.Finite.given

import temporal.features.activity.{deadline, timers, Failure, Timeout}
import temporal.features.activity.standalone.{activityWorker, client, protocol, MaxAttempts}
import temporal.features.activity.standalone.protocol.ActivityProtocol
import framework.outcomes.Outcome

class ActivityDispatchRegression extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  private def take(
      s: protocol.State,
      a: Action[EmptyTuple]
  ): List[Step[protocol.State, Outcome, protocol.Fact]] =
    ActivityProtocol.bindings
      .find(_.decl == a.decl) // scalafix:ok DisableSyntax.==
      .get
      .function
      .asInstanceOf[protocol.State => List[Step[protocol.State, Outcome, protocol.Fact]]](
        s
      ) // scalafix:ok DisableSyntax.asInstanceOf

  private def backoff(s: protocol.State): protocol.State =
    val started = take(s, activityWorker.poll).head.state
    failAttempt(started)

  private def failAttempt(s: protocol.State): protocol.State =
    ActivityProtocol.bindings
      .find(_.decl == activityWorker.respondFailed.decl) // scalafix:ok DisableSyntax.==
      .get
      .function
      .asInstanceOf[(protocol.State, Failure) => List[Step[protocol.State, Outcome, protocol.Fact]]](
        s,
        Failure.retryable
      )
      .head
      .state // scalafix:ok DisableSyntax.asInstanceOf

  private def delayedStart: protocol.State =
    ActivityProtocol.bindings
      .find(_.decl == client.start.decl) // scalafix:ok DisableSyntax.==
      .get
      .function
      .asInstanceOf[
        (protocol.State, Timeout, Timeout, Timeout, Timeout, Timeout, MaxAttempts) => List[
          Step[protocol.State, Outcome, protocol.Fact]
        ]
      ](
        ActivityProtocol.init,
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
    val waiting = backoff(ActivityProtocol.init.copy(phase = protocol.Phase.scheduled))
    val paused = take(waiting, client.pause).head.state
    val resumed = take(paused, client.unpause).head.state
    assertEquals(take(resumed, activityWorker.poll), Nil)
    val ready = take(resumed, timers.backoff).head.state
    assertEquals(take(ready, activityWorker.poll).head.state.phase, protocol.Phase.started)
  }

  test("schedule-to-start does not fire during retry backoff (model.go:312-322)") {
    val waiting = backoff(
      ActivityProtocol.init.copy(
        phase = protocol.Phase.scheduled,
        scheduleToStart = Timeout.expires
      )
    )
    assertEquals(take(waiting, deadline.scheduleToStart), Nil)
    val ready = take(waiting, timers.backoff).head.state
    assertEquals(take(ready, deadline.scheduleToStart).head.state.phase, protocol.Phase.timedOut)
  }

  test("start delay defers dispatch and both scheduling deadlines (model.go:312-335,355-365)") {
    val waiting = delayedStart
    assertEquals(waiting.dispatch, protocol.Dispatch.startDelay)
    assertEquals(take(waiting, activityWorker.poll), Nil)
    assertEquals(take(waiting, deadline.scheduleToStart), Nil)
    assertEquals(take(waiting, deadline.scheduleToClose), Nil)
    val ready = take(waiting, timers.startDelay).head.state
    assertEquals(ready.dispatch, protocol.Dispatch.now)
    assertEquals(take(ready, timers.startDelay), Nil)
    assertEquals(take(ready, activityWorker.poll).head.state.phase, protocol.Phase.started)
    for timer <- List(deadline.scheduleToStart, deadline.scheduleToClose) do
      assertEquals(take(ready, timer).head.state.phase, protocol.Phase.timedOut)
  }

  test("both dispatch delays can expire while paused (model.go:355-376)") {
    val retries = backoff(ActivityProtocol.init.copy(phase = protocol.Phase.scheduled))
    for (waiting, timer) <- List(delayedStart -> timers.startDelay, retries -> timers.backoff) do
      val paused = take(waiting, client.pause).head.state
      assertEquals(paused.dispatch, waiting.dispatch)
      assertEquals(take(paused, client.unpause).head.state.dispatch, waiting.dispatch)
      val elapsed = take(paused, timer).head.state
      assertEquals(elapsed.phase, protocol.Phase.paused)
      assertEquals(take(elapsed, activityWorker.poll), Nil)
      val resumed = take(elapsed, client.unpause).head.state
      assertEquals(take(resumed, activityWorker.poll).head.state.phase, protocol.Phase.started)
      assertEquals(take(resumed, timer), Nil)
  }

  test("a pause requested during an attempt retains its retry's backoff (model.go:130-137)") {
    val started =
      take(ActivityProtocol.init.copy(phase = protocol.Phase.scheduled), activityWorker.poll).head.state
    val requested = take(started, client.pause).head.state
    val paused = failAttempt(requested)
    assertEquals(paused.phase, protocol.Phase.paused)
    assertEquals(paused.dispatch, protocol.Dispatch.backoff)
    assertEquals(take(take(paused, client.unpause).head.state, activityWorker.poll), Nil)
  }

  test("every waiting dispatch maps to the Product's scheduled state") {
    assertEquals(summon[Finite[protocol.State]].values.size, 5616)
    // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
    for waiting <- Finite[protocol.State].values if waiting.phase == protocol.Phase.scheduled
    // scalafix:on DisableSyntax.==
    do
      assertEquals(
        ActivityProtocol.refinement.toProduct(waiting).phase,
        temporal.features.activity.standalone.product.Phase.scheduled
      )
  }
