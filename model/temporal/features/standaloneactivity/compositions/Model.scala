/* The admission designs composed with the task queue (temporal/shared/taskqueue), over its opaque
 * contract and over the matching, violating and storage-loss providers that replace it. The queue
 * and its providers are the queue's; this folder adds how the activity's dispatch, admission and
 * answer synchronize with it, and what the activity promises across both.
 */
package temporal
package features.standaloneactivity
package compositions

import umpire.*
import shared.taskqueue.*
import admission.*
import SystemFamily.given

// First written in System.scala: the state types keep the IR names they had there.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### The record as a member: a composition's Queries are not answered while a member names
// monitors, so the members are the designs without them. The designs are what refine the product.

val currentRecord = currentAdmission.unmonitored
val staleRecord = staleAdmission.unmonitored

// ### Over the opaque queue. The dispatch is the queue's enqueue, a delivery is admission's one
// input, and admission's answer is the acknowledgment: a commit, its answer and the acknowledgment
// are three points a fault can fall between.

final case class OverQueue(activity: AdmissionState, queue: QueueView)

/** The record's status sets, read through the composition's `activity` member. */
object OverQueue:
  def paused(s: OverQueue) = Admission.paused(s.activity)
  def running(s: OverQueue) = Admission.running(s.activity)
  def twoActive(s: OverQueue) = Admission.twoActive(s.activity)
  def phase(s: OverQueue) = s.activity.phase

val currentOverQueue =
  compose[OverQueue](_.activity -> currentRecord, _.queue -> dispatchQueue)
    .sync(_.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .ends(s => Admission.ends(s.activity))

val staleOverQueue = currentOverQueue.withMember(_.activity -> staleRecord)

// ### Over the detailed queue, which replaces the opaque one only within the composition: it holds
// where the detailed provider refines the interface it stands in for, and the checks then rely on
// that provider. Each later design swaps one member of the first for a provider of its interface.

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

/** The record's status sets, read through the composition's `activity` member. */
object OverMatching:
  def paused(s: OverMatching) = Admission.paused(s.activity)
  def running(s: OverMatching) = Admission.running(s.activity)
  def twoActive(s: OverMatching) = Admission.twoActive(s.activity)
  def phase(s: OverMatching) = s.activity.phase

val currentOverMatching =
  compose[OverMatching](_.activity -> currentRecord, _.queue -> matchingQueue)
    .sync(_.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .replaces(_.queue, dispatchQueue)
    .ends(s => Admission.ends(s.activity))

val staleOverMatching = currentOverMatching.withMember(_.activity -> staleRecord)

/** The corrected design over each violating provider: the replacement is what must fail. */
val currentOverForgetful = currentOverMatching.withMember(_.queue -> forgetfulQueue)
val currentOverVolatile = currentOverMatching.withMember(_.queue -> volatileQueue)

/** The corrected design where storage loss is assumed, over the interface that allows it. */
val currentOverLossyMatching = currentOverMatching.withMember(_.queue -> lossyMatchingQueue)
