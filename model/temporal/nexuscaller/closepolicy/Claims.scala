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
  before.caller != Caller.closed || after.state.caller != Caller.closed ||
    (after.state.known == before.known && after.state.intent == before.intent)

/** No step, a reset included, undoes what the handler did. */
def handlerEffectIsIrreversible(before: CloseResetState, after: CloseResetStep): Boolean =
  working(before.handler) || after.state.handler == before.handler

/** An outcome a history records is the handler's. */
def knownIsTheHandlersOutcome(after: CloseResetStep): Boolean = after.state.known match
  case Knowledge.original(r)  => after.state.handler == Handler.done(r)
  case Knowledge.successor(r) => after.state.handler == Handler.done(r)
  case _                      => true

def knows(k: Knowledge, r: Resolution): Boolean = k == Knowledge.original(r) ||
  k == Knowledge.successor(r)

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
  before.known == Knowledge.expired || after.state.known != Knowledge.expired ||
    !nothingOwed(before)

val four: Limits = Limits("four", steps = 4, actions = 4, search = 1 << 20)
val five: Limits = Limits("five", steps = 5, actions = 5, search = 1 << 20)

/**
 * Past the depth of every design's table, so a free search that verifies read every step.
 */
val twelve: Limits = Limits("twelve", steps = 12, actions = 12, search = 1 << 22)

