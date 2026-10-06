/* The Nexus caller close and reset designs: one logical operation whose caller closes or is reset
 * while the handler works. Reviewed as model/specimens/nexus.md, whose supported sketch
 * irgen/testdata/lifts/CloseReset.scala was; this Model keeps the sketch's vocabulary and
 * adds what the sketch left to it: the evidence of each step, the cancel request's principal and its
 * delivery to the handler, two faulty resets, a channel that redelivers once, and a schedule-to-close
 * deadline.
 *
 * Every machine here is an authored design. No server has a close policy, an operation-level
 * retention or a reset that reapplies it, and the two faulty policies are deliberate controls, not
 * claims about a known server defect. nexusProduct and nexusSystem model neither close nor reset,
 * and no design refines them.
 *
 * Three identities stay apart. The logical operation is the entity, named by a request identity
 * that spans runs. A run is the caller's role: the original, closed or not, and its reset successor.
 * A delivery is one report in the completion channel, which may arrive more than once.
 *
 * Read top to bottom: the types; the signature (the logical operation, the caller's and the
 * handler's actions, the bounds, and the assumptions the designs add or their progress claims are
 * conditional on); then one object per design -- RejectAfterClose, the first, whose status sets,
 * effects, monitors and declaring functions every design reads, and the eight designs derived from
 * it, each with its own progress claim and Queries. A subject of the System level, beside
 * system/System.scala (fn-126 decisions 16 and 22); its IR file is among the feature's exports, in
 * NexusCaller.scala. The faulty policies and resets are deliberately wrong designs, marked as
 * negative controls.
 */
package temporal
package features.nexuscaller
package system

// `Answer` here is the designs' delivery answer.
import umpire.*
import Answer.given

// ### Types: one logical operation, the original run and one reset successor

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

object Answer:
  // Kept in the companion, and imported by this file alone: the System level's other subjects, in
  // this package, answer the feature's Outcome.
  given Ok[Answer] = Ok(Answer.accepted)

/**
 * What a step records. The history events carry the baseline's names; the rest is what a design
 * would have to expose for a Run to read it.
 */
enum CloseFact derives Finite:
  case workflowClosed, workflowReset
  case cancelRequested(by: Principal)
  case cancelReceived
  case handlerFinished(result: Resolution)
  case nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut
  case outcomeRetained, outcomeReapplied, completionDropped

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

/** The outcomes the runs' histories have recorded. */
enum Outcomes derives Finite:
  case none
  case one(result: Resolution)
  case several

/** Who asked for the cancellation, and whether the history still says so. */
enum Asked derives Finite:
  case nobody
  case by(principal: Principal)
  case lost

// The claims each design is held to. Each set is declared on the design `m` its Queries ask, since
// a Property belongs to one machine.

/** Every claim of the specimen. */
final case class DesignClaims(
    outcomePreserved: Property[CloseResetState],
    ackOnlyWhenKept: Property[CloseResetState],
    closedHistoryIsFrozen: Property[CloseResetState],
    handlerEffectIsIrreversible: Property[CloseResetState],
    knownIsTheHandlersOutcome: Property[CloseResetState],
    knowledgeIsFinal: Property[CloseResetState],
    finishesAfterClose: Property[CloseResetState],
    requestedButUnreceived: Property[CloseResetState],
    receivedButSucceeded: Property[CloseResetState],
    canceledButUnknown: Property[CloseResetState],
    completionCancels: Property[CloseResetState],
    completionSucceeds: Property[CloseResetState],
    completionFails: Property[CloseResetState],
    rejectedTransiently: Property[CloseResetState],
    rejectedPermanently: Property[CloseResetState],
    lostAfterReset: Property[CloseResetState],
    reappliesRetained: Property[CloseResetState],
    routedToSuccessor: Property[CloseResetState]
)

/** The two promises every design keeps. */
final case class SafetyClaims(
    outcomePreserved: Property[CloseResetState],
    ackOnlyWhenKept: Property[CloseResetState]
)

/** The two promises and the claims of the schedule-to-close deadline. */
final case class DeadlineClaims(
    outcomePreserved: Property[CloseResetState],
    ackOnlyWhenKept: Property[CloseResetState],
    noUnnecessaryWait: Property[CloseResetState],
    expiresWithNothingOwed: Property[CloseResetState],
    expiresWhileOwed: Property[CloseResetState],
    lateCompletionIsDropped: Property[CloseResetState]
)

// ### Signature

/**
 * The logical operation. `operation` is keyed by the scheduled event of one run's history, which
 * cannot follow the operation into a reset successor; the request identity can.
 */
val nexusRequest = Entity(key = "requestId", refer = Map("owner" -> workflow))

val result = input[Resolution]

/**
 * The cancel request's input, in an object of its own: the caller's actions read it while they
 * initialize, and the assumption that makes the reset fair reads the caller's actions while the file
 * initializes.
 */
object Inputs:
  val principal = input[Principal]

// Who acts in the designs, grouped by side: the caller's and the handler's own actions here are
// taken by the actors the Nexus caller declares, whose actions the designs read too
// (`handler.complete`). The objects are not named `caller` and `handler`, which would name the
// same class files as the types `Caller` and `Handler` on a case-insensitive file system.

/** What the designs' caller workflow does: it closes, is reset, and requests a cancel. */
object callerSide:
  val close = action(caller) on workflow
  val reset = action(caller) on workflow
  val requestCancel = action(caller).on(operation).input(Inputs.principal)

/** The handler finishes the operation's work. */
object handlerSide:
  val finish = action(handler).on(operation).input(result)

