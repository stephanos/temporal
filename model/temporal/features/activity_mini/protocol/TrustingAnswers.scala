package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import Bounds.six
import AnswerDispatch.monitors.answers

val _ = () => summon[CanEqual[AnswerPhase, AnswerPhase]]

object TrustingAnswers extends Derived(AnswerDispatch.rebind(answers.deliver ~> AnswerDispatch.answering.effects.trust)), NegativeControl:
  object properties:
    val staleSettles = property when answers.deliver(Answer.firstCompleted) holds
      (after => after.state.phase == AnswerPhase.completed && after.state.attempts == 2 &&
        after.records(AnswerFact.acceptedStale))
    val noStaleSettlement = AnswerDispatch.properties.oldAnswerKeepsAttempt(TrustingAnswers)
  object queries:
    val stale = scenario.actions(activityWorker.poll, activityWorker.respondCompleted,
      fault.delayAnswer, timers.backoff, activityWorker.poll, answers.deliver(Answer.firstCompleted))
    val staleAnswerFound = query find properties.staleSettles in stale limits six total 450
    val staleAnswerRejected = query verify properties.noStaleSettlement in stale limits six total 450
