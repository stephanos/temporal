// The admission designs composed with the task queue (temporal/foundations/taskqueue), over its opaque
// contract and over the matching, violating and storage-loss providers that replace it. The queue
// and its providers are the queue's; this file adds how the activity's dispatch, admission and
// answer synchronize with it, and what the activity promises across both: the record's
// capability Properties, read through the `activity` member's projection, as the definitions in
// model/temporal/capabilities read them.
//
// Read top to bottom: the composed states and the claims they are held to; then the members,
// RecordMember and TrustingRecordMember, derived from the admission designs; then one object per
// composition, each before the compositions derived from it -- RecordOverQueue and TrustingRecordOverQueue
// over the opaque queue, RecordOverMatching and TrustingRecordOverMatching over the detailed one, and the
// corrected design over each violating provider, RecordOverForgetful, RecordOverVolatile and
// RecordOverLossyMatching. A composition reads end, then its sections in order: states, syncs,
// properties, capabilities and queries.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import Bounds.three
import foundations.taskqueue.{fault, queue, seven, twelve, Outstanding, QueueDetail, QueueView}
import foundations.taskqueue.product.TaskQueueProduct
import foundations.taskqueue.system.{
  ForgetfulQueue,
  LossyMatchingQueue,
  TaskQueueSystem,
  VolatileQueue
}

// ### Types

final case class OverQueue(activity: AdmissionState, queue: QueueView)

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

// The claims a design over the opaque queue is held to: the Properties its capabilities bring, the
// record's own count of active attempts over its member, and that a failed commit admits nothing and
// leaves its message queued. `notPaused` is the generated `<design>.pausedIsNotDispatched`.
final case class OverQueueClaims(
    notPaused: Property[OverQueue],
    oneActive: Property[OverQueue],
    failedCommit: Property[OverQueue]
)

final case class OverMatchingClaims(
    notPaused: Property[OverMatching],
    oneActive: Property[OverMatching]
)

abstract class OverQueueCapabilities(c: Composition[OverQueue])(using
    Declaring[OverQueue, String, String],
    Phasing[OverQueue, AdmissionPhase]
) extends Capabilities:
  val closable: Closable[OverQueue, AdmissionPhase, String] = Closable(
    rejected = c.composedOutcome(RecordMember, Outcome.rejected(Rejection.notFound))
  )
  val pausable: Capability = Pausable(
    pause = c.own(_.activity, client.pause),
    unpause = c.own(_.activity, client.unpause)
  )
  val pollable: Capability = Pollable(
    dispatch = c.synced(_.activity -> worker.poll)
  )
  except(Closable.closedIsRejectedUniformly, because = RecordOverQueue.states.queueStepsOn)

abstract class OverMatchingCapabilities(c: Composition[OverMatching])(using
    Declaring[OverMatching, String, String],
    Phasing[OverMatching, AdmissionPhase]
) extends Capabilities:
  val closable: Closable[OverMatching, AdmissionPhase, String] = Closable(
    rejected = c.composedOutcome(RecordMember, Outcome.rejected(Rejection.notFound))
  )
  val pausable: Capability = Pausable(
    pause = c.own(_.activity, client.pause),
    unpause = c.own(_.activity, client.unpause)
  )
  val pollable: Capability = Pollable(
    dispatch = c.synced(_.activity -> worker.poll)
  )
  except(Closable.closedIsRejectedUniformly, because = RecordOverQueue.states.queueStepsOn)

// ### The record as a member: a composition's Queries are not answered while a member names
// monitors, so the members are the designs without them. The designs are what refine the product.

object RecordMember extends Derived(ActivityRecord.unmonitored)

object TrustingRecordMember extends Derived(TrustingActivityRecord.unmonitored)

// ### Over the opaque queue. The dispatch is the queue's enqueue, a delivery is admission's one
// input, and admission's answer is the acknowledgment: a commit, its answer and the acknowledgment
// are three points a fault can fall between.