/** The server's own step: the cancel request reaches the handler. */
object history:
  val deliverCancel = internal

// The bounds. They search further than the caller's of the same names, `four` among them.

val four = Limits(steps = 4, actions = 4, search = 1 << 20)
val five = Limits(steps = 5, actions = 5, search = 1 << 20)

/**
 * Past the depth of every design's table, so a free search that verifies read every step.
 */
val twelve = Limits(steps = 12, actions = 12, search = 1 << 22)

// Assumptions no design makes of its own. Retention and the two channel variants are the
// assumptions of the designs that have them, which add them with `assuming`, so every result over
// such a design names them. The rest are what a progress claim is conditional on, named `under` it.

/**
 * Retention outlives a crash: no design has a step that loses a retained outcome.
 */
val retentionDurable = assume("retentionSurvivesCrash")

/**
 * A transient rejection is not repeated forever. A machine that assumes it has the channel that
 * redelivers once.
 */
val retryFair = assume("transientRejectionEventuallyAccepted")

/**
 * The schedule-to-close deadline is set and fires. Only a machine that assumes it has the timer.
 * No claim needs the timer fair: the machines that have it redeliver once, so none has a cycle.
 */
val deadlineExpires = assume("scheduleToCloseExpires")

/**
 * The handler reports until acknowledged or permanently rejected, as the channel holds the report.
 */
val reporting = assume("handlerReportsUntilAckOrPermanent")

/** A delivery that stays enabled is eventually made. */
val deliveryFair = assume("enabledDeliveryAndRecoveryActionsEventuallyRun")
  .fair(handler.complete)

/**
 * The closed run is eventually reset, and the reset reapplies what the operation retained.
 */
val recovery = assume("currentOwnerEventuallyRecoversAndReappliesRetainedOutcome")
  .fair(callerSide.reset)

// ### The designs differ only in the policy, the reset and the channel their rules pass
//
// RejectAfterClose is the first design: its status sets, effects, monitors and declaring functions
// are every design's. Each design after it is derived from one before it, rebinding the effect its
// policy, reset or channel changes and adding the assumptions it makes.

/**
 * The faulty policy: a closed run rejects a completion permanently. A deliberately wrong design, kept
 * so that the checks that must refute it are seen to.
 */
