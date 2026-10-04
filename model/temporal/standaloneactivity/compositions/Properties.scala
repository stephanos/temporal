package temporal
package standaloneactivity
package compositions

import umpire.*
import taskqueue.Outstanding
import admission.AdmissionFact

/**
 * The claims a design over the opaque queue is held to: the system contract's three promises over
 * the record's member, and that a failed commit admits nothing and leaves its message queued.
 */
final case class OverQueueClaims(
    notPaused: Property[OverQueue],
    oneActive: Property[OverQueue],
    terminal: Property[OverQueue],
    failedCommit: Property[OverQueue]
)

def overQueueClaims(c: Composition[OverQueue]): OverQueueClaims =
  val failedCommitKeepsTheMessage = c.property holds (after =>
    after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
      (after.state.queue.outstanding != Outstanding.empty &&
        !after.records(_.activity, AdmissionFact.attemptAdmitted))
  )
  OverQueueClaims(
    notAdmittedWhilePaused(c)(OverQueue.paused, OverQueue.running),
    atMostOneActive(c)(OverQueue.twoActive),
    terminalStays(c)(OverQueue.terminal, OverQueue.phase),
    failedCommitKeepsTheMessage
  )

final case class OverMatchingClaims(
    notPaused: Property[OverMatching],
    oneActive: Property[OverMatching],
    terminal: Property[OverMatching]
)

def overMatchingClaims(c: Composition[OverMatching]): OverMatchingClaims = OverMatchingClaims(
  notAdmittedWhilePaused(c)(OverMatching.paused, OverMatching.running),
  atMostOneActive(c)(OverMatching.twoActive),
  terminalStays(c)(OverMatching.terminal, OverMatching.phase)
)
