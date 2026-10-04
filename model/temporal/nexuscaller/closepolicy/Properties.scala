/* What the close and reset designs promise: the promises, the claims each design is held to, and
 * the conditional progress claims.
 */
package temporal
package nexuscaller
package closepolicy

import umpire.*

// ### What the designs promise
//
// Every promise below is an authored design promise, written once and declared as a Property on
// each design by the claims below. The monitors' promises are in Model.scala, beside the monitors.
// Two of the Properties are the baseline's, completionSucceeds and completionFails, which an open
// caller keeps as nexusProtocol does. A design names its monitors, and a verify also reports the
// first violation a watching monitor meets, whatever Property it asks.

/**
 * A closed run's history is frozen: no step of a caller that stays closed changes what it holds.
 */
def closedHistoryIsFrozen(before: CloseResetState, after: CloseResetStep): Boolean =
  before.caller == Caller.closed && after.state.caller == Caller.closed implies
    (after.state.known == before.known && after.state.intent == before.intent)

/** An outcome a history records is the handler's. */
def knownIsTheHandlersOutcome(after: CloseResetStep): Boolean = after.state.known match
  case Knowledge.original(r)  => after.state.handler == Handler.done(r)
  case Knowledge.successor(r) => after.state.handler == Handler.done(r)
  case _                      => true

def knows(k: Knowledge, r: Resolution): Boolean =
  k.in(Knowledge.original(r), Knowledge.successor(r))

/**
 * A recorded outcome stays recorded: a redelivery changes nothing, and a reset carries it over.
 */
def knowledgeIsFinal(before: CloseResetState, after: CloseResetStep): Boolean = before.known match
  case Knowledge.original(r)  => knows(after.state.known, r)
  case Knowledge.successor(r) => knows(after.state.known, r)
  case _                      => true

/** The handler's outcome is decided, and nothing holds or still carries it. */
def nothingOwed(s: CloseResetState): Boolean = s.handler match
  case Handler.done(_) => s.channel == Completion.none && s.retained == Retained.none
  case _               => false

/**
 * The deadline resolves only a wait that could still end otherwise: the handler is working, or its
 * report is still owed. A deadline that fires with nothing owed ends a wait for an outcome the
 * design already lost.
 */
def noUnnecessaryWait(before: CloseResetState, after: CloseResetStep): Boolean =
  before.known != Knowledge.expired && after.state.known == Knowledge.expired implies
    !nothingOwed(before)

// ### The claims each design is held to
//
// Each set is declared on the design `m` its Queries ask, since a Property belongs to one machine.

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

def designClaims(m: Machine[CloseResetState, Answer, Fact]): DesignClaims =
  // No step, a reset included, undoes what the handler did.
  val handlerEffectIsIrreversible = m.property.once(isDone).keeps(_.handler)
  // The handler's detached work goes on after the close.
  val finishesAfterClose = m.property when
    handlerFinish(Resolution.succeeded) holds
    (after =>
      after.state.caller == Caller.closed &&
        after.state.handler == Handler.done(Resolution.succeeded)
    )
  // A cancellation's request, its receipt, the handler's effect and the caller's knowledge.
  val requestedButUnreceived = m.property when
    requestCancel(Principal.callerWorkflow) holds
    (after =>
      after.state.intent == Intent.requested(Principal.callerWorkflow) &&
        after.state.handler == Handler.running
    )
  val receivedButSucceeded = m.property when
    handlerFinish(Resolution.succeeded) holds
    (after =>
      after.state.intent == Intent.requested(Principal.callerWorkflow) &&
        after.state.handler == Handler.done(Resolution.succeeded)
    )
  val canceledButUnknown = m.property when
    handlerFinish(Resolution.canceled) holds
    (after =>
      after.state.handler == Handler.done(Resolution.canceled) &&
        !ownerKnows(after.state, Resolution.canceled)
    )
  val completionCancels = m.property when complete(Resolution.canceled) holds
    (after =>
      ownerKnows(after.state, Resolution.canceled) &&
        after.records(Fact.nexusOperationCanceled)
    )
  // The baseline's two: an open caller records a completion by the baseline's event.
  val completionSucceeds = m.property when
    complete(Resolution.succeeded) holds
    (after => after.records(Fact.nexusOperationCompleted))
  val completionFails = m.property when complete(Resolution.failed) holds
    (after => after.records(Fact.nexusOperationFailed))
  // The two answers of a closed run, and the two resets, are kept apart.
  val rejectedTransiently = m.property when
    complete(Resolution.succeeded) holds
    (after =>
      after.outcome == Answer.rejectedTransient &&
        carries(after.state.channel, Resolution.succeeded)
    )
  val rejectedPermanently = m.property when
    complete(Resolution.succeeded) holds (after => after.outcome == Answer.rejectedPermanent)
  val lostAfterReset = m.property when reset holds
    (after => !outcomePreserved(after))
  val reappliesRetained = m.property when reset holds
    (after =>
      after.records(Fact.outcomeReapplied) &&
        after.state.known == Knowledge.successor(Resolution.succeeded)
    )
  val routedToSuccessor = m.property when complete(Resolution.failed) holds
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