/** Every claim and path of the specimen, declared on one design. */
def designQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val frozen = m.property("closedHistoryIsFrozen") holdsAcross closedHistoryIsFrozen
  val irreversible = m.property("handlerEffectIsIrreversible") holdsAcross
    handlerEffectIsIrreversible
  val handlers = m.property("knownIsTheHandlersOutcome") holds knownIsTheHandlersOutcome
  val knowledgeFinal = m.property("knowledgeIsFinal") holdsAcross knowledgeIsFinal
  // The handler's detached work goes on after the close.
  val finishesAfterClose = m.property("finishesAfterClose") when
    handlerFinish(Resolution.succeeded) holds
    (after =>
      after.state.caller == Caller.closed &&
        after.state.handler == Handler.done(Resolution.succeeded)
    )
  // A cancellation's request, its receipt, the handler's effect and the caller's knowledge.
  val requestedButUnreceived = m.property("requestedButUnreceived") when
    requestCancel(Principal.callerWorkflow) holds
    (after =>
      after.state.intent == Intent.requested(Principal.callerWorkflow) &&
        after.state.handler == Handler.running
    )
  val receivedButSucceeded = m.property("receivedButSucceeded") when
    handlerFinish(Resolution.succeeded) holds
    (after =>
      after.state.intent == Intent.requested(Principal.callerWorkflow) &&
        after.state.handler == Handler.done(Resolution.succeeded)
    )
  val canceledButUnknown = m.property("canceledButUnknown") when
    handlerFinish(Resolution.canceled) holds
    (after =>
      after.state.handler == Handler.done(Resolution.canceled) &&
        !ownerKnows(after.state, Resolution.canceled)
    )
  val completionCancels = m.property("completionCancels") when complete(Resolution.canceled) holds
    (after =>
      ownerKnows(after.state, Resolution.canceled) &&
        after.facts.contains(Fact.nexusOperationCanceled)
    )
  // The baseline's two: an open caller records a completion by the baseline's event.
  val completionSucceeds = m.property("completionSucceeds") when
    complete(Resolution.succeeded) holds
    (after => after.facts.contains(Fact.nexusOperationCompleted))
  val completionFails = m.property("completionFails") when complete(Resolution.failed) holds
    (after => after.facts.contains(Fact.nexusOperationFailed))
  // The two answers of a closed run, and the two resets, are kept apart.
  val rejectedTransiently = m.property("rejectedTransiently") when
    complete(Resolution.succeeded) holds
    (after =>
      after.outcome == Answer.rejectedTransient &&
        carries(after.state.channel, Resolution.succeeded)
    )
  val rejectedPermanently = m.property("rejectedPermanently") when
    complete(Resolution.succeeded) holds (after => after.outcome == Answer.rejectedPermanent)
  val lostAfterReset = m.property("lostAfterReset") when reset holds
    (after => !outcomePreserved(after))
  val reappliesRetained = m.property("reappliesRetained") when reset holds
    (after =>
      after.facts.contains(Fact.outcomeReapplied) &&
        after.state.known == Knowledge.successor(Resolution.succeeded)
    )
  val routedToSuccessor = m.property("routedToSuccessor") when complete(Resolution.failed) holds
    (after => after.state.known == Knowledge.successor(Resolution.failed))
  val closedThenFinished = m
    .scenario("closedThenFinished")
    .starts(opened)
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m
    .scenario("resetThenDelivered")
    .starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  // The request, its delivery to the handler and the handler's effect are three steps here.
  val canceledAcrossReset = m
    .scenario("canceledAcrossReset")
    .starts(opened)
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled),
      reset,
      complete(Resolution.canceled)
    )
  val ackedThenReset = m
    .scenario("ackedThenReset")
    .starts(opened)
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val duplicateCompletion = m
    .scenario("duplicateCompletion")
    .starts(opened)
    .actions(
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      complete(Resolution.succeeded)
    )
  // A reset between the commit and its acknowledgment, and between a rejection and the retry.
  val resetBetweenDeliveries = m
    .scenario("resetBetweenDeliveries")
    .starts(opened)
    .actions(
      handlerFinish(Resolution.failed),
      complete(Resolution.failed),
      reset,
      complete(Resolution.failed)
    )
  val detachedWork = m
    .scenario("detachedWork")
    .starts(opened)
    .actions(callerClose, handlerFinish(Resolution.succeeded))
  val cancelRequested = m
    .scenario("cancelRequested")
    .starts(opened)
    .actions(requestCancel(Principal.callerWorkflow))
  val cancelReceivedThenSucceeded = m
    .scenario("cancelReceivedThenSucceeded")
    .starts(opened)
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.succeeded)
    )
  val cancelReceivedThenCanceled = m
    .scenario("cancelReceivedThenCanceled")
    .starts(opened)
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled)
    )
  val canceledThenDelivered = m
    .scenario("canceledThenDelivered")
    .starts(opened)
    .actions(
      requestCancel(Principal.callerWorkflow),
      deliverCancel,
      handlerFinish(Resolution.canceled),
      complete(Resolution.canceled)
    )
  val finishedThenSucceeded = m
    .scenario("finishedThenSucceeded")
    .starts(opened)
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded))
  val finishedThenFailed = m
    .scenario("finishedThenFailed")
    .starts(opened)
    .actions(handlerFinish(Resolution.failed), complete(Resolution.failed))
  val deliveredToClosed = m
    .scenario("deliveredToClosed")
    .starts(opened)
    .actions(callerClose, handlerFinish(Resolution.succeeded), complete(Resolution.succeeded))
  val any = m.scenario("any").starts(opened).free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query(s"${m.name}.resetThenDelivered.ackOnlyWhenKept").verify(acked) in
      resetThenDelivered limits four total 20160,
    query(s"${m.name}.resetThenDelivered.outcomePreserved").verify(preserved) in
      resetThenDelivered limits four total 20160,
    query(s"${m.name}.canceledAcrossReset") verify preserved in
      canceledAcrossReset limits five total 33600,
    query(s"${m.name}.ackedThenReset") verify preserved in ackedThenReset limits four total 20160,
    query(s"${m.name}.duplicateCompletion").verify(knowledgeFinal) in duplicateCompletion limits
      four total 20160,
    query(s"${m.name}.resetBetweenCommitAndAcknowledgment").verify(acked) in
      resetBetweenDeliveries limits four total 26880,
    query(s"${m.name}.any.outcomePreserved") verify preserved in any limits twelve total 806400,
    query(s"${m.name}.any.ackOnlyWhenKept") verify acked in any limits twelve total 806400,
    query(s"${m.name}.any.closedHistoryIsFrozen") verify frozen in any limits twelve total 806400,
    query(s"${m.name}.any.handlerEffectIsIrreversible") verify irreversible in
      any limits twelve total 806400,
    query(s"${m.name}.any.knownIsTheHandlersOutcome") verify handlers in
      any limits twelve total 806400,
    query(s"${m.name}.any.knowledgeIsFinal") verify knowledgeFinal in
      any limits twelve total 806400,
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

