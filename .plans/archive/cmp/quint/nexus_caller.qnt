// -*- mode: quint -*-
//
// # The Nexus caller-side Model
//
// One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
// operation does, the protocol machine says how the server gets there and refines it, and the
// functional set runs one Query per side effect that settles the operation, once per value of
// the implementation switch.
//
// Read from top to bottom: vocabulary → the two machines → what they promise → what the set
// asks. Each machine is two modules: a `Table` module of types and pure step functions, which
// the pins and the refinement walk without dragging state along, and the machine module that
// holds the `var`s and turns the table into actions.
//
// Typecheck:  quint typecheck nexus_caller.qnt
// Scenarios:  quint test nexus_caller.qnt --main nexusProtocol
// Explore:    quint run nexus_caller.qnt --main nexusProtocol --invariant properties --mbt
// Verify:     quint verify nexus_caller.qnt --main nexusProtocol --invariant refinesProduct

// ### Entities and the input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes
// one class per assignment of them: `HandlerError(retryable)` is one constructor and two classes,
// which is the granularity an example is written at and what mirrors a protobuf oneof.
//
// Quint constructor labels are module-global, so `Resolution`'s members carry a `Resolved`
// prefix to live beside the phases of the same name; see README.
module nexusCallerVocabulary {
  import umpire.* from "./umpire"

  // An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
  // event: every history event of the operation carries that event's id.
  pure val workflow: Entity = { name: "workflow", key: "", refer: Set() }
  pure val operation: Entity =
    { name: "operation", key: "scheduledEvent", refer: Set(("caller", "workflow")) }

  type Reply = SyncSuccess | Async | OperationFailed | OperationCanceled | HandlerError(bool)

  type Resolution = ResolvedSucceeded | ResolvedFailed | ResolvedCanceled

  pure val replies: Set[Reply] =
    Set(SyncSuccess, Async, OperationFailed, OperationCanceled, HandlerError(true),
        HandlerError(false))

  pure val resolutions: Set[Resolution] =
    Set(ResolvedSucceeded, ResolvedFailed, ResolvedCanceled)

  // ### Actions
  //
  // Parties are names the feature declares by using them: `caller`, `handler`, `network`,
  // `worker`. The reserved party `system` is the server. A fault is an ordinary action of a
  // declared party, and a timer is `system` behavior the machine owns, so neither is a separate
  // kind. The action *types* are per machine (each machine's `Action` sum type); this catalog is
  // the part Quint cannot check: party, entity, schema, examples.
  pure val actions: Map[str, ActionDecl] = Map(
    "schedule" -> { party: Caller, creates: "operation", on: "",
      schema: "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
      examples: Set() },
    "handlerReply" -> { party: Handler, creates: "", on: "operation",
      schema: "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError",
      examples: Set(("HandlerError(false)", "BadRequest"), ("HandlerError(true)", "Internal")) },
    // The Nexus HTTP completion carries no protobuf message, so it declares no schema and its
    // classes are names the realization interprets.
    "complete" -> decl(Handler).with("on", "operation"),
    "transportFault" -> decl(Network).with("on", "operation"),
    // The handler's worker stops polling. An action that names no entity is behavior no entity
    // records: the Run records the fault, but nothing recorded names the operation, so the
    // machines keep their state and record nothing at it.
    "workerStop" -> decl(Worker),
  )

  // ### The derived observation
  //
  // A retryable attempt failure writes no history event, so the attempt count is read back
  // through `DescribeWorkflowExecution`. Every other evidence name resolves against the
  // realization's catalog, which is why only a derived observation is declared.
  pure val pendingAttempts: (str, str) = ("operation", "attempts")
}

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried
// to the protocol machine by the refinement declared there.
module nexusProductTable {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.*

  type ProductPhase = Scheduled | Started | Succeeded | Failed | Canceled | TimedOut

  type ProductState = { phase: ProductPhase }

  // The Lean `ProductOutcome` is `accepted | notFound`, which is `Delivery`; Quint uses it directly.

  type ProductFact =
    | NexusOperationScheduled
    | NexusOperationStarted
    | NexusOperationCompleted
    | NexusOperationFailed
    | NexusOperationCanceled
    | NexusOperationTimedOut

  type ProductStep = Step[ProductState, ProductFact]

  /// The action classes the product machine steps on, one constructor per action, with the
  /// finite inputs as payload.
  type Action =
    | HandlerReply(Reply)
    | Complete(Resolution)
    | TransportFault
    | WorkerStop
    | TimeoutFired