/** The two promises every design keeps. */
final case class SafetyClaims(
    outcomePreserved: Property[CloseResetState],
    ackOnlyWhenKept: Property[CloseResetState]
)

def safetyClaims(m: Machine[CloseResetState, Answer, Fact]): SafetyClaims = SafetyClaims(
  m.property("outcomePreserved") holds outcomePreserved,
  m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
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

def deadlineClaims(m: Machine[CloseResetState, Answer, Fact]): DeadlineClaims =
  val expiresWithNothingOwed = m.property when scheduleToClose holds
    (after => after.state.known == Knowledge.expired && nothingOwed(after.state))
  val expiresWhileOwed = m.property when scheduleToClose holds
    (after =>
      after.state.known == Knowledge.expired && carries(after.state.channel, Resolution.succeeded)
    )
  val lateCompletionIsDropped = m.property when
    complete(Resolution.succeeded) holds
    (after => after.outcome == Answer.rejectedPermanent && after.records(Fact.completionDropped))
  DeadlineClaims(
    m.property("outcomePreserved") holds outcomePreserved,
    m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept,
    m.property("noUnnecessaryWait") holdsAcross noUnnecessaryWait,
    expiresWithNothingOwed,
    expiresWhileOwed,
    lateCompletionIsDropped
  )

// ### Conditional progress
//
// From a decided outcome, a state a path may end in follows within six steps, under the assumptions
// each claim and its machine name. A machine that names no deadline has no timer, and one that does
// not assume the redelivery bound may be rejected transiently forever.

val rejectAfterCloseProgress: Progress[CloseResetState] = rejectAfterClose.leadsTo(
  "outcomeReachesOwner"
)(isDone, settled, within = 6, reporting, deliveryFair)

val ackByOriginalProgress: Progress[CloseResetState] = ackByOriginal.leadsTo(
  "outcomeReachesOwner"
)(isDone, settled, within = 6, reporting, deliveryFair)

val retainAndRouteProgress: Progress[CloseResetState] = retainAndRoute.leadsTo(
  "outcomeReachesOwner"
)(isDone, settled, within = 6, reporting, deliveryFair)

val retainAndRouteBoundedRetryProgress: Progress[CloseResetState] = retainAndRouteBoundedRetry
  .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

val rejectAfterCloseWithDeadlineProgress: Progress[CloseResetState] =
  rejectAfterCloseWithDeadline.leadsTo("outcomeReachesOwner")(
    isDone,
    settled,
    within = 6,
    reporting,
    deliveryFair
  )

val ackByOriginalWithDeadlineProgress: Progress[CloseResetState] = ackByOriginalWithDeadline
  .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

val retainAndRouteWithDeadlineProgress: Progress[CloseResetState] = retainAndRouteWithDeadline
  .leadsTo("outcomeReachesOwner")(isDone, settled, within = 6, reporting, deliveryFair)

/**
 * The operation retained the outcome for a closed run, and no owner knows it yet.
 */
def awaitingOwner(s: CloseResetState): Boolean = s.handler match
  case Handler.done(r) =>
    s.caller == Caller.closed && s.retained == Retained.pending(r) &&
    !ownerKnows(s, r)
  case _ => false

def ownerKnowsOutcome(s: CloseResetState): Boolean = s.handler match
  case Handler.done(r) => ownerKnows(s, r)
  case _               => false

// A retained outcome reaches an owner only through a reset. The claim is made twice over the channel
// that retries until acknowledged: under the recovery assumption, and without it. Over the channel
// that redelivers once, at most the one redelivery comes before the reset.

val retainedReachesOwner: Progress[CloseResetState] = retainAndRoute.leadsTo(
  "retainedReachesOwner"
)(awaitingOwner, ownerKnowsOutcome, within = 2, deliveryFair, recovery)

val retainedWaitsWithoutRecovery: Progress[CloseResetState] = retainAndRoute.leadsTo(
  "retainedWaitsWithoutRecovery"
)(awaitingOwner, ownerKnowsOutcome, within = 2, deliveryFair)

val retainedReachesOwnerBoundedRetry: Progress[CloseResetState] = retainAndRouteBoundedRetry
  .leadsTo("retainedReachesOwner")(
    awaitingOwner,
    ownerKnowsOutcome,
    within = 2,
    deliveryFair,
    recovery
  )
