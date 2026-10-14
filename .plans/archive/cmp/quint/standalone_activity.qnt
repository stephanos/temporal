// -*- mode: quint -*-
//
// # The standalone activity Model
//
// A Temporal activity started directly through `StartActivityExecution`, with no workflow.
// Grounded in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is
// deferred (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not
// modeled. Standalone activities write no history events, so every fact is a status read
// through `DescribeActivityExecution` or a result read through `PollActivityExecution`.
//
// Same layout as nexus_caller.qnt: vocabulary → product table and machine → protocol table and
// machine → properties, scenarios, queries, sets → composition.
//
// Typecheck:  quint typecheck standalone_activity.qnt
// Scenarios:  quint test standalone_activity.qnt --main activityProtocol
// Explore:    quint run standalone_activity.qnt --main activityProtocol --invariant properties --mbt
// Verify:     quint verify standalone_activity.qnt --main activityProtocol --invariant refinesProduct

// ### Entities and the input domains
//
// `AttemptResult`'s members carry an `Attempt` prefix for the same reason `Resolution`'s carry
// `Resolved` in the Nexus Model: the phases `Completed`, `Failed`, `Canceled` own the bare names.
module activityVocabulary {
  import umpire.* from "./umpire"

  pure val activity: Entity = { name: "activity", key: "activityId", refer: Set() }

  type AttemptResult = AttemptCompleted | AttemptFailed(bool) | AttemptCanceled

  type Control = Pause | Unpause | RequestCancel | Terminate

  pure val attemptResults: Set[AttemptResult] =
    Set(AttemptCompleted, AttemptFailed(true), AttemptFailed(false), AttemptCanceled)

  pure val controls: Set[Control] = Set(Pause, Unpause, RequestCancel, Terminate)

  // ### Actions (the part Quint cannot check: party, entity, schema, examples)
  pure val actions: Map[str, ActionDecl] = Map(
    "start" -> { party: Caller, creates: "activity", on: "",
      schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest", examples: Set() },
    // The worker's poll receives the task (`PollActivityTaskQueue`).
    "attemptStart" -> { party: Worker, creates: "", on: "activity",
      schema: "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse", examples: Set() },
    "attemptResult" -> { party: Worker, creates: "", on: "activity",
      schema: "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest",
      examples: Set(("AttemptFailed(false)", "ApplicationFailure nonRetryable"),
                    ("AttemptFailed(true)", "ApplicationFailure retryable")) },
    "control" -> { party: Caller, creates: "", on: "activity",
      schema: "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest",
      examples: Set() },
    "workerStop" -> decl(Worker),
  )

  // ### The derived observation: the attempt count, read from `DescribeActivityExecution`.
  pure val attemptCount: (str, str) = ("activity", "attempt")
}

// ### The product machine
//
// What the caller sees through Describe, with no account of how the server gets there.
module activityProductTable {
  import umpire.* from "./umpire"
  import activityVocabulary.*

  type ProductPhase =
    | Scheduled
    | Started
    | Paused
    | CancelRequested
    | Completed
    | Failed
    | Canceled
    | Terminated
    | TimedOut

  type ProductState = { phase: ProductPhase }

  type ProductFact =
    | StatusScheduled
    | StatusStarted
    | StatusPaused
    | StatusCancelRequested
    | StatusCompleted
    | StatusFailed
    | StatusCanceled
    | StatusTerminated
    | StatusTimedOut

  type ProductStep = Step[ProductState, ProductFact]

  type Action =
    | AttemptStart
    | AttemptResult(AttemptResult)
    | Control(Control)
    | WorkerStop
    | TimeoutFired

  pure def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] =
    accepted({ phase: phase }, [recorded])

  /// The five phases the product machine ends on.
  pure def productTerminal(s: ProductState): bool =
    Set(Completed, Failed, Canceled, Terminated, TimedOut).contains(s.phase)

  /// The worker's poll receives the task. Only a scheduled activity is dispatched.
  pure def attemptStartStep(s: ProductState): List[ProductStep] =
    if (s.phase != Scheduled) [] else productStep(Started, StatusStarted)

