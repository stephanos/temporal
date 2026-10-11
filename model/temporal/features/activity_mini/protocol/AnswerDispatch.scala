package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import Bounds.six
import AnswerDispatch.monitors.answers

object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], FailureModel, Answering:
  val init = AnswerState(AnswerPhase.waiting, UpTo(0), answers.empty)
  def end(s: State) = s.phase.in(AnswerPhase.completed, AnswerPhase.failed)
  val entity = activity
  object monitors:
    val answers = channel[Answer](capacity = 1, order = Order.fifo, loss = Loss.lossy)
    val cancelDuringBackoff = hole
    val environment = delayedAnswersPossible
  object rules extends Facets(answering.rules)
  object properties:
    def oldAnswerKeepsAttempt(m: Machine[AnswerState, Outcome, AnswerFact]): Property[AnswerState] =
      m.property("oldAnswerKeepsAttempt") when answers.deliver(Answer.firstCompleted) holdsAcross ((before, after) =>
        before.attempts != 2 || (after.state.phase == before.phase &&
          after.state.attempts == before.attempts && after.outcome == Outcome.rejected(Rejection.notFound)))
    val noStaleSettlement = oldAnswerKeepsAttempt(AnswerDispatch)
    val backoffCancellation = property when fault.interruptBackoff holds (_ => monitors.cancelDuringBackoff.reached)
    val lostAnswerLeavesBufferEmpty = property when answers.lose(Answer.firstCompleted) holds (_.state.answers.isEmpty)
  object queries:
    val stale = scenario.actions(activityWorker.poll, activityWorker.respondCompleted,
      fault.delayAnswer, timers.backoff, activityWorker.poll, answers.deliver(Answer.firstCompleted))
    val canceled = scenario.actions(activityWorker.poll, fault.delayAnswer, fault.interruptBackoff)
    val lost = scenario.actions(activityWorker.poll, activityWorker.respondCompleted, answers.lose(Answer.firstCompleted))
    val staleIsFenced = query verify properties.noStaleSettlement in stale limits six total 450
    val cancellationUnknown = query verify properties.backoffCancellation in canceled limits Bounds.three total 225
    val answerLoss = query verify properties.lostAnswerLeavesBufferEmpty in lost limits Bounds.three total 225
    val failedAnswer = witness(activityWorker.poll, activityWorker.respondFailed(Failure.fatal), answers.deliver(Answer.firstFailed))
      .records(AnswerFact.acceptedCurrent).ends(_.phase == AnswerPhase.failed).limits(Bounds.three).total(225)