object RejectAfterClose extends Machine[CloseResetState, Answer, CloseFact], NegativeControl:
  val entity = nexusRequest
  val init = CloseResetState(
    caller = Caller.open,
    intent = Intent.none,
    handler = Handler.running,
    channel = Completion.none,
    retained = Retained.none,
    known = Knowledge.none
  )
  def end(s: State) = states.settled(s)
  val evidence: PartialFunction[CloseFact, String] = {
    case CloseFact.cancelRequested(_) => "nexusOperationCancelRequested"
    case CloseFact.handlerFinished(_) => "handlerFinished"
  }

  /** What the designs read of a state: the handler's work, the channel, the owner and the promises. */
  object states:
    def working(h: Handler) = h.in(Handler.running, Handler.cancelReceived)

    /**
     * A cancel request may be made once, while the handler has neither finished nor been asked, by a
     * run still open.
     */
    def cancellable(s: State) =
      s.intent == Intent.none && s.caller.in(Caller.open, Caller.resetOpen) &&
        s.handler == Handler.running

    /**
     * A cancel request is in flight to a handler still working. A closed run's request is still
     * delivered.
     */
    def cancelInFlight(s: State) = s.intent != Intent.none && s.handler == Handler.running

    /**
     * The schedule-to-close deadline can fire: the owner knows no outcome, and the run is not closed,
     * whose history is frozen.
     */
    def expirable(s: State) =
      s.caller.in(Caller.open, Caller.resetOpen) && s.known == Knowledge.none

    /** The history event that records an outcome, by the baseline's names. */
    def recorded(r: Resolution) = r match
      case Resolution.succeeded => CloseFact.nexusOperationCompleted
      case Resolution.failed    => CloseFact.nexusOperationFailed
      case Resolution.canceled  => CloseFact.nexusOperationCanceled

    /**
     * A history records an outcome once: a second delivery of it records nothing.
     */
    def recordedOnce(s: State, k: Knowledge, r: Resolution) =
      if s.known == k then Nil else List(recorded(r))

    def carries(c: Completion, r: Resolution) =
      c.in(Completion.inFlight(r), Completion.retried(r))

    /**
     * Whether a delivery that does not end the report leaves it to be delivered again.
     */
    def redelivers(d: Redelivery, c: Completion, r: Resolution) = d == Redelivery.untilAck ||
      c == Completion.inFlight(r)

    /** The report as the channel holds it for its next delivery. */
    def again(d: Redelivery, r: Resolution) =
      if d == Redelivery.untilAck then Completion.inFlight(r) else Completion.retried(r)

    // The ways a delivery that leaves the report to be delivered again can go.
    val taken = choice
    val rejectedForNow = choice
    val ackLost = choice
    val refused = choice

    def fromRetention(x: Retained) = x match
      case Retained.pending(r) => Knowledge.successor(r)
      case Retained.none       => Knowledge.none

    def reapplied(rule: Reset, s: State) = s.known match
      case Knowledge.original(r) =>
        if rule == Reset.truncates then fromRetention(s.retained) else Knowledge.successor(r)
      case Knowledge.expired => Knowledge.expired
      case _                 => fromRetention(s.retained)

    def carried(rule: Reset, i: Intent) = if rule == Reset.forgetsCancel then Intent.none else i

    /**
     * What the successor's history records at its start: the reset, and the outcome it reapplies.
     */
    def resetFacts(k: Knowledge) = k match
      case Knowledge.successor(r) =>
        List(CloseFact.workflowReset, CloseFact.outcomeReapplied, recorded(r))
      case _ => List(CloseFact.workflowReset)

    // Promises: the outcome stays where an owner can learn it, an acknowledgment keeps it, and a
    // path ends only where the outcome is settled.

    /** The run that owns the operation records this outcome. */
    def ownerKnows(s: State, r: Resolution) =
      if s.caller == Caller.resetOpen then s.known == Knowledge.successor(r)
      else s.known == Knowledge.original(r)

    /**
     * A decided outcome is always somewhere a current or future owner can learn it: its history,
     * the operation's retention, or a report still in flight. An operation its deadline resolved
     * waits for no outcome.
     */
    def outcomePreserved(after: Step[CloseResetState, Answer, CloseFact]) =
      after.state.handler match
        case Handler.done(r) =>
          ownerKnows(after.state, r) ||
          after.state.retained == Retained.pending(r) || carries(after.state.channel, r) ||
          after.state.known == Knowledge.expired
        case _ => true

    def keptOrOwed(after: Step[CloseResetState, Answer, CloseFact], r: Resolution) =
      after.outcome.in(Answer.accepted, Answer.retained) implies
        (after.state.channel != Completion.none || ownerKnows(after.state, r) ||
          after.state.retained == Retained.pending(r))

    /**
     * An acknowledgment ends the handler's report only once the owner committed or the operation
     * retained the outcome.
     */
    def ackOnlyWhenKept(before: State, after: Step[CloseResetState, Answer, CloseFact]) =
      before.channel match
        case Completion.inFlight(r) => keptOrOwed(after, r)
        case Completion.retried(r)  => keptOrOwed(after, r)
        case Completion.none        => true

    def isDone(s: State) = !working(s.handler)

    /**
     * Where a path may end: the handler is still working, or its outcome reached the owner, or the
     * operation retained it for a successor a closed run may never get, or the deadline resolved the
     * wait. Any other state with no step is a lost outcome. `end` reads this named predicate.
     */
    def settled(s: State) = s.handler match
      case Handler.done(r) =>
        ownerKnows(s, r) ||
        (s.caller == Caller.closed && s.retained == Retained.pending(
          r
        )) || s.known == Knowledge.expired
      case _ => true

    def withOutcome(seen: Outcomes, r: Resolution) = seen match
      case Outcomes.none    => Outcomes.one(r)
      case Outcomes.one(x)  => if x == r then seen else Outcomes.several
      case Outcomes.several => Outcomes.several

    def outcomesAfter(seen: Outcomes, k: Knowledge) = k match
      case Knowledge.original(r)  => withOutcome(seen, r)
      case Knowledge.successor(r) => withOutcome(seen, r)
      case _                      => seen

    def askedBy(i: Intent) = i match
      case Intent.requested(p) => Asked.by(p)
      case Intent.none         => Asked.nobody

    def askedAfter(asked: Asked, i: Intent) = asked match
      case Asked.nobody => askedBy(i)
      case Asked.by(p)  => if i == Intent.requested(p) then asked else Asked.lost
      case Asked.lost   => Asked.lost

  /** What each step does. Where it fires is the rules'. */
  object effects:
    def close(s: State) = enter(s.copy(caller = Caller.closed), CloseFact.workflowClosed)

    /** One cancellation, by the principal that asks for it. */
    def requestCancel(s: State, p: Principal) =
      enter(s.copy(intent = Intent.requested(p)), CloseFact.cancelRequested(p))

    /** The request in flight reaches the handler, which it obliges to nothing. */
    def deliverCancel(s: State) =
      enter(s.copy(handler = Handler.cancelReceived), CloseFact.cancelReceived)

    /** The handler's irreversible effect, and its first report. */
    def finish(s: State, r: Resolution) =
      enter(
        s.copy(handler = Handler.done(r), channel = Completion.inFlight(r)),
        CloseFact.handlerFinished(r)
      )

    /**
     * The owner commits the outcome. Transient rejection keeps the report in flight; a lost
     * acknowledgment commits and keeps it in flight too, so the report arrives again.
     */
    def committed(
        d: Redelivery,
        s: State,
        k: Knowledge,
        r: Resolution
    ) =
      if states.redelivers(d, s.channel, r) then
        choose(
          states.taken -> enter(
            s.copy(known = k, channel = Completion.none),
            states.recordedOnce(s, k, r)*
          ),
          states.rejectedForNow -> List(
            Step(Answer.rejectedTransient, s.copy(channel = states.again(d, r)))
          ),
          states.ackLost -> enter(
            s.copy(known = k, channel = states.again(d, r)),
            states.recordedOnce(s, k, r)*
          )
            .because("the acknowledgment is lost")
        )
      else enter(s.copy(known = k, channel = Completion.none), states.recordedOnce(s, k, r)*)

    def keptAtOperation(d: Redelivery, s: State, r: Resolution) =
      if states.redelivers(d, s.channel, r) then
        choose(
          states.taken -> List(
            Step(
              Answer.retained,
              s.copy(retained = Retained.pending(r), channel = Completion.none),
              List(CloseFact.outcomeRetained)
            )
          ),
          states.rejectedForNow -> List(
            Step(Answer.rejectedTransient, s.copy(channel = states.again(d, r)))
          ),
          states.ackLost -> List(
            Step(
              Answer.retained,
              s.copy(retained = Retained.pending(r), channel = states.again(d, r)),
              List(CloseFact.outcomeRetained)
            )
          ).because("the acknowledgment is lost")
        )
      else
        List(
          Step(
            Answer.retained,
            s.copy(retained = Retained.pending(r), channel = Completion.none),
            List(CloseFact.outcomeRetained)
          )
        )

    /** A permanent rejection ends the report: the handler stops reporting. */
    def dropped(s: State) =
      List(
        Step(
          Answer.rejectedPermanent,
          s.copy(channel = Completion.none),
          List(CloseFact.completionDropped)
        )
      )

    def rejectedByClosed(d: Redelivery, s: State, r: Resolution) =
      if states.redelivers(d, s.channel, r) then
        choose(
          states.refused -> dropped(s),
          states.rejectedForNow -> List(
            Step(Answer.rejectedTransient, s.copy(channel = states.again(d, r)))
          )
        )
      else dropped(s)

    /**
     * A delivery of the report the channel carries, `r`, as the policy and the channel take it.
     */
    def deliver(p: Policy, d: Redelivery, s: State, r: Resolution) =
      // The deadline resolved the operation: a completion after it finds nothing to complete.
      if s.known == Knowledge.expired then dropped(s)
      else
        s.caller match
          case Caller.open   => committed(d, s, Knowledge.original(r), r)
          case Caller.closed =>
            if p == Policy.rejectAfterClose then rejectedByClosed(d, s, r)
            else keptAtOperation(d, s, r)
          case Caller.resetOpen =>
            if p == Policy.ackByOriginal then
              enter(s.copy(channel = Completion.none)).because("the original run acknowledges it")
            else committed(d, s, Knowledge.successor(r), r)

    /**
     * Reset builds the successor from a point after the start and before the close. It transfers
     * ownership, reapplies the outcome the original run recorded or the operation retained, and
     * keeps the cancel intent. It cannot undo the handler's effect.
     */
    def reset(rule: Reset, s: State) =
      enter(
        s.copy(
          caller = Caller.resetOpen,
          intent = states.carried(rule, s.intent),
          known = states.reapplied(rule, s),
          retained = Retained.none
        ),
        states.resetFacts(states.reapplied(rule, s))*
      )

    /** The schedule-to-close deadline resolves an operation whose owner knows no outcome. */
    def expire(s: State) =
      enter(s.copy(known = Knowledge.expired), CloseFact.nexusOperationTimedOut)

  // Monitors: the outcome stays where an owner can learn it, an acknowledgment keeps it, no run
  // records two outcomes, and a cancel request stays its principal's. Every design watches them.
  object monitors:
    /**
     * Whether a step lost a decided outcome: no owner knows it, and nothing retains or carries it.
     */
    val retainedOutcome = sticky(states.outcomePreserved)

    /**
     * Whether an acknowledgment ended the handler's report before the owner committed the outcome or
     * the operation retained it.
     */
    val ownerAcknowledgment = stickyAcross(states.ackOnlyWhenKept)

    val singleOutcome =
      monitor[CloseResetState, Answer, CloseFact, Outcomes](Outcomes.none)((seen, _, after) =>
        states.outcomesAfter(seen, after.state.known)
      )(seen => seen == Outcomes.several)

    /**
     * The cancellation principal is lost: a request made is no longer in the owner's history as
     * made.
     */
    val cancelPrincipal =
      monitor[CloseResetState, Answer, CloseFact, Asked](Asked.nobody)((asked, _, after) =>
        states.askedAfter(asked, after.state.intent)
      )(asked => asked == Asked.lost)

  object rules extends Rules(_.caller):
    on(callerSide.close)(in(Caller.open) ~> effects.close)
    on(callerSide.reset)(in(Caller.open, Caller.closed) ~> (effects.reset(Reset.reapplies, _)))
    on(callerSide.requestCancel)(where(states.cancellable) ~> effects.requestCancel)
    on(history.deliverCancel)(where(states.cancelInFlight) ~> effects.deliverCancel)

    // A canceled result needs the handler to have received the cancel request; having received it,
    // the handler may still succeed or fail.
    on(handlerSide.finish(Resolution.succeeded)) {
      where(_.handler == Handler.running) ~> (effects.finish(_, Resolution.succeeded))
    }
    on(handlerSide.finish(Resolution.failed)) {
      where(_.handler == Handler.running) ~> (effects.finish(_, Resolution.failed))
    }
    on(handlerSide.finish)(where(_.handler == Handler.cancelReceived) ~> effects.finish)

    // A completion is delivered while the channel carries its result.
    on(handler.complete(Resolution.succeeded)) {
      where(s => states.carries(s.channel, Resolution.succeeded)) ~> (effects.deliver(
        Policy.rejectAfterClose,
        Redelivery.untilAck,
        _,
        Resolution.succeeded
      ))
    }
    on(handler.complete(Resolution.failed)) {
      where(s => states.carries(s.channel, Resolution.failed)) ~> (effects.deliver(
        Policy.rejectAfterClose,
        Redelivery.untilAck,
        _,
        Resolution.failed
      ))
    }
    on(handler.complete(Resolution.canceled)) {
      where(s => states.carries(s.channel, Resolution.canceled)) ~> (effects.deliver(
        Policy.rejectAfterClose,
        Redelivery.untilAck,
        _,
        Resolution.canceled
      ))
    }

  // What the designs promise. Every promise below is an authored design promise, written once and
  // declared as a Property on each design by the claims below. The monitors' promises are the
  // vocabulary's, read by the monitors. Two of the Properties are the baseline's,
  // completionSucceeds and completionFails, which an open caller keeps as nexusSystem does. A
  // design names its monitors, and a verify also reports the first violation a watching monitor
  // meets, whatever Property it asks.
  object properties:
    /**
     * A closed run's history is frozen: no step of a caller that stays closed changes what it holds.
     */
    def closedHistoryIsFrozen(
        before: CloseResetState,
        after: Step[CloseResetState, Answer, CloseFact]
    ) =
      before.caller == Caller.closed && after.state.caller == Caller.closed implies
        (after.state.known == before.known && after.state.intent == before.intent)

    /** An outcome a history records is the handler's. */
    def knownIsTheHandlersOutcome(after: Step[CloseResetState, Answer, CloseFact]) =
      after.state.known match
        case Knowledge.original(r)  => after.state.handler == Handler.done(r)
        case Knowledge.successor(r) => after.state.handler == Handler.done(r)
        case _                      => true

    def knows(k: Knowledge, r: Resolution) =
      k.in(Knowledge.original(r), Knowledge.successor(r))

    /**
     * A recorded outcome stays recorded: a redelivery changes nothing, and a reset carries it over.
     */
    def knowledgeIsFinal(before: CloseResetState, after: Step[CloseResetState, Answer, CloseFact]) =
      before.known match
        case Knowledge.original(r)  => knows(after.state.known, r)
        case Knowledge.successor(r) => knows(after.state.known, r)
        case _                      => true

    /** The handler's outcome is decided, and nothing holds or still carries it. */
    def nothingOwed(s: CloseResetState) = s.handler match
      case Handler.done(_) => s.channel == Completion.none && s.retained == Retained.none
      case _               => false

    /**
     * The deadline resolves only a wait that could still end otherwise: the handler is working, or
     * its report is still owed. A deadline that fires with nothing owed ends a wait for an outcome
     * the design already lost.
     */
    def noUnnecessaryWait(
        before: CloseResetState,
        after: Step[CloseResetState, Answer, CloseFact]
    ) =
      before.known != Knowledge.expired && after.state.known == Knowledge.expired implies
        !nothingOwed(before)

    def designClaims(m: Machine[CloseResetState, Answer, CloseFact]) =
      // No step, a reset included, undoes what the handler did.
      val handlerEffectIsIrreversible = m.property.once(states.isDone).keeps(_.handler)
      // The handler's detached work goes on after the close.
      val finishesAfterClose = m.property when
        handlerSide.finish(Resolution.succeeded) holds
        (after =>
          after.state.caller == Caller.closed &&
            after.state.handler == Handler.done(Resolution.succeeded)
        )
      // A cancellation's request, its receipt, the handler's effect and the caller's knowledge.
      val requestedButUnreceived = m.property when
        callerSide.requestCancel(Principal.callerWorkflow) holds
        (after =>
          after.state.intent == Intent.requested(Principal.callerWorkflow) &&
            after.state.handler == Handler.running
        )
      val receivedButSucceeded = m.property when
        handlerSide.finish(Resolution.succeeded) holds
        (after =>
          after.state.intent == Intent.requested(Principal.callerWorkflow) &&
            after.state.handler == Handler.done(Resolution.succeeded)
        )
      val canceledButUnknown = m.property when
        handlerSide.finish(Resolution.canceled) holds
        (after =>
          after.state.handler == Handler.done(Resolution.canceled) &&
            !states.ownerKnows(after.state, Resolution.canceled)
        )
      val completionCancels = m.property when handler.complete(Resolution.canceled) holds
        (after =>
          states.ownerKnows(after.state, Resolution.canceled) &&
            after.records(CloseFact.nexusOperationCanceled)
        )
      // The baseline's two: an open caller records a completion by the baseline's event.
      val completionSucceeds = m.property when
        handler.complete(Resolution.succeeded) holds
        (after => after.records(CloseFact.nexusOperationCompleted))
      val completionFails = m.property when handler.complete(Resolution.failed) holds
        (after => after.records(CloseFact.nexusOperationFailed))
      // The two answers of a closed run, and the two resets, are kept apart.
      val rejectedTransiently = m.property when
        handler.complete(Resolution.succeeded) holds
        (after =>
          after.outcome == Answer.rejectedTransient &&
            states.carries(after.state.channel, Resolution.succeeded)
        )
      val rejectedPermanently = m.property when
        handler.complete(Resolution.succeeded) holds (after =>
          after.outcome == Answer.rejectedPermanent
        )
      val lostAfterReset = m.property when callerSide.reset holds
        (after => !states.outcomePreserved(after))
      val reappliesRetained = m.property when callerSide.reset holds
        (after =>
          after.records(CloseFact.outcomeReapplied) &&
            after.state.known == Knowledge.successor(Resolution.succeeded)
        )
      val routedToSuccessor = m.property when handler.complete(Resolution.failed) holds
        (after => after.state.known == Knowledge.successor(Resolution.failed))
      DesignClaims(
        m.property("outcomePreserved") holds states.outcomePreserved,
        m.property("ackOnlyWhenKept") holdsAcross states.ackOnlyWhenKept,
        m.property("closedHistoryIsFrozen") holdsAcross closedHistoryIsFrozen,
        handlerEffectIsIrreversible,
        m.property("knownIsTheHandlersOutcome") holds knownIsTheHandlersOutcome,
        m.property("knowledgeIsFinal") holdsAcross knowledgeIsFinal,
        finishesAfterClose,
        requestedButUnreceived,
        receivedButSucceeded,
        canceledButUnknown,
        completionCancels,
        completionSucceeds,
        completionFails,
        rejectedTransiently,
        rejectedPermanently,
        lostAfterReset,
        reappliesRetained,
        routedToSuccessor
      )

    def safetyClaims(m: Machine[CloseResetState, Answer, CloseFact]) = SafetyClaims(
      m.property("outcomePreserved") holds states.outcomePreserved,
      m.property("ackOnlyWhenKept") holdsAcross states.ackOnlyWhenKept
    )

    def deadlineClaims(m: Machine[CloseResetState, Answer, CloseFact]) =
      val expiresWithNothingOwed = m.property when deadline.scheduleToClose holds
        (after => after.state.known == Knowledge.expired && nothingOwed(after.state))
      val expiresWhileOwed = m.property when deadline.scheduleToClose holds
        (after =>
          after.state.known == Knowledge.expired &&
            states.carries(after.state.channel, Resolution.succeeded)
        )
      val lateCompletionIsDropped = m.property when
        handler.complete(Resolution.succeeded) holds
        (after =>
          after.outcome == Answer.rejectedPermanent && after.records(CloseFact.completionDropped)
        )
      DeadlineClaims(
        m.property("outcomePreserved") holds states.outcomePreserved,
        m.property("ackOnlyWhenKept") holdsAcross states.ackOnlyWhenKept,
        m.property("noUnnecessaryWait") holdsAcross noUnnecessaryWait,
        expiresWithNothingOwed,
        expiresWhileOwed,
        lateCompletionIsDropped
      )

    /**
     * The operation retained the outcome for a closed run, and no owner knows it yet.
     */
    def awaitingOwner(s: CloseResetState) = s.handler match
      case Handler.done(r) =>
        s.caller == Caller.closed && s.retained == Retained.pending(r) &&
        !states.ownerKnows(s, r)
      case _ => false

    def ownerKnowsOutcome(s: CloseResetState) = s.handler match
      case Handler.done(r) => states.ownerKnows(s, r)
      case _               => false

    // Conditional progress. From a decided outcome, a state a path may end in follows within six
    // steps, under the assumptions each claim and its machine name. A machine that names no
    // deadline has no timer, and one that does not assume the redelivery bound may be rejected
    // transiently forever. Each design makes the claim of its own.

    val rejectAfterCloseProgress = RejectAfterClose.leadsTo(
      "outcomeReachesOwner"
    )(states.isDone, states.settled, within = 6, reporting, deliveryFair)

  object queries:
    /** Every claim and path of the specimen, declared on one design. */
    def designQueries(m: Machine[CloseResetState, Answer, CloseFact]) =
      val claims = properties.designClaims(m)
      val closedThenFinished = m.scenario
        .actions(
          callerSide.close,
          handlerSide.finish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset
        )
      val resetThenDelivered = m.scenario
        .actions(
          handlerSide.finish(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed)
        )
      val deliveredToClosed = m.scenario
        .actions(
          callerSide.close,
          handlerSide.finish(Resolution.succeeded),
          handler.complete(Resolution.succeeded)
        )
      val canceledAcrossReset = m.scenario.actions(
        callerSide.requestCancel(Principal.callerWorkflow),
        history.deliverCancel,
        handlerSide.finish(Resolution.canceled),
        callerSide.reset,
        handler.complete(Resolution.canceled)
      )
      val ackedThenReset = m.scenario.actions(
        handlerSide.finish(Resolution.succeeded),
        handler.complete(Resolution.succeeded),
        callerSide.reset
      )
      val duplicateCompletion = m.scenario.actions(
        handlerSide.finish(Resolution.succeeded),
        handler.complete(Resolution.succeeded),
        handler.complete(Resolution.succeeded)
      )
      val resetBetweenDeliveries = m.scenario.actions(
        handlerSide.finish(Resolution.failed),
        handler.complete(Resolution.failed),
        callerSide.reset,
        handler.complete(Resolution.failed)
      )
      val detachedWork = m.scenario.actions(
        callerSide.close,
        handlerSide.finish(Resolution.succeeded)
      )
      val cancelRequested =
        m.scenario.actions(callerSide.requestCancel(Principal.callerWorkflow))
      val cancelReceivedThenSucceeded = m.scenario.actions(
        callerSide.requestCancel(Principal.callerWorkflow),
        history.deliverCancel,
        handlerSide.finish(Resolution.succeeded)
      )
      val cancelReceivedThenCanceled = m.scenario.actions(
        callerSide.requestCancel(Principal.callerWorkflow),
        history.deliverCancel,
        handlerSide.finish(Resolution.canceled)
      )
      val canceledThenDelivered = m.scenario.actions(
        callerSide.requestCancel(Principal.callerWorkflow),
        history.deliverCancel,
        handlerSide.finish(Resolution.canceled),
        handler.complete(Resolution.canceled)
      )
      val finishedThenSucceeded =
        m.scenario.actions(
          handlerSide.finish(Resolution.succeeded),
          handler.complete(Resolution.succeeded)
        )
      val finishedThenFailed =
        m.scenario.actions(
          handlerSide.finish(Resolution.failed),
          handler.complete(Resolution.failed)
        )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four,
        query verify claims.outcomePreserved in resetThenDelivered limits four,
        // The request, its delivery to the handler and the handler's effect are three steps here.
        query(s"${m.name}.canceledAcrossReset") verify claims.outcomePreserved in
          canceledAcrossReset limits five,
        query(s"${m.name}.ackedThenReset") verify claims.outcomePreserved in
          ackedThenReset limits four,
        query(s"${m.name}.duplicateCompletion") verify claims.knowledgeIsFinal in
          duplicateCompletion limits four,
        // A reset between the commit and its acknowledgment, and between a rejection and the retry.
        query(s"${m.name}.resetBetweenCommitAndAcknowledgment") verify claims.ackOnlyWhenKept in
          resetBetweenDeliveries limits four,
        query verify claims.outcomePreserved in any limits twelve,
        query verify claims.ackOnlyWhenKept in any limits twelve,
        query verify claims.closedHistoryIsFrozen in any limits twelve,
        query verify claims.handlerEffectIsIrreversible in any limits twelve,
        query verify claims.knownIsTheHandlersOutcome in any limits twelve,
        query verify claims.knowledgeIsFinal in any limits twelve,
        query(s"${m.name}.detachedWorkProceeds") find claims.finishesAfterClose in
          detachedWork limits four,
        query(s"${m.name}.intentWithoutReceipt") find claims.requestedButUnreceived in
          cancelRequested limits four,
        query(s"${m.name}.receiptWithoutEffect") find claims.receivedButSucceeded in
          cancelReceivedThenSucceeded limits four,
        query(s"${m.name}.effectWithoutKnowledge") find claims.canceledButUnknown in
          cancelReceivedThenCanceled limits four,
        query(s"${m.name}.canceledIsKnown") find claims.completionCancels in
          canceledThenDelivered limits four,
        query(s"${m.name}.asyncCompletion") find claims.completionSucceeds in
          finishedThenSucceeded limits four,
        query(s"${m.name}.asyncFailure") find claims.completionFails in
          finishedThenFailed limits four,
        query(s"${m.name}.transientRejectionAfterClose").find(claims.rejectedTransiently) in
          deliveredToClosed limits four,
        query(s"${m.name}.permanentRejectionAfterClose").find(claims.rejectedPermanently) in
          deliveredToClosed limits four,
        query(s"${m.name}.lostAfterReset") find claims.lostAfterReset in
          closedThenFinished limits four,
        query(s"${m.name}.resetAfterRetention")
          .find(claims.reappliesRetained) in closedThenFinished limits
          four,
        query(s"${m.name}.resetBeforeRetention")
          .find(claims.routedToSuccessor) in resetThenDelivered limits
          four
      )

    // The two pinned controls and the two promises, over another channel.

    /** The two promises and the two pinned controls, declared on one design. */
    def safetyQueries(m: Machine[CloseResetState, Answer, CloseFact]) =
      val claims = properties.safetyClaims(m)
      val closedThenFinished = m.scenario.actions(
        callerSide.close,
        handlerSide.finish(Resolution.succeeded),
        handler.complete(Resolution.succeeded),
        callerSide.reset
      )
      val resetThenDelivered = m.scenario.actions(
        handlerSide.finish(Resolution.failed),
        callerSide.reset,
        handler.complete(Resolution.failed)
      )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four,
        query verify claims.outcomePreserved in any limits twelve,
        query verify claims.ackOnlyWhenKept in any limits twelve
      )

    // The deadline. With a schedule-to-close deadline the state a lost outcome leaves has a step.
    // The timeout that ends that wait is found with nothing owed, and is told apart from one that
    // beats a report still in flight, which loses nothing the design promised.

    def deadlineQueries(m: Machine[CloseResetState, Answer, CloseFact]) =
      val claims = properties.deadlineClaims(m)
      val closedThenFinished = m.scenario.actions(
        callerSide.close,
        handlerSide.finish(Resolution.succeeded),
        handler.complete(Resolution.succeeded),
        callerSide.reset
      )
      val resetThenDelivered = m.scenario.actions(
        handlerSide.finish(Resolution.failed),
        callerSide.reset,
        handler.complete(Resolution.failed)
      )
      val closedLossThenExpired = m.scenario.actions(
        callerSide.close,
        handlerSide.finish(Resolution.succeeded),
        handler.complete(Resolution.succeeded),
        callerSide.reset,
        deadline.scheduleToClose
      )
      val resetLossThenExpired = m.scenario.actions(
        handlerSide.finish(Resolution.failed),
        callerSide.reset,
        handler.complete(Resolution.failed),
        deadline.scheduleToClose
      )
      val reportedThenExpired = m.scenario.actions(
        handlerSide.finish(Resolution.succeeded),
        deadline.scheduleToClose
      )
      val expiredThenDelivered = m.scenario.actions(
        handlerSide.finish(Resolution.succeeded),
        deadline.scheduleToClose,
        handler.complete(Resolution.succeeded)
      )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four,
        query verify claims.outcomePreserved in any limits twelve,
        query verify claims.ackOnlyWhenKept in any limits twelve,
        query verify claims.noUnnecessaryWait in any limits twelve,
        query(s"${m.name}.expiredAfterClosedLoss").find(claims.expiresWithNothingOwed) in
          closedLossThenExpired limits five,
        query(s"${m.name}.expiredAfterResetLoss").find(claims.expiresWithNothingOwed) in
          resetLossThenExpired limits five,
        query(s"${m.name}.expiredWhileReported").find(claims.expiresWhileOwed) in
          reportedThenExpired limits four,
        query(s"${m.name}.lateCompletionIsDropped").find(claims.lateCompletionIsDropped) in
          expiredThenDelivered limits four
      )

    val rejectAfterCloseQueries = designQueries(RejectAfterClose)

