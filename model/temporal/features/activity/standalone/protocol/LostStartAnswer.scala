package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.realize.satisfied
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object LostStartAnswer
    extends Machine[AdmissionResponseState, Outcome, AdmissionResponseFact],
      FailureModel,
      ResponseLoss:
  val entity = activity
  val init =
    AdmissionResponseState(record = ActivityRecord.init, lossAvailable = true)
  def end(s: State) = !s.lossAvailable

  object rules extends Facets(responseLoss.rules)

  // The lost response, as a server's Run is checked.
  object queries:
    val lostAdmissionResponseQuery =
      witness("lostStartAnswer.committed")(
        history.dispatch,
        foundations.taskqueue.fault.ackLoss
      )
        .records(AdmissionResponseFact.attemptAdmitted)
        .limits(three)
        .live(satisfied)
