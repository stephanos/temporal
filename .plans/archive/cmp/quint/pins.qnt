// -*- mode: quint -*-
//
// # What the two Models say: the pins
//
// The Lean `#guard`s run at compile time against the machine's table. Quint has no static state
// count, but the tables are `pure val`s over finite sets, so the same numbers are pure
// expressions the simulator evaluates in a `run` of assertions:
//
//   quint test pins.qnt --main nexusPins
//   quint test pins.qnt --main activityPins
//
// A `run` here is deterministic (no `nondet`, no state read), so one sample is one evaluation;
// pass `--max-samples 1`. Anything that depends on randomness (an exploration's coverage) is
// pinned by the Go side from ITF traces under a fixed `--seed`, not here.
//
// The find Queries are pinned by their own `run`s: `quint test nexus_caller.qnt --main
// nexusProtocol` and `quint test standalone_activity.qnt --main activityProtocol` failing is the
// pin failing. They are re-exported below so one command runs everything.

module nexusPins {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller"
  import nexusProductTable as product from "./nexus_caller"
  import nexusProtocolTable.* from "./nexus_caller"
  import nexusProtocol as nexus from "./nexus_caller"

  /// A state written the way a reader names one: the phase, and whichever fields are not at
  /// the value the operation begins with.
  pure def at(phase: Phase, attempts: int): ProtocolState =
    { ...unscheduled, phase: phase, attempts: attempts }

  // ### The product machine

  // Six phases, and the four the design ends on. Every action class the machine steps on: six
  // replies, three resolutions, the two faults it cannot see, and the one timer.
  run productTable = all {
    assert(product::states.size() == 6),
    assert(product::ends.size() == 4),
    assert(product::actionClasses.size() == 12),
    // A retryable handler error is invisible here: it is the protocol machine that backs off.
    assert(product::handlerReplyStep({ phase: product::Scheduled }, HandlerError(true)) == []),
    // What the Model actually reaches: every phase.
    assert(reachableFrom(product::starts, product::states.size(),
      s => successors(s, product::actionClasses, product::steps)) == product::states),
    assert(stuckStates(product::states, product::actionClasses, product::productTerminal,
      product::steps) == Set()),
  }

  // ### The protocol machine

  // Eight phases, three attempt counts and three deadlines, and the four phases the design ends
  // on. Every action class: eight schedule commands, one per assignment of the three deadlines,
  // six replies, three resolutions, the two faults and the four timers.
  run protocolTable = all {
    assert(states.size() == 8 * (attemptBound + 1) * 2 * 2 * 2),
    assert(ends.size() == 4 * (attemptBound + 1) * 2 * 2 * 2),
    assert(actionClasses.size() == 8 + 6 + 3 + 1 + 1 + 4),
    // The machine begins before the operation exists, with every deadline at its first value.
    assert(starts == Set(unscheduled)),
    // One row per state, class and step: what the exploration's `rows` target counts.
    assert(rowCount(states, actionClasses, steps) == 1152),
    // Nothing is stuck: the operation's own phase decides which steps are enabled.
    assert(stuckStates(states, actionClasses, s => terminalPhase(s.phase), steps) == Set()),
    // Not every state is reachable: the count and the deadlines are fields of the state, so
    // states that disagree about them exist in the type and no run produces them.
    assert(reachableFrom(starts, states.size(),
      s => successors(s, actionClasses, steps)).size() == 158),
  }