/**
 * The faulty policy: after a reset the original run acknowledges the completion, and the successor
 * never learns it. A deliberately wrong design.
 */
object AckByOriginal
    extends Derived(
      RejectAfterClose
        .assuming(retentionDurable)
        .rebind(
          handler.complete ~> (RejectAfterClose.effects
            .deliver(Policy.ackByOriginal, Redelivery.untilAck, _, _))
        )
    ),
      NegativeControl:
  object properties:
    val ackByOriginalProgress = AckByOriginal.leadsTo("outcomeReachesOwner")(
      RejectAfterClose.states.isDone,
      RejectAfterClose.states.settled,
      within = 6,
      reporting,
      deliveryFair
    )

  object queries:
    val ackByOriginalQueries = RejectAfterClose.queries.designQueries(AckByOriginal)

/** The corrected policy: retain at the operation and route to the current owner. */
object RetainAndRoute
    extends Derived(
      RejectAfterClose
        .assuming(retentionDurable)
        .rebind(
          handler.complete ~> (RejectAfterClose.effects
            .deliver(Policy.retainAndRoute, Redelivery.untilAck, _, _))
        )
    ):
  object properties:
    val retainAndRouteProgress = RetainAndRoute.leadsTo("outcomeReachesOwner")(
      RejectAfterClose.states.isDone,
      RejectAfterClose.states.settled,
      within = 6,
      reporting,
      deliveryFair
    )

    // A retained outcome reaches an owner only through a reset. The claim is made twice over the
    // channel that retries until acknowledged: under the recovery assumption, and without it.

    val retainedReachesOwner = RetainAndRoute.leadsTo("retainedReachesOwner")(
      RejectAfterClose.properties.awaitingOwner,
      RejectAfterClose.properties.ownerKnowsOutcome,
      within = 2,
      deliveryFair,
      recovery
    )

    val retainedWaitsWithoutRecovery = RetainAndRoute.leadsTo("retainedWaitsWithoutRecovery")(
      RejectAfterClose.properties.awaitingOwner,
      RejectAfterClose.properties.ownerKnowsOutcome,
      within = 2,
      deliveryFair
    )

  object queries:
    val retainAndRouteQueries = RejectAfterClose.queries.designQueries(RetainAndRoute)

