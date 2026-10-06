/* The admission designs composed with the task queue (temporal/shared/taskqueue), over its opaque
 * contract and over the matching, violating and storage-loss providers that replace it. The queue
 * and its providers are the queue's; this file adds how the activity's dispatch, admission and
 * answer synchronize with it, and what the activity promises across both: the record's
 * capabilities, read through the `activity` member's projection, as the laws of
 * model/temporal/capabilities read them.
 *
 * Read top to bottom: the composed states and the claims they are held to; then the members,
 * CurrentRecord and StaleRecord, derived from the admission designs; then one object per
 * composition, each before the compositions derived from it -- CurrentOverQueue and StaleOverQueue
 * over the opaque queue, CurrentOverMatching and StaleOverMatching over the detailed one, and the
 * corrected design over each violating provider, CurrentOverForgetful, CurrentOverVolatile and
 * CurrentOverLossyMatching. A composition reads end, then its sections in order: states, syncs,
 * properties, implements and queries.
 */
package temporal
package features.standaloneactivity
package system

import umpire.*
import temporal.capabilities.{given, *}
import shared.Bounds.three
import shared.taskqueue.{fault, queue, seven, twelve, Outstanding, QueueDetail, QueueView}
import shared.taskqueue.product.DispatchQueue
import shared.taskqueue.system.{ForgetfulQueue, LossyMatchingQueue, MatchingQueue, VolatileQueue}

// ### Types

final case class OverQueue(activity: AdmissionState, queue: QueueView)

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

/**
 * The claims a design over the opaque queue is held to: the laws its capabilities bring, the
 * record's own count of active attempts over its member, and that a failed commit admits nothing and
 * leaves its message queued. `notPaused` is the generated `<design>.pausedIsNotDispatched`.
 */
final case class OverQueueClaims(
    notPaused: Property[OverQueue],
    oneActive: Property[OverQueue],
    failedCommit: Property[OverQueue]
)

final case class OverMatchingClaims(
    notPaused: Property[OverMatching],
    oneActive: Property[OverMatching]
)

// ### The record as a member: a composition's Queries are not answered while a member names
// monitors, so the members are the designs without them. The designs are what refine the product.

object CurrentRecord extends Derived(CurrentAdmission.unmonitored)

object StaleRecord extends Derived(StaleAdmission.unmonitored)

// ### Over the opaque queue. The dispatch is the queue's enqueue, a delivery is admission's one
// input, and admission's answer is the acknowledgment: a commit, its answer and the acknowledgment
// are three points a fault can fall between.

object CurrentOverQueue
    extends Composition[OverQueue](_.activity -> CurrentRecord, _.queue -> DispatchQueue):
  def end(s: State) = CurrentAdmission.end(s.activity)

  /** What the designs over a queue answer and waive. */
  object states:
    /**
     * The composed outcome of the record's answer to a control of a closed activity: Closable's
     * `rejected`, which only closedIsRejectedUniformly reads, and the designs over a queue waive it.
     */
    val closedAnswer = "activity_notFound"

    /**
     * Why a design over a queue waives closedIsRejectedUniformly: the queue member keeps its own
     * steps after the record closes, so a composed step moves the state; the record's own
     * declaration (Record.scala) holds the record to the law.
     */
    val queueStepsOn =
      "the queue member keeps stepping after the record closes; admissionCapabilities holds the record"

  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.attemptStart, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerDelivery, _.queue -> queue.acknowledge)

  object properties:
    def overQueueClaims(c: Composition[State]) =
      val declared = implements.overQueueCapabilities(c)
      val failedCommitKeepsTheMessage = c.property holds (after =>
        after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
          (after.state.queue.outstanding != Outstanding.empty &&
            !after.records(_.activity, AdmissionFact.attemptAdmitted))
      )
      OverQueueClaims(
        declared.claim(pausedIsNotDispatched),
        CurrentAdmission.properties.atMostOneActive(c)(
          through(_.activity, CurrentAdmission.states.twoActive)
        ),
        failedCommitKeepsTheMessage
      )

  object implements:
    /**
     * A design over the opaque queue as the laws read it: the record's capabilities, read through
     * the `activity` member's projection. Its free Queries run within `five`.
     */
    def overQueueCapabilities(c: Composition[State]) = capabilities(c, limits = five)(
      Closable(
        status = through(_.activity, CurrentAdmission.states.phase),
        terminal = CurrentAdmission.states.terminal,
        rejected = states.closedAnswer
      ),
      Pausable(
        pause = c.own(_.activity, caller.control(Control.pause)),
        unpause = c.own(_.activity, caller.control(Control.unpause)),
        paused = through(_.activity, CurrentAdmission.states.paused)
      ),
      Pollable(
        dispatch = c.synced(_.activity -> worker.attemptStart),
        running = through(_.activity, CurrentAdmission.states.running)
      )
    ).except(closedIsRejectedUniformly, because = states.queueStepsOn)

  object queries:
    /** Over the opaque queue: every claim and path of the design `c`. */
    def overQueueQueries(c: Composition[State]) =
      val claims = properties.overQueueClaims(c)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.activity, caller.control(Control.pause)),
        c.synced(_.activity -> worker.attemptStart)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> worker.attemptStart),
        c.own(_.activity, caller.control(Control.pause))
      )
      val duplicateDelivery = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.synced(_.activity -> worker.attemptStart),
        c.synced(_.activity -> worker.attemptStart)
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

    val currentOverQueueQueries = overQueueQueries(CurrentOverQueue)

