// The Nexus specimen's reviewed supported block (model/specimens/nexus.md), lifted as written,
// with its outcome-retention and acknowledgment promises also stated as passive monitors, as its
// proposed E1 block does, in the framework's `monitor` declaration. Only the package, this header,
// the Monitors section and each design's `monitors` line differ from the reviewed text. The lifter's
// tests lift the three designs' Queries and compare the IR with expected/closereset.json.
package fixture.specimens.closereset

import temporal.nexuscaller.{Resolution, caller, complete, handler, operation, workflow}
import temporal.nexuscaller.given
import umpire.*

val Family: umpire.Family = umpire.Family("temporal.nexus.caller.closereset")

// ### Design vocabulary: one logical operation, the original run and one reset successor

/** The caller's runs. `closed` is the original run closed with its history frozen; `resetOpen` is the
  * successor, which owns the operation from the reset on. */
enum Caller derives Finite:
  case open, closed, resetOpen

/** What the handler has done. `done` is an irreversible effect: no reset undoes it. */
enum Handler derives Finite:
  case running
  case done(result: Resolution)

/** The one completion report in the bounded channel from handler to caller. While it is in flight
  * the handler still owes the report. */
enum Completion derives Finite:
  case none
  case inFlight(result: Resolution)

/** An outcome retained durably at the operation, keyed by the stable operation identity rather than a run. */
enum Retained derives Finite:
  case none
  case pending(result: Resolution)

/** Which run's history records the outcome. */
enum Knowledge derives Finite:
  case none
  case original(result: Resolution)
  case successor(result: Resolution)

final case class CloseResetState(
    caller: Caller,
    cancel: Boolean,
    handler: Handler,
    channel: Completion,
    retained: Retained,
    known: Knowledge,
) derives Finite

/** What a delivery attempt reports back to the handler. */
enum Answer derives Finite:
  case accepted, retained, rejectedTransient, rejectedPermanent

/** A close/reset design records nothing a Run reads yet: its evidence is task 5's to declare. */
type CloseResetStep = Step[CloseResetState, Answer, Nothing]

/** Where a completion goes once the original run can no longer take it. */
enum Policy derives Finite:
  /** Faulty: a closed run rejects the completion permanently, and the handler stops reporting. */
  case rejectAfterClose
  /** Faulty: after a reset the original run acknowledges the completion; the successor never learns it. */
  case ackByOriginal
  /** Corrected: retain at the operation and route to the current owner. */
  case retainAndRoute

val callerClose = action("callerClose", caller) on workflow
val reset = action("reset", caller) on workflow
val requestCancel = action("requestCancel", caller) on operation
val handlerFinish = action("handlerFinish", handler).on(operation).input[Resolution]("result")

val opened: CloseResetState = CloseResetState(Caller.open, false, Handler.running, Completion.none, Retained.none, Knowledge.none)

// ### Step functions

def closeStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller != Caller.open then Nil else List(Step(Answer.accepted, s.copy(caller = Caller.closed)))

def cancelStep(s: CloseResetState): List[CloseResetStep] =
  if s.cancel || s.caller == Caller.closed || s.handler != Handler.running then Nil
  else List(Step(Answer.accepted, s.copy(cancel = true)))

