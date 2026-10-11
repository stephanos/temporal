package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import temporal.capabilities.product.*
import Bounds.{five, three}
import foundations.taskqueue.{queue, Outstanding}
import foundations.taskqueue.product.TaskQueueProduct

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RecordOverQueue
    extends Composition[OverQueue](_.activity -> RecordMember, _.queue -> TaskQueueProduct),
      Phased[OverQueue, AdmissionPhase](_.activity.phase):
  override def end(s: State) = ActivityRecord.end(s.activity)

  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> activityWorker.poll, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerMatching, _.queue -> queue.acknowledge)

  object properties:
    def overQueueClaims(c: Composition[State], declared: Capabilities[State, String, String]) =
      val failedCommitKeepsTheMessage = c.property holds (after =>
        after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
          (after.state.queue.outstanding != Outstanding.empty && // scalafix:ok DisableSyntax.!=; strictEquality checks native comparisons
            !after.records(_.activity, AdmissionFact.attemptAdmitted))
      )
      OverQueueClaims(
        declared.claim(Pausable.pausedIsNotDispatched[State, AdmissionPhase]),
        ActivityRecord.properties.atMostOneActive(c)(
          through(_.activity, ActivityRecord.states.twoActive)
        ),
        failedCommitKeepsTheMessage
      )

  object capabilities extends OverQueueCapabilities(this)

  object queries:
    capabilities.bound(five)

    // Over the opaque queue: every claim and path of the design `c`.
    def overQueueQueries(c: Composition[State], declared: Capabilities[State, String, String]) =
      val claims = properties.overQueueClaims(c, declared)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.activity, client.pause),
        c.synced(_.activity -> activityWorker.poll)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> activityWorker.poll),
        c.own(_.activity, client.pause)
      )
      val duplicateDelivery = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> activityWorker.poll),
        c.synced(_.activity -> activityWorker.poll)
      )
      val any = c.scenario.free
      Vector(
        query(s"${c.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits three,
        query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits three,
        query(s"${c.name}.duplicateDelivery") verify claims.oneActive in
          duplicateDelivery limits three,
        query(s"${c.name}.failedCommit") verify claims.failedCommit in
          duplicateDelivery limits three,
        query verify claims.oneActive in any limits five
      )

    val recordOverQueueQueries = overQueueQueries(RecordOverQueue, capabilities)