  /// The worker reports the attempt. Unlike the Nexus Model, the retry is visible here: a
  /// retryable failure reschedules the activity and Describe reads SCHEDULED again
  /// (TransitionRescheduled), or, once a cancel was requested, reads CANCELED. A cancellation
  /// is honored only where one was requested.
  pure def attemptResultStep(s: ProductState, result: AttemptResult): List[ProductStep] =
    if (s.phase != Started and s.phase != CancelRequested) [] else
    match result {
      | AttemptCompleted => productStep(Completed, StatusCompleted)
      | AttemptFailed(retryable) =>
        if (not(retryable)) productStep(Failed, StatusFailed)
        else if (s.phase == Started) productStep(Scheduled, StatusScheduled)
        else productStep(Canceled, StatusCanceled)
      | AttemptCanceled =>
        if (s.phase == CancelRequested) productStep(Canceled, StatusCanceled) else []
    }

  /// A caller control. On a terminal activity every control is not found and changes nothing.
  /// `requestCancel` is idempotent in `CancelRequested` and records the status again.
  pure def controlStep(s: ProductState, control: Control): List[ProductStep] =
    if (productTerminal(s)) notFound(s) else
    match control {
      | Pause =>
        if (s.phase == Scheduled or s.phase == Started) productStep(Paused, StatusPaused) else []
      | Unpause =>
        if (s.phase == Paused) productStep(Scheduled, StatusScheduled) else []
      | RequestCancel =>
        if (Set(Scheduled, Started, Paused, CancelRequested).contains(s.phase))
          productStep(CancelRequested, StatusCancelRequested)
        else []
      | Terminate => productStep(Terminated, StatusTerminated)
    }

  /// Invisible, as in the Nexus Model: a kept state with no facts would read as a stutter.
  pure def workerStopStep(s: ProductState): List[ProductStep] = []

  /// One of the activity's deadlines firing, whichever it was.
  pure def timeoutStep(s: ProductState): List[ProductStep] =
    if (Set(Scheduled, Started, CancelRequested, Paused).contains(s.phase))
      productStep(TimedOut, StatusTimedOut)
    else []

  pure def steps(s: ProductState, a: Action): List[ProductStep] =
    match a {
      | AttemptStart => attemptStartStep(s)
      | AttemptResult(result) => attemptResultStep(s, result)
      | Control(control) => controlStep(s, control)
      | WorkerStop => workerStopStep(s)
      | TimeoutFired => timeoutStep(s)
    }

  pure val phases: Set[ProductPhase] =
    Set(Scheduled, Started, Paused, CancelRequested, Completed, Failed, Canceled, Terminated,
        TimedOut)
  pure val states: Set[ProductState] = phases.map(p => { phase: p })
  pure val starts: Set[ProductState] = Set({ phase: Scheduled })
  pure val ends: Set[ProductState] = states.filter(productTerminal)
  pure val timers: Set[Action] = Set(TimeoutFired)

  pure val actionClasses: Set[Action] =
    Set(AttemptStart, WorkerStop, TimeoutFired)
      .union(attemptResults.map(r => AttemptResult(r)))
      .union(controls.map(c => Control(c)))

  /// `evidence:` every fact is a Describe status read.
  pure def evidence(f: ProductFact): str =
    match f {
      | StatusScheduled => "statusScheduled"
      | StatusStarted => "statusStarted"
      | StatusPaused => "statusPaused"
      | StatusCancelRequested => "statusCancelRequested"
      | StatusCompleted => "statusCompleted"
      | StatusFailed => "statusFailed"
      | StatusCanceled => "statusCanceled"
      | StatusTerminated => "statusTerminated"
      | StatusTimedOut => "statusTimedOut"
    }

  // Once an activity is over, no step changes its phase.
  pure def terminalIsFinalRow(before: ProductState, after: ProductState): bool =
    not(productTerminal(before)) or after.phase == before.phase

  // A paused activity is not dispatched: no step takes it straight to started.
  pure def pausedIsNotDispatchedRow(before: ProductState, after: ProductState): bool =
    before.phase == Paused implies after.phase != Started
}

module activityProduct {
  import umpire.* from "./umpire"
  import activityVocabulary.*
  import activityProductTable.*

