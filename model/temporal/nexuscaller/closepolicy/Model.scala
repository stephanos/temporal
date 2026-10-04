/* The Nexus caller close and reset designs: one logical operation whose caller closes or is reset
 * while the handler works. Reviewed as model/specimens/nexus.md, whose supported sketch
 * lifter/testdata/lifts/CloseReset.scala is; this Model keeps the sketch's vocabulary and
 * adds what the sketch left to it: the evidence of each step, the cancel request's principal and its
 * delivery to the handler, two faulty resets, a channel that redelivers once, and a schedule-to-close
 * deadline.
 *
 * Every machine here is an authored design. No server has a close policy, an operation-level
 * retention or a reset that reapplies it, and the two faulty policies are deliberate controls, not
 * claims about a known server defect. nexusProduct and nexusProtocol model neither close nor reset,
 * and no design refines them.
 *
 * Three identities stay apart. The logical operation is the entity, named by a request identity
 * that spans runs. A run is the caller's role: the original, closed or not, and its reset successor.
 * A delivery is one report in the completion channel, which may arrive more than once.
 */
package temporal
package nexuscaller
package closepolicy

import umpire.*

/** The family of the designs' machines. */
given Family = Family("temporal.nexus.caller.closepolicy")

/**
 * The logical operation. `operation` is keyed by the scheduled event of one run's history, which
 * cannot follow the operation into a reset successor; the request identity can.
 */
val nexusRequest: Entity = Entity(key = "requestId", refer = Map("owner" -> workflow))

// ### Design vocabulary: one logical operation, the original run and one reset successor

/**
 * The caller's runs. `closed` is the original run closed with its history frozen; `resetOpen` is
 * the successor, which owns the operation from the reset on.
 */
enum Caller derives Finite:
  case open, closed, resetOpen

/**
 * Who asked for a cancellation. One principal is in scope, as one cancellation is: the caller
 * workflow whose command requests it. Which principals a server has is not this Model's claim.
 */
enum Principal derives Finite:
  case callerWorkflow

/** The cancel request the owning run's history records, with its principal. */
enum Intent derives Finite:
  case none
  case requested(by: Principal)

/**
 * What the handler has done. `cancelReceived` is the cancel request delivered to a handler still
 * working, which obliges it to nothing. `done` is an irreversible effect: no reset undoes it.
 */
enum Handler derives Finite:
  case running, cancelReceived
  case done(result: Resolution)

/**
 * The one completion report in the bounded channel from handler to caller. While it is in flight
 * the handler still owes the report. `retried` is the report delivered again by a channel that
 * redelivers once.
 */
enum Completion derives Finite:
  case none
  case inFlight(result: Resolution)
  case retried(result: Resolution)

/**
 * An outcome retained durably at the operation, keyed by the stable operation identity rather than
 * a run, together with the obligation to deliver it to an owner.
 */
enum Retained derives Finite:
  case none
  case pending(result: Resolution)

/**
 * Which run's history records the outcome. `expired` is the schedule-to-close deadline recorded in
 * its place: it belongs to the operation, so a reset keeps it.
 */
enum Knowledge derives Finite:
  case none
  case original(result: Resolution)
  case successor(result: Resolution)
  case expired

final case class CloseResetState(
    caller: Caller,
    intent: Intent,
    handler: Handler,
    channel: Completion,
    retained: Retained,
    known: Knowledge
) derives Finite

/** What a delivery attempt reports back to the handler. */
enum Answer derives Finite:
  case accepted, retained, rejectedTransient, rejectedPermanent

given Accepted[Answer] = Accepted(Answer.accepted)

/**
 * What a step records. The history events carry the baseline's names; the rest is what a design
 * would have to expose for a Run to read it.
 */
enum Fact derives Finite:
  case workflowClosed, workflowReset
  case cancelRequested(by: Principal)
  case cancelReceived
  case handlerFinished(result: Resolution)
  case nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut
  case outcomeRetained, outcomeReapplied, completionDropped

type CloseResetStep = Step[CloseResetState, Answer, Fact]

/** Where a completion goes once the original run can no longer take it. */
enum Policy derives Finite:
  /**
   * Faulty: a closed run rejects the completion permanently, and the handler stops reporting.
   */
  case rejectAfterClose

  /**
   * Faulty: after a reset the original run acknowledges the completion; the successor never learns
   * it.
   */
  case ackByOriginal

  /** Corrected: retain at the operation and route to the current owner. */
  case retainAndRoute

