package temporal
package standaloneactivity
package compositions

import umpire.*
import taskqueue.Outstanding
import admission.AdmissionFact

// ### What a design over the opaque queue promises

/** The claims a design over the opaque queue is held to, declared once by `overQueueClaims`. */
final case class OverQueueClaims(
    notPaused: Property[OverQueue],
    oneActive: Property[OverQueue],
    terminal: Property[OverQueue],
    failedCommit: Property[OverQueue]
)

/**
 * The three promises of the system contract over the record's member, and a failed commit, which
 * admits nothing and leaves its message with the queue.
 */
def overQueueClaims(c: Composition[OverQueue]): OverQueueClaims = OverQueueClaims(
  notAdmittedWhilePaused(c)(OverQueue.paused, OverQueue.running),
  atMostOneActive(c)(OverQueue.twoActive),
  terminalStays(c)(OverQueue.terminal, OverQueue.phase),
  c.property("failedCommitKeepsTheMessage") holds (after =>
    after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
      (after.state.queue.outstanding != Outstanding.empty &&
        !after.records(_.activity, AdmissionFact.attemptAdmitted))
  )
)

// ### What a design over the detailed queue promises

/** The claims a design over the detailed queue is held to, declared once by `overMatchingClaims`. */
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
