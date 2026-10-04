package temporal
package standaloneactivity
package admission

import umpire.*
import umpire.laws.terminalStatesAreFinal
import temporal.laws.pausedIsNotDispatched

/**
 * The claims every admission design is held to: the three promises of the system contract over the
 * record's status sets, and each deadline timing the activity out with the status that says which.
 */
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    terminal: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

def admissionClaims(m: Machine[AdmissionState, Outcome, AdmissionFact]): AdmissionClaims =
  val notAdmittedWhilePaused = pausedIsNotDispatched(m)(Admission.paused, Admission.running)
  val terminalStays = terminalStatesAreFinal(m)(Admission.phase, Admission.terminal)
  val scheduleToStartTimesOut = m.property when scheduleToStart holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
  )
  val scheduleToCloseTimesOut = m.property when scheduleToClose holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
  )
  AdmissionClaims(
    notAdmittedWhilePaused,
    atMostOneActive(m)(Admission.twoActive),
    terminalStays,
    scheduleToStartTimesOut,
    scheduleToCloseTimesOut
  )

// ### What the machines a server's Run is checked against promise

/** Admission met the stale message and rejected it. */
val staleDeliveryRejected =
  heldAdmission.property when attemptStart holds (_.records(AdmissionFact.admissionRejected))

/** A lost response still leaves the attempt admitted when the update committed. */
val committedDespiteLostResponse =
  admissionResponseLoss.property when taskqueue.ackLoss holds (after =>
    after.records(AdmissionResponseFact.attemptAdmitted)
  )