/**
 * The corrected policy with a faulty reset that forgets the cancel request, and with it who made
 * it. A deliberately wrong design.
 */
object ForgetsCancelOnReset
    extends Derived(
      RetainAndRoute.rebind(
        callerSide.reset ~> (RejectAfterClose.effects.reset(Reset.forgetsCancel, _))
      )
    ),
      NegativeControl:
  object queries:
    val forgetsCancelOnResetQueries = RejectAfterClose.queries.designQueries(ForgetsCancelOnReset)

/**
 * The corrected policy with a faulty reset that does not reapply what the original run recorded. A
 * deliberately wrong design.
 */
object TruncatesOnReset
    extends Derived(
      RetainAndRoute.rebind(
        callerSide.reset ~> (RejectAfterClose.effects.reset(Reset.truncates, _))
      )
    ),
      NegativeControl:
  object queries:
    val truncatesOnResetQueries = RejectAfterClose.queries.designQueries(TruncatesOnReset)

/** The corrected design over the channel that redelivers once. */
object RetainAndRouteBoundedRetry
    extends Derived(
      RetainAndRoute
        .assuming(retryFair)
        .rebind(
          handler.complete ~> (RejectAfterClose.effects
            .deliver(Policy.retainAndRoute, Redelivery.once, _, _))
        )
    ):
  object properties:
    val retainAndRouteBoundedRetryProgress =
      RetainAndRouteBoundedRetry.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

    // Over the channel that redelivers once, at most the one redelivery comes before the reset.
    val retainedReachesOwnerBoundedRetry =
      RetainAndRouteBoundedRetry.leadsTo("retainedReachesOwner")(
        RejectAfterClose.properties.awaitingOwner,
        RejectAfterClose.properties.ownerKnowsOutcome,
        within = 2,
        deliveryFair,
        recovery
      )

  object queries:
    val retainAndRouteBoundedRetryQueries =
      RejectAfterClose.queries.safetyQueries(RetainAndRouteBoundedRetry)