val rejectAfterCloseQueries: Vector[Query] = designQueries(rejectAfterCloseDesign)
val ackByOriginalQueries: Vector[Query] = designQueries(ackByOriginalDesign)
val retainAndRouteQueries: Vector[Query] = designQueries(retainAndRouteDesign)
val forgetsCancelOnResetQueries: Vector[Query] = designQueries(forgetsCancelOnResetDesign)
val truncatesOnResetQueries: Vector[Query] = designQueries(truncatesOnResetDesign)

// ### The two pinned controls and the two promises, over another channel

/** The two promises and the two pinned controls, declared on one design. */
def safetyQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val closedThenFinished = m
    .scenario("closedThenFinished")
    .starts(opened)
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m
    .scenario("resetThenDelivered")
    .starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val any = m.scenario("any").starts(opened).free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query(s"${m.name}.resetThenDelivered.ackOnlyWhenKept").verify(acked) in
      resetThenDelivered limits four total 20160,
    query(s"${m.name}.any.outcomePreserved") verify preserved in any limits twelve total 806400,
    query(s"${m.name}.any.ackOnlyWhenKept") verify acked in any limits twelve total 806400
  )

val retainAndRouteBoundedRetryQueries: Vector[Query] =
  safetyQueries(retainAndRouteBoundedRetryDesign)

// ### The deadline
//
// With a schedule-to-close deadline the state a lost outcome leaves has a step. The timeout that
// ends that wait is found with nothing owed, and is told apart from one that beats a report still
// in flight, which loses nothing the design promised.

def deadlineQueries(m: Machine[CloseResetState, Answer, Fact]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val necessary = m.property("noUnnecessaryWait") holdsAcross noUnnecessaryWait
  val expiresWithNothingOwed = m.property("expiresWithNothingOwed") when scheduleToClose holds
    (after => after.state.known == Knowledge.expired && nothingOwed(after.state))
  val expiresWhileOwed = m.property("expiresWhileOwed") when scheduleToClose holds
    (after =>
      after.state.known == Knowledge.expired && carries(after.state.channel, Resolution.succeeded)
    )
  val lateCompletionIsDropped = m.property("lateCompletionIsDropped") when
    complete(Resolution.succeeded) holds
    (after =>
      after.outcome == Answer.rejectedPermanent && after.facts.contains(Fact.completionDropped)
    )
  val closedThenFinished = m
    .scenario("closedThenFinished")
    .starts(opened)
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset
    )
  val resetThenDelivered = m
    .scenario("resetThenDelivered")
    .starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val closedLossThenExpired = m
    .scenario("closedLossThenExpired")
    .starts(opened)
    .actions(
      callerClose,
      handlerFinish(Resolution.succeeded),
      complete(Resolution.succeeded),
      reset,
      scheduleToClose
    )
  val resetLossThenExpired = m
    .scenario("resetLossThenExpired")
    .starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed), scheduleToClose)
  val reportedThenExpired = m
    .scenario("reportedThenExpired")
    .starts(opened)
    .actions(handlerFinish(Resolution.succeeded), scheduleToClose)
  val expiredThenDelivered = m
    .scenario("expiredThenDelivered")
    .starts(opened)
    .actions(handlerFinish(Resolution.succeeded), scheduleToClose, complete(Resolution.succeeded))
  val any = m.scenario("any").starts(opened).free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in
      closedThenFinished limits four total 26880,
    query(s"${m.name}.resetThenDelivered.ackOnlyWhenKept").verify(acked) in
      resetThenDelivered limits four total 20160,
    query(s"${m.name}.any.outcomePreserved") verify preserved in any limits twelve total 887040,
    query(s"${m.name}.any.ackOnlyWhenKept") verify acked in any limits twelve total 887040,
    query(s"${m.name}.any.noUnnecessaryWait") verify necessary in any limits twelve total 887040,
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
  deadlineQueries(rejectAfterCloseWithDeadlineDesign)
