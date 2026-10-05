package temporal
package features.standaloneactivity
package admission

import umpire.*
import temporal.capabilities.pausedIsNotDispatched

/**
 * The claims every admission design is held to: the laws its capabilities bring, the record's own
 * count of active attempts, and each deadline timing the activity out with the status that says which.
 * `notPaused` is the generated `<design>.pausedIsNotDispatched`, which the pinned paths read.
 */
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

def admissionClaims(m: Machine[AdmissionState, Outcome, AdmissionFact]) =
  val declared = admissionCapabilities(m)
  val scheduleToStartTimesOut = m.property when scheduleToStart holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
  )
  val scheduleToCloseTimesOut = m.property when scheduleToClose holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
  )
  AdmissionClaims(
    declared.claim(pausedIsNotDispatched),
    atMostOneActive(m)(Admission.twoActive),
    scheduleToStartTimesOut,
    scheduleToCloseTimesOut
  )

// ### What the machines a server's Run is checked against promise

/** Admission met the stale message and rejected it. */
val staleDeliveryRejected =
  heldAdmission.property when attemptStart holds (_.records(AdmissionFact.admissionRejected))

/** A lost response still leaves the attempt admitted when the update committed. */
val committedDespiteLostResponse =
  admissionResponseLoss.property when shared.taskqueue.ackLoss holds (after =>
    after.records(AdmissionResponseFact.attemptAdmitted)
  )
