// The ResponseLoss facet of LostStartAnswer (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import framework.outcomes.Outcome

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait ResponseLoss extends Facet[AdmissionResponseState, Outcome, AdmissionResponseFact]:
  object responseLoss:
    object effects:
      def sendDispatch(s: State) = enter(s, AdmissionResponseFact.dispatchSent)

      def loseResponse(s: State) = choose(
        committedThenLost -> enter(
          AdmissionResponseState(
            record = ActivityRecord.states.admitted(s.record),
            lossAvailable = false
          ),
          AdmissionResponseFact.attemptAdmitted
        ),
        failedThenLost -> enter(s.copy(lossAvailable = false))
          .because("the durable update failed before its answer was lost")
      )

    // Both steps take the budget's state: once a loss consumes it, neither fires.
    object rules extends Rules:
      on(history.dispatch) {
        where(_.lossAvailable) ~> effects.sendDispatch
      }
      on(foundations.taskqueue.fault.ackLoss) {
        where(_.lossAvailable) ~> effects.loseResponse
      }
