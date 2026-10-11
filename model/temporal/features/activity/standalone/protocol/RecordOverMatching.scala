package temporal
package features.activity
package standalone
package protocol

import framework.*
import temporal.capabilities.system.*
import framework.Finite.given
import temporal.capabilities.product.*
import Bounds.{eight, five, seven, twelve}
import foundations.taskqueue.{fault, queue}
import foundations.taskqueue.product.TaskQueueProduct
import foundations.taskqueue.protocol.TaskQueueProtocol

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RecordOverMatching
    extends Composition[OverMatching](_.activity -> RecordMember, _.queue -> TaskQueueProtocol),
      Phased[OverMatching, AdmissionPhase](_.activity.phase),
      FailureModel:
  override def end(s: State) = ActivityRecord.end(s.activity)

  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> activityWorker.poll, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerMatching, _.queue -> queue.acknowledge)
    replaces(_.queue, TaskQueueProduct)

  object properties:
    def overMatchingClaims(c: Composition[State], declared: Capabilities[State, String, String]) =
      OverMatchingClaims(
        declared.claim(Pausable.pausedIsNotDispatched[State, AdmissionPhase]),
        Exclusive.atMostOneActive(c)(
          through(_.activity, ActivityRecord.states.twoActive)
        )
      )

  object capabilities extends OverMatchingCapabilities(this)

  object queries:
    capabilities.bound(twelve)

    // Over the detailed queue: every claim and path of the design `c`.
    def overMatchingQueries(c: Composition[State], declared: Capabilities[State, String, String]) =
      val claims = properties.overMatchingClaims(c, declared)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.own(_.activity, client.pause),
        c.synced(_.activity -> activityWorker.poll)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> activityWorker.poll),
        c.own(_.activity, client.pause)
      )
      // The answer to the poller is lost after the commit, so the persisted task is handed out again.
      val deliveredAgainAfterLostAck = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> activityWorker.poll),
        c.own(_.queue, fault.ackLoss),
        c.synced(_.activity -> activityWorker.poll)
      )
      // A crash after the admission commit, before the acknowledgment: history retries the sync
      // match.
      val crashAfterAdmissionCommit = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> activityWorker.poll),
        c.own(_.queue, fault.crash),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> activityWorker.poll)
      )
      val any = c.scenario.free
      Vector(
        query(s"${c.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits five,
        query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits five,
        query(s"${c.name}.deliveredAgainAfterLostAck") verify claims.oneActive in
          deliveredAgainAfterLostAck limits seven,
        query(s"${c.name}.crashAfterAdmissionCommit") verify claims.oneActive in
          crashAfterAdmissionCommit limits eight,
        query verify claims.oneActive in any limits twelve
      )

    val recordOverMatchingQueries = overMatchingQueries(RecordOverMatching, capabilities)
