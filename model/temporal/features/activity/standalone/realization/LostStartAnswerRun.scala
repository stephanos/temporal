package temporal
package features.activity
package standalone
package realization

import protocol.{history, AdmissionResponseFact, LostStartAnswer}


import framework.realize.*
import temporal.realize.*

// One lost admission answer, with its durable decision observed before the response is replaced.
object LostStartAnswerRun
    extends Realizes(
      LostStartAnswer,
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(LostStartAnswer.queries)
    ):

  object controller
      extends Controller(
        everyCase(startUnreached),
        perform(history.dispatch -> holdDispatch),
        perform(foundations.taskqueue.fault.ackLoss -> loseAdmissionResponse),
        everyCase(readAttemptCount)
      )

  object evidence
      extends Evidences(
        answered(AdmissionResponseFact.dispatchSent, holdDispatch),
        // A lost answer's record also numbers the attempt admission committed, which must be one.
        committed(
          AdmissionResponseFact.attemptAdmitted,
          loseAdmissionResponse,
          fields = Vector(attemptField(Field(_.getDeliveryAdmission.attempt)))
        )(admitted, Condition.greater(Field(_.getDeliveryAdmission.attempt), Operand.integer(0)))
      )

  object controls extends Controls(dispatchHold)
