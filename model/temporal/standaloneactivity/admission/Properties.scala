package temporal
package standaloneactivity
package admission

import umpire.*
import umpire.laws.closedIsRejectedUniformly
import temporal.laws.{pausedIsNotDispatched, given}

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

/**
 * What an admission design is as the laws read it, the record as a machine of its own: it closes, it
 * pauses before an attempt is admitted, and its work is handed out by a worker's poll. Its free
 * Queries run within `five`, as the system contract's did. It waives closedIsRejectedUniformly, the
 * one law the record does not keep.
 */
def admissionCapabilities(m: Machine[AdmissionState, Outcome, AdmissionFact]) =
  capabilities(m, limits = five)(
    Closable(status = Admission.phase, terminal = Admission.terminal, rejected = Outcome.notFound),
    Pausable(
      pause = control(Control.pause),
      unpause = control(Control.unpause),
      paused = Admission.paused
    ),
    Pollable(dispatch = attemptStart, running = Admission.running)
  ).except(closedIsRejectedUniformly, because = deliveryAfterClose)

/**
 * Why the record waives closedIsRejectedUniformly: a delivery that reaches a closed record is
 * admission's to reject, and rejecting it records `admissionRejected` and owes matching the answer, so
 * the record steps where the law has it stay (chasm/lib/activity/activity.go HandleStarted).
 */
val deliveryAfterClose =
  "admission rejects a delivery to a closed record and owes matching its answer: activity.go HandleStarted"

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
  admissionResponseLoss.property when taskqueue.ackLoss holds (after =>
    after.records(AdmissionResponseFact.attemptAdmitted)
  )
