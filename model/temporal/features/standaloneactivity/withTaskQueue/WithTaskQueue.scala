/* The admission designs composed with the task queue (temporal/shared/taskqueue), over its opaque
 * contract and over the matching, violating and storage-loss providers that replace it. The queue
 * and its providers are the queue's; this folder adds how the activity's dispatch, admission and
 * answer synchronize with it, and what the activity promises across both: the record's
 * capabilities, read through the `activity` member's projection, as the laws of
 * model/temporal/capabilities read them.
 *
 * Read top to bottom: the composed states and the claims they are held to; then the members,
 * CurrentRecord and StaleRecord; then one object per composition, each before the designs derived
 * from it -- CurrentOverQueue and StaleOverQueue over the opaque queue, CurrentOverMatching and
 * StaleOverMatching over the detailed one, and the corrected design over each violating provider,
 * CurrentOverForgetful, CurrentOverVolatile and CurrentOverLossyMatching.
 */
package temporal
package features.standaloneactivity
package withTaskQueue

import umpire.*
import temporal.capabilities.{given, *}
import shared.taskqueue.*
import record.*
import SystemFamily.given

// First written in System.scala: the state types keep the IR names they had there.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

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

object CurrentRecord:
  val currentRecord = Admission.currentAdmission.unmonitored

object StaleRecord:
  val staleRecord = StaleAdmission.staleAdmission.unmonitored

// ### Over the opaque queue. The dispatch is the queue's enqueue, a delivery is admission's one
// input, and admission's answer is the acknowledgment: a commit, its answer and the acknowledgment
// are three points a fault can fall between.

