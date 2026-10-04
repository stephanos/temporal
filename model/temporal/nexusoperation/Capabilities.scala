/* What the standalone Nexus operation is, as the laws of model/temporal/capabilities read it: it
 * closes, a caller terminates it and requests its cancel, and DescribeNexusOperationExecution
 * reports its status. It receives the laws without listing them, each named `nexusOperation.<law>`.
 */
package temporal
package nexusoperation

import umpire.*
import temporal.capabilities.{given, *}

/**
 * Why the operation overrides closedIsRejectedUniformly: the server answers a control that repeats
 * a request id the operation took OK, after it closed too (operation.go RequestCancel, Terminate).
 */
val repeatedRequestsAnswer =
  "a repeated request id is answered OK after close: operation.go RequestCancel and Terminate"

/**
 * The rejection is the operation's own: alreadyCompleted, a FailedPrecondition, where the activity
 * answers NotFound (operation.go ErrOperationAlreadyCompleted). Each functional law's find starts
 * the operation, which no handler answers, so it stays running, then takes the control.
 */
val operationCapabilities = capabilities(nexusOperation, limits = three)(
  Closable(
    status = Operation.phase,
    terminal = Operation.terminal,
    rejected = Outcome.alreadyCompleted
  ),
  Terminable(
    terminate = terminate,
    settled = OperationFact.statusTerminated,
    reach = Seq(start),
    expect = inconclusive(explanationsDisagree)
  ),
  Cancelable(
    requestCancel = requestCancel,
    requested = OperationFact.statusCancelRequested,
    reach = Seq(start),
    expect = inconclusive(explanationsDisagree)
  ),
  Describable(status = OperationRealization.operationStatus)
).overriding(closedIsRejectedUniformly -> closedRejectsOrRepeats, because = repeatedRequestsAnswer)
