/* What the Nexus caller's machines promise, the cross-entity claim of the operation and its
 * handler's worker, and the claim the forged control breaks.
 */
package temporal
package nexuscaller

import umpire.*
import worker.Phase as WorkerPhase

// A same-step claim names the action it is about under `when` and holds of the step that action
// produces; a transition claim holds of the state before and the step after. A functional Query
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.

/**
 * Once an operation is over, no step changes its phase. Declared on the product machine and read on
 * the protocol machine through the map.
 */
val terminalIsFinal = nexusProduct.property.once(Product.productTerminal).keeps(_.phase)

/** A synchronous reply settles the operation as succeeded, and the completed event records it. */
val syncSucceeds = nexusProtocol.property when handlerReply(Reply.syncSuccess) holds { s =>
  s.state.phase == Phase.succeeded && s.records(ProtocolFact.nexusOperationCompleted)
}

/** An asynchronous reply starts the operation, and the started event records it. */
val asyncStarts = nexusProtocol.property when handlerReply(Reply.async) holds { s =>
  s.state.phase == Phase.started && s.records(ProtocolFact.nexusOperationStarted)
}

/**
 * A successful completion is recorded by the completed event. Neither the phase nor the outcome is
 * fixed: a completion resolves any running phase, and accepted is every earlier step's outcome
 * too, so a clause fixing it would be answered before the completion.
 */
val completionSucceeds = nexusProtocol.property when complete(Resolution.succeeded) holds
  (_.records(ProtocolFact.nexusOperationCompleted))

/** A failed completion is recorded by the failed event. */
val completionFails = nexusProtocol.property when complete(Resolution.failed) holds
  (_.records(ProtocolFact.nexusOperationFailed))

/** A non-retryable handler error settles the operation as failed, and the failed event records it. */
val handlerErrorFails =
  nexusProtocol.property when handlerReply(Reply.handlerError(false)) holds { s =>
    s.state.phase == Phase.failed && s.records(ProtocolFact.nexusOperationFailed)
  }

/**
 * Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
 * so every field is named.
 */
val succeededOnRetry =
  ProtocolState(Phase.succeeded, 1, Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * A synchronous reply to the retried attempt settles the operation as succeeded on its second
 * attempt: the count the retryable failure raised is still one, and the completed event records the
 * reply.
 */
val retrySucceeds = nexusProtocol.property when handlerReply(Reply.syncSuccess) holds { s =>
  s.state == succeededOnRetry && s.records(ProtocolFact.nexusOperationCompleted)
}

/**
 * The schedule-to-start deadline settles an operation no handler started as timed out, and the
 * timed-out event records which deadline it was.
 */
val scheduleToStartFires = nexusProtocol.property when scheduleToStart holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart))
}

/** The start-to-close deadline settles a started operation no handler completed as timed out. */
val startToCloseFires = nexusProtocol.property when startToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose))
}

/**
 * The cross-entity claim: every reply, of any class, leaves the handler's worker polling, so no
 * handler replies while its worker is stopped.
 */
val repliedByPollingWorker = nexusCaller.property
  .whenAction(nexusCaller.synced(_.operation -> handlerReply))
  .holds(_.state.worker.phase == WorkerPhase.polling)

/** A failed completion is recorded as completed: what the control predicts and no runtime sends. */
val forgedSuccess = Control.forgedCompletion.property when complete(Resolution.failed) holds
  (_.records(ProtocolFact.nexusOperationCompleted))
