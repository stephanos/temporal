module activityProductTable {
  // Quint 0.33.0 flattens private dependencies into the caller's namespace. Unique names
  // keep product transitions distinct from protocol transitions during TLA+ compilation.
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"

  type ActivityProductPhase =
    | ActivityScheduled
    | ActivityStarted
    | ActivityPaused
    | ActivityCancelRequested
    | ActivityCompleted
    | ActivityFailed
    | ActivityCanceled
    | ActivityTerminated
    | ActivityTimedOut

  type ActivityProductState = { phase: ActivityProductPhase }

  type ActivityProductFact =
    | ActivityStatusScheduled
    | ActivityStatusStarted
    | ActivityStatusPaused
    | ActivityStatusCancelRequested
    | ActivityStatusCompleted
    | ActivityStatusFailed
    | ActivityStatusCanceled
    | ActivityStatusTerminated
    | ActivityStatusTimedOut

  type ActivityProductStep = Step[ActivityProductState, ActivityProductFact]

  type ActivityAction =
    | ActivityAttemptStart
    | ActivityAttemptResultAction(AttemptResult)
    | ActivityControlAction(Control)
    | ActivityWorkerStop
    | ActivityTimeoutFired

  pure def activityProductStep(phase: ActivityProductPhase, recorded: ActivityProductFact): List[ActivityProductStep] =
    accepted({ phase: phase }, [recorded])

  /// The five phases the product machine ends on.
  pure def activityProductTerminal(s: ActivityProductState): bool =
    Set(ActivityCompleted, ActivityFailed, ActivityCanceled, ActivityTerminated, ActivityTimedOut).contains(s.phase)

  /// The worker's poll receives the task. Only a scheduled activity is dispatched.
  pure def activityAttemptStartStep(s: ActivityProductState): List[ActivityProductStep] =
    if (s.phase != ActivityScheduled) [] else activityProductStep(ActivityStarted, ActivityStatusStarted)

  /// The worker reports the attempt. Unlike the Nexus Model, the retry is visible here: a
  /// retryable failure reschedules the activity and Describe reads SCHEDULED again
  /// (TransitionRescheduled), or, once a cancel was requested, reads CANCELED. A cancellation
  /// is honored only where one was requested.
  pure def activityAttemptResultStep(s: ActivityProductState, result: AttemptResult): List[ActivityProductStep] =
    if (s.phase != ActivityStarted and s.phase != ActivityCancelRequested) [] else
    match result {
      | AttemptCompleted => activityProductStep(ActivityCompleted, ActivityStatusCompleted)
      | AttemptFailed(retryable) =>
        if (not(retryable)) activityProductStep(ActivityFailed, ActivityStatusFailed)
        else if (s.phase == ActivityStarted) activityProductStep(ActivityScheduled, ActivityStatusScheduled)
        else activityProductStep(ActivityCanceled, ActivityStatusCanceled)
      | AttemptCanceled =>
        if (s.phase == ActivityCancelRequested) activityProductStep(ActivityCanceled, ActivityStatusCanceled) else []
    }

  /// A caller control. On a terminal activity every control is not found and changes nothing.
  /// `requestCancel` is idempotent in `CancelRequested` and records the status again.
  pure def activityControlStep(s: ActivityProductState, control: Control): List[ActivityProductStep] =
    if (activityProductTerminal(s)) notFound(s) else
    match control {
      | Pause =>
        if (s.phase == ActivityScheduled or s.phase == ActivityStarted) activityProductStep(ActivityPaused, ActivityStatusPaused) else []
      | Unpause =>
        if (s.phase == ActivityPaused) activityProductStep(ActivityScheduled, ActivityStatusScheduled) else []
      | RequestCancel =>
        if (Set(ActivityScheduled, ActivityStarted, ActivityPaused, ActivityCancelRequested).contains(s.phase))
          activityProductStep(ActivityCancelRequested, ActivityStatusCancelRequested)
        else []
      | Terminate => activityProductStep(ActivityTerminated, ActivityStatusTerminated)
    }

  /// Invisible, as in the Nexus Model: a kept state with no facts would read as a stutter.
  pure def activityWorkerStopStep(s: ActivityProductState): List[ActivityProductStep] = []

  /// One of the activity's deadlines firing, whichever it was.
  pure def activityTimeoutStep(s: ActivityProductState): List[ActivityProductStep] =
    if (Set(ActivityScheduled, ActivityStarted, ActivityCancelRequested, ActivityPaused).contains(s.phase))
      activityProductStep(ActivityTimedOut, ActivityStatusTimedOut)
    else []

  pure def activitySteps(s: ActivityProductState, a: ActivityAction): List[ActivityProductStep] =
    match a {
      | ActivityAttemptStart => activityAttemptStartStep(s)
      | ActivityAttemptResultAction(result) => activityAttemptResultStep(s, result)
      | ActivityControlAction(control) => activityControlStep(s, control)
      | ActivityWorkerStop => activityWorkerStopStep(s)
      | ActivityTimeoutFired => activityTimeoutStep(s)
    }

  pure val activityPhases: Set[ActivityProductPhase] =
    Set(ActivityScheduled, ActivityStarted, ActivityPaused, ActivityCancelRequested, ActivityCompleted, ActivityFailed, ActivityCanceled, ActivityTerminated,
        ActivityTimedOut)
  pure val activityStates: Set[ActivityProductState] = activityPhases.map(p => { phase: p })
  pure val activityStarts: Set[ActivityProductState] = Set({ phase: ActivityScheduled })
  pure val activityEnds: Set[ActivityProductState] = activityStates.filter(activityProductTerminal)
  pure val activityTimers: Set[ActivityAction] = Set(ActivityTimeoutFired)

  pure val activityActionClasses: Set[ActivityAction] =
    Set(ActivityAttemptStart, ActivityWorkerStop, ActivityTimeoutFired)
      .union(attemptResults.map(r => ActivityAttemptResultAction(r)))
      .union(controls.map(c => ActivityControlAction(c)))

  /// `evidence:` every fact is a Describe status read.
  pure def activityEvidence(f: ActivityProductFact): str =
    match f {
      | ActivityStatusScheduled => "statusScheduled"
      | ActivityStatusStarted => "statusStarted"
      | ActivityStatusPaused => "statusPaused"
      | ActivityStatusCancelRequested => "statusCancelRequested"
      | ActivityStatusCompleted => "statusCompleted"
      | ActivityStatusFailed => "statusFailed"
      | ActivityStatusCanceled => "statusCanceled"
      | ActivityStatusTerminated => "statusTerminated"
      | ActivityStatusTimedOut => "statusTimedOut"
    }

  // Once an activity is over, no step changes its phase.
  pure def activityTerminalIsFinalRow(before: ActivityProductState, after: ActivityProductState): bool =
    not(activityProductTerminal(before)) or after.phase == before.phase

  // A paused activity is not dispatched: no step takes it straight to started.
  pure def activityPausedIsNotDispatchedRow(before: ActivityProductState, after: ActivityProductState): bool =
    before.phase == ActivityPaused implies after.phase != ActivityStarted
  pure val activityProductScheduled: ActivityProductState = { phase: ActivityScheduled }
  pure val activityProductStarted: ActivityProductState = { phase: ActivityStarted }
  pure val activityProductPaused: ActivityProductState = { phase: ActivityPaused }
  pure val activityProductCancelRequested: ActivityProductState = { phase: ActivityCancelRequested }
  pure val activityProductCompleted: ActivityProductState = { phase: ActivityCompleted }
  pure val activityProductFailed: ActivityProductState = { phase: ActivityFailed }
  pure val activityProductCanceled: ActivityProductState = { phase: ActivityCanceled }
  pure val activityProductTerminated: ActivityProductState = { phase: ActivityTerminated }
  pure val activityProductTimedOut: ActivityProductState = { phase: ActivityTimedOut }

  pure def activityProductAllows(before: ActivityProductState, after: ActivityProductState): bool =
    refinementRow(before, after, rowsFrom(before, activityActionClasses, activitySteps))

  type ProductPhase = ActivityProductPhase
  type ProductState = ActivityProductState
  type ProductFact = ActivityProductFact
  type ProductStep = ActivityProductStep
  type Action = ActivityAction
  pure def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] = activityProductStep(phase, recorded)
  pure def productTerminal(s: ProductState): bool = activityProductTerminal(s)
  pure def attemptStartStep(s: ProductState): List[ProductStep] = activityAttemptStartStep(s)
  pure def attemptResultStep(s: ProductState, result: AttemptResult): List[ProductStep] = activityAttemptResultStep(s, result)
  pure def controlStep(s: ProductState, control: Control): List[ProductStep] = activityControlStep(s, control)
  pure def workerStopStep(s: ProductState): List[ProductStep] = activityWorkerStopStep(s)
  pure def timeoutStep(s: ProductState): List[ProductStep] = activityTimeoutStep(s)
  pure def steps(s: ProductState, a: Action): List[ProductStep] = activitySteps(s, a)
  pure val phases = activityPhases
  pure val states = activityStates
  pure val starts = activityStarts
  pure val ends = activityEnds
  pure val timers = activityTimers
  pure val actionClasses = activityActionClasses
  pure def evidence(f: ProductFact): str = activityEvidence(f)
  pure def terminalIsFinalRow(before: ProductState, after: ProductState): bool = activityTerminalIsFinalRow(before, after)
  pure def pausedIsNotDispatchedRow(before: ProductState, after: ProductState): bool = activityPausedIsNotDispatchedRow(before, after)
  pure val productScheduled = activityProductScheduled
  pure val productStarted = activityProductStarted
  pure val productPaused = activityProductPaused
  pure val productCancelRequested = activityProductCancelRequested
  pure val productCompleted = activityProductCompleted
  pure val productFailed = activityProductFailed
  pure val productCanceled = activityProductCanceled
  pure val productTerminated = activityProductTerminated
  pure val productTimedOut = activityProductTimedOut
  pure def productAllows(before: ProductState, after: ProductState): bool = activityProductAllows(before, after)
  pure val Scheduled = ActivityScheduled
  pure val Started = ActivityStarted
  pure val Paused = ActivityPaused
  pure val CancelRequested = ActivityCancelRequested
  pure val Completed = ActivityCompleted
  pure val Failed = ActivityFailed
  pure val Canceled = ActivityCanceled
  pure val Terminated = ActivityTerminated
  pure val TimedOut = ActivityTimedOut
  pure val StatusScheduled = ActivityStatusScheduled
  pure val StatusStarted = ActivityStatusStarted
  pure val StatusPaused = ActivityStatusPaused
  pure val StatusCancelRequested = ActivityStatusCancelRequested
  pure val StatusCompleted = ActivityStatusCompleted
  pure val StatusFailed = ActivityStatusFailed
  pure val StatusCanceled = ActivityStatusCanceled
  pure val StatusTerminated = ActivityStatusTerminated
  pure val StatusTimedOut = ActivityStatusTimedOut
  pure val AttemptStart = ActivityAttemptStart
  pure def AttemptResultAction(value: AttemptResult): Action = ActivityAttemptResultAction(value)
  pure def ControlAction(value: Control): Action = ActivityControlAction(value)
  pure val WorkerStop = ActivityWorkerStop
  pure val TimeoutFired = ActivityTimeoutFired

}