/** What a reset carries into the successor. */
enum Reset derives Finite:
  /**
   * Reapplies the recorded or retained outcome and keeps the cancel request.
   */
  case reapplies

  /**
   * Faulty: rebuilds from before the completion and does not reapply what the original recorded.
   */
  case truncates

  /** Faulty: forgets the cancel request, and with it who made it. */
  case forgetsCancel

/** How often the channel delivers a report whose delivery did not end it. */
enum Redelivery derives Finite:
  /**
   * Until acknowledged or permanently rejected: a transient rejection may repeat forever.
   */
  case untilAck

  /**
   * Once more, whether the first delivery was rejected transiently or its acknowledgment lost.
   */
  case once

val principal = input[Principal]
val result = input[Resolution]

val callerClose = action(caller) on workflow
val reset = action(caller) on workflow
val requestCancel = action(caller).on(operation).input(principal)
val handlerFinish = action(handler).on(operation).input(result)

/** The cancel request reaches the handler. */
val deliverCancel = internal

val opened: CloseResetState = CloseResetState(
  Caller.open,
  Intent.none,
  Handler.running,
  Completion.none,
  Retained.none,
  Knowledge.none
)

// ### Step functions

def closeStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller != Caller.open then disabled
  else accept(s.copy(caller = Caller.closed), Fact.workflowClosed)

/**
 * One cancellation, while the handler has neither finished nor been asked, by a run still open.
 */
def cancelStep(s: CloseResetState, p: Principal): List[CloseResetStep] =
  if s.intent != Intent.none || s.caller == Caller.closed || s.handler != Handler.running then
    disabled
  else accept(s.copy(intent = Intent.requested(p)), Fact.cancelRequested(p))

/**
 * The request in flight reaches a handler still working. A closed run's request is still delivered.
 */
def cancelDeliveryStep(s: CloseResetState): List[CloseResetStep] =
  if s.intent == Intent.none || s.handler != Handler.running then disabled
  else accept(s.copy(handler = Handler.cancelReceived), Fact.cancelReceived)

def working(h: Handler): Boolean = h.in(Handler.running, Handler.cancelReceived)

/**
 * The handler's irreversible effect, and its first report. A canceled result needs the handler to
 * have received the cancel request; having received it, the handler may still succeed or fail.
 */