  pure def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] =
    accepted({ phase: phase }, [recorded])

  /// The handler's reply to the server's start request. An operation that has not started yet is
  /// the only one a reply can move.
  pure def handlerReplyStep(s: ProductState, reply: Reply): List[ProductStep] =
    if (s.phase != Scheduled) [] else
    match reply {
      | SyncSuccess => productStep(Succeeded, NexusOperationCompleted)
      | Async => productStep(Started, NexusOperationStarted)
      | OperationFailed => productStep(Failed, NexusOperationFailed)
      | OperationCanceled => productStep(Canceled, NexusOperationCanceled)
      // A retryable handler error leaves the operation where it is: the product machine does not
      // know about backing off, which is the whole of what the protocol machine adds.
      | HandlerError(retryable) =>
        if (retryable) [] else productStep(Failed, NexusOperationFailed)
    }

  /// The four phases the product machine ends on.
  pure def productTerminal(s: ProductState): bool =
    Set(Succeeded, Failed, Canceled, TimedOut).contains(s.phase)

  /// An asynchronous completion. A completion that arrives after the operation is over is not
  /// found, and changes nothing.
  pure def completeStep(s: ProductState, resolution: Resolution): List[ProductStep] =
    if (productTerminal(s)) notFound(s) else
    match resolution {
      | ResolvedSucceeded => productStep(Succeeded, NexusOperationCompleted)
      | ResolvedFailed => productStep(Failed, NexusOperationFailed)
      | ResolvedCanceled => productStep(Canceled, NexusOperationCanceled)
    }

  /// A transport fault is an ordinary action of the network. The product machine cannot see one:
  /// whether a delivery was retried is the protocol's account of how, not what.
  pure def transportFaultStep(s: ProductState): List[ProductStep] = []

  /// The handler's worker stopping is a fault the Run records and the operation does not feel.
  /// The product machine cannot see it, like the transport fault: a step that kept the state and
  /// recorded nothing would be indistinguishable from a stutter, and the refinement would read
  /// every stutter as this step.
  pure def workerStopStep(s: ProductState): List[ProductStep] = []

  /// One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
  /// the product machine has one timer, and it fires while the operation runs.
  pure def timeoutStep(s: ProductState): List[ProductStep] =
    if (s.phase == Scheduled or s.phase == Started)
      productStep(TimedOut, NexusOperationTimedOut)
    else []

  // The `machine nexusProduct` block, as a table: `steps:` is the dispatch, the rest are sets.

  pure def steps(s: ProductState, a: Action): List[ProductStep] =
    match a {
      | HandlerReply(reply) => handlerReplyStep(s, reply)
      | Complete(resolution) => completeStep(s, resolution)
      | TransportFault => transportFaultStep(s)
      | WorkerStop => workerStopStep(s)
      | TimeoutFired => timeoutStep(s)
    }

  pure val phases: Set[ProductPhase] = Set(Scheduled, Started, Succeeded, Failed, Canceled, TimedOut)
  pure val states: Set[ProductState] = phases.map(p => { phase: p })
  pure val starts: Set[ProductState] = Set({ phase: Scheduled })
  pure val ends: Set[ProductState] = states.filter(productTerminal)
  pure val timers: Set[Action] = Set(TimeoutFired)

  pure val actionClasses: Set[Action] =
    replies.map(r => HandlerReply(r))
      .union(resolutions.map(r => Complete(r)))
      .union(Set(TransportFault, WorkerStop, TimeoutFired))

  /// `evidence:` fact → recorded event name.
  pure def evidence(f: ProductFact): str =
    match f {
      | NexusOperationScheduled => "nexusOperationScheduled"
      | NexusOperationStarted => "nexusOperationStarted"
      | NexusOperationCompleted => "nexusOperationCompleted"
      | NexusOperationFailed => "nexusOperationFailed"
      | NexusOperationCanceled => "nexusOperationCanceled"
      | NexusOperationTimedOut => "nexusOperationTimedOut"
    }

  // Once an operation is over, no step changes its phase. Declared on the product machine, as a
  // transition claim on (before, after), and read on the protocol machine through the map.
  pure def terminalIsFinalRow(before: ProductState, after: ProductState): bool =
    not(productTerminal(before)) or after.phase == before.phase
}

module nexusProduct {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.*
  import nexusProductTable.*

