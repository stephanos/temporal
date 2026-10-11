package temporal
package features.activity
package standalone
package realization

import protocol.ActivityProtocol

import framework.realize.*
import temporal.realize.*

object ScheduledCompletionByIDRun
    extends Realizes(
      ActivityProtocol,
      learned = Vector(activityExecutionRun),
      observations = Vector(correlated, publicAttemptCount),
      runs = Seq(
        ActivityProtocol.completion.queries.scheduledCompletedByID
      )
    ):

  object controller
      extends Controller(
        perform(client.start() -> startExternal),
        perform(completionClient.respondCompletedByID -> respondCompletedById),
        everyCase(awaitExternalCompleted),
        everyCase(readExternalAttemptCount)
      )

  object evidence
      extends Evidences(
        answered(protocol.Fact.statusScheduled, startExternal),
        externalCompleted
      )

  object serverSteps extends ServerSteps(scheduledExternalSettlement)
