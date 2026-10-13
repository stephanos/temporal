package temporal
package features.activity
package mini
package protocol

import framework.dsl.*
import framework.dsl.Finite.given
import foundations.taskqueue.{QueueView, Outstanding, queue as foundationQueue}
import Bounds.three

val _ = () => summon[CanEqual[Outstanding, Outstanding]]

final case class OverQueue(activity: State, queue: QueueView)

object taskQueue:
  val schedule = internal.input(scheduleToStart).on(foundations.taskqueue.taskQueueEntity)

object Queue extends Composition[OverQueue](
    _.activity -> ProtocolMember, _.queue -> QueueScheduled):
  def end(s: State) = s.activity.phase.in[Closed]
  object syncs extends Syncs:
    sync(_.activity -> activityWorker.poll, _.queue -> foundationQueue.deliver)
  object properties:
    object safety:
      val deliveredWasScheduled = property.whenAction(synced(_.activity -> activityWorker.poll))
        .holds(_.state.queue.outstanding.in(Outstanding.deliveredOnce, Outstanding.deliveredTwice))
  object queries:
    val delivery = scenario.actions(own(_.activity, client.start()), own(_.queue, taskQueue.schedule()),
      synced(_.activity -> activityWorker.poll))
    val queueDelivery = query verify properties.safety.deliveredWasScheduled in delivery limits three
