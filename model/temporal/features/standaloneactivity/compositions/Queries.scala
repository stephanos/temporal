package temporal
package features.standaloneactivity
package compositions

import umpire.*
import shared.taskqueue.*
import admission.dispatch

/** Over the opaque queue: every claim and path of the design `c`. */
def overQueueQueries(c: Composition[OverQueue]) =
  val claims = overQueueClaims(c)
  val staleDeliveryAfterPause = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.own(_.activity, control(Control.pause)),
    c.synced(_.activity -> attemptStart)
  )
  val admittedBeforePause = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.synced(_.activity -> attemptStart),
    c.own(_.activity, control(Control.pause))
  )
  val duplicateDelivery = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.synced(_.activity -> attemptStart),
    c.synced(_.activity -> attemptStart)
  )
  val any = c.scenario.free
  Vector(
    query(s"${c.name}.staleDelivery") verify claims.notPaused in
      staleDeliveryAfterPause limits three total 432,
    query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
      admittedBeforePause limits three total 432,
    query(s"${c.name}.duplicateDelivery") verify claims.oneActive in
      duplicateDelivery limits three total 432,
    query(s"${c.name}.failedCommit") verify claims.failedCommit in
      duplicateDelivery limits three total 432,
    query verify claims.oneActive in any limits five total 9360
  )

val currentOverQueueQueries = overQueueQueries(currentOverQueue)
val staleOverQueueQueries = overQueueQueries(staleOverQueue)

/** Over the detailed queue: `anyTotal` is the static combination count of the `any` Queries. */
def overMatchingQueries(c: Composition[OverMatching], anyTotal: Int) =
  val claims = overMatchingClaims(c)
  val staleDeliveryAfterPause = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.own(_.queue, addActivityTask),
    c.own(_.queue, persistTask),
    c.own(_.activity, control(Control.pause)),
    c.synced(_.activity -> attemptStart)
  )
  val admittedBeforePause = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.own(_.queue, addActivityTask),
    c.own(_.queue, persistTask),
    c.synced(_.activity -> attemptStart),
    c.own(_.activity, control(Control.pause))
  )
  // The answer to the poller is lost after the commit, so the persisted task is handed out again.
  val deliveredAgainAfterLostAck = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.own(_.queue, addActivityTask),
    c.own(_.queue, persistTask),
    c.synced(_.activity -> attemptStart),
    c.own(_.queue, ackLoss),
    c.synced(_.activity -> attemptStart)
  )
  // A crash after the admission commit, before the acknowledgment: history retries the sync match.
  val crashAfterAdmissionCommit = c.scenario.actions(
    c.synced(_.activity -> dispatch),
    c.own(_.queue, addActivityTask),
    c.own(_.queue, syncMatch),
    c.synced(_.activity -> attemptStart),
    c.own(_.queue, crash),
    c.own(_.queue, addActivityTask),
    c.own(_.queue, syncMatch),
    c.synced(_.activity -> attemptStart)
  )
  val any = c.scenario.free
  Vector(
    query(s"${c.name}.staleDelivery") verify claims.notPaused in
      staleDeliveryAfterPause limits five total 5400,
    query(s"${c.name}.admittedBeforePause") verify claims.notPaused in
      admittedBeforePause limits five total 5400,
    query(s"${c.name}.deliveredAgainAfterLostAck") verify claims.oneActive in
      deliveredAgainAfterLostAck limits seven total 6480,
    query(s"${c.name}.crashAfterAdmissionCommit") verify claims.oneActive in
      crashAfterAdmissionCommit limits eight total 8640,
    query verify claims.oneActive in any limits twelve total anyTotal
  )

val currentOverMatchingQueries = overMatchingQueries(currentOverMatching, anyTotal = 233280)
val staleOverMatchingQueries = overMatchingQueries(staleOverMatching, anyTotal = 233280)
val currentOverLossyMatchingQueries =
  overMatchingQueries(currentOverLossyMatching, anyTotal = 246240)
