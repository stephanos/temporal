package temporal
package features.activity
package standalone
package realization

import protocol.{history, AdmissionFact, HeldDispatch}

import framework.realize.*
import temporal.realize.*

// The stale dispatch of one paused activity, held, then delivered to admission. The machine starts
// scheduled, so every Case carries the start; its one deadline, a start-to-close no Case lives to
// see, competes with no delivery.
object HeldDispatchRun
    extends Realizes(
      HeldDispatch,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(HeldDispatch.queries)
    ):

  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(client.pause -> pauseActivity),
        onPath(client.pause)(pauseDescribed.await(AdmissionFact.statusPaused)),
        perform(activityWorker.poll -> releaseDispatch),
        everyCase(readAttemptCount)
      )

  object evidence
      extends Evidences(
        answered(AdmissionFact.dispatchSent, holdDispatch),
        pauseDescribed(AdmissionFact.statusPaused),
        committed(AdmissionFact.admissionRejected, releaseDispatch)(rejected),
        committed(AdmissionFact.attemptAdmitted, releaseDispatch, exhaustive = true)(admitted)
      )

  object controls extends Controls(dispatchHold)
