/* What the standalone activity's machines promise of their own: the protocol's settlement claims,
 * the cross-entity claim of the activity and its worker, and the system contract's promise of one
 * active attempt, written once for admission/ and compositions/. The laws they receive are declared
 * in Capabilities.scala.
 */
package temporal
package features.standaloneactivity

import umpire.*
import shared.worker.Phase as WorkerPhase

val completes = activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
  s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
}

val nonRetryableFails =
  activityProtocol.property when attemptResult(AttemptResult.failed(false)) holds { s =>
    s.state.phase == Phase.failed && s.records(ProtocolFact.statusFailed)
  }

/** Completed on the second attempt of an activity with no deadline set. */
val completedOnRetry =
  ProtocolState(Phase.completed, UpTo(attemptBound), Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * The attempt count saturates at `attemptBound`, so the claim is bounded by it: a completion on any
 * later attempt than the second reads as this one.
 */
val retryCompletes =
  activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
    s.state == completedOnRetry && s.records(ProtocolFact.statusCompleted)
  }

val cancelRequestedWhileStarted =
  activityProtocol.property when control(Control.requestCancel) holds { s =>
    s.state.phase == Phase.cancelRequested && s.records(ProtocolFact.statusCancelRequested)
  }

val canceledByWorker =
  activityProtocol.property when attemptResult(AttemptResult.canceled) holds { s =>
    s.state.phase == Phase.canceled && s.records(ProtocolFact.statusCanceled)
  }

val terminated = activityProtocol.property when control(Control.terminate) holds { s =>
  s.state.phase == Phase.terminated && s.records(ProtocolFact.statusTerminated)
}

// Each deadline times the activity out and the status records which it was. With both
// schedule-to-start and schedule-to-close set and no attempt started, either may fire first.
val scheduleToStartFires = activityProtocol.property when scheduleToStart holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
}

val scheduleToCloseFires = activityProtocol.property when scheduleToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
}

val startToCloseFires = activityProtocol.property when startToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.startToClose))
}

/** The cross-entity claim: no stopped worker starts an attempt. */
val startedByPollingWorker = standaloneActivity.property
  .whenAction(standaloneActivity.synced(_.activity -> attemptStart))
  .holds(_.state.worker.phase == WorkerPhase.polling)

// The system contract's promises are declared on each admission design and each composition with
// the record: the laws its capabilities bring, and the record's own count of active attempts. Each
// takes the states it speaks of as predicates, so one definition serves the record and a
// composition, which reads the record through its `activity` member.

/** No step leaves two admitted attempts active. */
def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
  m.property("atMostOneActive").never(s => twoActive(s.state))
