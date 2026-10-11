package temporal
package features.activity
package mini
package protocol

import framework.*

enum AnswerResult:
  case completed, failed
enum Answer:
  case firstCompleted, firstFailed, secondCompleted, secondFailed
enum AnswerPhase:
  case waiting, held, backingOff, completed, failed
final case class AnswerState(phase: AnswerPhase, attempts: UpTo[2], answers: Inbox[Answer])
enum AnswerFact:
  case acceptedCurrent, acceptedStale

given Finite[AnswerState] =
  given Finite[Inbox[Answer]] = AnswerDispatch.monitors.answers.contents
  new Finite[AnswerState]:
    def values: IndexedSeq[AnswerState] = Finite.derived[AnswerState].values

object fault extends Actor:
  val delayAnswer = action(this).on(activity)
  val interruptBackoff = action(this).on(activity)

val delayedAnswersPossible = assume
