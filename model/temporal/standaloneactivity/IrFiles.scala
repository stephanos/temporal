/* The checked-in IR files of the standalone activity Models (umpire.irFile). */
package temporal
package standaloneactivity

import umpire.*

// The activity Model. Its cross-entity Query, stoppedWorkerStartsNothing, carries the composition
// and its claim.
val activityFile = irFile("activity")(
  standaloneActivity,
  activityProduct,
  Functional.all,
  productCapabilities,
  protocolCapabilities,
  cancelRequest,
  stoppedWorkerStartsNothing,
  ActivityRealization.standalone
)

// Its system contract, the admission designs, and the shared task queue's providers it composes. A
// composition no Query runs over is a root of its own.
val activitySystemFile = irFile("activity-system")(
  admission.currentQueries,
  admission.staleQueries,
  competingTimers,
  taskqueue.matchingQueueQueries,
  taskqueue.forgetfulQueueQueries,
  taskqueue.volatileQueueQueries,
  taskqueue.lossyMatchingQueueQueries,
  taskqueue.storageLossQuery,
  compositions.currentOverQueueQueries,
  compositions.staleOverQueueQueries,
  compositions.currentOverMatchingQueries,
  compositions.staleOverMatchingQueries,
  compositions.currentOverLossyMatchingQueries,
  compositions.currentOverForgetful,
  compositions.currentOverVolatile
)

// The held race a server is run through, and the realization that runs it. It is a Model of its
// own, so the system contract's Queries are the ones its checkers were given.
val activityRaceFile = irFile("activity-race")(
  admission.heldStaleDelivery,
  ActivityRealization.heldDelivery,
  admission.lostAdmissionResponseQuery,
  ActivityRealization.lostAdmissionResponse
)
