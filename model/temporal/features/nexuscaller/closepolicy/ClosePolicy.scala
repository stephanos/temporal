/* The Nexus caller close and reset designs: one logical operation whose caller closes or is reset
 * while the handler works. Reviewed as model/specimens/nexus.md, whose supported sketch
 * irgen/testdata/lifts/CloseReset.scala was; this Model keeps the sketch's vocabulary and
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
 *
 * Read top to bottom: the types; the signature (the logical operation, the caller's and the
 * handler's actions, and the bounds); then RejectAfterClose, the first design, whose step functions,
 * monitors and assumptions every design reads, with the nine designs derived from it, their
 * promises and their Queries; and last Files, its IR file.
 */
package temporal
package features.nexuscaller
package closepolicy

// `Answer` here is the designs' delivery answer.
import umpire.*

// Moved from temporal.nexuscaller.closepolicy; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexuscaller.closepolicy.Model$package$")

/** The family of the designs' machines. */
given Family = Family("temporal.nexus.caller.closepolicy")

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

given Ok[Answer] = Ok(Answer.accepted)

val principal = input[Principal]
val result = input[Resolution]

// Who acts in the designs, grouped by side: the caller's and the handler's own actions here are
// taken by the parties the Nexus caller declares, whose actions the designs read too
// (`handler.complete`). The sections are not named `caller` and `handler`, which would name the
// same class files as the types `Caller` and `Handler` on a case-insensitive file system. Sections
// are transparent to Definition IDs, so each action keeps the ID this file's pin gives it.

/** What the designs' caller workflow does: it closes, is reset, and requests a cancel. */
object callerSide extends Section:
  val callerClose = action(caller) on workflow
  val reset = action(caller) on workflow
  val requestCancel = action(caller).on(operation).input(principal)

/** The handler finishes the operation's work. */
object handlerSide extends Section:
  val handlerFinish = action(handler).on(operation).input(result)

/** The server's own step: the cancel request reaches the handler. */
object history extends Section:
  val deliverCancel = internal

// The bounds. They search further than the caller's of the same names, `four` among them.

val four = Limits(steps = 4, actions = 4, search = 1 << 20)
val five = Limits(steps = 5, actions = 5, search = 1 << 20)

/**
 * Past the depth of every design's table, so a free search that verifies read every step.
 */
val twelve = Limits(steps = 12, actions = 12, search = 1 << 22)

// ### The designs differ only in the policy, the reset and the channel their steps pass
//
// Each design after the first is derived from one before it, changing the step functions its
// policy, reset or channel changes and the assumptions it adds.

