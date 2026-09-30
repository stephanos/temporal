module activityProtocol {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"
  import activityProtocolTable.* from "./activity_protocol_table"
  import activityProductTable.ProductState from "./activity_product_table"
  import activityProductTable.productScheduled from "./activity_product_table"
  import activityProductTable.productStarted from "./activity_product_table"
  import activityProductTable.productPaused from "./activity_product_table"
  import activityProductTable.productCancelRequested from "./activity_product_table"
  import activityProductTable.productCompleted from "./activity_product_table"
  import activityProductTable.productFailed from "./activity_product_table"
  import activityProductTable.productCanceled from "./activity_product_table"
  import activityProductTable.productTerminated from "./activity_product_table"
  import activityProductTable.productTimedOut from "./activity_product_table"
  import activityProductTable.productAllows from "./activity_product_table"
  import activityProductTable.terminalIsFinalRow from "./activity_product_table"
  import activityProductTable.pausedIsNotDispatchedRow from "./activity_product_table"

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

  action start(scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout): bool =
    fire(Start({ scheduleToClose: scheduleToClose, scheduleToStart: scheduleToStart,
                 startToClose: startToClose }))
  action attemptStart = fire(AttemptStart)
  action attemptResult(result: AttemptResult): bool = fire(AttemptResultAction(result))
  action control(c: Control): bool = fire(ControlAction(c))
  action workerStop = fire(WorkerStop)
  action backoff = fire(Backoff)
  action scheduleToClose = fire(ScheduleToClose)
  action scheduleToStart = fire(ScheduleToStart)
  action startToClose = fire(StartToClose)

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

  val controlDelivery = match last {
    | None => true
    | Some(l) => match l.actionClass {
      | ControlAction(_) =>
        if (terminalPhase(l.before.phase))
          l.outcome == NotFound and state == l.before and l.facts == []
        else l.outcome == Accepted
      | _ => true
    }
  }

  val refinesProduct = match last {
    | None => true
    | Some(l) =>
      val before = productOf(l.before)
      productAllows(before, productOf(state))
  }

  // ### What the machines promise

  // The two product transition claims, read here through the map.
  val terminalIsFinal = match last {
    | None => true
    | Some(l) => terminalIsFinalRow(productOf(l.before), productOf(state))
  }

  val pausedIsNotDispatched = match last {
    | None => true
    | Some(l) => pausedIsNotDispatchedRow(productOf(l.before), productOf(state))
  }

  // A completed attempt settles the activity as completed, and Describe reads it.
  val completes = claim(fired(AttemptResultAction(AttemptCompleted)),
    state.phase == Completed and recorded(StatusCompleted))

  // A non-retryable failure settles the activity as failed.
  val nonRetryableFails = claim(fired(AttemptResultAction(AttemptFailed(false))),
    state.phase == Failed and recorded(StatusFailed))

  /// Completed on the second attempt with no deadline set; a claim fixes one state.
  pure val completedOnRetry: ProtocolState =
    { phase: Completed, attempts: 2, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  // The retried attempt completes: the count the retryable failure raised is two.
  val retryCompletes = claim(fired(AttemptResultAction(AttemptCompleted)),
    state == completedOnRetry and recorded(StatusCompleted))

  // A cancel request against a running attempt is recorded and waits for the worker.
  val cancelRequestedWhileStarted = claim(fired(ControlAction(RequestCancel)),
    state.phase == CancelRequested and recorded(StatusCancelRequested))

  // The worker honors the request, and Describe reads canceled.
  val canceledByWorker = claim(fired(AttemptResultAction(AttemptCanceled)),
    state.phase == Canceled and recorded(StatusCanceled))

  // Termination settles any live activity.
  val terminated = claim(fired(ControlAction(Terminate)),
    state.phase == Terminated and recorded(StatusTerminated))

  // The deadlines, typed: as in the Nexus Model, with `StatusTimedOut(type)`.
  val scheduleToStartFires = claim(fired(ScheduleToStart),
    state.phase == TimedOut and recorded(StatusTimedOut(ByScheduleToStart)))

  val startToCloseFires = claim(fired(StartToClose),
    state.phase == TimedOut and recorded(StatusTimedOut(ByStartToClose)))

  val properties = and {
    terminalIsFinal,
    pausedIsNotDispatched,
    refinesProduct,
    asInvariant(completes),
    asInvariant(nonRetryableFails),
    controlDelivery,
    acceptedResult implies asInvariant(cancelRequestedWhileStarted),
    asInvariant(canceledByWorker),
    acceptedResult implies asInvariant(terminated),
    asInvariant(scheduleToStartFires),
    asInvariant(startToCloseFires),
    // `retryCompletes` fixes one state, so it is a Query's claim, not an invariant.
  }

  // ### The paths the Queries run (start unstarted; `start(Unset, Unset, Unset)` unless said)

  run completed =
    init.then(start(Unset, Unset, Unset)).then(attemptStart).then(attemptResult(AttemptCompleted))

  run nonRetryable =
    init.then(start(Unset, Unset, Unset)).then(attemptStart)
      .then(attemptResult(AttemptFailed(false)))

  // The retryable failure backs the activity off; the backoff timer fires silently; the second
  // poll receives the task again and the attempt completes.
  run retriedThenCompleted =
    init.then(start(Unset, Unset, Unset)).then(attemptStart)
      .then(attemptResult(AttemptFailed(true))).then(backoff).then(attemptStart)
      .then(attemptResult(AttemptCompleted))

  run cancelRequestedThenCanceled =
    init.then(start(Unset, Unset, Unset)).then(attemptStart).then(control(RequestCancel))
      .then(attemptResult(AttemptCanceled))

  run terminatedWhileScheduled =
    init.then(start(Unset, Unset, Unset)).then(workerStop).then(control(Terminate))

  run pausedThenCompleted =
    init.then(start(Unset, Unset, Unset)).then(control(Pause)).then(control(Unpause))
      .then(attemptStart).then(attemptResult(AttemptCompleted))

  run scheduleToStartExpires =
    init.then(start(Unset, Expires, Unset)).then(workerStop).then(scheduleToStart)

  run startToCloseExpires =
    init.then(start(Unset, Unset, Expires)).then(attemptStart).then(startToClose)

  pure val three: Limits = { steps: 3, actions: 3, search: 4096 }
  pure val four: Limits = { steps: 4, actions: 4, search: 32768 }
  pure val six: Limits = { steps: 6, actions: 6, search: 262144 }

  // ### The Queries

  run completion = completed.expect(found(completes))
  run nonRetryableFailure = nonRetryable.expect(found(nonRetryableFails))
  run retry = retriedThenCompleted.expect(found(retryCompletes))
  run cancel = cancelRequestedThenCanceled.expect(found(canceledByWorker))
  run terminate = terminatedWhileScheduled.expect(found(terminated))
  run pauseResume = pausedThenCompleted.expect(found(completes))
  run scheduleToStartTimeout = scheduleToStartExpires.expect(found(scheduleToStartFires))
  run startToCloseTimeout = startToCloseExpires.expect(found(startToCloseFires))

  // verify terminalIsFinal in completed, three.
  run terminalHolds =
    init.then(start(Unset, Unset, Unset)).expect(terminalIsFinal)
      .then(attemptStart).expect(terminalIsFinal)
      .then(attemptResult(AttemptCompleted)).expect(terminalIsFinal)

  // verify pausedIsNotDispatched in pausedThenCompleted, six.
  run pauseHolds =
    init.then(start(Unset, Unset, Unset)).expect(pausedIsNotDispatched)
      .then(control(Pause)).expect(pausedIsNotDispatched)
      .then(control(Unpause)).expect(pausedIsNotDispatched)
      .then(attemptStart).expect(pausedIsNotDispatched)
      .then(attemptResult(AttemptCompleted)).expect(pausedIsNotDispatched)

  // ### The sets (no repeat: standalone activities are CHASM only)

  pure val standaloneActivityTests: SetDecl =
    setOf(Functional, Set((Caller, Driven), (Worker, Driven)),
      Set("completion", "nonRetryableFailure", "retry", "cancel", "terminate", "pauseResume",
          "scheduleToStartTimeout", "startToCloseTimeout"))

  pure val standaloneActivityCanary: SetDecl =
    setOf(Canary, Set((Caller, Driven), (Worker, Observed)), Set("completion", "cancel"))

  pure val standaloneActivityExploration: SetDecl =
    setOf(Exploratory, Set((Caller, Driven), (Worker, Driven)), Set())
      .with("machine", "activityProtocol")
      .with("cover", Set(Rows, Results, ClassMembers))
      .with("budget", "four")
}
