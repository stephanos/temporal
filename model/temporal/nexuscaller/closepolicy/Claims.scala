package temporal
package nexuscaller
package closepolicy

// `Answer` here is the designs' delivery answer.
import umpire.*

// ### What the designs promise
//
// Every Property below is an authored design promise, written once and declared on each design: a
// Property belongs to one machine. Two are the baseline's, completionSucceeds and completionFails,
// which an open caller keeps as nexusProtocol does. A design names its monitors, and a verify also
// reports the first violation a watching monitor meets, whatever Property it asks.

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

val four = Limits(steps = 4, actions = 4, search = 1 << 20)
val five = Limits(steps = 5, actions = 5, search = 1 << 20)

/**
 * Past the depth of every design's table, so a free search that verifies read every step.
 */
val twelve = Limits(steps = 12, actions = 12, search = 1 << 22)

/** Every claim and path of the specimen, declared on one design. */
def designQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val frozen = m.property("closedHistoryIsFrozen") holdsAcross closedHistoryIsFrozen
  // No step, a reset included, undoes what the handler did.
  val handlerEffectIsIrreversible = m.property.once(isDone).keeps(_.handler)
  val handlers = m.property("knownIsTheHandlersOutcome") holds knownIsTheHandlersOutcome
  val knowledgeFinal = m.property("knowledgeIsFinal") holdsAcross knowledgeIsFinal
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
  val closedThenFinished = m.scenario
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m.scenario
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  // The request, its delivery to the handler and the handler's effect are three steps here.
  val canceledAcrossReset = m.scenario
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled),
      reset,
      complete(Resolution.canceled)
    )
  val ackedThenReset = m.scenario
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val duplicateCompletion = m.scenario
    .actions(
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      complete(Resolution.succeeded)
    )
  // A reset between the commit and its acknowledgment, and between a rejection and the retry.
  val resetBetweenDeliveries = m.scenario
    .actions(
      handlerFinish(Resolution.failed),
      complete(Resolution.failed),
      reset,
      complete(Resolution.failed)
    )
  val detachedWork = m.scenario
    .actions(callerClose, handlerFinish(Resolution.succeeded))
  val cancelRequested = m.scenario
    .actions(requestCancel(Principal.callerWorkflow))
  val cancelReceivedThenSucceeded = m.scenario
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.succeeded)
    )
  val cancelReceivedThenCanceled = m.scenario
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled)
    )
  val canceledThenDelivered = m.scenario
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled),
      complete(Resolution.canceled)
    )
  val finishedThenSucceeded = m.scenario
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded))
  val finishedThenFailed = m.scenario
    .actions(handlerFinish(Resolution.failed), complete(Resolution.failed))
  val deliveredToClosed = m.scenario
    .actions(callerClose, handlerFinish(Resolution.succeeded), complete(Resolution.succeeded))
  val any = m.scenario.free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query verify acked in resetThenDelivered limits four total 20160,
    query verify preserved in resetThenDelivered limits four total 20160,
    query(s"${m.name}.canceledAcrossReset") verify preserved in
      canceledAcrossReset limits five total 33600,
    query(s"${m.name}.ackedThenReset") verify preserved in ackedThenReset limits four total 20160,
    query(s"${m.name}.duplicateCompletion").verify(knowledgeFinal) in duplicateCompletion limits
      four total 20160,
    query(s"${m.name}.resetBetweenCommitAndAcknowledgment").verify(acked) in
      resetBetweenDeliveries limits four total 26880,
    query verify preserved in any limits twelve total 806400,
    query verify acked in any limits twelve total 806400,
    query verify frozen in any limits twelve total 806400,
    query verify handlerEffectIsIrreversible in any limits twelve total 806400,
    query verify handlers in any limits twelve total 806400,
    query verify knowledgeFinal in any limits twelve total 806400,
    query(s"${m.name}.detachedWorkProceeds") find finishesAfterClose in
      detachedWork limits four total 13440,
    query(s"${m.name}.intentWithoutReceipt").find(requestedButUnreceived) in cancelRequested limits
      four total 6720,
    query(s"${m.name}.receiptWithoutEffect").find(receivedButSucceeded) in
      cancelReceivedThenSucceeded limits four total 20160,
    query(s"${m.name}.effectWithoutKnowledge").find(canceledButUnknown) in
      cancelReceivedThenCanceled limits four total 20160,
    query(s"${m.name}.canceledIsKnown") find completionCancels in
      canceledThenDelivered limits four total 26880,
    query(s"${m.name}.asyncCompletion").find(completionSucceeds) in finishedThenSucceeded limits
      four total 13440,
    query(s"${m.name}.asyncFailure") find completionFails in
      finishedThenFailed limits four total 13440,
    query(s"${m.name}.transientRejectionAfterClose").find(rejectedTransiently) in
      deliveredToClosed limits four total 20160,
    query(s"${m.name}.permanentRejectionAfterClose").find(rejectedPermanently) in
      deliveredToClosed limits four total 20160,
    query(s"${m.name}.lostAfterReset") find lostAfterReset in
      closedThenFinished limits four total 26880,
    query(s"${m.name}.resetAfterRetention").find(reappliesRetained) in closedThenFinished limits
      four total 26880,
    query(s"${m.name}.resetBeforeRetention").find(routedToSuccessor) in resetThenDelivered limits
      four total 20160
  )

