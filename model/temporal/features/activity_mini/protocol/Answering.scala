package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import scala.annotation.unused
import AnswerDispatch.monitors.answers

val _ = () => summon[CanEqual[AnswerPhase, AnswerPhase]]

trait Answering extends Facet[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:

  object answering:

    object states:
      def held(s: AnswerState): Boolean = s.phase == AnswerPhase.held
      def current(s: AnswerState, answer: Answer): Boolean =
        if s.attempts == 1 then answer.in(Answer.firstCompleted, Answer.firstFailed)
        else answer.in(Answer.secondCompleted, Answer.secondFailed)
      def completed(answer: Answer): Boolean =
        answer.in(Answer.firstCompleted, Answer.secondCompleted)

    object effects:
      def poll(s: AnswerState) = enter(
        s.copy(phase = AnswerPhase.held, attempts = UpTo(s.attempts + 1))
      )
      def send(s: AnswerState, result: AnswerResult) =
        val answer = if s.attempts == 1 then
          if result == AnswerResult.completed then Answer.firstCompleted else Answer.firstFailed
        else if result == AnswerResult.completed then Answer.secondCompleted
        else Answer.secondFailed
        enter(s.copy(answers = s.answers.send(answer)))
      def complete(s: AnswerState) = send(s, AnswerResult.completed)
      def fail(s: AnswerState, @unused failure: Failure) = send(s, AnswerResult.failed)
      def receive(s: AnswerState, answer: Answer) =
        if !states.current(s, answer) then reject(Outcome.rejected(Rejection.notFound), s)
        else
          enter(
            s.copy(phase = if states.completed(answer) then AnswerPhase.completed
            else AnswerPhase.failed),
            AnswerFact.acceptedCurrent
          )
      def trust(s: AnswerState, answer: Answer) =
        enter(
          s.copy(phase = if states.completed(answer) then AnswerPhase.completed
          else AnswerPhase.failed),
          if states.current(s, answer) then AnswerFact.acceptedCurrent else AnswerFact.acceptedStale
        )
      def lose(s: AnswerState, @unused answer: Answer) = stay(s)
      def interrupt(@unused s: AnswerState): List[Step[AnswerState, Outcome, AnswerFact]] =
        AnswerDispatch.monitors.cancelDuringBackoff.reached

    object rules extends Rules:
      on(activityWorker.poll) {
        where(s => s.phase == AnswerPhase.waiting && s.attempts < 2) ~> effects.poll
      }
      on(activityWorker.respondCompleted) {
        where(s => states.held(s) && !s.answers.isFull) ~> effects.complete
      }
      on(activityWorker.respondFailed) {
        where(s => states.held(s) && !s.answers.isFull) ~> effects.fail
      }
      on(fault.delayAnswer) {
        where(s => states.held(s) && s.attempts < 2) ~> enters(AnswerPhase.backingOff)
      }
      on(timers.backoff) {
        where(_.phase == AnswerPhase.backingOff) ~> enters(AnswerPhase.waiting)
      }
      on(answers.deliver) {
        where(states.held) ~> effects.receive
      }
      on(answers.lose) {
        always ~> effects.lose
      }
      on(fault.interruptBackoff) {
        where(_.phase == AnswerPhase.backingOff) ~> effects.interrupt
      }
