package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol


import framework.realize.*
import temporal.realize.*

object HeldFailureByIDRun
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.retry.queries.heldFailedByID
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitFailurePublication),
        everyCase(awaitExternalStarted),
        perform(completionClient.respondFailedByID(Failure.fatal) -> respondFailedById),
        everyCase(awaitExternalFailed),
        everyCase(readExternalAttemptCount)
      )

  object workers extends Workers(externalFailureAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalStarted,
        delivered(
          protocol.Fact.attemptCount,
          externalFailureAttempts,
          attempt = 2,
          after = startExternal
        ),
        externalFailed
      )

  object serverSteps extends ServerSteps(fatalExternalSettlement)