object StaleOverQueue
    extends Composition(CurrentOverQueue.withMember(_.activity -> StaleRecord)),
      NegativeControl:
  object queries:
    val staleOverQueueQueries = CurrentOverQueue.queries.overQueueQueries(StaleOverQueue)

// ### Over the detailed queue, which replaces the opaque one only within the composition: it holds
// where the detailed provider refines the interface it stands in for, and the checks then rely on
// that provider. Each later design swaps one member of the first for a provider of its interface.

object CurrentOverMatching
    extends Composition[OverMatching](_.activity -> CurrentRecord, _.queue -> MatchingQueue),
      FailureModel:
  def end(s: State) = CurrentAdmission.end(s.activity)

  object syncs extends Syncs:
    sync(_.activity -> history.dispatch, _.queue -> queue.enqueue)
    sync("admit", _.activity -> worker.attemptStart, _.queue -> queue.deliver)
    sync("settle", _.activity -> history.answerDelivery, _.queue -> queue.acknowledge)
    replaces(_.queue, DispatchQueue)

  object properties:
    def overMatchingClaims(c: Composition[State]) =
      val declared = implements.overMatchingCapabilities(c)
      OverMatchingClaims(
        declared.claim(pausedIsNotDispatched),
        CurrentAdmission.properties.atMostOneActive(c)(
          through(_.activity, CurrentAdmission.states.twoActive)
        )
      )

  object implements:
    /**
     * A design over the detailed queue as the laws read it, through the `activity` member's
     * projection. Its free Queries run within `twelve`, the detailed provider's depth.
     */
    def overMatchingCapabilities(c: Composition[State]) = capabilities(c, limits = twelve)(
      Closable(
        status = through(_.activity, CurrentAdmission.states.phase),
        terminal = CurrentAdmission.states.terminal,
        rejected = CurrentOverQueue.states.closedAnswer
      ),
      Pausable(
        pause = c.own(_.activity, caller.control(Control.pause)),
        unpause = c.own(_.activity, caller.control(Control.unpause)),
        paused = through(_.activity, CurrentAdmission.states.paused)
      ),
      Pollable(
        dispatch = c.synced(_.activity -> worker.attemptStart),
        running = through(_.activity, CurrentAdmission.states.running)
      )
    ).except(closedIsRejectedUniformly, because = CurrentOverQueue.states.queueStepsOn)

  object queries:
    /** Over the detailed queue: every claim and path of the design `c`. */
    def overMatchingQueries(c: Composition[State]) =
      val claims = properties.overMatchingClaims(c)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.own(_.activity, caller.control(Control.pause)),
        c.synced(_.activity -> worker.attemptStart)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> worker.attemptStart),
        c.own(_.activity, caller.control(Control.pause))
      )
      // The answer to the poller is lost after the commit, so the persisted task is handed out again.
      val deliveredAgainAfterLostAck = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.persistTask),
        c.synced(_.activity -> worker.attemptStart),
        c.own(_.queue, fault.ackLoss),
        c.synced(_.activity -> worker.attemptStart)
      )
      // A crash after the admission commit, before the acknowledgment: history retries the sync
      // match.
      val crashAfterAdmissionCommit = c.scenario.actions(
        c.synced(_.activity -> history.dispatch),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> worker.attemptStart),
        c.own(_.queue, fault.crash),
        c.own(_.queue, queue.addActivityTask),
        c.own(_.queue, queue.syncMatch),
        c.synced(_.activity -> worker.attemptStart)
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

    val currentOverMatchingQueries = overMatchingQueries(CurrentOverMatching)

object StaleOverMatching
    extends Composition(CurrentOverMatching.withMember(_.activity -> StaleRecord)),
      NegativeControl:
  object queries:
    val staleOverMatchingQueries =
      CurrentOverMatching.queries.overMatchingQueries(StaleOverMatching)

// ### The corrected design over each violating provider: the replacement is what must fail.

object CurrentOverForgetful
    extends Composition(CurrentOverMatching.withMember(_.queue -> ForgetfulQueue)),
      NegativeControl

object CurrentOverVolatile
    extends Composition(CurrentOverMatching.withMember(_.queue -> VolatileQueue)),
      NegativeControl

/** The corrected design where storage loss is assumed, over the interface that allows it. */
object CurrentOverLossyMatching
    extends Composition(CurrentOverMatching.withMember(_.queue -> LossyMatchingQueue)),
      FailureModel:
  object queries:
    val currentOverLossyMatchingQueries =
      CurrentOverMatching.queries.overMatchingQueries(CurrentOverLossyMatching)
