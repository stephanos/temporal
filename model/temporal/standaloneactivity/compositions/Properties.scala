package temporal
package standaloneactivity
package compositions

import umpire.*
import temporal.capabilities.pausedIsNotDispatched
import taskqueue.Outstanding
import admission.AdmissionFact

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

def overQueueClaims(c: Composition[OverQueue]) =
  val declared = overQueueCapabilities(c)
  val failedCommitKeepsTheMessage = c.property holds (after =>
    after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
      (after.state.queue.outstanding != Outstanding.empty &&
        !after.records(_.activity, AdmissionFact.attemptAdmitted))
  )
  OverQueueClaims(
    declared.claim(pausedIsNotDispatched),
    atMostOneActive(c)(OverQueue.twoActive),
    failedCommitKeepsTheMessage
  )

final case class OverMatchingClaims(
    notPaused: Property[OverMatching],
    oneActive: Property[OverMatching]
)

def overMatchingClaims(c: Composition[OverMatching]) =
  val declared = overMatchingCapabilities(c)
  OverMatchingClaims(
    declared.claim(pausedIsNotDispatched),
    atMostOneActive(c)(OverMatching.twoActive)
  )
