// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the
// same actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state record has no "no instance yet"
// member, so `Unscheduled` is that member, and it is what makes the three deadline fields
// reachable at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79),
// and the concurrency-limit rejection. The limit exists, but a step function does not read the
// setup, the key and value differ per switch value, and the rejection names no operation, so it
// is not modeled until a Query needs it.
module nexusProtocolTable {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProductTable.ProductState from "./nexus_product_table"
  import nexusProductTable.productScheduled from "./nexus_product_table"
  import nexusProductTable.productStarted from "./nexus_product_table"
  import nexusProductTable.productSucceeded from "./nexus_product_table"
  import nexusProductTable.productFailed from "./nexus_product_table"
  import nexusProductTable.productCanceled from "./nexus_product_table"
  import nexusProductTable.productTimedOut from "./nexus_product_table"
  import nexusProductTable.productAllows from "./nexus_product_table"
  import nexusProductTable.terminalIsFinalRow from "./nexus_product_table"

  type Phase =
    | Unscheduled
    | Scheduled
    | BackingOff
    | Started
    | Succeeded
    | Failed
    | Canceled
    | TimedOut

  /// Which timer fired. The history event records it, so a Contract that did not check it would
  /// pass a run that timed out on the wrong deadline.
  type TimeoutType = ByScheduleToClose | ByScheduleToStart | ByStartToClose

  // `attemptBound` is `umpire::attemptBound` = 2; `attempts` ranges over `attemptCounts`.

  type ProtocolState = {
    phase: Phase,
    attempts: int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
  }

  type ProtocolFact =
    | NexusOperationScheduled
    | NexusOperationStarted
    | NexusOperationCompleted
    | NexusOperationFailed
    | NexusOperationCanceled
    | NexusOperationTimedOut(TimeoutType)
    | PendingAttempts

  type ProtocolStep = Step[ProtocolState, ProtocolFact]

  type Deadlines = { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout }

  type Action =
    | Schedule(Deadlines)
    | HandlerReply(Reply)
    | Complete(Resolution)
    | TransportFault
    | WorkerStop
    | Backoff
    | ScheduleToClose
    | ScheduleToStart
    | StartToClose

  /// The four phases the design ends on. A completion that arrives after one of them is not found.
  pure def terminalPhase(phase: Phase): bool =
    Set(Succeeded, Failed, Canceled, TimedOut).contains(phase)

  /// Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
  pure def running(phase: Phase): bool =
    Set(Scheduled, BackingOff, Started).contains(phase)

  pure def moves(s: ProtocolState, phase: Phase, recorded: List[ProtocolFact]): List[ProtocolStep] =
    accepted(s.with("phase", phase), recorded)

  /// The caller's schedule command. It names the operation's three deadlines, and every one of
  /// them is a state field because whether a timer fires is a question about the operation and
  /// not about the command that started it.
  pure def scheduleStep(s: ProtocolState, d: Deadlines): List[ProtocolStep] =
    if (s.phase != Unscheduled) [] else
    accepted({ phase: Scheduled, attempts: 0, scheduleToClose: d.scheduleToClose,
               scheduleToStart: d.scheduleToStart, startToClose: d.startToClose },
             [NexusOperationScheduled])

  /// The handler's reply to the server's start request. What the product machine cannot see is
  /// the last arm: a retryable failure backs the operation off and raises its attempt count, and
  /// the count is read back through the `pendingAttempts` observation because no history event
  /// records it.
  pure def protocolHandlerReplyStep(s: ProtocolState, reply: Reply): List[ProtocolStep] =
    if (s.phase != Scheduled) [] else
    match reply {
      | SyncSuccess => moves(s, Succeeded, [NexusOperationCompleted])
      | Async => moves(s, Started, [NexusOperationStarted])
      | OperationFailed => moves(s, Failed, [NexusOperationFailed])
      | OperationCanceled => moves(s, Canceled, [NexusOperationCanceled])
      | HandlerError(retryable) =>
        if (not(retryable)) moves(s, Failed, [NexusOperationFailed])
        else accepted({ ...s, phase: BackingOff, attempts: saturatingSucc(s.attempts) },
                      [PendingAttempts])
    }

  /// A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
  pure def protocolTransportFaultStep(s: ProtocolState): List[ProtocolStep] =
    if (s.phase != Scheduled) [] else
    accepted({ ...s, phase: BackingOff, attempts: saturatingSucc(s.attempts) }, [PendingAttempts])

  /// The handler's worker stopping is a fault the Run records and the operation does not feel,
  /// so the step keeps the state and records nothing. On a path it is confirmed by the evidence
  /// of the step after it, and the Case says so in a Known Gap.
  pure def protocolWorkerStopStep(s: ProtocolState): List[ProtocolStep] = accepted(s, [])

  /// An asynchronous completion. Before a start, the server records a Started event first, which
  /// is why the evidence is two facts and not one -- and why the product machine, which has no
  /// `BackingOff` phase to have skipped, could write the completion alone.
  pure def protocolCompleteStep(s: ProtocolState, resolution: Resolution): List[ProtocolStep] =
    if (terminalPhase(s.phase)) notFound(s)
    else if (s.phase == Unscheduled) []
    else
      val startedFirst: List[ProtocolFact] = if (s.phase == Started) [] else [NexusOperationStarted]
      match resolution {
        | ResolvedSucceeded => moves(s, Succeeded, startedFirst.append(NexusOperationCompleted))
        | ResolvedFailed => moves(s, Failed, startedFirst.append(NexusOperationFailed))
        | ResolvedCanceled => moves(s, Canceled, startedFirst.append(NexusOperationCanceled))
      }