  var state: ProductState
  var last: Option[Fired[Action, ProductState, ProductFact]]

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None }
  }

  /// Take the one row the table has for this class, or be disabled.
  action fire(a: Action): bool = {
    val rows = steps(state, a)
    all {
      rows.length() > 0,
      state' = rows[0].state,
      last' = Some({ action: a, before: state, outcome: rows[0].outcome, facts: rows[0].facts }),
    }
  }

  action handlerReply(reply: Reply): bool = fire(HandlerReply(reply))
  action complete(resolution: Resolution): bool = fire(Complete(resolution))
  action transportFault = fire(TransportFault)
  action workerStop = fire(WorkerStop)
  action timeout = fire(TimeoutFired)

  /// Exploration: any class of the catalog.
  action step = {
    nondet a = oneOf(actionClasses)
    fire(a)
  }

  def fired(a: Action): bool = last.firedIn(a)
  def recorded(f: ProductFact): bool = last.recordedIn(f)

  val terminalIsFinal = match last {
    | None => true
    | Some(l) => nexusProductTable::terminalIsFinalRow(l.before, state)
  }
}

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
  import nexusCallerVocabulary.*
  import nexusProductTable as product

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
  pure def productOf(s: ProtocolState): product::ProductState =
    { phase: match s.phase {
        | Unscheduled => product::Scheduled
        | Scheduled => product::Scheduled
        | BackingOff => product::Scheduled
        | Started => product::Started
        | Succeeded => product::Succeeded
        | Failed => product::Failed
        | Canceled => product::Canceled
        | TimedOut => product::TimedOut
      } }

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
    refines(states, actionClasses, steps, productOf, product::actionClasses, product::steps)
}

module nexusProtocol {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.*
  import nexusProtocolTable.*
  import nexusProductTable as product

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
    nondet a = oneOf(actionClasses)
    fire(a)
  }

  def fired(a: Action): bool = last.firedIn(a)
  def recorded(f: ProtocolFact): bool = last.recordedIn(f)

  /// The refinement as a state invariant: the last row taken maps to a product row or stutter.
  val refinesProduct = match last {
    | None => true
    | Some(l) =>
      val before = productOf(l.before)
      refinementRow(before, productOf(state),
        rowsFrom(before, product::actionClasses, product::steps))
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
    | Some(l) => product::terminalIsFinalRow(productOf(l.before), productOf(state))
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
    asInvariant(completionSucceeds),
    asInvariant(completionFails),
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

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase
// change and every reply is the worker serving, so a reply has a row only while the worker
// polls. No set names the composition; it is what the cross-entity claim is verified over.
//
// Quint composes by importing two stateful modules and writing the synchronized actions as
// `all { member1, member2 }`; a member action no `sync` names fires alone through its qualifier.
module nexusCaller {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.*
  import nexusProtocolTable.*
  import nexusProtocol as operation
  import Worker(taskQueue = "handler") as worker from "./umpire"

  // The caller's view of the handler's worker: it stops and it serves. It never resumes, because
  // an action no `sync` line names would stay executable on its own and admit a stop, a resume
  // and then a reply; the operation's timers settle every state a stop leaves.
  // `handlerWorker = polling restricted to [workerStop, serve]`: the composition's `step` below
  // names only these two of the worker's actions.
  pure val handlerWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  // `NexusCallerState` is the pair of member states; Quint keeps them as `operation::state` and
  // `worker::state` rather than one record.
  action init = all { operation::init, worker::init }

  // sync: workerStop = operation.workerStop ∥ worker.workerStop
  action workerStop = all { operation::workerStop, worker::workerStop }
  // sync: handlerReply = operation.handlerReply ∥ worker.serve
  action handlerReply(reply: Reply): bool = all { operation::handlerReply(reply), worker::serve }

  /// The operation's classes no `sync` names fire alone.
  pure val unsynced: Set[Action] = actionClasses.filter(a =>
    match a {
      | HandlerReply(_) => false
      | WorkerStop => false
      | _ => true
    })

  action step = any {
    { nondet a = oneOf(unsynced) operation::fire(a) },
    { nondet r = oneOf(replies) handlerReply(r) },
    workerStop,
  }

  // starts: [operation.unscheduled, worker.polling]; ends: the operation's four terminals.
  val ended = terminalPhase(operation::state.phase)

  def firedReply = match operation::last {
    | None => false
    | Some(l) => match l.action {
      | HandlerReply(_) => true
      | _ => false
    }
  }

  // Every reply, of any class, leaves the handler's worker polling: no handler replies while its
  // worker is stopped.
  val repliedByPollingWorker = claim(firedReply, worker::state.phase == worker::Polling)

  // A retryable reply backs the operation off; the handler's worker then stops, so the retried
  // attempt is never answered and the schedule-to-start deadline fires.
  run repliedThenStopped =
    init.then(operation::schedule(Unset, Expires, Unset)).then(handlerReply(HandlerError(true)))
      .then(workerStop).then(operation::scheduleToStart)

  val pollingWorkerReplies = asInvariant(repliedByPollingWorker)

  // verify repliedByPollingWorker in repliedThenStopped, limits four. Over the whole
  // composition: `quint verify nexus_caller.qnt --main nexusCaller --invariant
  // pollingWorkerReplies --max-steps 4`.
  run stoppedWorkerRepliesNothing =
    init.then(operation::schedule(Unset, Expires, Unset)).expect(pollingWorkerReplies)
      .then(handlerReply(HandlerError(true))).expect(pollingWorkerReplies)
      .then(workerStop).expect(pollingWorkerReplies)
      .then(operation::scheduleToStart).expect(pollingWorkerReplies)
}
