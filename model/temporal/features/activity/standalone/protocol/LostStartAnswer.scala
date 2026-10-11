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
  val init = AdmissionResponseState(record = ActivityRecord.init, lossAvailable = true)
  def end(s: State) = !s.lossAvailable

  object rules extends Facets(responseLoss.rules)

  object properties:
    // A lost response still leaves the attempt admitted when the update committed.
    val committedDespiteLostResponse =
      property when foundations.taskqueue.fault.ackLoss holds (after =>
        after.records(AdmissionResponseFact.attemptAdmitted)
      )

  // The lost response, as a server's Run is checked.
  object queries:
    val oneLostResponse = scenario.actions(history.dispatch, foundations.taskqueue.fault.ackLoss)
    val lostAdmissionResponseQuery =
      (query("lostStartAnswer.committed") find properties.committedDespiteLostResponse in
        oneLostResponse limits three)
        .live(satisfied)
