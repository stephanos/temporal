/* What the standalone Nexus operation promises of its own: its reading of closed rejection, which
 * its capabilities (Capabilities.scala) put in place of the law's.
 */
package temporal
package nexusoperation

import umpire.*

/**
 * A closed operation keeps its state, and answers a control alreadyCompleted, or OK where it repeats
 * a request the operation took, a recorded cancel or the terminate that closed it: the operation's
 * own reading of closedIsRejectedUniformly.
 */
def closedRejectsOrRepeats(m: Machine[OperationState, Outcome, OperationFact])(
    status: OperationState => Phase,
    terminal: Phase => Boolean,
    rejected: Outcome
): Property[OperationState] =
  m.property holdsAcross ((before, after) =>
    !terminal(status(before)) ||
      (after.state == before && (after.outcome == rejected || after.outcome == Outcome.accepted &&
        (before.cancelRequested || status(before) == Phase.terminated)))
  )
