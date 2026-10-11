package temporal
package features.nexus
package standalone
package realization

import protocol.{Fact as OperationFact, NexusProtocol}
import framework.realize.*
import temporal.realize.*

// One standalone operation a controller starts on the Case's endpoint.
object StandaloneNexusRun
    extends Realizes(
      NexusProtocol,
      requiredSettings = Vector(standaloneEnabled),
      runs = Seq(NexusProtocol.queries, NexusProtocol.capabilities)
    ):

  object controller
      extends Controller(
        perform(client.start -> startOperation),
        perform(client.requestCancel -> requestCancelOperation),
        perform(client.terminate -> terminateOperation),
        onPath(client.terminate)(described.await(OperationFact.nexusOperationTerminated))
      )

  object evidence
      extends Evidences(
        answered(OperationFact.statusScheduled, startOperation),
        answered(OperationFact.statusCancelRequested, requestCancelOperation),
        described(OperationFact.nexusOperationTerminated)
      )