  var state: ProductState
  var last: Option[Fired[Action, ProductState, ProductFact]]

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None }
  }

  action fire(a: Action): bool = {
    val rows = steps(state, a)
    all {
      rows.length() > 0,
      state' = rows[0].state,
      last' = Some({ action: a, before: state, outcome: rows[0].outcome, facts: rows[0].facts }),
    }
  }

  action attemptStart = fire(AttemptStart)
  action attemptResult(result: AttemptResult): bool = fire(AttemptResult(result))
  action control(c: Control): bool = fire(Control(c))
  action workerStop = fire(WorkerStop)
  action timeout = fire(TimeoutFired)

  action step = {
    nondet a = oneOf(actionClasses)
    fire(a)
  }

  val terminalIsFinal = match last {
    | None => true
    | Some(l) => activityProductTable::terminalIsFinalRow(l.before, state)
  }

  val pausedIsNotDispatched = match last {
    | None => true
    | Some(l) => activityProductTable::pausedIsNotDispatchedRow(l.before, state)
  }
}

// ### The protocol machine
//
// How the server gets there: the retry the caller cannot see, the pause requested of a running
// attempt, the three timers the start request sets, and the attempt count. `Unstarted` is the
// "no instance yet" member, as `Unscheduled` is in the Nexus Model.
module activityProtocolTable {
  import umpire.* from "./umpire"
  import activityVocabulary.*
  import activityProductTable as product

  type Phase =
    | Unstarted
    | Scheduled
    | BackingOff
    | Started
    | Paused
    | PauseRequested
    | CancelRequested
    | Completed
    | Failed
    | Canceled
    | Terminated
    | TimedOut

  type TimeoutType = ByScheduleToClose | ByScheduleToStart | ByStartToClose

  type ProtocolState = {
    phase: Phase,
    attempts: int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
  }

  /// As the product's facts, with the timed-out status typed by the deadline, plus the
  /// attempt-count observation.
  type ProtocolFact =
    | StatusScheduled
    | StatusStarted
    | StatusPaused
    | StatusCancelRequested
    | StatusCompleted
    | StatusFailed
    | StatusCanceled
    | StatusTerminated
    | StatusTimedOut(TimeoutType)
    | AttemptCount

  type ProtocolStep = Step[ProtocolState, ProtocolFact]

  type Deadlines = { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout }

  type Action =
    | Start(Deadlines)
    | AttemptStart
    | AttemptResult(AttemptResult)
    | Control(Control)
    | WorkerStop
    | Backoff
    | ScheduleToClose
    | ScheduleToStart
    | StartToClose

  /// The five phases the design ends on.
  pure def terminalPhase(phase: Phase): bool =
    Set(Completed, Failed, Canceled, Terminated, TimedOut).contains(phase)

  /// Started and not yet over: the phases the schedule-to-close deadline covers.
  pure def running(phase: Phase): bool =
    Set(Scheduled, BackingOff, Started, Paused, PauseRequested, CancelRequested).contains(phase)

  pure def moves(s: ProtocolState, phase: Phase, recorded: List[ProtocolFact]): List[ProtocolStep] =
    accepted(s.with("phase", phase), recorded)

  /// The caller's start request: it names the three deadlines, which become state fields.
  pure def startStep(s: ProtocolState, d: Deadlines): List[ProtocolStep] =
    if (s.phase != Unstarted) [] else
    accepted({ phase: Scheduled, attempts: 0, scheduleToClose: d.scheduleToClose,
               scheduleToStart: d.scheduleToStart, startToClose: d.startToClose },
             [StatusScheduled])

  /// The poll receives the task: the attempt count rises, and Describe shows both.
  pure def protocolAttemptStartStep(s: ProtocolState): List[ProtocolStep] =
    if (s.phase != Scheduled) [] else
    accepted({ ...s, phase: Started, attempts: saturatingSucc(s.attempts) },
             [StatusStarted, AttemptCount])