object CurrentOverQueue:
  /**
   * The composed outcome of the record's answer to a control of a closed activity: Closable's
   * `rejected`, which only closedIsRejectedUniformly reads, and the designs over a queue waive it.
   */
  val closedAnswer = "activity_notFound"

  /**
   * Why a design over a queue waives closedIsRejectedUniformly: the queue member keeps its own steps
   * after the record closes, so a composed step moves the state; the record's own declaration
   * (record/) holds the record to the law.
   */
  val queueStepsOn =
    "the queue member keeps stepping after the record closes; admissionCapabilities holds the record"

  val currentOverQueue =
    compose[OverQueue](_.activity -> CurrentRecord.currentRecord, _.queue -> dispatchQueue)
      .sync(_.activity -> dispatch, _.queue -> enqueue)
      .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
      .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
      .ends(s => Admission.end(s.activity))

  object properties:
    def overQueueClaims(c: Composition[OverQueue]) =
      val declared = laws.overQueueCapabilities(c)
      val failedCommitKeepsTheMessage = c.property holds (after =>
        after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
          (after.state.queue.outstanding != Outstanding.empty &&
            !after.records(_.activity, AdmissionFact.attemptAdmitted))
      )
      OverQueueClaims(
        declared.claim(pausedIsNotDispatched),
        Admission.properties.atMostOneActive(c)(through(_.activity, Admission.twoActive)),
        failedCommitKeepsTheMessage
      )

  object laws:
    /**
     * A design over the opaque queue as the laws read it: the record's capabilities, read through
     * the `activity` member's projection. Its free Queries run within `five`.
     */
    def overQueueCapabilities(c: Composition[OverQueue]) = capabilities(c, limits = five)(
      Closable(
        status = through(_.activity, Admission.phase),
        terminal = Admission.terminal,
        rejected = closedAnswer
      ),
      Pausable(
        pause = c.own(_.activity, control(Control.pause)),
        unpause = c.own(_.activity, control(Control.unpause)),
        paused = through(_.activity, Admission.paused)
      ),
      Pollable(
        dispatch = c.synced(_.activity -> attemptStart),
        running = through(_.activity, Admission.running)
      )
    ).except(closedIsRejectedUniformly, because = queueStepsOn)

  object queries:
    /** Over the opaque queue: every claim and path of the design `c`. */
    def overQueueQueries(c: Composition[OverQueue]) =
      val claims = properties.overQueueClaims(c)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.own(_.activity, control(Control.pause)),
        c.synced(_.activity -> attemptStart)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.synced(_.activity -> attemptStart),
        c.own(_.activity, control(Control.pause))
      )
      val duplicateDelivery = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.synced(_.activity -> attemptStart),
        c.synced(_.activity -> attemptStart)
      )
      val any = c.scenario.free
      Vector(
        query(s"${c.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits three total 432,
        query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits three total 432,
        query(s"${c.name}.duplicateDelivery") verify claims.oneActive in
          duplicateDelivery limits three total 432,
        query(s"${c.name}.failedCommit") verify claims.failedCommit in
          duplicateDelivery limits three total 432,
        query verify claims.oneActive in any limits five total 9360
      )

    val currentOverQueueQueries = overQueueQueries(currentOverQueue)

object StaleOverQueue:
  val staleOverQueue =
    CurrentOverQueue.currentOverQueue.withMember(_.activity -> StaleRecord.staleRecord)

  object queries:
    val staleOverQueueQueries = CurrentOverQueue.queries.overQueueQueries(staleOverQueue)

// ### Over the detailed queue, which replaces the opaque one only within the composition: it holds
// where the detailed provider refines the interface it stands in for, and the checks then rely on
// that provider. Each later design swaps one member of the first for a provider of its interface.

object CurrentOverMatching:
  val currentOverMatching =
    compose[OverMatching](_.activity -> CurrentRecord.currentRecord, _.queue -> matchingQueue)
      .sync(_.activity -> dispatch, _.queue -> enqueue)
      .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
      .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
      .replaces(_.queue, dispatchQueue)
      .ends(s => Admission.end(s.activity))

  object properties:
    def overMatchingClaims(c: Composition[OverMatching]) =
      val declared = laws.overMatchingCapabilities(c)
      OverMatchingClaims(
        declared.claim(pausedIsNotDispatched),
        Admission.properties.atMostOneActive(c)(through(_.activity, Admission.twoActive))
      )

  object laws:
    /**
     * A design over the detailed queue as the laws read it, through the `activity` member's
     * projection. Its free Queries run within `twelve`, the detailed provider's depth.
     */
    def overMatchingCapabilities(c: Composition[OverMatching]) = capabilities(c, limits = twelve)(
      Closable(
        status = through(_.activity, Admission.phase),
        terminal = Admission.terminal,
        rejected = CurrentOverQueue.closedAnswer
      ),
      Pausable(
        pause = c.own(_.activity, control(Control.pause)),
        unpause = c.own(_.activity, control(Control.unpause)),
        paused = through(_.activity, Admission.paused)
      ),
      Pollable(
        dispatch = c.synced(_.activity -> attemptStart),
        running = through(_.activity, Admission.running)
      )
    ).except(closedIsRejectedUniformly, because = CurrentOverQueue.queueStepsOn)

  object queries:
    /** Over the detailed queue: `anyTotal` is the static combination count of the `any` Queries. */
    def overMatchingQueries(c: Composition[OverMatching], anyTotal: Int) =
      val claims = properties.overMatchingClaims(c)
      val staleDeliveryAfterPause = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.own(_.queue, addActivityTask),
        c.own(_.queue, persistTask),
        c.own(_.activity, control(Control.pause)),
        c.synced(_.activity -> attemptStart)
      )
      val admittedBeforePause = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.own(_.queue, addActivityTask),
        c.own(_.queue, persistTask),
        c.synced(_.activity -> attemptStart),
        c.own(_.activity, control(Control.pause))
      )
      // The answer to the poller is lost after the commit, so the persisted task is handed out again.
      val deliveredAgainAfterLostAck = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.own(_.queue, addActivityTask),
        c.own(_.queue, persistTask),
        c.synced(_.activity -> attemptStart),
        c.own(_.queue, ackLoss),
        c.synced(_.activity -> attemptStart)
      )
      // A crash after the admission commit, before the acknowledgment: history retries the sync
      // match.
      val crashAfterAdmissionCommit = c.scenario.actions(
        c.synced(_.activity -> dispatch),
        c.own(_.queue, addActivityTask),
        c.own(_.queue, syncMatch),
        c.synced(_.activity -> attemptStart),
        c.own(_.queue, crash),
        c.own(_.queue, addActivityTask),
        c.own(_.queue, syncMatch),
        c.synced(_.activity -> attemptStart)
      )
      val any = c.scenario.free
      Vector(
        query(s"${c.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits five total 5400,
        query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits five total 5400,
        query(s"${c.name}.deliveredAgainAfterLostAck") verify claims.oneActive in
          deliveredAgainAfterLostAck limits seven total 6480,
        query(s"${c.name}.crashAfterAdmissionCommit") verify claims.oneActive in
          crashAfterAdmissionCommit limits eight total 8640,
        query verify claims.oneActive in any limits twelve total anyTotal
      )

    val currentOverMatchingQueries = overMatchingQueries(currentOverMatching, anyTotal = 233280)

object StaleOverMatching:
  val staleOverMatching =
    CurrentOverMatching.currentOverMatching.withMember(_.activity -> StaleRecord.staleRecord)

  object queries:
    val staleOverMatchingQueries =
      CurrentOverMatching.queries.overMatchingQueries(staleOverMatching, anyTotal = 233280)

// ### The corrected design over each violating provider: the replacement is what must fail.

object CurrentOverForgetful:
  val currentOverForgetful =
    CurrentOverMatching.currentOverMatching.withMember(_.queue -> forgetfulQueue)

object CurrentOverVolatile:
  val currentOverVolatile =
    CurrentOverMatching.currentOverMatching.withMember(_.queue -> volatileQueue)

/** The corrected design where storage loss is assumed, over the interface that allows it. */
object CurrentOverLossyMatching:
  val currentOverLossyMatching =
    CurrentOverMatching.currentOverMatching.withMember(_.queue -> lossyMatchingQueue)

  object queries:
    val currentOverLossyMatchingQueries =
      CurrentOverMatching.queries.overMatchingQueries(currentOverLossyMatching, anyTotal = 246240)