/** The handler's irreversible effect, and its first report. A canceled result needs a cancel intent. */
def finishStep(s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if s.handler != Handler.running then Nil
  else if r == Resolution.canceled && !s.cancel then Nil
  else List(Step(Answer.accepted, s.copy(handler = Handler.done(r), channel = Completion.inFlight(r))))

/** The owner commits the outcome. Transient rejection keeps the report in flight; a lost
  * acknowledgment commits and keeps it in flight too, so the report arrives again. */
def committed(s: CloseResetState, k: Knowledge): List[CloseResetStep] = List(
  Step(Answer.accepted, s.copy(known = k, channel = Completion.none)),
  Step(Answer.rejectedTransient, s),
  Step(Answer.accepted, s.copy(known = k), Nil, "the acknowledgment is lost"),
)

def keptAtOperation(s: CloseResetState, r: Resolution): List[CloseResetStep] = List(
  Step(Answer.retained, s.copy(retained = Retained.pending(r), channel = Completion.none)),
  Step(Answer.rejectedTransient, s),
  Step(Answer.retained, s.copy(retained = Retained.pending(r)), Nil, "the acknowledgment is lost"),
)

def deliverStep(p: Policy, s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if s.channel != Completion.inFlight(r) then Nil
  else s.caller match
    case Caller.open => committed(s, Knowledge.original(r))
    case Caller.closed =>
      if p == Policy.rejectAfterClose then
        List(Step(Answer.rejectedPermanent, s.copy(channel = Completion.none)), Step(Answer.rejectedTransient, s))
      else keptAtOperation(s, r)
    case Caller.resetOpen =>
      if p == Policy.ackByOriginal then
        List(Step(Answer.accepted, s.copy(channel = Completion.none), Nil, "the original run acknowledges it"))
      else committed(s, Knowledge.successor(r))

/** Reset builds the successor from a point after the start and before the close. It transfers
  * ownership, reapplies the outcome the original run recorded or the operation retained, and keeps
  * the cancel intent. It cannot undo the handler's effect. */
def resetStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller == Caller.resetOpen then Nil
  else List(Step(Answer.accepted, s.copy(caller = Caller.resetOpen, known = reapplied(s), retained = Retained.none)))

def reapplied(s: CloseResetState): Knowledge = s.known match
  case Knowledge.original(r) => Knowledge.successor(r)
  case _ =>
    s.retained match
      case Retained.pending(r) => Knowledge.successor(r)
      case Retained.none    => Knowledge.none

/** Where a path may end: the handler is still working, or its outcome reached the owner, or the
  * operation retained it for a successor a closed run may never get. Any other state with no step is
  * a lost outcome. */
def settled(s: CloseResetState): Boolean = s.handler match
  case Handler.running => true
  case Handler.done(r) => ownerKnows(s, r) || (s.caller == Caller.closed && s.retained == Retained.pending(r))

// ### Monitors: the outcome stays where an owner can learn it, and an acknowledgment keeps it

/** Whether a step lost a decided outcome: no owner knows it, and nothing retains or carries it. */
val retainedOutcome: Monitor[CloseResetState, Answer, Nothing, Boolean] =
  monitor[CloseResetState, Answer, Nothing, Boolean]("retainedOutcome", false)((lost, _, after) =>
    lost || !outcomePreserved(after))(lost => lost)

/** Whether an acknowledgment ended the handler's report before the owner committed the outcome or
  * the operation retained it. */
val ownerAcknowledgment: Monitor[CloseResetState, Answer, Nothing, Boolean] =
  monitor[CloseResetState, Answer, Nothing, Boolean]("ownerAcknowledgment", false)((broken, before, after) =>
    broken || !ackOnlyWhenKept(before, after))(broken => broken)

// ### The three designs differ only in the policy their delivery step passes

val rejectAfterCloseDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "rejectAfterClose") {
    forEntity(operation)
    monitors(retainedOutcome, ownerAcknowledgment)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.rejectAfterClose, s, r)))
  }

val ackByOriginalDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "ackByOriginal") {
    forEntity(operation)
    monitors(retainedOutcome, ownerAcknowledgment)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.ackByOriginal, s, r)))
  }

val retainAndRouteDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "retainAndRoute") {
    forEntity(operation)
    monitors(retainedOutcome, ownerAcknowledgment)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.retainAndRoute, s, r)))
  }

// ### Promises

/** The run that owns the operation records this outcome. */
def ownerKnows(s: CloseResetState, r: Resolution): Boolean =
  if s.caller == Caller.resetOpen then s.known == Knowledge.successor(r) else s.known == Knowledge.original(r)

/** A decided outcome is always somewhere a current or future owner can learn it: its history, the
  * operation's retention, or a report still in flight. */
def outcomePreserved(after: CloseResetStep): Boolean = after.state.handler match
  case Handler.running => true
  case Handler.done(r) =>
    ownerKnows(after.state, r) || after.state.retained == Retained.pending(r) || after.state.channel == Completion.inFlight(r)

/** An acknowledgment ends the handler's report only once the owner committed or the operation retained the outcome. */
def ackOnlyWhenKept(before: CloseResetState, after: CloseResetStep): Boolean = before.channel match
  case Completion.inFlight(r) =>
    (after.outcome != Answer.accepted && after.outcome != Answer.retained) || after.state.channel != Completion.none ||
      ownerKnows(after.state, r) || after.state.retained == Retained.pending(r)
  case Completion.none => true

val four: Limits = Limits("four", steps = 4, actions = 4, search = 1 << 20)
val six: Limits = Limits("six", steps = 6, actions = 6, search = 1 << 22)

def closeResetQueries(m: Machine[CloseResetState, Answer, Nothing]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val closedThenFinished = m.scenario("closedThenFinished").starts(opened)
    .actions(callerClose, handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val resetThenDelivered = m.scenario("resetThenDelivered").starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val canceledAcrossReset = m.scenario("canceledAcrossReset").starts(opened)
    .actions(requestCancel, handlerFinish(Resolution.canceled), reset, complete(Resolution.canceled))
  val ackedThenReset = m.scenario("ackedThenReset").starts(opened)
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val any = m.scenario("any").starts(opened).free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in closedThenFinished limits four,
    query(s"${m.name}.resetThenDelivered.ackOnlyWhenKept") verify acked in resetThenDelivered limits four,
    query(s"${m.name}.resetThenDelivered.outcomePreserved") verify preserved in resetThenDelivered limits four,
    query(s"${m.name}.canceledAcrossReset") verify preserved in canceledAcrossReset limits four,
    query(s"${m.name}.ackedThenReset") verify preserved in ackedThenReset limits four,
    query(s"${m.name}.any.outcomePreserved") verify preserved in any limits six,
    query(s"${m.name}.any.ackOnlyWhenKept") verify acked in any limits six,
  )

val rejectAfterCloseQueries: Vector[Query] = closeResetQueries(rejectAfterCloseDesign)
val ackByOriginalQueries: Vector[Query] = closeResetQueries(ackByOriginalDesign)
val retainAndRouteQueries: Vector[Query] = closeResetQueries(retainAndRouteDesign)