def finishStep(s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if !working(s.handler) then disabled
  else if r == Resolution.canceled && s.handler != Handler.cancelReceived then disabled
  else
    accept(
      s.copy(handler = Handler.done(r), channel = Completion.inFlight(r)),
      Fact.handlerFinished(r)
    )

/** The history event that records an outcome, by the baseline's names. */
def recorded(r: Resolution): Fact = r match
  case Resolution.succeeded => Fact.nexusOperationCompleted
  case Resolution.failed    => Fact.nexusOperationFailed
  case Resolution.canceled  => Fact.nexusOperationCanceled

/**
 * A history records an outcome once: a second delivery of it records nothing.
 */
def recordedOnce(s: CloseResetState, k: Knowledge, r: Resolution): List[Fact] =
  if s.known == k then Nil else List(recorded(r))

def carries(c: Completion, r: Resolution): Boolean =
  c.in(Completion.inFlight(r), Completion.retried(r))

/**
 * Whether a delivery that does not end the report leaves it to be delivered again.
 */
def redelivers(d: Redelivery, c: Completion, r: Resolution): Boolean = d == Redelivery.untilAck ||
  c == Completion.inFlight(r)

/** The report as the channel holds it for its next delivery. */
def again(d: Redelivery, r: Resolution): Completion =
  if d == Redelivery.untilAck then Completion.inFlight(r) else Completion.retried(r)

// The ways a delivery that leaves the report to be delivered again can go.
val taken = choice
val rejectedForNow = choice
val ackLost = choice
val refused = choice

/**
 * The owner commits the outcome. Transient rejection keeps the report in flight; a lost
 * acknowledgment commits and keeps it in flight too, so the report arrives again.
 */
def committed(
    d: Redelivery,
    s: CloseResetState,
    k: Knowledge,
    r: Resolution
): List[CloseResetStep] =
  if redelivers(d, s.channel, r) then
    choose(
      taken -> accept(s.copy(known = k, channel = Completion.none), recordedOnce(s, k, r)*),
      rejectedForNow -> List(Step(Answer.rejectedTransient, s.copy(channel = again(d, r)))),
      ackLost -> accept(s.copy(known = k, channel = again(d, r)), recordedOnce(s, k, r)*)
        .because("the acknowledgment is lost")
    )
  else accept(s.copy(known = k, channel = Completion.none), recordedOnce(s, k, r)*)

def keptAtOperation(d: Redelivery, s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if redelivers(d, s.channel, r) then
    choose(
      taken -> List(
        Step(
          Answer.retained,
          s.copy(retained = Retained.pending(r), channel = Completion.none),
          List(Fact.outcomeRetained)
        )
      ),
      rejectedForNow -> List(Step(Answer.rejectedTransient, s.copy(channel = again(d, r)))),
      ackLost -> List(
        Step(
          Answer.retained,
          s.copy(retained = Retained.pending(r), channel = again(d, r)),
          List(Fact.outcomeRetained)
        )
      ).because("the acknowledgment is lost")
    )
  else
    List(
      Step(
        Answer.retained,
        s.copy(retained = Retained.pending(r), channel = Completion.none),
        List(Fact.outcomeRetained)
      )
    )

/**
 * A permanent rejection ends the report: the handler stops reporting. A `choose` writes this step
 * out in its alternative, since an alternative is one step written out, not a helper's.
 */
def dropped(s: CloseResetState): List[CloseResetStep] =
  List(
    Step(
      Answer.rejectedPermanent,
      s.copy(channel = Completion.none),
      List(Fact.completionDropped)
    )
  )

def rejectedByClosed(d: Redelivery, s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if redelivers(d, s.channel, r) then
    choose(
      refused -> List(
        Step(
          Answer.rejectedPermanent,
          s.copy(channel = Completion.none),
          List(Fact.completionDropped)
        )
      ),
      rejectedForNow -> List(Step(Answer.rejectedTransient, s.copy(channel = again(d, r))))
    )
  else dropped(s)

def deliverStep(p: Policy, d: Redelivery, s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if !carries(s.channel, r) then disabled
  // The deadline resolved the operation: a completion after it finds nothing to complete.
  else if s.known == Knowledge.expired then dropped(s)
  else
    s.caller match
      case Caller.open   => committed(d, s, Knowledge.original(r), r)
      case Caller.closed =>
        if p == Policy.rejectAfterClose then rejectedByClosed(d, s, r) else keptAtOperation(d, s, r)
      case Caller.resetOpen =>
        if p == Policy.ackByOriginal then
          accept(s.copy(channel = Completion.none)).because("the original run acknowledges it")
        else committed(d, s, Knowledge.successor(r), r)

def fromRetention(x: Retained): Knowledge = x match
  case Retained.pending(r) => Knowledge.successor(r)
  case Retained.none       => Knowledge.none

def reapplied(rule: Reset, s: CloseResetState): Knowledge = s.known match
  case Knowledge.original(r) =>
    if rule == Reset.truncates then fromRetention(s.retained) else Knowledge.successor(r)
  case Knowledge.expired => Knowledge.expired
  case _                 => fromRetention(s.retained)

def carried(rule: Reset, i: Intent): Intent = if rule == Reset.forgetsCancel then Intent.none else i

/**
 * What the successor's history records at its start: the reset, and the outcome it reapplies.
 */
def resetFacts(k: Knowledge): List[Fact] = k match
  case Knowledge.successor(r) => List(Fact.workflowReset, Fact.outcomeReapplied, recorded(r))
  case _                      => List(Fact.workflowReset)

/**
 * Reset builds the successor from a point after the start and before the close. It transfers
 * ownership, reapplies the outcome the original run recorded or the operation retained, and keeps
 * the cancel intent. It cannot undo the handler's effect.
 */
def resetStep(rule: Reset, s: CloseResetState): List[CloseResetStep] =
  if s.caller == Caller.resetOpen then disabled
  else
    accept(
      s.copy(
        caller = Caller.resetOpen,
        intent = carried(rule, s.intent),
        known = reapplied(rule, s),
        retained = Retained.none
      ),
      resetFacts(reapplied(rule, s))*
    )

/**
 * The schedule-to-close deadline resolves an operation whose owner knows no outcome. A closed run's
 * history is frozen, so no deadline fires in it.
 */
def expireStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller == Caller.closed || s.known != Knowledge.none then disabled
  else accept(s.copy(known = Knowledge.expired), Fact.nexusOperationTimedOut)

// ### Promises

/** The run that owns the operation records this outcome. */
def ownerKnows(s: CloseResetState, r: Resolution): Boolean =
  if s.caller == Caller.resetOpen then s.known == Knowledge.successor(r)
  else s.known == Knowledge.original(r)

/**
 * A decided outcome is always somewhere a current or future owner can learn it: its history, the
 * operation's retention, or a report still in flight. An operation its deadline resolved waits for
 * no outcome.
 */
def outcomePreserved(after: CloseResetStep): Boolean = after.state.handler match
  case Handler.done(r) =>
    ownerKnows(after.state, r) ||
    after.state.retained == Retained.pending(r) || carries(after.state.channel, r) ||
    after.state.known == Knowledge.expired
  case _ => true

def keptOrOwed(after: CloseResetStep, r: Resolution): Boolean =
  after.outcome.in(Answer.accepted, Answer.retained) implies
    (after.state.channel != Completion.none || ownerKnows(after.state, r) ||
      after.state.retained == Retained.pending(r))

/**
 * An acknowledgment ends the handler's report only once the owner committed or the operation
 * retained the outcome.
 */
def ackOnlyWhenKept(before: CloseResetState, after: CloseResetStep): Boolean = before.channel match
  case Completion.inFlight(r) => keptOrOwed(after, r)
  case Completion.retried(r)  => keptOrOwed(after, r)
  case Completion.none        => true

def isDone(s: CloseResetState): Boolean = !working(s.handler)

/**
 * Where a path may end: the handler is still working, or its outcome reached the owner, or the
 * operation retained it for a successor a closed run may never get, or the deadline resolved the
 * wait. Any other state with no step is a lost outcome.
 */
def settled(s: CloseResetState): Boolean = s.handler match
  case Handler.done(r) =>
    ownerKnows(s, r) ||
    (s.caller == Caller.closed && s.retained == Retained.pending(r)) || s.known == Knowledge.expired
  case _ => true

// ### Monitors: the outcome stays where an owner can learn it, an acknowledgment keeps it, no run
// records two outcomes, and a cancel request stays its principal's

/**
 * Whether a step lost a decided outcome: no owner knows it, and nothing retains or carries it.
 */
val retainedOutcome = sticky(outcomePreserved)

/**
 * Whether an acknowledgment ended the handler's report before the owner committed the outcome or
 * the operation retained it.
 */
val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)

/** The outcomes the runs' histories have recorded. */
enum Outcomes derives Finite:
  case none
  case one(result: Resolution)
  case several

def withOutcome(seen: Outcomes, r: Resolution): Outcomes = seen match
  case Outcomes.none    => Outcomes.one(r)
  case Outcomes.one(x)  => if x == r then seen else Outcomes.several
  case Outcomes.several => Outcomes.several

def outcomesAfter(seen: Outcomes, k: Knowledge): Outcomes = k match
  case Knowledge.original(r)  => withOutcome(seen, r)
  case Knowledge.successor(r) => withOutcome(seen, r)
  case _                      => seen

val singleOutcome =
  monitor[CloseResetState, Answer, Fact, Outcomes](Outcomes.none)((seen, _, after) =>
    outcomesAfter(seen, after.state.known)
  )(seen => seen == Outcomes.several)

/** Who asked for the cancellation, and whether the history still says so. */
enum Asked derives Finite:
  case nobody
  case by(principal: Principal)
  case lost

def askedBy(i: Intent): Asked = i match
  case Intent.requested(p) => Asked.by(p)
  case Intent.none         => Asked.nobody

def askedAfter(asked: Asked, i: Intent): Asked = asked match
  case Asked.nobody => askedBy(i)
  case Asked.by(p)  => if i == Intent.requested(p) then asked else Asked.lost
  case Asked.lost   => Asked.lost

/**
 * The cancellation principal is lost: a request made is no longer in the owner's history as made.
 */
val cancelPrincipal =
  monitor[CloseResetState, Answer, Fact, Asked](Asked.nobody)((asked, _, after) =>
    askedAfter(asked, after.state.intent)
  )(asked => asked == Asked.lost)

// ### Assumptions
//
// Retention and the two channel variants are assumptions of the machines that have them, so every
// result over such a machine names them. The rest are what a progress claim is conditional on.

/**
 * Retention outlives a crash: no design has a step that loses a retained outcome.
 */
val retentionDurable: Assumption = assume("retentionSurvivesCrash")

/**
 * A transient rejection is not repeated forever. A machine that assumes it has the channel that
 * redelivers once.
 */
val retryFair: Assumption = assume("transientRejectionEventuallyAccepted")

/**
 * The schedule-to-close deadline is set and fires. Only a machine that assumes it has the timer. No
 * claim needs the timer fair: the machines that have it redeliver once, so none has a cycle.
 */
val deadlineExpires: Assumption = assume("scheduleToCloseExpires")

/**
 * The handler reports until acknowledged or permanently rejected, as the channel holds the report.
 */
val reporting: Assumption = assume("handlerReportsUntilAckOrPermanent")

/** A delivery that stays enabled is eventually made. */
val deliveryFair: Assumption = assume("enabledDeliveryAndRecoveryActionsEventuallyRun")
  .fair(complete)

/**
 * The closed run is eventually reset, and the reset reapplies what the operation retained.
 */
val recovery: Assumption = assume("currentOwnerEventuallyRecoversAndReappliesRetainedOutcome")
  .fair(reset)

// ### The designs differ only in the policy, the reset and the channel their steps pass
//
// Each design after the first is derived from one before it, changing the step functions its
// policy, reset or channel changes and the assumptions it adds.

val rejectAfterClose = machine[CloseResetState, Answer, Fact] {
  forEntity(nexusRequest)
  monitors(retainedOutcome, ownerAcknowledgment, singleOutcome, cancelPrincipal)
  starts(opened)
  ends(settled)
  evidence {
    case Fact.cancelRequested(_) => "nexusOperationCancelRequested"
    case Fact.handlerFinished(_) => "handlerFinished"
  }
  steps(
    callerClose ~> closeStep,
    reset ~> (s => resetStep(Reset.reapplies, s)),
    requestCancel ~> cancelStep,
    deliverCancel ~> cancelDeliveryStep,
    handlerFinish ~> finishStep,
    complete ~> ((s, r) => deliverStep(Policy.rejectAfterClose, Redelivery.untilAck, s, r))
  )
}

val ackByOriginal = rejectAfterClose
  .assuming(retentionDurable)
  .rebind(complete ~> ((s, r) => deliverStep(Policy.ackByOriginal, Redelivery.untilAck, s, r)))

val retainAndRoute = rejectAfterClose
  .assuming(retentionDurable)
  .rebind(complete ~> ((s, r) => deliverStep(Policy.retainAndRoute, Redelivery.untilAck, s, r)))

/** The corrected policy with a reset that forgets the cancel request. */
val forgetsCancelOnReset = retainAndRoute.rebind(reset ~> (s => resetStep(Reset.forgetsCancel, s)))

/**
 * The corrected policy with a reset that does not reapply what the original run recorded.
 */
val truncatesOnReset = retainAndRoute.rebind(reset ~> (s => resetStep(Reset.truncates, s)))

/** The corrected design over the channel that redelivers once. */
val retainAndRouteBoundedRetry = retainAndRoute
  .assuming(retryFair)
  .rebind(complete ~> ((s, r) => deliverStep(Policy.retainAndRoute, Redelivery.once, s, r)))

// ### The three policies with a schedule-to-close deadline, over the channel that redelivers once

val rejectAfterCloseWithDeadline = rejectAfterClose
  .assuming(retryFair, deadlineExpires)
  .rebind(complete ~> ((s, r) => deliverStep(Policy.rejectAfterClose, Redelivery.once, s, r)))
  .extend(scheduleToClose ~> expireStep)

val ackByOriginalWithDeadline = ackByOriginal
  .assuming(retryFair, deadlineExpires)
  .rebind(complete ~> ((s, r) => deliverStep(Policy.ackByOriginal, Redelivery.once, s, r)))
  .extend(scheduleToClose ~> expireStep)

val retainAndRouteWithDeadline = retainAndRouteBoundedRetry
  .assuming(deadlineExpires)
  .extend(scheduleToClose ~> expireStep)