val rejectAfterCloseQueries: Vector[Query] = designQueries(rejectAfterClose)
val ackByOriginalQueries: Vector[Query] = designQueries(ackByOriginal)
val retainAndRouteQueries: Vector[Query] = designQueries(retainAndRoute)
val forgetsCancelOnResetQueries: Vector[Query] = designQueries(forgetsCancelOnReset)
val truncatesOnResetQueries: Vector[Query] = designQueries(truncatesOnReset)

// ### The two pinned controls and the two promises, over another channel

/** The two promises and the two pinned controls, declared on one design. */
def safetyQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val closedThenFinished = m.scenario
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m.scenario
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val any = m.scenario.free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query verify acked in resetThenDelivered limits four total 20160,
    query verify preserved in any limits twelve total 806400,
    query verify acked in any limits twelve total 806400
  )

val retainAndRouteBoundedRetryQueries: Vector[Query] =
  safetyQueries(retainAndRouteBoundedRetry)

// ### The deadline
//
// With a schedule-to-close deadline the state a lost outcome leaves has a step. The timeout that
// ends that wait is found with nothing owed, and is told apart from one that beats a report still
// in flight, which loses nothing the design promised.

def deadlineQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val necessary = m.property("noUnnecessaryWait") holdsAcross noUnnecessaryWait
  val expiresWithNothingOwed = m.property when scheduleToClose holds
    (after => after.state.known == Knowledge.expired && nothingOwed(after.state))
  val expiresWhileOwed = m.property when scheduleToClose holds
    (after =>
      after.state.known == Knowledge.expired && carries(after.state.channel, Resolution.succeeded)
    )
  val lateCompletionIsDropped = m.property when
    complete(Resolution.succeeded) holds
    (after => after.outcome == Answer.rejectedPermanent && after.records(Fact.completionDropped))
  val closedThenFinished = m.scenario
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m.scenario
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val closedLossThenExpired = m.scenario
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset,
      scheduleToClose
    )
  val resetLossThenExpired = m.scenario
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed), scheduleToClose)
  val reportedThenExpired = m.scenario
    .actions(handlerFinish(Resolution.succeeded), scheduleToClose)
  val expiredThenDelivered = m.scenario
    .actions(handlerFinish(Resolution.succeeded), scheduleToClose, complete(Resolution.succeeded))
  val any = m.scenario.free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query verify acked in resetThenDelivered limits four total 20160,
    query verify preserved in any limits twelve total 887040,
    query verify acked in any limits twelve total 887040,
    query verify necessary in any limits twelve total 887040,
    query(s"${m.name}.expiredAfterClosedLoss").find(expiresWithNothingOwed) in
      closedLossThenExpired limits five total 33600,
    query(s"${m.name}.expiredAfterResetLoss").find(expiresWithNothingOwed) in
      resetLossThenExpired limits five total 26880,
    query(s"${m.name}.expiredWhileReported").find(expiresWhileOwed) in reportedThenExpired limits
      four total 13440,
    query(s"${m.name}.lateCompletionIsDropped").find(lateCompletionIsDropped) in
      expiredThenDelivered limits four total 20160
  )

val rejectAfterCloseWithDeadlineQueries: Vector[Query] =
  deadlineQueries(rejectAfterCloseWithDeadline)
val ackByOriginalWithDeadlineQueries: Vector[Query] =
  deadlineQueries(ackByOriginalWithDeadline)
val retainAndRouteWithDeadlineQueries: Vector[Query] =
  deadlineQueries(retainAndRouteWithDeadline)

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
