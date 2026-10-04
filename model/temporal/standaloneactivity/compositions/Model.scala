/* The admission designs composed with the shared task queue (temporal/taskqueue): first over its
 * opaque contract, then over the detailed matching provider that replaces it, and over the
 * deliberately violating and storage-loss providers. The queue, its providers and what they promise
 * are the queue's; what this folder adds is how the activity's dispatch, admission and answer
 * synchronize with it and what the activity promises across both.
 */
package temporal
package standaloneactivity
package compositions

import umpire.*
import taskqueue.*
import admission.*
import SystemFamily.given

// The compositions were first written in the standalone activity's System.scala. Their state types
// keep the IR names they had there.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### The record as a member
//
// A composition's Queries are not answered while a member names monitors, so the members are the
// two designs again without them. They declare no refinement: the designs are what refine the
// product.

val currentRecord = currentAdmission.unmonitored
val staleRecord = staleAdmission.unmonitored

// ### A design over the opaque queue
//
// The dispatch is the queue's enqueue, a delivery is admission's one input, and admission's answer
// is the acknowledgment. Split that way, a commit, its answer and the acknowledgment are three
// points a fault can fall between.

final case class OverQueue(activity: AdmissionState, queue: QueueView)

/** The record's status sets, read through the composition's `activity` member. */
object OverQueue:
  def paused(s: OverQueue): Boolean = Admission.paused(s.activity)
  def running(s: OverQueue): Boolean = Admission.running(s.activity)
  def terminal(s: OverQueue): Boolean = Admission.terminal(s.activity)
  def twoActive(s: OverQueue): Boolean = Admission.twoActive(s.activity)
  def phase(s: OverQueue): AdmissionPhase = s.activity.phase

val currentOverQueue: Composition[OverQueue] =
  compose[OverQueue](_.activity -> currentRecord, _.queue -> dispatchQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .ends(s => Admission.ends(s.activity))

val staleOverQueue: Composition[OverQueue] =
  currentOverQueue.withMember(_.activity -> staleRecord)

// ### A design over the detailed queue, which replaces the opaque one
//
// The replacement is scoped to the composition: it holds only where the detailed provider refines
// the interface it stands in for, and its checks then rely on that provider, not on the interface.
// Each later design replaces one member of the first, and the interface it replaces is the one its
// new provider refines.

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

/** The record's status sets, read through the composition's `activity` member. */
object OverMatching:
  def paused(s: OverMatching): Boolean = Admission.paused(s.activity)
  def running(s: OverMatching): Boolean = Admission.running(s.activity)
  def terminal(s: OverMatching): Boolean = Admission.terminal(s.activity)
  def twoActive(s: OverMatching): Boolean = Admission.twoActive(s.activity)
  def phase(s: OverMatching): AdmissionPhase = s.activity.phase

val currentOverMatching: Composition[OverMatching] =
  compose[OverMatching](_.activity -> currentRecord, _.queue -> matchingQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .replaces(_.queue, dispatchQueue)
    .ends(s => Admission.ends(s.activity))

val staleOverMatching: Composition[OverMatching] =
  currentOverMatching.withMember(_.activity -> staleRecord)

/** The corrected design over each violating provider: the replacement is what must fail. */
val currentOverForgetful: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> forgetfulQueue)

val currentOverVolatile: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> volatileQueue)

/** The corrected design where storage loss is assumed, over the interface that allows it. */
val currentOverLossyMatching: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> lossyMatchingQueue)
