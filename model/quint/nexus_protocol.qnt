module nexusProtocol {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProtocolTable.* from "./nexus_protocol_table"
  import nexusProductTable.ProductState from "./nexus_product_table"
  import nexusProductTable.productScheduled from "./nexus_product_table"
  import nexusProductTable.productStarted from "./nexus_product_table"
  import nexusProductTable.productSucceeded from "./nexus_product_table"
  import nexusProductTable.productFailed from "./nexus_product_table"
  import nexusProductTable.productCanceled from "./nexus_product_table"
  import nexusProductTable.productTimedOut from "./nexus_product_table"
  import nexusProductTable.productAllows from "./nexus_product_table"
  import nexusProductTable.terminalIsFinalRow from "./nexus_product_table"

  var state: ProtocolState
  var last: Option[Fired[Action, ProtocolState, ProtocolFact]]

  var occurrence: bool

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None, occurrence' = false }
  }

  action fire(a: Action): bool = {
    val rows = steps(state, a)
    if (rows.length() == 0) all {
      false, state' = state, last' = last, occurrence' = occurrence,
    } else {
      nondet row = oneOf(rows.toSet())
      all {
        state' = row.state,
        occurrence' = not(occurrence),
        last' = Some({ actionClass: a, before: state, outcome: row.outcome, facts: row.facts }),
      }
    }
  }

  action schedule(scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout): bool =
    fire(Schedule({ scheduleToClose: scheduleToClose, scheduleToStart: scheduleToStart,
                    startToClose: startToClose }))
  action handlerReply(reply: Reply): bool = fire(HandlerReply(reply))
  action complete(resolution: Resolution): bool = fire(Complete(resolution))
  action transportFault = fire(TransportFault)
  action workerStop = fire(WorkerStop)
  action backoff = fire(Backoff)
  action scheduleToClose = fire(ScheduleToClose)
  action scheduleToStart = fire(ScheduleToStart)
  action startToClose = fire(StartToClose)

  /// Exploration: any class of the catalog. `cover: rows | results | classMembers` is what
  /// `quint run --max-samples` does with this, cut at `--max-steps`.
  action step = {
    nondet a = oneOf(actionClasses.filter(a => steps(state, a).length() > 0))
    fire(a)
  }

  def fired(a: Action): bool = last.firedIn(a)
  def recorded(f: ProtocolFact): bool = last.recordedIn(f)

  val acceptedResult = match last {
    | None => false
    | Some(l) => l.outcome == Accepted
  }

  val completionDelivery = match last {
    | None => true
    | Some(l) => match l.actionClass {
      | Complete(_) =>
        if (terminalPhase(l.before.phase))
          l.outcome == NotFound and state == l.before and l.facts == []
        else l.outcome == Accepted
      | _ => true
    }
  }

  /// The refinement as a state invariant: the last row taken maps to a product row or stutter.
  val refinesProduct = match last {
    | None => true
    | Some(l) =>
      val before = productOf(l.before)
      productAllows(before, productOf(state))
  }

  // ### What the machines promise
  //
  // A same-step claim names the action it is about under `when` and holds of the step that
  // action produces; a transition claim holds of the step before and the step after. A
  // functional Query realizes a same-step claim, because the Case's Contract is the claim's
  // clause triggered by the action the Case performs; a transition claim is searched and
  // verified, never realized.

  // Once an operation is over, no step changes its phase. The product claim, read here through
  // the map.
  val terminalIsFinal = match last {
    | None => true
    | Some(l) => terminalIsFinalRow(productOf(l.before), productOf(state))
  }

  // A synchronous reply settles the operation as succeeded, and the completed event records it.
  val syncSucceeds = claim(fired(HandlerReply(SyncSuccess)),
    state.phase == Succeeded and recorded(NexusOperationCompleted))

  // An asynchronous reply starts the operation, and the started event records it.
  val asyncStarts = claim(fired(HandlerReply(Async)),
    state.phase == Started and recorded(NexusOperationStarted))

  // A successful completion is recorded by the completed event. Neither the phase nor the
  // outcome is fixed: a completion resolves any running phase, and `Accepted` is every earlier
  // step's outcome too, so a clause fixing it would be answered before the completion.
  val completionSucceeds = claim(fired(Complete(ResolvedSucceeded)),
    recorded(NexusOperationCompleted))

  // A failed completion is recorded by the failed event.
  val completionFails = claim(fired(Complete(ResolvedFailed)), recorded(NexusOperationFailed))

  // A non-retryable handler error settles the operation as failed, and the failed event records it.
  val handlerErrorFails = claim(fired(HandlerReply(HandlerError(false))),
    state.phase == Failed and recorded(NexusOperationFailed))

  /// Succeeded on the second attempt of an operation with no deadline set. A claim fixes one
  /// state, so every field is named.
  pure val succeededOnRetry: ProtocolState =
    { phase: Succeeded, attempts: 1, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  // A synchronous reply to the retried attempt settles the operation as succeeded on its second
  // attempt: the count the retryable failure raised is still one, and the completed event
  // records the reply.
  val retrySucceeds = claim(fired(HandlerReply(SyncSuccess)),
    state == succeededOnRetry and recorded(NexusOperationCompleted))

  // The schedule-to-start deadline settles an operation no handler started as timed out, and the
  // timed-out event records which deadline it was.
  val scheduleToStartFires = claim(fired(ScheduleToStart),
    state.phase == TimedOut and recorded(NexusOperationTimedOut(ByScheduleToStart)))

  // The start-to-close deadline settles a started operation no handler completed as timed out.
  val startToCloseFires = claim(fired(StartToClose),
    state.phase == TimedOut and recorded(NexusOperationTimedOut(ByStartToClose)))

  /// Every same-step claim as an invariant, for `quint run --invariant properties`.
  val properties = and {
    terminalIsFinal,
    refinesProduct,
    asInvariant(syncSucceeds),
    asInvariant(asyncStarts),
    completionDelivery,
    acceptedResult implies asInvariant(completionSucceeds),
    acceptedResult implies asInvariant(completionFails),
    asInvariant(handlerErrorFails),
    asInvariant(scheduleToStartFires),
    asInvariant(startToCloseFires),
    // `retrySucceeds` fixes one state, so it is a Query's claim, not an invariant.
  }

  // ### The paths the Queries run
  //
  // A protocol Scenario is a `run`: its start is `init` (the one start, `Unscheduled`) and its
  // classed actions follow in order. Each path below is one upstream functional test's shape:
  // the schedule command with no deadline set, then the side effects that settle the operation.

  run syncReplied =
    init.then(schedule(Unset, Unset, Unset)).then(handlerReply(SyncSuccess))

  run asyncThenSucceeded =
    init.then(schedule(Unset, Unset, Unset)).then(handlerReply(Async))
      .then(complete(ResolvedSucceeded))

  run asyncThenFailed =
    init.then(schedule(Unset, Unset, Unset)).then(handlerReply(Async))
      .then(complete(ResolvedFailed))

  run nonRetryableError =
    init.then(schedule(Unset, Unset, Unset)).then(handlerReply(HandlerError(false)))

  // The retryable error backs the operation off; the backoff timer fires and records nothing;
  // the retried attempt is answered synchronously.
  run retriedThenSucceeded =
    init.then(schedule(Unset, Unset, Unset)).then(handlerReply(HandlerError(true)))
      .then(backoff).then(handlerReply(SyncSuccess))

  // The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
  // nothing answers the start request; the deadline fires. The worker stops after the schedule
  // in the operation's order, where the stop changes nothing; the realization stops it before
  // the workflow starts, where the stop cannot race the dispatch.
  run scheduleToStartExpires =
    init.then(schedule(Unset, Expires, Unset)).then(workerStop).then(scheduleToStart)

  // The schedule command sets the start-to-close deadline; the handler accepts asynchronously
  // and never completes; the deadline fires.
  run startToCloseExpires =
    init.then(schedule(Unset, Unset, Expires)).then(handlerReply(Async)).then(startToClose)

  // Nine actions are enabled before the operation is scheduled and eleven once it is, so an
  // exact sequence of two is found among ninety-nine candidates, one of three among about a
  // thousand and one of four among about ten thousand. Quint does not search for a scenario: a
  // `run` is the path. The Limits bound the exploratory `step` (`--max-steps`, `--max-samples`)
  // and travel to the Case as data.
  pure val two: Limits = { steps: 2, actions: 2, search: 512 }
  pure val three: Limits = { steps: 3, actions: 3, search: 4096 }
  pure val four: Limits = { steps: 4, actions: 4, search: 32768 }

  // ### The Queries
  //
  // A find Query is the scenario `run` followed by `.expect(found(claim))`: the claim's action
  // fired last and the claim holds. A verify Query checks the transition claim after every step
  // of the path. `quint test --main nexusProtocol` runs all of them; `--match retry` runs one.
  // `quint test --out-itf` writes the trace the Go runtime realizes.

  run syncCompletion = syncReplied.expect(found(syncSucceeds))
  run asyncCompletion = asyncThenSucceeded.expect(found(completionSucceeds))
  run asyncFailure = asyncThenFailed.expect(found(completionFails))
  run handlerError = nonRetryableError.expect(found(handlerErrorFails))
  run retry = retriedThenSucceeded.expect(found(retrySucceeds))
  run scheduleToStartTimeout = scheduleToStartExpires.expect(found(scheduleToStartFires))
  run startToCloseTimeout = startToCloseExpires.expect(found(startToCloseFires))

  // The product claim is verified over every step of the asynchronous path. Verified over the
  // whole machine instead: `quint verify --invariant terminalIsFinal --max-steps 3`.
  run terminalHolds =
    init.then(schedule(Unset, Unset, Unset)).expect(terminalIsFinal)
      .then(handlerReply(Async)).expect(terminalIsFinal)
      .then(complete(ResolvedSucceeded)).expect(terminalIsFinal)

  // ### The sets
  //
  // Data the Go side reads; Quint checks only that it is well-typed. Every party but `system`
  // is bound: the Case drives the caller, the handler and the worker, and observes the network.
  // The functional set repeats over the implementation switch, so each Query's Case runs once
  // under HSM and once under CHASM.

  pure val nexusCallerTests: SetDecl =
    setOf(Functional,
      Set((Caller, Driven), (Handler, Driven), (Network, Observed), (Worker, Driven)),
      Set("syncCompletion", "asyncCompletion", "asyncFailure", "handlerError", "retry",
          "scheduleToStartTimeout", "startToCloseTimeout"))
      .with("repeat", "implementation")

  // A canary runs a Query against a deployment that performs the handler's part itself: the
  // handler is `Observed`, so the verifier reads which reply occurred and checks the machine
  // allows it. A path with a silent step -- the backoff, the worker stop -- is a capability gap
  // no deployment closes, so a canary naming it is rejected (on the Go side; Quint has no
  // evidence notion to reject it with).
  pure val nexusCallerCanary: SetDecl =
    setOf(Canary,
      Set((Caller, Driven), (Handler, Observed), (Network, Observed), (Worker, Driven)),
      Set("syncCompletion", "asyncCompletion"))

  // An exploration covers the protocol machine rather than listing Queries: `quint run --mbt
  // --out-itf --max-steps 4 --n-traces N` under `step`, and the Go side counts rows, results and
  // class members off the traces.
  pure val nexusCallerExploration: SetDecl =
    setOf(Exploratory,
      Set((Caller, Driven), (Handler, Driven), (Network, Observed), (Worker, Driven)), Set())
      .with("machine", "nexusProtocol")
      .with("cover", Set(Rows, Results, ClassMembers))
      .with("budget", "four")
}
