/* The admission designs' capabilities: the record, as a machine of its own, as the laws of
 * model/temporal/capabilities read it.
 */
package temporal
package standaloneactivity
package admission

import umpire.*
import temporal.capabilities.{given, *}

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