  /// The worker reports the attempt. From `Started`, a retryable failure backs off and only the
  /// attempt count records it, and a cancellation nobody requested is not a row. From
  /// `CancelRequested`, faithful to statemachine.go, where CANCEL_REQUESTED is a source of
  /// Completed, Failed, Canceled, TimedOut and Terminated: a retryable failure transitions to
  /// Canceled. From `PauseRequested`, a retryable failure lands in Paused
  /// (TransitionAttemptFailedWhilePauseRequested).
  pure def protocolAttemptResultStep(s: ProtocolState, result: AttemptResult): List[ProtocolStep] =
    match s.phase {
      | Started => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, BackingOff, [AttemptCount])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => []
        }
      | CancelRequested => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, Canceled, [StatusCanceled])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => moves(s, Canceled, [StatusCanceled])
        }
      | PauseRequested => match result {
          | AttemptCompleted => moves(s, Completed, [StatusCompleted])
          | AttemptFailed(retryable) =>
            if (retryable) moves(s, Paused, [StatusPaused])
            else moves(s, Failed, [StatusFailed])
          | AttemptCanceled => []
        }
      | _ => []
    }

  /// A caller control. Pausing a running attempt only requests the pause; the Describe status
  /// reads PAUSE_REQUESTED, but the fact is `StatusPaused` for both, as the product cannot tell
  /// them apart. Cancel and terminate are not found once the activity is over.
  pure def protocolControlStep(s: ProtocolState, control: Control): List[ProtocolStep] =
    match control {
      | Pause =>
        if (s.phase == Scheduled or s.phase == BackingOff) moves(s, Paused, [StatusPaused])
        else if (s.phase == Started) moves(s, PauseRequested, [StatusPaused])
        else []
      | Unpause =>
        if (s.phase == Paused) moves(s, Scheduled, [StatusScheduled])
        else if (s.phase == PauseRequested) moves(s, Started, [StatusStarted])
        else []
      | RequestCancel =>
        if (terminalPhase(s.phase)) notFound(s)
        else if (s.phase == Unstarted) []
        else moves(s, CancelRequested, [StatusCancelRequested])
      | Terminate =>
        if (terminalPhase(s.phase)) notFound(s)
        else if (s.phase == Unstarted) []
        else moves(s, Terminated, [StatusTerminated])
    }

  /// A stutter: the Run records the fault, nothing recorded names the activity.
  pure def protocolWorkerStopStep(s: ProtocolState): List[ProtocolStep] = accepted(s, [])

  /// The backoff timer records nothing: a retry writes no status change the caller reads.
  pure def backoffStep(s: ProtocolState): List[ProtocolStep] =
    if (s.phase != BackingOff) [] else moves(s, Scheduled, [])

  pure def scheduleToCloseStep(s: ProtocolState): List[ProtocolStep] =
    if (running(s.phase) and s.scheduleToClose == Expires)
      moves(s, TimedOut, [StatusTimedOut(ByScheduleToClose)])
    else []

  pure def scheduleToStartStep(s: ProtocolState): List[ProtocolStep] =
    if ((s.phase == Scheduled or s.phase == BackingOff) and s.scheduleToStart == Expires)
      moves(s, TimedOut, [StatusTimedOut(ByScheduleToStart)])
    else []

  pure def startToCloseStep(s: ProtocolState): List[ProtocolStep] =
    if (Set(Started, PauseRequested, CancelRequested).contains(s.phase) and s.startToClose == Expires)
      moves(s, TimedOut, [StatusTimedOut(ByStartToClose)])
    else []

  /// How a protocol state reads through Describe: not yet started, scheduled and backing off
  /// all read as scheduled; a requested pause is still a running attempt, so it reads as
  /// started; the rest by name.
  pure def productOf(s: ProtocolState): product::ProductState =
    { phase: match s.phase {
        | Unstarted => product::Scheduled
        | Scheduled => product::Scheduled
        | BackingOff => product::Scheduled
        | Started => product::Started
        | Paused => product::Paused
        | PauseRequested => product::Started
        | CancelRequested => product::CancelRequested
        | Completed => product::Completed
        | Failed => product::Failed
        | Canceled => product::Canceled
        | Terminated => product::Terminated
        | TimedOut => product::TimedOut
      } }

  pure def steps(s: ProtocolState, a: Action): List[ProtocolStep] =
    match a {
      | Start(d) => startStep(s, d)
      | AttemptStart => protocolAttemptStartStep(s)
      | AttemptResult(result) => protocolAttemptResultStep(s, result)
      | Control(control) => protocolControlStep(s, control)
      | WorkerStop => protocolWorkerStopStep(s)
      | Backoff => backoffStep(s)
      | ScheduleToClose => scheduleToCloseStep(s)
      | ScheduleToStart => scheduleToStartStep(s)
      | StartToClose => startToCloseStep(s)
    }

  pure val phases: Set[Phase] =
    Set(Unstarted, Scheduled, BackingOff, Started, Paused, PauseRequested, CancelRequested,
        Completed, Failed, Canceled, Terminated, TimedOut)

  pure val states: Set[ProtocolState] =
    tuples(phases, attemptCounts, timeouts, timeouts, timeouts).map(((p, n, c, s, t)) =>
      { phase: p, attempts: n, scheduleToClose: c, scheduleToStart: s, startToClose: t })

  pure val unstarted: ProtocolState =
    { phase: Unstarted, attempts: 0, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  pure val starts: Set[ProtocolState] = Set(unstarted)
  pure val ends: Set[ProtocolState] = states.filter(s => terminalPhase(s.phase))
  pure val timers: Set[Action] = Set(Backoff, ScheduleToClose, ScheduleToStart, StartToClose)
  pure val unobservable: Set[Action] = Set(Backoff)

  pure val deadlines: Set[Deadlines] =
    tuples(timeouts, timeouts, timeouts).map(((c, s, t)) =>
      { scheduleToClose: c, scheduleToStart: s, startToClose: t })

  pure val actionClasses: Set[Action] =
    deadlines.map(d => Start(d))
      .union(Set(AttemptStart, WorkerStop))
      .union(attemptResults.map(r => AttemptResult(r)))
      .union(controls.map(c => Control(c)))
      .union(timers)

  pure def evidence(f: ProtocolFact): str =
    match f {
      | StatusScheduled => "statusScheduled"
      | StatusStarted => "statusStarted"
      | StatusPaused => "statusPaused"
      | StatusCancelRequested => "statusCancelRequested"
      | StatusCompleted => "statusCompleted"
      | StatusFailed => "statusFailed"
      | StatusCanceled => "statusCanceled"
      | StatusTerminated => "statusTerminated"
      | StatusTimedOut(_) => "statusTimedOut"
      | AttemptCount => "attemptCount"
    }

  /// `refines: activityProduct` with `map: productOf`, over the whole table. The rows that
  /// need the rule to be by mapped states: a requested pause resumed (`Started` to `Started`, a
  /// stutter), a retryable failure while pause was requested (`Started` to `Paused`, which the
  /// product's pause provides), and the backoff after a retryable failure (`Started` to
  /// `Scheduled`, the product's visible retry).
  pure val refinement: bool =
    refines(states, actionClasses, steps, productOf, product::actionClasses, product::steps)
}

