package temporal
package standaloneactivity
package admission

import umpire.*

// ### What each design promises

/** The claims every admission design is held to, declared once per design by `admissionClaims`. */
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    terminal: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

/**
 * The claims of the design `m`: the three promises of the system contract over the record's status
 * sets, and each deadline timing the activity out with the status that says which it was. Each
 * takes its name explicitly, so every design's instance keeps the name its checks read.
 */
def admissionClaims(m: Machine[AdmissionState, Outcome, AdmissionFact]): AdmissionClaims =
  AdmissionClaims(
    notAdmittedWhilePaused(m)(Admission.paused, Admission.running),
    atMostOneActive(m)(Admission.twoActive),
    terminalStays(m)(Admission.terminal, Admission.phase),
    m.property("scheduleToStartTimesOut") when scheduleToStart holds (after =>
      after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
    ),
    m.property("scheduleToCloseTimesOut") when scheduleToClose holds (after =>
      after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
    )
  )

// ### What the machines a server's Run is checked against promise

/** Admission met the stale message and rejected it. */
val staleDeliveryRejected =
  heldAdmission.property when attemptStart holds (after =>
    after.records(AdmissionFact.admissionRejected)
  )

/** A lost response still leaves the attempt admitted when the update committed. */
val committedDespiteLostResponse =
  admissionResponseLoss.property when taskqueue.ackLoss holds (after =>
    after.records(AdmissionResponseFact.attemptAdmitted)
  )