// The three policies with a schedule-to-close deadline, over the channel that redelivers once.

/** The faulty policy that rejects after close, with the deadline. A deliberately wrong design. */
object RejectAfterCloseWithDeadline
    extends Derived(
      RejectAfterClose
        .assuming(retryFair, deadlineExpires)
        .rebind(
          handler.complete ~> (RejectAfterClose.effects
            .deliver(Policy.rejectAfterClose, Redelivery.once, _, _))
        )
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.effects.expire
          }
        )
    ),
      NegativeControl:
  object properties:
    val rejectAfterCloseWithDeadlineProgress =
      RejectAfterCloseWithDeadline.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val rejectAfterCloseWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(RejectAfterCloseWithDeadline)

/** The faulty policy the original run acknowledges, with the deadline. A deliberately wrong design. */
object AckByOriginalWithDeadline
    extends Derived(
      AckByOriginal
        .assuming(retryFair, deadlineExpires)
        .rebind(
          handler.complete ~> (RejectAfterClose.effects
            .deliver(Policy.ackByOriginal, Redelivery.once, _, _))
        )
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.effects.expire
          }
        )
    ),
      NegativeControl:
  object properties:
    val ackByOriginalWithDeadlineProgress =
      AckByOriginalWithDeadline.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val ackByOriginalWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(AckByOriginalWithDeadline)

/** The corrected design over the channel that redelivers once, with the deadline. */
object RetainAndRouteWithDeadline
    extends Derived(
      RetainAndRouteBoundedRetry
        .assuming(deadlineExpires)
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.effects.expire
          }
        )
    ):
  object properties:
    val retainAndRouteWithDeadlineProgress =
      RetainAndRouteWithDeadline.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val retainAndRouteWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(RetainAndRouteWithDeadline)
