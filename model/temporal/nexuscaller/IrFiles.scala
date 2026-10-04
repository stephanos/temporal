/* The checked-in IR files of the Nexus caller Model (umpire.irFile). */
package temporal
package nexuscaller

import umpire.*

// The functional Queries and the realization that runs them are roots beside the machines: Go
// lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
// The product claim read on a protocol path and the cross-entity Query, which carries the
// composition with the handler's worker and its claim, are roots too.
val nexusCallerFile = irFile("nexus-caller")(
  nexusProduct,
  nexusProtocol,
  handlerWorker,
  nexusCaller,
  worker.polling,
  functionalQueries,
  terminalHolds,
  stoppedWorkerRepliesNothing,
  NexusRealization.asyncNexus
)

// The forged completion a caller must refuse, and the realization that offers it.
val nexusControlFile =
  irFile("nexus-control")(Control.forgedCompletion, NexusRealization.forgedCompletion)
