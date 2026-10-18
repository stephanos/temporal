package temporal
package features.nexus
package workflow
package realization

import temporal.realize.*

import protocol.NexusProtocol
import CallerDeclarations.*

// One Nexus operation scheduled by a controller-started workflow and answered by a handler inside
// the Case's own worker. Its server steps are derived: a timeout class fires at the deadline its
// schedule command sets.
object WorkflowNexusRun
    extends Realizes(
      NexusProtocol,
      learned = Vector(completionAuthority),
      observations = Vector(historyEvent, correlated),
      runs = Seq(NexusProtocol.queries, NexusProtocol.capabilities)
    ):

  object controller extends Controller(callerItems*)

  object workers extends Workers(workflowScript, handlerScript)

  object evidence extends Evidences((Vector(scheduled) ++ historyKinds.evidence :+ pending)*)
