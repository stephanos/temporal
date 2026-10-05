/* The bounds, paths and Queries of the close and reset designs. */
package temporal
package features.nexuscaller
package closepolicy

// `Answer` here is the designs' delivery answer.
import umpire.*

val four = Limits(steps = 4, actions = 4, search = 1 << 20)
val five = Limits(steps = 5, actions = 5, search = 1 << 20)

/**
 * Past the depth of every design's table, so a free search that verifies read every step.
 */
val twelve = Limits(steps = 12, actions = 12, search = 1 << 22)

/** Every claim and path of the specimen, declared on one design. */
def designQueries(m: Machine[CloseResetState, Answer, Fact]) =
  val claims = designClaims(m)
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

// ### The two pinned controls and the two promises, over another channel

/** The two promises and the two pinned controls, declared on one design. */
def safetyQueries(m: Machine[CloseResetState, Answer, Fact]) =
  val claims = safetyClaims(m)
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
    query(s"${m.name}.closedThenFinished") verify claims.outcomePreserved in
      closedThenFinished limits four total 26880,
    query verify claims.ackOnlyWhenKept in resetThenDelivered limits four total 20160,
    query verify claims.outcomePreserved in any limits twelve total 806400,
    query verify claims.ackOnlyWhenKept in any limits twelve total 806400
  )

val retainAndRouteBoundedRetryQueries =
  safetyQueries(retainAndRouteBoundedRetry)

// ### The deadline
//
// With a schedule-to-close deadline the state a lost outcome leaves has a step. The timeout that
// ends that wait is found with nothing owed, and is told apart from one that beats a report still
// in flight, which loses nothing the design promised.

def deadlineQueries(m: Machine[CloseResetState, Answer, Fact]) =
  val claims = deadlineClaims(m)
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