object RecordOverQueue
    extends Composition[OverQueue](_.activity -> RecordMember, _.queue -> TaskQueueProduct),
      Phased[OverQueue, AdmissionPhase](_.activity.phase):
  override def end(s: State) = ActivityRecord.end(s.activity)
  // What the designs over a queue answer and waive.
  object states:
    // Why a design over a queue waives closedIsRejectedUniformly: the queue member keeps its own
    // steps after the record closes, so a composed step moves the state; the record's own
    // declaration (Record.scala) holds the record to that Property.
    val queueStepsOn =
      "the queue member keeps stepping after the record closes; admissionCapabilities holds the record"

  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.poll, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerMatching, _.queue -> queue.acknowledge)

  object properties:
    def overQueueClaims(c: Composition[State], declared: Capabilities[State, String, String]) =
      val failedCommitKeepsTheMessage = c.property holds (after =>
        after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
          (after.state.queue.outstanding != Outstanding.empty &&
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
        c.synced(_.activity -> worker.poll)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> worker.poll),
        c.own(_.activity, client.pause)
      )
      val duplicateDelivery = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> worker.poll),
        c.synced(_.activity -> worker.poll)
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

object TrustingRecordOverQueue
    extends DerivedComposition(RecordOverQueue.withMember(_.activity -> TrustingRecordMember)),
      NegativeControl:
  object capabilities extends OverQueueCapabilities(this)
  object queries:
    capabilities.bound(five)
    val trustingRecordOverQueueQueries =
      RecordOverQueue.queries.overQueueQueries(TrustingRecordOverQueue, capabilities)

// ### Over the detailed queue, which replaces the opaque one only within the composition: it holds
// where the detailed provider refines the interface it stands in for, and the checks then rely on
// that provider. Each later design swaps one member of the first for a provider of its interface.

object RecordOverMatching
    extends Composition[OverMatching](_.activity -> RecordMember, _.queue -> TaskQueueSystem),
      Phased[OverMatching, AdmissionPhase](_.activity.phase),
      FailureModel:
  override def end(s: State) = ActivityRecord.end(s.activity)
  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.poll, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerMatching, _.queue -> queue.acknowledge)
    replaces(_.queue, TaskQueueProduct)

  object properties:
    def overMatchingClaims(c: Composition[State], declared: Capabilities[State, String, String]) =
      OverMatchingClaims(
        declared.claim(Pausable.pausedIsNotDispatched[State, AdmissionPhase]),
        ActivityRecord.properties.atMostOneActive(c)(
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
        c.synced(_.activity -> worker.poll)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> worker.poll),
        c.own(_.activity, client.pause)
      )
      // The answer to the poller is lost after the commit, so the persisted task is handed out again.
      val deliveredAgainAfterLostAck = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> worker.poll),
        c.own(_.queue, fault.ackLoss),
        c.synced(_.activity -> worker.poll)
      )
      // A crash after the admission commit, before the acknowledgment: history retries the sync
      // match.
      val crashAfterAdmissionCommit = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> worker.poll),
        c.own(_.queue, fault.crash),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> worker.poll)
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

object TrustingRecordOverMatching
    extends DerivedComposition(RecordOverMatching.withMember(_.activity -> TrustingRecordMember)),
      NegativeControl:
  object capabilities extends OverMatchingCapabilities(this)
  object queries:
    capabilities.bound(twelve)
    val trustingRecordOverMatchingQueries =
      RecordOverMatching.queries.overMatchingQueries(TrustingRecordOverMatching, capabilities)

// ### The corrected design over each violating provider: the replacement is what must fail.

object RecordOverForgetful
    extends Composition(RecordOverMatching.withMember(_.queue -> ForgetfulQueue)),
      NegativeControl

object RecordOverVolatile
    extends Composition(RecordOverMatching.withMember(_.queue -> VolatileQueue)),
      NegativeControl

// The corrected design where storage loss is assumed, over the interface that allows it.
object RecordOverLossyMatching
    extends DerivedComposition(RecordOverMatching.withMember(_.queue -> LossyMatchingQueue)),
      FailureModel:
  object capabilities extends OverMatchingCapabilities(this)
  object queries:
    capabilities.bound(twelve)
    val recordOverLossyMatchingQueries =
      RecordOverMatching.queries.overMatchingQueries(RecordOverLossyMatching, capabilities)
