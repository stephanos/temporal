module nexusProductTable {
  // Quint 0.33.0 flattens private dependencies into the caller's namespace. Unique names
  // keep product transitions distinct from protocol transitions during TLA+ compilation.
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"

  type NexusProductPhase = NexusScheduled | NexusStarted | NexusSucceeded | NexusFailed | NexusCanceled | NexusTimedOut

  type NexusProductState = { phase: NexusProductPhase }

  // The Lean `ProductOutcome` is `accepted | notFound`, which is `Delivery`; Quint uses it directly.

  type NexusProductFact =
    | NexusNexusOperationScheduled
    | NexusNexusOperationStarted
    | NexusNexusOperationCompleted
    | NexusNexusOperationFailed
    | NexusNexusOperationCanceled
    | NexusNexusOperationTimedOut

  type NexusProductStep = Step[NexusProductState, NexusProductFact]

  /// The action classes the product machine steps on, one constructor per action, with the
  /// finite inputs as payload.
  type NexusAction =
    | NexusHandlerReply(Reply)
    | NexusComplete(Resolution)
    | NexusTransportFault
    | NexusWorkerStop
    | NexusTimeoutFired

  pure def nexusProductStep(phase: NexusProductPhase, recorded: NexusProductFact): List[NexusProductStep] =
    accepted({ phase: phase }, [recorded])

  /// The handler's reply to the server's start request. An operation that has not started yet is
  /// the only one a reply can move.
  pure def nexusHandlerReplyStep(s: NexusProductState, reply: Reply): List[NexusProductStep] =
    if (s.phase != NexusScheduled) [] else
    match reply {
      | SyncSuccess => nexusProductStep(NexusSucceeded, NexusNexusOperationCompleted)
      | Async => nexusProductStep(NexusStarted, NexusNexusOperationStarted)
      | OperationFailed => nexusProductStep(NexusFailed, NexusNexusOperationFailed)
      | OperationCanceled => nexusProductStep(NexusCanceled, NexusNexusOperationCanceled)
      // A retryable handler error leaves the operation where it is: the product machine does not
      // know about backing off, which is the whole of what the protocol machine adds.
      | HandlerError(retryable) =>
        if (retryable) [] else nexusProductStep(NexusFailed, NexusNexusOperationFailed)
    }

  /// The four phases the product machine ends on.
  pure def nexusProductTerminal(s: NexusProductState): bool =
    Set(NexusSucceeded, NexusFailed, NexusCanceled, NexusTimedOut).contains(s.phase)

  /// An asynchronous completion. A completion that arrives after the operation is over is not
  /// found, and changes nothing.
  pure def nexusCompleteStep(s: NexusProductState, resolution: Resolution): List[NexusProductStep] =
    if (nexusProductTerminal(s)) notFound(s) else
    match resolution {
      | ResolvedSucceeded => nexusProductStep(NexusSucceeded, NexusNexusOperationCompleted)
      | ResolvedFailed => nexusProductStep(NexusFailed, NexusNexusOperationFailed)
      | ResolvedCanceled => nexusProductStep(NexusCanceled, NexusNexusOperationCanceled)
    }

  /// A transport fault is an ordinary action of the network. The product machine cannot see one:
  /// whether a delivery was retried is the protocol's account of how, not what.
  pure def nexusTransportFaultStep(s: NexusProductState): List[NexusProductStep] = []

  /// The handler's worker stopping is a fault the Run records and the operation does not feel.
  /// The product machine cannot see it, like the transport fault: a step that kept the state and
  /// recorded nothing would be indistinguishable from a stutter, and the refinement would read
  /// every stutter as this step.
  pure def nexusWorkerStopStep(s: NexusProductState): List[NexusProductStep] = []

  /// One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
  /// the product machine has one timer, and it fires while the operation runs.
  pure def nexusTimeoutStep(s: NexusProductState): List[NexusProductStep] =
    if (s.phase == NexusScheduled or s.phase == NexusStarted)
      nexusProductStep(NexusTimedOut, NexusNexusOperationTimedOut)
    else []

  // The `machine nexusProduct` block, as a table: `steps:` is the dispatch, the rest are sets.

  pure def nexusSteps(s: NexusProductState, a: NexusAction): List[NexusProductStep] =
    match a {
      | NexusHandlerReply(reply) => nexusHandlerReplyStep(s, reply)
      | NexusComplete(resolution) => nexusCompleteStep(s, resolution)
      | NexusTransportFault => nexusTransportFaultStep(s)
      | NexusWorkerStop => nexusWorkerStopStep(s)
      | NexusTimeoutFired => nexusTimeoutStep(s)
    }

  pure val nexusPhases: Set[NexusProductPhase] = Set(NexusScheduled, NexusStarted, NexusSucceeded, NexusFailed, NexusCanceled, NexusTimedOut)
  pure val nexusStates: Set[NexusProductState] = nexusPhases.map(p => { phase: p })
  pure val nexusStarts: Set[NexusProductState] = Set({ phase: NexusScheduled })
  pure val nexusEnds: Set[NexusProductState] = nexusStates.filter(nexusProductTerminal)
  pure val nexusTimers: Set[NexusAction] = Set(NexusTimeoutFired)

  pure val nexusActionClasses: Set[NexusAction] =
    replies.map(r => NexusHandlerReply(r))
      .union(resolutions.map(r => NexusComplete(r)))
      .union(Set(NexusTransportFault, NexusWorkerStop, NexusTimeoutFired))

  /// `evidence:` fact → recorded event name.
  pure def nexusEvidence(f: NexusProductFact): str =
    match f {
      | NexusNexusOperationScheduled => "nexusOperationScheduled"
      | NexusNexusOperationStarted => "nexusOperationStarted"
      | NexusNexusOperationCompleted => "nexusOperationCompleted"
      | NexusNexusOperationFailed => "nexusOperationFailed"
      | NexusNexusOperationCanceled => "nexusOperationCanceled"
      | NexusNexusOperationTimedOut => "nexusOperationTimedOut"
    }

  // Once an operation is over, no step changes its phase. Declared on the product machine, as a
  // transition claim on (before, after), and read on the protocol machine through the map.
  pure def nexusTerminalIsFinalRow(before: NexusProductState, after: NexusProductState): bool =
    not(nexusProductTerminal(before)) or after.phase == before.phase
  pure val nexusProductScheduled: NexusProductState = { phase: NexusScheduled }
  pure val nexusProductStarted: NexusProductState = { phase: NexusStarted }
  pure val nexusProductSucceeded: NexusProductState = { phase: NexusSucceeded }
  pure val nexusProductFailed: NexusProductState = { phase: NexusFailed }
  pure val nexusProductCanceled: NexusProductState = { phase: NexusCanceled }
  pure val nexusProductTimedOut: NexusProductState = { phase: NexusTimedOut }

  pure def nexusProductAllows(before: NexusProductState, after: NexusProductState): bool =
    refinementRow(before, after, rowsFrom(before, nexusActionClasses, nexusSteps))

  type ProductPhase = NexusProductPhase
  type ProductState = NexusProductState
  type ProductFact = NexusProductFact
  type ProductStep = NexusProductStep
  type Action = NexusAction
  pure def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] = nexusProductStep(phase, recorded)
  pure def handlerReplyStep(s: ProductState, reply: Reply): List[ProductStep] = nexusHandlerReplyStep(s, reply)
  pure def productTerminal(s: ProductState): bool = nexusProductTerminal(s)
  pure def completeStep(s: ProductState, resolution: Resolution): List[ProductStep] = nexusCompleteStep(s, resolution)
  pure def transportFaultStep(s: ProductState): List[ProductStep] = nexusTransportFaultStep(s)
  pure def workerStopStep(s: ProductState): List[ProductStep] = nexusWorkerStopStep(s)
  pure def timeoutStep(s: ProductState): List[ProductStep] = nexusTimeoutStep(s)
  pure def steps(s: ProductState, a: Action): List[ProductStep] = nexusSteps(s, a)
  pure val phases = nexusPhases
  pure val states = nexusStates
  pure val starts = nexusStarts
  pure val ends = nexusEnds
  pure val timers = nexusTimers
  pure val actionClasses = nexusActionClasses
  pure def evidence(f: ProductFact): str = nexusEvidence(f)
  pure def terminalIsFinalRow(before: ProductState, after: ProductState): bool = nexusTerminalIsFinalRow(before, after)
  pure val productScheduled = nexusProductScheduled
  pure val productStarted = nexusProductStarted
  pure val productSucceeded = nexusProductSucceeded
  pure val productFailed = nexusProductFailed
  pure val productCanceled = nexusProductCanceled
  pure val productTimedOut = nexusProductTimedOut
  pure def productAllows(before: ProductState, after: ProductState): bool = nexusProductAllows(before, after)
  pure val Scheduled = NexusScheduled
  pure val Started = NexusStarted
  pure val Succeeded = NexusSucceeded
  pure val Failed = NexusFailed
  pure val Canceled = NexusCanceled
  pure val TimedOut = NexusTimedOut
  pure val NexusOperationScheduled = NexusNexusOperationScheduled
  pure val NexusOperationStarted = NexusNexusOperationStarted
  pure val NexusOperationCompleted = NexusNexusOperationCompleted
  pure val NexusOperationFailed = NexusNexusOperationFailed
  pure val NexusOperationCanceled = NexusNexusOperationCanceled
  pure val NexusOperationTimedOut = NexusNexusOperationTimedOut
  pure def HandlerReply(value: Reply): Action = NexusHandlerReply(value)
  pure def Complete(value: Resolution): Action = NexusComplete(value)
  pure val TransportFault = NexusTransportFault
  pure val WorkerStop = NexusWorkerStop
  pure val TimeoutFired = NexusTimeoutFired

}
