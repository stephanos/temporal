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
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.product.*
import foundations.taskqueue.{QueueDetail, QueueView}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// ### Types

final case class OverQueue(activity: AdmissionState, queue: QueueView) derives CanEqual

final case class OverMatching(activity: AdmissionState, queue: QueueDetail) derives CanEqual

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
    dispatch = c.synced(_.activity -> activityWorker.poll)
  ) // format: off
  except(Closable.closedIsRejectedUniformly[OverQueue, AdmissionPhase, String], because = dispatchWaivers.queueStepsOn) // format: on

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
    dispatch = c.synced(_.activity -> activityWorker.poll)
  ) // format: off
  except(Closable.closedIsRejectedUniformly[OverMatching, AdmissionPhase, String], because = dispatchWaivers.queueStepsOn) // format: on

// ### The record as a member: a composition's Queries are not answered while a member names
// monitors, so the members are the designs without them. The designs are what refine the product.

object RecordMember extends Derived(ActivityRecord.unmonitored)
