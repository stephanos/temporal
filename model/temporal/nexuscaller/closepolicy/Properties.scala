/* What the close and reset designs promise: the transition and same-step promises their Queries
 * ask, and the conditional progress claims.
 */
package temporal
package nexuscaller
package closepolicy

import umpire.*

// ### What the designs promise
//
// Every promise below is an authored design promise, written once and declared as a Property on
// each design by the shared def that asks it (Queries.scala): a Property belongs to one machine.
// The monitors' promises are in Model.scala, beside the monitors. Two of the Properties are the
// baseline's, completionSucceeds and completionFails, which an open caller keeps as nexusProtocol
// does. A design names its monitors, and a verify also reports the first violation a watching
// monitor meets, whatever Property it asks.

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