  run protocolRows = all {
    // A retryable handler error backs the operation off and raises the attempt count. No
    // history event records that, which is why its evidence is the `pendingAttempts` observation.
    assert(protocolHandlerReplyStep(at(Scheduled, 0), HandlerError(true)) ==
      [{ outcome: Accepted, state: at(BackingOff, 1), facts: [PendingAttempts] }]),
    // The count saturates rather than wrapping.
    assert(protocolHandlerReplyStep(at(Scheduled, attemptBound), HandlerError(true))[0]
      .state.attempts == attemptBound),
    // A completion that arrives before the start records the Started event first, and one that
    // arrives after it does not. Both record the completion.
    assert(protocolCompleteStep(at(BackingOff, 1), ResolvedSucceeded)[0].facts ==
      [NexusOperationStarted, NexusOperationCompleted]),
    assert(protocolCompleteStep(at(Started, 0), ResolvedSucceeded)[0].facts ==
      [NexusOperationCompleted]),
    // A completion after the operation is over is not found and changes nothing.
    assert(protocolCompleteStep(at(TimedOut, 0), ResolvedSucceeded) ==
      [{ outcome: NotFound, state: at(TimedOut, 0), facts: [] }]),
    // A timer fires only when the schedule command set it, and each covers its own span.
    assert(startToCloseStep({ ...at(Scheduled, 0), startToClose: Expires }) == []),
    assert(startToCloseStep({ ...at(Started, 0), startToClose: Expires })[0].state.phase == TimedOut),
    assert(scheduleToCloseStep(at(Started, 0)) == []),
    // Which timer fired is recorded.
    assert(scheduleToStartStep({ ...at(Scheduled, 0), scheduleToStart: Expires })[0].facts ==
      [NexusOperationTimedOut(ByScheduleToStart)]),
    // The worker stopping keeps the state and records nothing; the product does not see it.
    assert(protocolWorkerStopStep(at(Scheduled, 0)) ==
      [{ outcome: Accepted, state: at(Scheduled, 0), facts: [] }]),
    assert(product::workerStopStep({ phase: product::Scheduled }) == []),
  }

  // ### The refinement
  //
  // `refines: nexusProduct` with `map: productOf` walks every protocol row through the map: a
  // row whose mapped states are equal is a stutter, and every other row must be a product row
  // between the mapped states under some product class.

  /// The rows the product machine does not see: mapped states equal.
  pure val stutters: Set[(ProtocolState, Action)] =
    tuples(states, actionClasses).filter(((s, a)) =>
      steps(s, a).toSet().exists(row => productOf(s) == productOf(row.state)))

  pure def isComplete(a: Action): bool =
    match a {
      | Complete(_) => true
      | _ => false
    }

  run refinementPins = all {
    assert(refinement),
    assert(rejectedRows(states, actionClasses, steps, productOf, product::actionClasses,
      product::steps) == Set()),
    // Every schedule command, every retry, every transport fault, every backoff, every worker
    // stop, and every completion that arrives after the end and is not found.
    assert(stutters.size() == 24 * 8 + 24 + 24 + 24 + 192 + 288),
    // **Stutter invariance.** A product Property read on the protocol machine is checked on
    // every row, stutters included. `terminalIsFinal` triggers only at a terminal prior state,
    // and the only stutters that leave one are the worker stop's and the late completion's,
    // which keep the phase: every other stutter leaves from a phase that reads as `Scheduled`.
    assert(stutters.forall(((s, a)) =>
      Set(Unscheduled, Scheduled, BackingOff).contains(s.phase) or a == WorkerStop or isComplete(a))),
    // A deadline firing is the product's one timer, whichever deadline it was: the mapped row
    // `Started` to `TimedOut` exists in the product.
    assert(rowsFrom({ phase: product::Started }, product::actionClasses, product::steps)
      .exists(r => r.state.phase == product::TimedOut)),
  }

  // ### The Properties and the Queries
  //
  // Each functional Query finds its claim on its path, and `terminalHolds` verifies: the
  // machine module's `run`s, re-exported so `quint test pins.qnt --main nexusPins` runs them.
  run syncCompletion = nexus::syncCompletion
  run asyncCompletion = nexus::asyncCompletion
  run asyncFailure = nexus::asyncFailure
  run handlerError = nexus::handlerError
  run retry = nexus::retry
  run scheduleToStartTimeout = nexus::scheduleToStartTimeout
  run startToCloseTimeout = nexus::startToCloseTimeout
  run terminalHolds = nexus::terminalHolds

  // A timer is named under `when` like any action, and a Scenario lists it where it fires: the
  // retry path ends on a reply with the count still one, the timeout path on a timer.
  run retryShape = nexus::retriedThenSucceeded
    .expect(nexus::fired(HandlerReply(SyncSuccess)) and nexus::state.attempts == 1)
  run timeoutShape = nexus::scheduleToStartExpires.expect(nexus::fired(ScheduleToStart))
}

module activityPins {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./standalone_activity"
  import activityProductTable as product from "./standalone_activity"
  import activityProtocolTable.* from "./standalone_activity"
  import activityProtocol as act from "./standalone_activity"

  pure def at(phase: Phase, attempts: int): ProtocolState =
    { ...unstarted, phase: phase, attempts: attempts }

