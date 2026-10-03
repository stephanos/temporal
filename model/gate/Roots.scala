package umpire.gate

/**
 * The declarations the gate lifts into each checked-in IR file of model/ir: the file's name, and the
 * fully qualified names of the `val`s that are its roots.
 */
object Roots:
  // The functional Queries and the realization that runs them are roots beside the machines: Go
  // lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
  // The product claim read on a protocol path and the cross-entity Query, which carries the
  // composition with the handler's worker and its claim, are roots too.
  private val nexusCaller = Seq(
    "temporal.nexuscaller.Model$package$.nexusProduct",
    "temporal.nexuscaller.Model$package$.nexusProtocol",
    "temporal.nexuscaller.Model$package$.handlerWorker",
    "temporal.nexuscaller.Model$package$.nexusCaller",
    "temporal.worker.Worker$package$.polling",
    "temporal.nexuscaller.Claims$package$.functionalQueries",
    "temporal.nexuscaller.Claims$package$.terminalHolds",
    "temporal.nexuscaller.Claims$package$.stoppedWorkerRepliesNothing",
    "temporal.nexuscaller.NexusRealization$.asyncNexus"
  )
  private val nexusControl = Seq(
    "temporal.nexuscaller.Control$.forgedCompletion",
    "temporal.nexuscaller.NexusRealization$.forgedCompletion"
  )

  // The standalone activity Model: ir/activity.json. Its cross-entity Query,
  // stoppedWorkerStartsNothing, carries the composition and its claim.
  private val activity = Seq(
    "temporal.standaloneactivity.Model$package$.standaloneActivity",
    "temporal.standaloneactivity.Model$package$.activityProduct",
    "temporal.standaloneactivity.Claims$package$.functionalQueries",
    "temporal.standaloneactivity.Claims$package$.terminalHolds",
    "temporal.standaloneactivity.Claims$package$.pauseHolds",
    "temporal.standaloneactivity.Claims$package$.cancelRequest",
    "temporal.standaloneactivity.Claims$package$.stoppedWorkerStartsNothing",
    "temporal.standaloneactivity.ActivityRealization$.standalone"
  )

  // Its system contract, the admission designs and the dispatch queue's providers:
  // ir/activity-system.json. A composition no Query runs over is a root of its own.
  private val activitySystem = Seq(
    "currentQueries",
    "staleQueries",
    "competingTimers",
    "matchingQueueQueries",
    "forgetfulQueueQueries",
    "volatileQueueQueries",
    "lossyMatchingQueueQueries",
    "storageLossQuery",
    "currentOverQueueQueries",
    "staleOverQueueQueries",
    "currentOverMatchingQueries",
    "staleOverMatchingQueries",
    "currentOverLossyMatchingQueries",
    "currentOverForgetful",
    "currentOverVolatile"
  ).map("temporal.standaloneactivity.System$package$." + _)

  // The held race a server is run through, and the realization that runs it: ir/activity-race.json.
  // It is a Model of its own, so the system contract's Queries are the ones its checkers were given.
  private val activityRace = Seq(
    "temporal.standaloneactivity.System$package$.heldStaleDelivery",
    "temporal.standaloneactivity.ActivityRealization$.heldDelivery",
    "temporal.standaloneactivity.System$package$.lostAdmissionResponseQuery",
    "temporal.standaloneactivity.ActivityRealization$.lostAdmissionResponse"
  )

  // The Nexus caller close and reset designs: ir/nexus-close.json. Each design's Queries are a root,
  // and so is each progress claim.
  private val nexusClose = Seq(
    "rejectAfterCloseQueries",
    "ackByOriginalQueries",
    "retainAndRouteQueries",
    "forgetsCancelOnResetQueries",
    "truncatesOnResetQueries",
    "retainAndRouteBoundedRetryQueries",
    "rejectAfterCloseWithDeadlineQueries",
    "ackByOriginalWithDeadlineQueries",
    "retainAndRouteWithDeadlineQueries",
    "rejectAfterCloseProgress",
    "ackByOriginalProgress",
    "retainAndRouteProgress",
    "retainAndRouteBoundedRetryProgress",
    "rejectAfterCloseWithDeadlineProgress",
    "ackByOriginalWithDeadlineProgress",
    "retainAndRouteWithDeadlineProgress",
    "retainedReachesOwner",
    "retainedWaitsWithoutRecovery",
    "retainedReachesOwnerBoundedRetry"
  ).map("temporal.nexuscaller.closepolicy.Claims$package$." + _)

  val ir: Seq[(String, Seq[String])] = Seq(
    "nexus-caller.json" -> nexusCaller,
    "nexus-control.json" -> nexusControl,
    "activity.json" -> activity,
    "activity-system.json" -> activitySystem,
    "activity-race.json" -> activityRace,
    "nexus-close.json" -> nexusClose
  )