module activityProtocol {
  import umpire.* from "./umpire"
  import activityVocabulary.*
  import activityProtocolTable.*
  import activityProductTable as product

  var state: ProtocolState
  var last: Option[Fired[Action, ProtocolState, ProtocolFact]]

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None }
  }

  action fire(a: Action): bool = {
    val rows = steps(state, a)
    all {
      rows.length() > 0,
      state' = rows[0].state,
      last' = Some({ action: a, before: state, outcome: rows[0].outcome, facts: rows[0].facts }),
    }
  }

  action start(scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout): bool =
    fire(Start({ scheduleToClose: scheduleToClose, scheduleToStart: scheduleToStart,
                 startToClose: startToClose }))
  action attemptStart = fire(AttemptStart)
  action attemptResult(result: AttemptResult): bool = fire(AttemptResult(result))
  action control(c: Control): bool = fire(Control(c))
  action workerStop = fire(WorkerStop)
  action backoff = fire(Backoff)
  action scheduleToClose = fire(ScheduleToClose)
  action scheduleToStart = fire(ScheduleToStart)
  action startToClose = fire(StartToClose)

  action step = {
    nondet a = oneOf(actionClasses)
    fire(a)
  }

  def fired(a: Action): bool = last.firedIn(a)
  def recorded(f: ProtocolFact): bool = last.recordedIn(f)

  val refinesProduct = match last {
    | None => true
    | Some(l) =>
      val before = productOf(l.before)
      refinementRow(before, productOf(state),
        rowsFrom(before, product::actionClasses, product::steps))
  }

  // ### What the machines promise

  // The two product transition claims, read here through the map.
  val terminalIsFinal = match last {
    | None => true
    | Some(l) => product::terminalIsFinalRow(productOf(l.before), productOf(state))
  }

  val pausedIsNotDispatched = match last {
    | None => true
    | Some(l) => product::pausedIsNotDispatchedRow(productOf(l.before), productOf(state))
  }

  // A completed attempt settles the activity as completed, and Describe reads it.
  val completes = claim(fired(AttemptResult(AttemptCompleted)),
    state.phase == Completed and recorded(StatusCompleted))

  // A non-retryable failure settles the activity as failed.
  val nonRetryableFails = claim(fired(AttemptResult(AttemptFailed(false))),
    state.phase == Failed and recorded(StatusFailed))

  /// Completed on the second attempt with no deadline set; a claim fixes one state.
  pure val completedOnRetry: ProtocolState =
    { phase: Completed, attempts: 2, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  // The retried attempt completes: the count the retryable failure raised is two.
  val retryCompletes = claim(fired(AttemptResult(AttemptCompleted)),
    state == completedOnRetry and recorded(StatusCompleted))

  // A cancel request against a running attempt is recorded and waits for the worker.
  val cancelRequestedWhileStarted = claim(fired(Control(RequestCancel)),
    state.phase == CancelRequested and recorded(StatusCancelRequested))

  // The worker honors the request, and Describe reads canceled.
  val canceledByWorker = claim(fired(AttemptResult(AttemptCanceled)),
    state.phase == Canceled and recorded(StatusCanceled))

  // Termination settles any live activity.
  val terminated = claim(fired(Control(Terminate)),
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
    asInvariant(cancelRequestedWhileStarted),
    asInvariant(canceledByWorker),
    asInvariant(terminated),
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

// ### The activity and its worker
//
// `activityWorker = polling restricted to [workerStop, serve]`; every dispatch is the worker
// serving, so `attemptStart` has a row only while the worker polls.
module standaloneActivity {
  import umpire.* from "./umpire"
  import activityVocabulary.*
  import activityProtocolTable.*
  import activityProtocol as activity
  import Worker(taskQueue = "activity") as worker from "./umpire"

  pure val activityWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  action init = all { activity::init, worker::init }

  // sync: workerStop = activity.workerStop ∥ worker.workerStop
  action workerStop = all { activity::workerStop, worker::workerStop }
  // sync: attemptStart = activity.attemptStart ∥ worker.serve
  action attemptStart = all { activity::attemptStart, worker::serve }

  pure val unsynced: Set[Action] = actionClasses.filter(a =>
    match a {
      | AttemptStart => false
      | WorkerStop => false
      | _ => true
    })

  action step = any {
    { nondet a = oneOf(unsynced) activity::fire(a) },
    attemptStart,
    workerStop,
  }

  // starts: [activity.unstarted, worker.polling]; ends: the five terminals.
  val ended = terminalPhase(activity::state.phase)

  // Every dispatch leaves the worker polling: no attempt starts while its worker is stopped.
  val startedByPollingWorker =
    claim(activity::fired(AttemptStart), worker::state.phase == worker::Polling)

  val pollingWorkerStarts = asInvariant(startedByPollingWorker)

  // The start request sets the schedule-to-start deadline; the first poll dispatches and the
  // attempt fails retryably; the backoff fires; the worker stops before the retry, so nothing
  // polls again and the deadline fires. The claim is exercised by the dispatch, not vacuous.
  run stoppedBeforeRetry =
    init.then(activity::start(Unset, Expires, Unset)).then(attemptStart)
      .then(activity::attemptResult(AttemptFailed(true))).then(activity::backoff)
      .then(workerStop).then(activity::scheduleToStart)

  // verify startedByPollingWorker in stoppedBeforeRetry, six.
  run stoppedWorkerStartsNothing =
    init.then(activity::start(Unset, Expires, Unset)).expect(pollingWorkerStarts)
      .then(attemptStart).expect(pollingWorkerStarts)
      .then(activity::attemptResult(AttemptFailed(true))).expect(pollingWorkerStarts)
      .then(activity::backoff).expect(pollingWorkerStarts)
      .then(workerStop).expect(pollingWorkerStarts)
      .then(activity::scheduleToStart).expect(pollingWorkerStarts)
}