  // ### The product machine: nine phases, five ends.
  run productTable = all {
    assert(product::states.size() == 9),
    assert(product::ends.size() == 5),
    assert(product::actionClasses.size() == 1 + 4 + 4 + 1 + 1),
    assert(reachableFrom(product::starts, product::states.size(),
      s => successors(s, product::actionClasses, product::steps)) == product::states),
    // The product's own transition claims hold on every product row.
    assert(tuples(product::states, product::actionClasses).forall(((s, a)) =>
      product::steps(s, a).toSet().forall(row =>
        product::terminalIsFinalRow(s, row.state) and product::pausedIsNotDispatchedRow(s, row.state)))),
  }

  // ### The protocol machine: twelve phases, three counts, three deadlines; five ends.
  run protocolTable = all {
    assert(states.size() == 12 * (attemptBound + 1) * 2 * 2 * 2),
    assert(ends.size() == 5 * (attemptBound + 1) * 2 * 2 * 2),
    assert(actionClasses.size() == 8 + 1 + 4 + 4 + 1 + 4),
    assert(starts == Set(unstarted)),
    assert(stuckStates(states, actionClasses, s => terminalPhase(s.phase), steps) == Set()),
    // Every phase is reachable, the requested pause included.
    assert(reachableFrom(starts, states.size(),
      s => successors(s, actionClasses, steps)).map(s => s.phase) == phases),
  }

  run protocolRows = all {
    // A cancellation nobody requested is not a row.
    assert(protocolAttemptResultStep(at(Started, 1), AttemptCanceled) == []),
    // A retryable failure backs off and only the attempt count records it.
    assert(protocolAttemptResultStep(at(Started, 1), AttemptFailed(true)) ==
      [{ outcome: Accepted, state: at(BackingOff, 1), facts: [AttemptCount] }]),
    // The poll raises the count and Describe shows both.
    assert(protocolAttemptStartStep(at(Scheduled, 1))[0] ==
      { outcome: Accepted, state: at(Started, 2), facts: [StatusStarted, AttemptCount] }),
    // A pause of a running attempt is only requested; the fact is the same.
    assert(protocolControlStep(at(Started, 1), Pause)[0].state.phase == PauseRequested),
    assert(protocolControlStep(at(Started, 1), Pause)[0].facts == [StatusPaused]),
    // Controls on a finished activity are not found; before the start, not rows at all.
    assert(protocolControlStep(at(Completed, 1), Terminate)[0].outcome == NotFound),
    assert(protocolControlStep(unstarted, RequestCancel) == []),
  }

  // ### The refinement
  //
  // `refines: activityProduct` with `map: productOf`. A requested pause reads as `Started`, so
  // resuming it is a stutter and a result while it is requested is a product row from
  // `Started`; the retry is visible, so the backoff after a retryable failure is the product's
  // `Started` to `Scheduled` row.
  run refinementPins = all {
    assert(refinement),
    assert(rejectedRows(states, actionClasses, steps, productOf, product::actionClasses,
      product::steps) == Set()),
    assert(productOf(at(PauseRequested, 1)) == { phase: product::Started }),
    // Unlike the Nexus product, the activity product sees the retry: Describe reads SCHEDULED
    // again, or CANCELED once a cancel was requested.
    assert(product::attemptResultStep({ phase: product::Started }, AttemptFailed(true)) ==
      [{ outcome: Accepted, state: { phase: product::Scheduled }, facts: [product::StatusScheduled] }]),
    assert(product::attemptResultStep({ phase: product::CancelRequested }, AttemptFailed(true)) ==
      [{ outcome: Accepted, state: { phase: product::Canceled }, facts: [product::StatusCanceled] }]),
  }

  // ### The Properties and the Queries
  run completion = act::completion
  run nonRetryableFailure = act::nonRetryableFailure
  run retry = act::retry
  run cancel = act::cancel
  run terminate = act::terminate
  run pauseResume = act::pauseResume
  run scheduleToStartTimeout = act::scheduleToStartTimeout
  run startToCloseTimeout = act::startToCloseTimeout
  run terminalHolds = act::terminalHolds
  run pauseHolds = act::pauseHolds

  // `pausedIsNotDispatched`, read through the map, holds on every protocol row, not only on
  // the `pausedThenCompleted` path: no row leaves a phase that reads as `Paused` for `Started`.
  run pausedOverTheTable = all {
    assert(tuples(states, actionClasses).forall(((s, a)) =>
      steps(s, a).toSet().forall(row =>
        product::pausedIsNotDispatchedRow(productOf(s), productOf(row.state))))),
  }
}