object RejectAfterClose:
  // Moved from temporal.nexuscaller.closepolicy, as the file's declarations were: its monitors and
  // assumptions keep the Definition IDs they had there.
  given DefinitionScope = DefinitionScope("temporal.nexuscaller.closepolicy.Model$package$")

  val opened = CloseResetState(
    Caller.open,
    Intent.none,
    Handler.running,
    Completion.none,
    Retained.none,
    Knowledge.none
  )

  def working(h: Handler) = h.in(Handler.running, Handler.cancelReceived)

  /** The history event that records an outcome, by the baseline's names. */
  def recorded(r: Resolution) = r match
    case Resolution.succeeded => Fact.nexusOperationCompleted
    case Resolution.failed    => Fact.nexusOperationFailed
    case Resolution.canceled  => Fact.nexusOperationCanceled

  /**
   * A history records an outcome once: a second delivery of it records nothing.
   */
  def recordedOnce(s: CloseResetState, k: Knowledge, r: Resolution) =
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

  def reapplied(rule: Reset, s: CloseResetState) = s.known match
    case Knowledge.original(r) =>
      if rule == Reset.truncates then fromRetention(s.retained) else Knowledge.successor(r)
    case Knowledge.expired => Knowledge.expired
    case _                 => fromRetention(s.retained)

  def carried(rule: Reset, i: Intent) = if rule == Reset.forgetsCancel then Intent.none else i

  /**
   * What the successor's history records at its start: the reset, and the outcome it reapplies.
   */
  def resetFacts(k: Knowledge) = k match
    case Knowledge.successor(r) => List(Fact.workflowReset, Fact.outcomeReapplied, recorded(r))
    case _                      => List(Fact.workflowReset)

  // Promises: the outcome stays where an owner can learn it, an acknowledgment keeps it, and a path
  // ends only where the outcome is settled.

  /** The run that owns the operation records this outcome. */
  def ownerKnows(s: CloseResetState, r: Resolution) =
    if s.caller == Caller.resetOpen then s.known == Knowledge.successor(r)
    else s.known == Knowledge.original(r)

  /**
   * A decided outcome is always somewhere a current or future owner can learn it: its history, the
   * operation's retention, or a report still in flight. An operation its deadline resolved waits for
   * no outcome.
   */
  def outcomePreserved(after: CloseResetStep) = after.state.handler match
    case Handler.done(r) =>
      ownerKnows(after.state, r) ||
      after.state.retained == Retained.pending(r) || carries(after.state.channel, r) ||
      after.state.known == Knowledge.expired
    case _ => true

  def keptOrOwed(after: CloseResetStep, r: Resolution) =
    after.outcome.in(Answer.accepted, Answer.retained) implies
      (after.state.channel != Completion.none || ownerKnows(after.state, r) ||
        after.state.retained == Retained.pending(r))

  /**
   * An acknowledgment ends the handler's report only once the owner committed or the operation
   * retained the outcome.
   */
  def ackOnlyWhenKept(before: CloseResetState, after: CloseResetStep) = before.channel match
    case Completion.inFlight(r) => keptOrOwed(after, r)
    case Completion.retried(r)  => keptOrOwed(after, r)
    case Completion.none        => true

  def isDone(s: CloseResetState) = !working(s.handler)

  /**
   * Where a path may end: the handler is still working, or its outcome reached the owner, or the
   * operation retained it for a successor a closed run may never get, or the deadline resolved the
   * wait. Any other state with no step is a lost outcome.
   */
  def settled(s: CloseResetState) = s.handler match
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

  object effects:
    def closeStep(s: CloseResetState) =
      if s.caller != Caller.open then disabled
      else enter(s.copy(caller = Caller.closed), Fact.workflowClosed)

    /**
     * One cancellation, while the handler has neither finished nor been asked, by a run still open.
     */
    def cancelStep(s: CloseResetState, p: Principal) =
      if s.intent != Intent.none || s.caller == Caller.closed || s.handler != Handler.running then
        disabled
      else enter(s.copy(intent = Intent.requested(p)), Fact.cancelRequested(p))

    /**
     * The request in flight reaches a handler still working. A closed run's request is still
     * delivered.
     */
    def cancelDeliveryStep(s: CloseResetState) =
      if s.intent == Intent.none || s.handler != Handler.running then disabled
      else enter(s.copy(handler = Handler.cancelReceived), Fact.cancelReceived)

    /**
     * The handler's irreversible effect, and its first report. A canceled result needs the handler
     * to have received the cancel request; having received it, the handler may still succeed or
     * fail.
     */
    def finishStep(s: CloseResetState, r: Resolution) =
      if !working(s.handler) then disabled
      else if r == Resolution.canceled && s.handler != Handler.cancelReceived then disabled
      else
        enter(
          s.copy(handler = Handler.done(r), channel = Completion.inFlight(r)),
          Fact.handlerFinished(r)
        )

    /**
     * The owner commits the outcome. Transient rejection keeps the report in flight; a lost
     * acknowledgment commits and keeps it in flight too, so the report arrives again.
     */
    def committed(
        d: Redelivery,
        s: CloseResetState,
        k: Knowledge,
        r: Resolution
    ) =
      if redelivers(d, s.channel, r) then
        choose(
          taken -> enter(s.copy(known = k, channel = Completion.none), recordedOnce(s, k, r)*),
          rejectedForNow -> List(Step(Answer.rejectedTransient, s.copy(channel = again(d, r)))),
          ackLost -> enter(s.copy(known = k, channel = again(d, r)), recordedOnce(s, k, r)*)
            .because("the acknowledgment is lost")
        )
      else enter(s.copy(known = k, channel = Completion.none), recordedOnce(s, k, r)*)

    def keptAtOperation(d: Redelivery, s: CloseResetState, r: Resolution) =
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

    /** A permanent rejection ends the report: the handler stops reporting. */
    def dropped(s: CloseResetState): List[CloseResetStep] =
      List(
        Step(
          Answer.rejectedPermanent,
          s.copy(channel = Completion.none),
          List(Fact.completionDropped)
        )
      )

    def rejectedByClosed(d: Redelivery, s: CloseResetState, r: Resolution) =
      if redelivers(d, s.channel, r) then
        choose(
          refused -> dropped(s),
          rejectedForNow -> List(Step(Answer.rejectedTransient, s.copy(channel = again(d, r))))
        )
      else dropped(s)

    def deliverStep(p: Policy, d: Redelivery, s: CloseResetState, r: Resolution) =
      if !carries(s.channel, r) then disabled
      // The deadline resolved the operation: a completion after it finds nothing to complete.
      else if s.known == Knowledge.expired then dropped(s)
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
    def resetStep(rule: Reset, s: CloseResetState) =
      if s.caller == Caller.resetOpen then disabled
      else
        enter(
          s.copy(
            caller = Caller.resetOpen,
            intent = carried(rule, s.intent),
            known = reapplied(rule, s),
            retained = Retained.none
          ),
          resetFacts(reapplied(rule, s))*
        )

    /**
     * The schedule-to-close deadline resolves an operation whose owner knows no outcome. A closed
     * run's history is frozen, so no deadline fires in it.
     */
    def expireStep(s: CloseResetState) =
      if s.caller == Caller.closed || s.known != Knowledge.none then disabled
      else enter(s.copy(known = Knowledge.expired), Fact.nexusOperationTimedOut)

  // Monitors: the outcome stays where an owner can learn it, an acknowledgment keeps it, no run
  // records two outcomes, and a cancel request stays its principal's.

  /**
   * Whether a step lost a decided outcome: no owner knows it, and nothing retains or carries it.
   */
  val retainedOutcome = sticky(outcomePreserved)

  /**
   * Whether an acknowledgment ended the handler's report before the owner committed the outcome or
   * the operation retained it.
   */
  val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)

  val singleOutcome =
    monitor[CloseResetState, Answer, Fact, Outcomes](Outcomes.none)((seen, _, after) =>
      outcomesAfter(seen, after.state.known)
    )(seen => seen == Outcomes.several)

  /**
   * The cancellation principal is lost: a request made is no longer in the owner's history as made.
   */
  val cancelPrincipal =
    monitor[CloseResetState, Answer, Fact, Asked](Asked.nobody)((asked, _, after) =>
      askedAfter(asked, after.state.intent)
    )(asked => asked == Asked.lost)

  // Assumptions. Retention and the two channel variants are assumptions of the machines that have
  // them, so every result over such a machine names them. The rest are what a progress claim is
  // conditional on.

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
   * The handler reports until acknowledged or permanently rejected, as the channel holds the
   * report.
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
      callerSide.callerClose ~> effects.closeStep,
      callerSide.reset ~> (s => effects.resetStep(Reset.reapplies, s)),
      callerSide.requestCancel ~> effects.cancelStep,
      history.deliverCancel ~> effects.cancelDeliveryStep,
      handlerSide.handlerFinish ~> effects.finishStep,
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.rejectAfterClose, Redelivery.untilAck, s, r)
      )
    )
  }

  val ackByOriginal = rejectAfterClose
    .assuming(retentionDurable)
    .rebind(
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.ackByOriginal, Redelivery.untilAck, s, r)
      )
    )

  val retainAndRoute = rejectAfterClose
    .assuming(retentionDurable)
    .rebind(
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.retainAndRoute, Redelivery.untilAck, s, r)
      )
    )

  /** The corrected policy with a reset that forgets the cancel request. */
  val forgetsCancelOnReset =
    retainAndRoute.rebind(callerSide.reset ~> (s => effects.resetStep(Reset.forgetsCancel, s)))

  /**
   * The corrected policy with a reset that does not reapply what the original run recorded.
   */
  val truncatesOnReset =
    retainAndRoute.rebind(callerSide.reset ~> (s => effects.resetStep(Reset.truncates, s)))

  /** The corrected design over the channel that redelivers once. */
  val retainAndRouteBoundedRetry = retainAndRoute
    .assuming(retryFair)
    .rebind(
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.retainAndRoute, Redelivery.once, s, r)
      )
    )

  // The three policies with a schedule-to-close deadline, over the channel that redelivers once.

  val rejectAfterCloseWithDeadline = rejectAfterClose
    .assuming(retryFair, deadlineExpires)
    .rebind(
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.rejectAfterClose, Redelivery.once, s, r)
      )
    )
    .extend(deadline.scheduleToClose ~> effects.expireStep)

  val ackByOriginalWithDeadline = ackByOriginal
    .assuming(retryFair, deadlineExpires)
    .rebind(
      handler.complete ~> ((s, r) =>
        effects.deliverStep(Policy.ackByOriginal, Redelivery.once, s, r)
      )
    )
    .extend(deadline.scheduleToClose ~> effects.expireStep)

  val retainAndRouteWithDeadline = retainAndRouteBoundedRetry
    .assuming(deadlineExpires)
    .extend(deadline.scheduleToClose ~> effects.expireStep)

  // What the designs promise. Every promise below is an authored design promise, written once and
  // declared as a Property on each design by the claims below. The monitors' promises are above,
  // beside the monitors. Two of the Properties are the baseline's, completionSucceeds and
  // completionFails, which an open caller keeps as nexusProtocol does. A design names its monitors,
  // and a verify also reports the first violation a watching monitor meets, whatever Property it
  // asks.

  object properties:
    /**
     * A closed run's history is frozen: no step of a caller that stays closed changes what it holds.
     */
    def closedHistoryIsFrozen(before: CloseResetState, after: CloseResetStep) =
      before.caller == Caller.closed && after.state.caller == Caller.closed implies
        (after.state.known == before.known && after.state.intent == before.intent)

    /** An outcome a history records is the handler's. */
    def knownIsTheHandlersOutcome(after: CloseResetStep) = after.state.known match
      case Knowledge.original(r)  => after.state.handler == Handler.done(r)
      case Knowledge.successor(r) => after.state.handler == Handler.done(r)
      case _                      => true

    def knows(k: Knowledge, r: Resolution) =
      k.in(Knowledge.original(r), Knowledge.successor(r))

    /**
     * A recorded outcome stays recorded: a redelivery changes nothing, and a reset carries it over.
     */
    def knowledgeIsFinal(before: CloseResetState, after: CloseResetStep) = before.known match
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
    def noUnnecessaryWait(before: CloseResetState, after: CloseResetStep) =
      before.known != Knowledge.expired && after.state.known == Knowledge.expired implies
        !nothingOwed(before)

    def designClaims(m: Machine[CloseResetState, Answer, Fact]) =
      // No step, a reset included, undoes what the handler did.
      val handlerEffectIsIrreversible = m.property.once(isDone).keeps(_.handler)
      // The handler's detached work goes on after the close.
      val finishesAfterClose = m.property when
        handlerSide.handlerFinish(Resolution.succeeded) holds
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
        handlerSide.handlerFinish(Resolution.succeeded) holds
        (after =>
          after.state.intent == Intent.requested(Principal.callerWorkflow) &&
            after.state.handler == Handler.done(Resolution.succeeded)
        )
      val canceledButUnknown = m.property when
        handlerSide.handlerFinish(Resolution.canceled) holds
        (after =>
          after.state.handler == Handler.done(Resolution.canceled) &&
            !ownerKnows(after.state, Resolution.canceled)
        )
      val completionCancels = m.property when handler.complete(Resolution.canceled) holds
        (after =>
          ownerKnows(after.state, Resolution.canceled) &&
            after.records(Fact.nexusOperationCanceled)
        )
      // The baseline's two: an open caller records a completion by the baseline's event.
      val completionSucceeds = m.property when
        handler.complete(Resolution.succeeded) holds
        (after => after.records(Fact.nexusOperationCompleted))
      val completionFails = m.property when handler.complete(Resolution.failed) holds
        (after => after.records(Fact.nexusOperationFailed))
      // The two answers of a closed run, and the two resets, are kept apart.
      val rejectedTransiently = m.property when
        handler.complete(Resolution.succeeded) holds
        (after =>
          after.outcome == Answer.rejectedTransient &&
            carries(after.state.channel, Resolution.succeeded)
        )
      val rejectedPermanently = m.property when
        handler.complete(Resolution.succeeded) holds (after =>
          after.outcome == Answer.rejectedPermanent
        )
      val lostAfterReset = m.property when callerSide.reset holds
        (after => !outcomePreserved(after))
      val reappliesRetained = m.property when callerSide.reset holds
        (after =>
          after.records(Fact.outcomeReapplied) &&
            after.state.known == Knowledge.successor(Resolution.succeeded)
        )
      val routedToSuccessor = m.property when handler.complete(Resolution.failed) holds
        (after => after.state.known == Knowledge.successor(Resolution.failed))
      DesignClaims(
        m.property("outcomePreserved") holds outcomePreserved,
        m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept,
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

    def safetyClaims(m: Machine[CloseResetState, Answer, Fact]) = SafetyClaims(
      m.property("outcomePreserved") holds outcomePreserved,
      m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
    )

    def deadlineClaims(m: Machine[CloseResetState, Answer, Fact]) =
      val expiresWithNothingOwed = m.property when deadline.scheduleToClose holds
        (after => after.state.known == Knowledge.expired && nothingOwed(after.state))
      val expiresWhileOwed = m.property when deadline.scheduleToClose holds
        (after =>
          after.state.known == Knowledge.expired &&
            carries(after.state.channel, Resolution.succeeded)
        )
      val lateCompletionIsDropped = m.property when
        handler.complete(Resolution.succeeded) holds
        (after =>
          after.outcome == Answer.rejectedPermanent && after.records(Fact.completionDropped)
        )
      DeadlineClaims(
        m.property("outcomePreserved") holds outcomePreserved,
        m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept,
        m.property("noUnnecessaryWait") holdsAcross noUnnecessaryWait,
        expiresWithNothingOwed,
        expiresWhileOwed,
        lateCompletionIsDropped
      )

    // Conditional progress. From a decided outcome, a state a path may end in follows within six
    // steps, under the assumptions each claim and its machine name. A machine that names no
    // deadline has no timer, and one that does not assume the redelivery bound may be rejected
    // transiently forever.

    val rejectAfterCloseProgress = rejectAfterClose.leadsTo(
      "outcomeReachesOwner"
    )(isDone, settled, within = 6, reporting, deliveryFair)

    val ackByOriginalProgress = ackByOriginal.leadsTo(
      "outcomeReachesOwner"
    )(isDone, settled, within = 6, reporting, deliveryFair)

    val retainAndRouteProgress = retainAndRoute.leadsTo(
      "outcomeReachesOwner"
    )(isDone, settled, within = 6, reporting, deliveryFair)

    val retainAndRouteBoundedRetryProgress = retainAndRouteBoundedRetry
      .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

    val rejectAfterCloseWithDeadlineProgress =
      rejectAfterCloseWithDeadline.leadsTo("outcomeReachesOwner")(
        isDone,
        settled,
        within = 6,
        reporting,
        deliveryFair
      )

    val ackByOriginalWithDeadlineProgress = ackByOriginalWithDeadline
      .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

    val retainAndRouteWithDeadlineProgress = retainAndRouteWithDeadline
      .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

    /**
     * The operation retained the outcome for a closed run, and no owner knows it yet.
     */
    def awaitingOwner(s: CloseResetState) = s.handler match
      case Handler.done(r) =>
        s.caller == Caller.closed && s.retained == Retained.pending(r) &&
        !ownerKnows(s, r)
      case _ => false

    def ownerKnowsOutcome(s: CloseResetState) = s.handler match
      case Handler.done(r) => ownerKnows(s, r)
      case _               => false

    // A retained outcome reaches an owner only through a reset. The claim is made twice over the
    // channel that retries until acknowledged: under the recovery assumption, and without it. Over
    // the channel that redelivers once, at most the one redelivery comes before the reset.

    val retainedReachesOwner = retainAndRoute.leadsTo(
      "retainedReachesOwner"
    )(awaitingOwner, ownerKnowsOutcome, within = 2, deliveryFair, recovery)

    val retainedWaitsWithoutRecovery = retainAndRoute.leadsTo(
      "retainedWaitsWithoutRecovery"
    )(awaitingOwner, ownerKnowsOutcome, within = 2, deliveryFair)

    val retainedReachesOwnerBoundedRetry = retainAndRouteBoundedRetry
      .leadsTo("retainedReachesOwner")(
        awaitingOwner,
        ownerKnowsOutcome,
        within = 2,
        deliveryFair,
        recovery
      )

  object queries:
    /** Every claim and path of the specimen, declared on one design. */
    def designQueries(m: Machine[CloseResetState, Answer, Fact]) =
      val claims = properties.designClaims(m)
      val closedThenFinished = m.scenario
        .actions(
          callerSide.callerClose,
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset
        )
      val resetThenDelivered = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed)
        )
      // The request, its delivery to the handler and the handler's effect are three steps here.
      val canceledAcrossReset = m.scenario
        .actions(
          callerSide.requestCancel(Principal.callerWorkflow),
          history.deliverCancel,
          handlerSide.handlerFinish(Resolution.canceled),
          callerSide.reset,
          handler.complete(Resolution.canceled)
        )
      val ackedThenReset = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset
        )
      val duplicateCompletion = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          handler.complete(Resolution.succeeded)
        )
      // A reset between the commit and its acknowledgment, and between a rejection and the retry.
      val resetBetweenDeliveries = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.failed),
          handler.complete(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed)
        )
      val detachedWork = m.scenario
        .actions(callerSide.callerClose, handlerSide.handlerFinish(Resolution.succeeded))
      val cancelRequested = m.scenario
        .actions(callerSide.requestCancel(Principal.callerWorkflow))
      val cancelReceivedThenSucceeded = m.scenario
        .actions(
          callerSide.requestCancel(Principal.callerWorkflow),
          history.deliverCancel,
          handlerSide.handlerFinish(Resolution.succeeded)
        )
      val cancelReceivedThenCanceled = m.scenario
        .actions(
          callerSide.requestCancel(Principal.callerWorkflow),
          history.deliverCancel,
          handlerSide.handlerFinish(Resolution.canceled)
        )
      val canceledThenDelivered = m.scenario
        .actions(
          callerSide.requestCancel(Principal.callerWorkflow),
          history.deliverCancel,
          handlerSide.handlerFinish(Resolution.canceled),
          handler.complete(Resolution.canceled)
        )
      val finishedThenSucceeded = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded)
        )
      val finishedThenFailed = m.scenario
        .actions(handlerSide.handlerFinish(Resolution.failed), handler.complete(Resolution.failed))
      val deliveredToClosed = m.scenario
        .actions(
          callerSide.callerClose,
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded)
        )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four total 26880,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four total 20160,
        query verify claims.outcomePreserved in resetThenDelivered limits four total 20160,
        query(s"${m.name}.canceledAcrossReset") verify claims.outcomePreserved in
          canceledAcrossReset limits five total 33600,
        query(
          s"${m.name}.ackedThenReset"
        ) verify claims.outcomePreserved in ackedThenReset limits four total 20160,
        query(s"${m.name}.duplicateCompletion")
          .verify(claims.knowledgeIsFinal) in duplicateCompletion limits
          four total 20160,
        query(s"${m.name}.resetBetweenCommitAndAcknowledgment").verify(claims.ackOnlyWhenKept) in
          resetBetweenDeliveries limits four total 26880,
        query verify claims.outcomePreserved in any limits twelve total 806400,
        query verify claims.ackOnlyWhenKept in any limits twelve total 806400,
        query verify claims.closedHistoryIsFrozen in any limits twelve total 806400,
        query verify claims.handlerEffectIsIrreversible in any limits twelve total 806400,
        query verify claims.knownIsTheHandlersOutcome in any limits twelve total 806400,
        query verify claims.knowledgeIsFinal in any limits twelve total 806400,
        query(s"${m.name}.detachedWorkProceeds") find claims.finishesAfterClose in
          detachedWork limits four total 13440,
        query(s"${m.name}.intentWithoutReceipt")
          .find(claims.requestedButUnreceived) in cancelRequested limits
          four total 6720,
        query(s"${m.name}.receiptWithoutEffect").find(claims.receivedButSucceeded) in
          cancelReceivedThenSucceeded limits four total 20160,
        query(s"${m.name}.effectWithoutKnowledge").find(claims.canceledButUnknown) in
          cancelReceivedThenCanceled limits four total 20160,
        query(s"${m.name}.canceledIsKnown") find claims.completionCancels in
          canceledThenDelivered limits four total 26880,
        query(s"${m.name}.asyncCompletion")
          .find(claims.completionSucceeds) in finishedThenSucceeded limits
          four total 13440,
        query(s"${m.name}.asyncFailure") find claims.completionFails in
          finishedThenFailed limits four total 13440,
        query(s"${m.name}.transientRejectionAfterClose").find(claims.rejectedTransiently) in
          deliveredToClosed limits four total 20160,
        query(s"${m.name}.permanentRejectionAfterClose").find(claims.rejectedPermanently) in
          deliveredToClosed limits four total 20160,
        query(s"${m.name}.lostAfterReset") find claims.lostAfterReset in
          closedThenFinished limits four total 26880,
        query(s"${m.name}.resetAfterRetention")
          .find(claims.reappliesRetained) in closedThenFinished limits
          four total 26880,
        query(s"${m.name}.resetBeforeRetention")
          .find(claims.routedToSuccessor) in resetThenDelivered limits
          four total 20160
      )

    val rejectAfterCloseQueries = designQueries(rejectAfterClose)
    val ackByOriginalQueries = designQueries(ackByOriginal)
    val retainAndRouteQueries = designQueries(retainAndRoute)
    val forgetsCancelOnResetQueries = designQueries(forgetsCancelOnReset)
    val truncatesOnResetQueries = designQueries(truncatesOnReset)

    // The two pinned controls and the two promises, over another channel.

    /** The two promises and the two pinned controls, declared on one design. */
    def safetyQueries(m: Machine[CloseResetState, Answer, Fact]) =
      val claims = properties.safetyClaims(m)
      val closedThenFinished = m.scenario
        .actions(
          callerSide.callerClose,
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset
        )
      val resetThenDelivered = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed)
        )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four total 26880,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four total 20160,
        query verify claims.outcomePreserved in any limits twelve total 806400,
        query verify claims.ackOnlyWhenKept in any limits twelve total 806400
      )

    val retainAndRouteBoundedRetryQueries =
      safetyQueries(retainAndRouteBoundedRetry)

    // The deadline. With a schedule-to-close deadline the state a lost outcome leaves has a step.
    // The timeout that ends that wait is found with nothing owed, and is told apart from one that
    // beats a report still in flight, which loses nothing the design promised.

    def deadlineQueries(m: Machine[CloseResetState, Answer, Fact]) =
      val claims = properties.deadlineClaims(m)
      val closedThenFinished = m.scenario
        .actions(
          callerSide.callerClose,
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset
        )
      val resetThenDelivered = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed)
        )
      val closedLossThenExpired = m.scenario
        .actions(
          callerSide.callerClose,
          handlerSide.handlerFinish(Resolution.succeeded),
          handler.complete(Resolution.succeeded),
          callerSide.reset,
          deadline.scheduleToClose
        )
      val resetLossThenExpired = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.failed),
          callerSide.reset,
          handler.complete(Resolution.failed),
          deadline.scheduleToClose
        )
      val reportedThenExpired = m.scenario
        .actions(handlerSide.handlerFinish(Resolution.succeeded), deadline.scheduleToClose)
      val expiredThenDelivered = m.scenario
        .actions(
          handlerSide.handlerFinish(Resolution.succeeded),
          deadline.scheduleToClose,
          handler.complete(Resolution.succeeded)
        )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
          closedThenFinished limits four total 26880,
        query verify claims.ackOnlyWhenKept in resetThenDelivered limits four total 20160,
        query verify claims.outcomePreserved in any limits twelve total 887040,
        query verify claims.ackOnlyWhenKept in any limits twelve total 887040,
        query verify claims.noUnnecessaryWait in any limits twelve total 887040,
        query(s"${m.name}.expiredAfterClosedLoss").find(claims.expiresWithNothingOwed) in
          closedLossThenExpired limits five total 33600,
        query(s"${m.name}.expiredAfterResetLoss").find(claims.expiresWithNothingOwed) in
          resetLossThenExpired limits five total 26880,
        query(s"${m.name}.expiredWhileReported")
          .find(claims.expiresWhileOwed) in reportedThenExpired limits
          four total 13440,
        query(s"${m.name}.lateCompletionIsDropped").find(claims.lateCompletionIsDropped) in
          expiredThenDelivered limits four total 20160
      )

    val rejectAfterCloseWithDeadlineQueries =
      deadlineQueries(rejectAfterCloseWithDeadline)
    val ackByOriginalWithDeadlineQueries =
      deadlineQueries(ackByOriginalWithDeadline)
    val retainAndRouteWithDeadlineQueries =
      deadlineQueries(retainAndRouteWithDeadline)

