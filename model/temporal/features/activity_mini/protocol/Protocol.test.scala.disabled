package temporal.features.activity.mini.protocol

import framework.dsl.*
import framework.dsl.outcomes.Outcome
import temporal.features.activity.{Failure, Timeout}
import temporal.features.activity.mini.{MaxAttempts, client, scheduleToStart}
import AnswerDispatch.monitors.answers

class MiniProtocolTest extends munit.FunSuite:
  test("shared scheduling input and backoff preserve the Product view") {
    assert(client.start.decl.tokens.head.exists(_ eq scheduleToStart))
    assert(taskQueue.schedule.decl.tokens.head.exists(_ eq scheduleToStart))
    val scheduled = ActivityProtocol.dispatching.effects.schedule(
      ActivityProtocol.init, Timeout.expires, Timeout.unset, MaxAttempts.two).head.state
    val started = ActivityProtocol.dispatching.effects.startAttempt(scheduled).head.state
    val retried = ActivityProtocol.completion.effects.fail(started, Failure.retryable).head
    assertEquals(retried.state.phase, Phase.backingOff)
    assertEquals(Phase.backingOff.status, Phase.scheduled.status)
    assertEquals(ActivityProtocol.refinement.toProduct(retried.state).phase,
      temporal.features.activity.mini.product.Phase.scheduled)
    assertEquals(retried.facts, List(Fact.statusScheduled))
    val second = ActivityProtocol.dispatching.effects.startAttempt(retried.state.copy(phase = Phase.scheduled)).head.state
    assertEquals(second.attempts.toInt, 2)
    assertEquals(ActivityProtocol.completion.effects.fail(second, Failure.retryable).head.state.phase, Phase.failed)
  }

  test("answer dispatch fences an old attempt while the control trusts it") {
    val second = AnswerDispatch.init.copy(phase = AnswerPhase.held, attempts = UpTo(2))
    val old = Answer.firstCompleted
    val good = AnswerDispatch.answering.effects.receive(second, old).head
    val bad = AnswerDispatch.answering.effects.trust(second, old).head
    assertEquals(good.state.phase, AnswerPhase.held)
    assertEquals(good.outcome, Outcome.rejected(framework.dsl.outcomes.Rejection.notFound))
    assertEquals(bad.state.phase, AnswerPhase.completed)
    assertEquals(answers.capacity, 1)
    assertEquals(answers.loss, Loss.lossy)
    assert(answers.empty.send(old).isFull)
  }
