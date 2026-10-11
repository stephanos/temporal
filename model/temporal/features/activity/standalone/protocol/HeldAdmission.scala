// The HeldAdmission facet of HeldDispatch (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import framework.outcomes.Outcome

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait HeldAdmission extends Facet[AdmissionState, Outcome, AdmissionFact], Phased[AdmissionState, AdmissionPhase]:
  object heldAdmission:
    object effects:
      // Admission as the corrected design decides it, with no failure of its durable update.
      def admitCommitted(s: State) =
        enter(
          ActivityRecord.states.admitted(s),
          AdmissionFact.statusStarted,
          AdmissionFact.attemptAdmitted
        )

    object rules extends Rules:
      on(history.dispatch) {
        when(AdmissionPhase.scheduled) ~> ActivityRecord.admission.effects.sendDispatch
      }
      on(client.pause) {
        when(AdmissionPhase.scheduled) ~> ActivityRecord.admission.effects.pause
        when(AdmissionPhase.started) ~> ActivityRecord.admission.effects.pauseHeld
      }
      on(activityWorker.poll) {
        when(AdmissionPhase.scheduled) ~> effects.admitCommitted
        when(
          AdmissionPhase.paused,
          AdmissionPhase.pausedWhileHeld,
          AdmissionPhase.started,
          AdmissionPhase.completed,
          AdmissionPhase.timedOut
        ) ~> ActivityRecord.admission.effects.rejectDelivery
      }
      on(history.answerMatching) { // format: off
        where(_.answer == Answer.owed) ~> ActivityRecord.admission.effects.answerMatching // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on
