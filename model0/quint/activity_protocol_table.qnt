// ### The protocol machine
//
// How the server gets there: the retry the caller cannot see, the pause requested of a running
// attempt, the three timers the start request sets, and the attempt count. `Unstarted` is the
// "no instance yet" member, as `Unscheduled` is in the Nexus Model.
module activityProtocolTable {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"
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
    | AttemptResultAction(AttemptResult)
    | ControlAction(Control)
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
            if (retryable) moves(s, BackingOff, [StatusScheduled, AttemptCount])
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
    if (terminalPhase(s.phase)) notFound(s)
    else if (s.phase == Unstarted) []
    else match control {
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
  pure def productOf(s: ProtocolState): ProductState =
    match s.phase {
        | Unstarted => productScheduled
        | Scheduled => productScheduled
        | BackingOff => productScheduled
        | Started => productStarted
        | Paused => productPaused
        | PauseRequested => productStarted
        | CancelRequested => productCancelRequested
        | Completed => productCompleted
        | Failed => productFailed
        | Canceled => productCanceled
        | Terminated => productTerminated
        | TimedOut => productTimedOut
      }

  pure def steps(s: ProtocolState, a: Action): List[ProtocolStep] =
    match a {
      | Start(d) => startStep(s, d)
      | AttemptStart => protocolAttemptStartStep(s)
      | AttemptResultAction(result) => protocolAttemptResultStep(s, result)
      | ControlAction(control) => protocolControlStep(s, control)
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
      .union(attemptResults.map(r => AttemptResultAction(r)))
      .union(controls.map(c => ControlAction(c)))
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
    tuples(states, actionClasses).forall(((s, a)) =>
      steps(s, a).toSet().forall(row => productAllows(productOf(s), productOf(row.state))))
}