val ackByOriginalWithDeadlineQueries: Vector[Query] =
  deadlineQueries(ackByOriginalWithDeadlineDesign)
val retainAndRouteWithDeadlineQueries: Vector[Query] =
  deadlineQueries(retainAndRouteWithDeadlineDesign)

// ### Conditional progress
//
// From a decided outcome, a state a path may end in follows within six steps, under the assumptions
// each claim and its machine name. A machine that names no deadline has no timer, and one that does
// not assume the redelivery bound may be rejected transiently forever.

val rejectAfterCloseProgress: Progress[CloseResetState] = rejectAfterCloseDesign.leadsTo(
  "outcomeReachesOwner"
)(s => isDone(s), s => settled(s), within = 6, reporting, deliveryFair)

val ackByOriginalProgress: Progress[CloseResetState] = ackByOriginalDesign.leadsTo(
  "outcomeReachesOwner"
)(s => isDone(s), s => settled(s), within = 6, reporting, deliveryFair)

val retainAndRouteProgress: Progress[CloseResetState] = retainAndRouteDesign.leadsTo(
  "outcomeReachesOwner"
)(s => isDone(s), s => settled(s), within = 6, reporting, deliveryFair)

val retainAndRouteBoundedRetryProgress: Progress[CloseResetState] = retainAndRouteBoundedRetryDesign
  .leadsTo("outcomeReachesOwner")(
    s => isDone(s),
    s => settled(s),
    within = 6,
    reporting,
    deliveryFair
  )

val rejectAfterCloseWithDeadlineProgress: Progress[CloseResetState] =
  rejectAfterCloseWithDeadlineDesign.leadsTo("outcomeReachesOwner")(
    s => isDone(s),
    s => settled(s),
    within = 6,
    reporting,
    deliveryFair
  )

val ackByOriginalWithDeadlineProgress: Progress[CloseResetState] = ackByOriginalWithDeadlineDesign
  .leadsTo("outcomeReachesOwner")(
    s => isDone(s),
    s => settled(s),
    within = 6,
    reporting,
    deliveryFair
  )

val retainAndRouteWithDeadlineProgress: Progress[CloseResetState] = retainAndRouteWithDeadlineDesign
  .leadsTo("outcomeReachesOwner")(
    s => isDone(s),
    s => settled(s),
    within = 6,
    reporting,
    deliveryFair
  )

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

val retainedReachesOwner: Progress[CloseResetState] = retainAndRouteDesign.leadsTo(
  "retainedReachesOwner"
)(s => awaitingOwner(s), s => ownerKnowsOutcome(s), within = 2, deliveryFair, recovery)

val retainedWaitsWithoutRecovery: Progress[CloseResetState] = retainAndRouteDesign.leadsTo(
  "retainedWaitsWithoutRecovery"
)(s => awaitingOwner(s), s => ownerKnowsOutcome(s), within = 2, deliveryFair)

val retainedReachesOwnerBoundedRetry: Progress[CloseResetState] = retainAndRouteBoundedRetryDesign
  .leadsTo("retainedReachesOwner")(
    s => awaitingOwner(s),
    s => ownerKnowsOutcome(s),
    within = 2,
    deliveryFair,
    recovery
  )