  /// The backoff timer. It is what makes `BackingOff` a phase the operation leaves rather than a
  /// state it is stuck in, and it records nothing: a retry writes no history event.
  pure def backoffStep(s: ProtocolState): List[ProtocolStep] =
    if (s.phase != BackingOff) [] else moves(s, Scheduled, [])

  /// The schedule-to-close deadline covers the whole operation, so it fires in every running
  /// phase -- and only when the schedule command set it.
  pure def scheduleToCloseStep(s: ProtocolState): List[ProtocolStep] =
    if (running(s.phase) and s.scheduleToClose == Expires)
      moves(s, TimedOut, [NexusOperationTimedOut(ByScheduleToClose)])
    else []

  /// The schedule-to-start deadline covers the wait for the handler to accept, so it stops at
  /// the start.
  pure def scheduleToStartStep(s: ProtocolState): List[ProtocolStep] =
    if ((s.phase == Scheduled or s.phase == BackingOff) and s.scheduleToStart == Expires)
      moves(s, TimedOut, [NexusOperationTimedOut(ByScheduleToStart)])
    else []

  /// The start-to-close deadline covers the handler's own work, so it begins at the start.
  pure def startToCloseStep(s: ProtocolState): List[ProtocolStep] =
    if (s.phase == Started and s.startToClose == Expires)
      moves(s, TimedOut, [NexusOperationTimedOut(ByStartToClose)])
    else []

  /// How a protocol state reads as a product state. A phase of the same name is that phase;
  /// backing off is still scheduled, because the product machine cannot see a retry; and an
  /// operation not yet scheduled reads as scheduled, because the product machine begins there.
  /// Every other field is hidden, which is what a map that does not read it says.
  pure def productOf(s: ProtocolState): ProductState =
    match s.phase {
        | Unscheduled => productScheduled
        | Scheduled => productScheduled
        | BackingOff => productScheduled
        | Started => productStarted
        | Succeeded => productSucceeded
        | Failed => productFailed
        | Canceled => productCanceled
        | TimedOut => productTimedOut
      }

  // The `machine nexusProtocol` block, as a table.

  pure def steps(s: ProtocolState, a: Action): List[ProtocolStep] =
    match a {
      | Schedule(d) => scheduleStep(s, d)
      | HandlerReply(reply) => protocolHandlerReplyStep(s, reply)
      | Complete(resolution) => protocolCompleteStep(s, resolution)
      | TransportFault => protocolTransportFaultStep(s)
      | WorkerStop => protocolWorkerStopStep(s)
      | Backoff => backoffStep(s)
      | ScheduleToClose => scheduleToCloseStep(s)
      | ScheduleToStart => scheduleToStartStep(s)
      | StartToClose => startToCloseStep(s)
    }

  pure val phases: Set[Phase] =
    Set(Unscheduled, Scheduled, BackingOff, Started, Succeeded, Failed, Canceled, TimedOut)

  pure val states: Set[ProtocolState] =
    tuples(phases, attemptCounts, timeouts, timeouts, timeouts).map(((p, n, c, s, t)) =>
      { phase: p, attempts: n, scheduleToClose: c, scheduleToStart: s, startToClose: t })

  pure val unscheduled: ProtocolState =
    { phase: Unscheduled, attempts: 0, scheduleToClose: Unset, scheduleToStart: Unset,
      startToClose: Unset }

  pure val starts: Set[ProtocolState] = Set(unscheduled)
  pure val ends: Set[ProtocolState] = states.filter(s => terminalPhase(s.phase))
  pure val timers: Set[Action] = Set(Backoff, ScheduleToClose, ScheduleToStart, StartToClose)
  pure val unobservable: Set[Action] = Set(Backoff)

  pure val deadlines: Set[Deadlines] =
    tuples(timeouts, timeouts, timeouts).map(((c, s, t)) =>
      { scheduleToClose: c, scheduleToStart: s, startToClose: t })

  pure val actionClasses: Set[Action] =
    deadlines.map(d => Schedule(d))
      .union(replies.map(r => HandlerReply(r)))
      .union(resolutions.map(r => Complete(r)))
      .union(Set(TransportFault, WorkerStop))
      .union(timers)

  /// `evidence:` fact → recorded event or observation name.
  pure def evidence(f: ProtocolFact): str =
    match f {
      | NexusOperationScheduled => "nexusOperationScheduled"
      | NexusOperationStarted => "nexusOperationStarted"
      | NexusOperationCompleted => "nexusOperationCompleted"
      | NexusOperationFailed => "nexusOperationFailed"
      | NexusOperationCanceled => "nexusOperationCanceled"
      | NexusOperationTimedOut(_) => "nexusOperationTimedOut"
      | PendingAttempts => "pendingAttempts"
    }

  /// `refines: nexusProduct` with `map: productOf`, over the whole table: every row's mapped
  /// states are equal or are a product row under some class. `quint test` evaluates it in
  /// pins.qnt; the machine module checks the same row predicate as an invariant.
  pure val refinement: bool =
    tuples(states, actionClasses).forall(((s, a)) =>
      steps(s, a).toSet().forall(row => productAllows(productOf(s), productOf(row.state))))
}
