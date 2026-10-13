package temporal
package features.activity
package mini
package protocol

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome
import temporal.features.activity.{Timeout, timers}
import temporal.capabilities.product.Pollable
import scala.annotation.unused
import foundations.taskqueue.{QueueView, QueueOutcome, QueueFact}
import foundations.taskqueue.product.TaskQueueProduct

trait Dispatching extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object dispatching:
    object states:
      def waiting(s: State): Boolean = s.phase == Phase.scheduled
      def firstSchedule(s: State): Boolean = s.attempts == 0

    object effects:
      def scheduleQueue(s: QueueView, @unused scheduleToStart: Timeout): List[Step[QueueView, QueueOutcome, QueueFact]] =
        TaskQueueProduct.queueing.effects.enqueueView(s)
      def schedule(s: State, scheduleToStart: Timeout, startToClose: Timeout, maxAttempts: MaxAttempts) =
        enter(s.copy(scheduleToStart = scheduleToStart, startToClose = startToClose,
          maxAttempts = maxAttempts), Fact.statusScheduled)
      def startAttempt(s: State) = enter(s.copy(phase = Phase.started,
        attempts = UpTo((s.attempts + 1).min(2))), Fact.statusStarted)

    object rules extends Rules:
      on(client.start) {
        when(Phase.scheduled).where(states.firstSchedule) ~> effects.schedule
      }
      on(activityWorker.poll) {
        when(Phase.scheduled).where(ActivityProtocol.states.retriesRemaining) ~> effects.startAttempt
      }
      on(timers.backoff) {
        when(Phase.backingOff) ~> enters(Phase.scheduled)
      }

    object capabilities extends Capabilities:
      val pollable: Capability = Pollable(dispatch = activityWorker.poll)
