package temporal
package taskqueue

import umpire.*

val seven = Limits(steps = 7, actions = 7, search = 262144)

/** Past the depth of the detailed queue's table and of a design composed with it, which is ten. */
val twelve = Limits(steps = 12, actions = 12, search = 262144)

// ### The crash cuts

/**
 * One crash at each point of the route, and the delivery that must still follow it: after the
 * invocation, after the sync match, after persistence and after a delivery. After the
 * acknowledgment nothing is left to deliver. `anyTotal` is the static combination count of its free
 * `any` Query.
 */
def providerQueries(
    m: Machine[QueueDetail, QueueOutcome, QueueFact],
    anyTotal: Int
) =
  val laws = queueLaws(m)
  val afterInvocation = m
    .scenario("crashAfterInvocation")
    .actions(enqueue, addActivityTask, crash, addActivityTask, persistTask, deliver)
  val afterSyncMatch = m
    .scenario("crashAfterSyncMatch")
    .actions(enqueue, addActivityTask, syncMatch, crash, addActivityTask, syncMatch, deliver)
  val afterPersistence = m
    .scenario("crashAfterPersistence")
    .actions(enqueue, addActivityTask, persistTask, crash, deliver)
  val afterDelivery = m
    .scenario("crashAfterDelivery")
    .actions(enqueue, addActivityTask, persistTask, deliver, crash, deliver)
  val afterAcknowledgment = m
    .scenario("crashAfterAcknowledgment")
    .actions(enqueue, addActivityTask, persistTask, deliver, acknowledge, crash)
  val any = m.scenario("any").free
  Vector(
    query(s"${m.name}.crashAfterInvocation") find laws.delivers in
      afterInvocation limits seven total 180,
    query(s"${m.name}.crashAfterSyncMatch") find laws.delivers in
      afterSyncMatch limits seven total 210,
    query(s"${m.name}.crashAfterPersistence") find laws.delivers in
      afterPersistence limits seven total 150,
    query(s"${m.name}.crashAfterDelivery") find laws.delivers in
      afterDelivery limits seven total 180,
    query(s"${m.name}.crashAfterAcknowledgment") verify laws.committedStays in
      afterAcknowledgment limits seven total 180,
    query verify laws.committedStays in any limits twelve total anyTotal
  )

// Eight bound actions for every provider but the lossy one, which binds storage loss as a ninth.
val matchingQueueQueries = providerQueries(matchingQueue, anyTotal = 2880)
val forgetfulQueueQueries = providerQueries(forgetfulQueue, anyTotal = 2880)
val volatileQueueQueries = providerQueries(volatileQueue, anyTotal = 2880)
val lossyMatchingQueueQueries = providerQueries(lossyMatchingQueue, anyTotal = 3240)

// ### Storage loss

val persistedThenLost =
  lossyMatchingQueue.scenario.actions(enqueue, addActivityTask, persistTask, storageLoss)

val storageLossQuery =
  query("lossyMatchingQueue.storageLoss") find storageLossDrops in
    persistedThenLost limits seven total 120
