package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol


import framework.realize.*
import temporal.realize.*

object HeldCancellationByIDRun
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.cancellation.queries.heldCanceledByID,
        ActivityProtocol.cancellation.queries.cancelIsRequested
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        everyCase(awaitCancellationPublication),
        everyCase(awaitExternalStarted),
        perform(client.requestCancel -> requestExternalCancellation),
        everyCase(awaitExternalCancelRequested),
        perform(completionClient.respondCanceledByID -> respondCanceledById),
        everyCase(awaitExternalCanceled),
        everyCase(readExternalAttemptCount)
      )

  object workers extends Workers(externalCancellationAttempts)

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalStarted,
        delivered(
          protocol.Fact.attemptCount,
          externalCancellationAttempts,
          attempt = 2,
          after = startExternal
        ),
        externalCancelRequested,
        externalCanceled
      )

  object serverSteps extends ServerSteps(canceledExternalSettlement)
