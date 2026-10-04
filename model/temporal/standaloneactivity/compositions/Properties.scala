package temporal
package standaloneactivity
package compositions

import umpire.*
import umpire.laws.terminalStatesAreFinal
import temporal.laws.pausedIsNotDispatched
import taskqueue.Outstanding
import admission.{Admission, AdmissionFact}

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
  val notAdmittedWhilePaused = pausedIsNotDispatched(c)(OverQueue.paused, OverQueue.running)
  val terminalStays = terminalStatesAreFinal(c)(OverQueue.phase, Admission.terminal)
  OverQueueClaims(
    notAdmittedWhilePaused,
    atMostOneActive(c)(OverQueue.twoActive),
    terminalStays,
    failedCommitKeepsTheMessage
  )

final case class OverMatchingClaims(
    notPaused: Property[OverMatching],
    oneActive: Property[OverMatching],
    terminal: Property[OverMatching]
)

def overMatchingClaims(c: Composition[OverMatching]): OverMatchingClaims =
  val notAdmittedWhilePaused = pausedIsNotDispatched(c)(OverMatching.paused, OverMatching.running)
  val terminalStays = terminalStatesAreFinal(c)(OverMatching.phase, Admission.terminal)
  OverMatchingClaims(
    notAdmittedWhilePaused,
    atMostOneActive(c)(OverMatching.twoActive),
    terminalStays
  )
