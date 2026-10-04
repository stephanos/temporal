/* What the standalone activity's machines promise: the product's and the protocol's Properties, the
 * cross-entity claim of the activity and its worker, and the three promises the system contract
 * declares on the admission record and on both composition families from one definition each.
 */
package temporal
package standaloneactivity

import umpire.*
import worker.Phase as WorkerPhase

// ### What the product machine promises

/**
 * Once an activity is over, no step changes its phase. Declared on the product machine and read on
 * the protocol machine through the map.
 */
val terminalIsFinal = activityProduct.property.once(Product.terminal).keeps(_.phase)

/**
 * A paused activity is dispatched to no worker: nothing moves it straight to started.
 *
 * The predicate fixes no state, outcome or fact: "not started" only says what must not follow
 * paused. It is kept as a function and verified as one, so it claims something all the same:
 * it fails the moment a row from paused to started appears, which is the regression it
 * guards.
 */
val pausedIsNotDispatched =
  activityProduct.property.never(s => Product.running(s.state)).from(Product.paused)

// ### What the protocol machine promises

/** A completed answer settles the activity as completed, and the status records it. */
val completes = activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
  s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
}

/** A non-retryable failure settles the activity as failed. */
val nonRetryableFails =
  activityProtocol.property when attemptResult(AttemptResult.failed(false)) holds { s =>
    s.state.phase == Phase.failed && s.records(ProtocolFact.statusFailed)
  }

/** Completed on the second attempt of an activity with no deadline set. */
val completedOnRetry: ProtocolState =
  ProtocolState(Phase.completed, UpTo(attemptBound), Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * The retried attempt's completion settles the activity on its second attempt. The attempt count
 * saturates at `attemptBound`, so the claim is bounded by it: a completion on any later attempt reads
 * as this one.
 */
val retryCompletes =
  activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
    s.state == completedOnRetry && s.records(ProtocolFact.statusCompleted)
  }

/** A cancel request is recorded as requested. */
val cancelRequestedWhileStarted =
  activityProtocol.property when control(Control.requestCancel) holds { s =>
    s.state.phase == Phase.cancelRequested && s.records(ProtocolFact.statusCancelRequested)
  }

/** The worker's canceled answer settles a cancel-requested activity. */
val canceledByWorker =
  activityProtocol.property when attemptResult(AttemptResult.canceled) holds { s =>
    s.state.phase == Phase.canceled && s.records(ProtocolFact.statusCanceled)
  }

/** A terminate settles the activity as terminated. */
val terminated = activityProtocol.property when control(Control.terminate) holds { s =>
  s.state.phase == Phase.terminated && s.records(ProtocolFact.statusTerminated)
}

/**
 * The schedule-to-start deadline times the activity out and the status records which deadline it
 * was.
 */
val scheduleToStartFires = activityProtocol.property when scheduleToStart holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
}

/**
 * The schedule-to-close deadline times the activity out and the status records which it was. With
 * both deadlines set and no attempt started, either may fire first.
 */
val scheduleToCloseFires = activityProtocol.property when scheduleToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
}

/** The start-to-close deadline times a held attempt out. */
val startToCloseFires = activityProtocol.property when startToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.startToClose))
}

// ### The cross-entity claim

/** Every attempt start leaves the worker polling: no stopped worker starts an attempt. */
val startedByPollingWorker = standaloneActivity.property
  .whenAction(standaloneActivity.synced(_.activity -> attemptStart))
  .holds(_.state.worker.phase == WorkerPhase.polling)

// ### Promises of the system contract, written once
//
// Each is declared on each admission design (admission/) and on each composition with the record
// (compositions/). Each takes the states it speaks of as named predicates, so one definition serves
// the record and a composition, which reads the record through its `activity` member.

/** No step admits a paused activity: nothing moves it from paused straight to started. */
def notAdmittedWhilePaused[S](m: Declares[S])(
    paused: S => Boolean,
    running: S => Boolean
): Property[S] =
  m.property("notAdmittedWhilePaused").never(s => running(s.state)).from(paused)

/** No step leaves two admitted attempts active. */
def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
  m.property("atMostOneActive").never(s => twoActive(s.state))

/** Once the activity is over, no step changes its phase. */
def terminalStays[S, P](m: Declares[S])(terminal: S => Boolean, phase: S => P): Property[S] =
  m.property("terminalStays").once(terminal).keeps(phase)