// ### The checked-in IR file of the Nexus caller close and reset designs (umpire.irFile).

object Files:
  // Each design's Queries are a root, and so is each progress claim.
  val nexusCloseFile = irFile("nexus-close")(
    RejectAfterClose.queries.rejectAfterCloseQueries,
    RejectAfterClose.queries.ackByOriginalQueries,
    RejectAfterClose.queries.retainAndRouteQueries,
    RejectAfterClose.queries.forgetsCancelOnResetQueries,
    RejectAfterClose.queries.truncatesOnResetQueries,
    RejectAfterClose.queries.retainAndRouteBoundedRetryQueries,
    RejectAfterClose.queries.rejectAfterCloseWithDeadlineQueries,
    RejectAfterClose.queries.ackByOriginalWithDeadlineQueries,
    RejectAfterClose.queries.retainAndRouteWithDeadlineQueries,
    RejectAfterClose.properties.rejectAfterCloseProgress,
    RejectAfterClose.properties.ackByOriginalProgress,
    RejectAfterClose.properties.retainAndRouteProgress,
    RejectAfterClose.properties.retainAndRouteBoundedRetryProgress,
    RejectAfterClose.properties.rejectAfterCloseWithDeadlineProgress,
    RejectAfterClose.properties.ackByOriginalWithDeadlineProgress,
    RejectAfterClose.properties.retainAndRouteWithDeadlineProgress,
    RejectAfterClose.properties.retainedReachesOwner,
    RejectAfterClose.properties.retainedWaitsWithoutRecovery,
    RejectAfterClose.properties.retainedReachesOwnerBoundedRetry
  )
