package temporal

import framework.*
import munit.FunSuite

class PropertyGroupMigrationTest extends FunSuite:
  private lazy val safety: List[(String, Property[?])] = List(
    "activityProtocol.cancelRequested" -> features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.cancelRequested,
    "activityProtocol.canceledByWorker" -> features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.canceledByWorker,
    "activityProtocol.canceledByID" -> features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.canceledByID,
    "activityProtocol.completes" -> features.activity.standalone.protocol.ActivityProtocol.completion.properties.safety.completes,
    "activityProtocol.dispatchRequiresReady" -> features.activity.standalone.protocol.ActivityProtocol.dispatching.properties.safety.dispatchRequiresReady,
    "activityProtocol.scheduleToStartRequiresDispatch" -> features.activity.standalone.protocol.ActivityProtocol.dispatching.properties.safety.scheduleToStartRequiresDispatch,
    "activityProtocol.cancelIsNotUndone" -> features.activity.standalone.protocol.ActivityProtocol.properties.safety.cancelIsNotUndone,
    "activityProtocol.resetSettles" -> features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetSettles,
    "activityProtocol.resetResumes" -> features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetResumes,
    "activityProtocol.resetKeepsPaused" -> features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetKeepsPaused,
    "activityProtocol.controlPrecedence" -> features.activity.standalone.protocol.ActivityProtocol.properties.safety.controlPrecedence,
    "activityProtocol.resetStaysPending" -> features.activity.standalone.protocol.ActivityProtocol.reset.properties.safety.resetStaysPending,
    "activityProtocol.retryExhausts" -> features.activity.standalone.protocol.ActivityProtocol.retry.properties.safety.retryExhausts,
    "standaloneActivity.startedByPollingWorker" -> features.activity.standalone.protocol.StandaloneActivity.properties.safety.startedByPollingWorker,
    "activityProtocol.scheduleToStartFires" -> features.activity.standalone.protocol.ActivityProtocol.timeouts.properties.safety.scheduleToStartFires,
    "activityProtocol.scheduleToCloseTimesOut" -> features.activity.standalone.protocol.ActivityProtocol.timeouts.properties.safety.scheduleToCloseTimesOut,
    "nexusProduct.terminalIsFinal" -> features.nexus.shared.product.NexusProduct.properties.safety.terminalIsFinal,
    "nexusCaller.repliedByPollingWorker" -> features.nexus.workflow.protocol.NexusCaller.properties.safety.repliedByPollingWorker,
    "nexusProtocol.asyncStarts" -> features.nexus.workflow.protocol.NexusProtocol.properties.safety.asyncStarts,
    "nexusProtocol.completionSucceeds" -> features.nexus.workflow.protocol.NexusProtocol.properties.safety.completionSucceeds
  )

  private lazy val liveness: List[Progress[?]] = List(
    features.activity.standalone.protocol.ServedDispatch.properties.liveness.startsWithinOneStep,
    features.activity.standalone.protocol.UnservedDispatch.properties.liveness.startsWithinOneStep,
    features.nexus.workflow.protocol.AckByOriginal.properties.liveness.ackByOriginalProgress,
    features.nexus.workflow.protocol.AckByOriginalWithDeadline.properties.liveness.ackByOriginalWithDeadlineProgress,
    features.nexus.workflow.protocol.RejectAfterClose.properties.liveness.rejectAfterCloseProgress,
    features.nexus.workflow.protocol.RejectAfterCloseWithDeadline.properties.liveness.rejectAfterCloseWithDeadlineProgress,
    features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainAndRouteProgress,
    features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainedReachesOwner,
    features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainedWaitsWithoutRecovery,
    features.nexus.workflow.protocol.RetainAndRouteBoundedRetry.properties.liveness.retainAndRouteBoundedRetryProgress,
    features.nexus.workflow.protocol.RetainAndRouteBoundedRetry.properties.liveness.retainedReachesOwnerBoundedRetry,
    features.nexus.workflow.protocol.RetainAndRouteWithDeadline.properties.liveness.retainAndRouteWithDeadlineProgress
  )

  test("every authored direct claim is initialized once in its declared group") {
    assertEquals(safety.size, 20)
    assertEquals(safety.map(_._2.decl).distinct.size, safety.size)
    assertEquals(
      safety.map((name, p) => s"${p.decl.machine.name}.${name.split('.').last}").sorted,
      List(
        "activityProtocol.cancelIsNotUndone",
        "activityProtocol.cancelRequested",
        "activityProtocol.canceledByID",
        "activityProtocol.canceledByWorker",
        "activityProtocol.completes",
        "activityProtocol.controlPrecedence",
        "activityProtocol.dispatchRequiresReady",
        "activityProtocol.resetKeepsPaused",
        "activityProtocol.resetResumes",
        "activityProtocol.resetSettles",
        "activityProtocol.resetStaysPending",
        "activityProtocol.retryExhausts",
        "activityProtocol.scheduleToCloseTimesOut",
        "activityProtocol.scheduleToStartFires",
        "activityProtocol.scheduleToStartRequiresDispatch",
        "nexusCaller.repliedByPollingWorker",
        "nexusProduct.terminalIsFinal",
        "nexusProtocol.asyncStarts",
        "nexusProtocol.completionSucceeds",
        "standaloneActivity.startedByPollingWorker"
      )
    )

    assertEquals(liveness.size, 12)
    assertEquals(liveness.distinct.size, liveness.size)
    assertEquals(
      liveness.map(p => s"${p.machine.name}.${p.name}" -> p.within).sorted,
      List(
        "ackByOriginal.outcomeReachesOwner" -> 6,
        "ackByOriginalWithDeadline.outcomeReachesOwner" -> 6,
        "rejectAfterClose.outcomeReachesOwner" -> 6,
        "rejectAfterCloseWithDeadline.outcomeReachesOwner" -> 6,
        "retainAndRoute.outcomeReachesOwner" -> 6,
        "retainAndRoute.retainedReachesOwner" -> 2,
        "retainAndRoute.retainedWaitsWithoutRecovery" -> 2,
        "retainAndRouteBoundedRetry.outcomeReachesOwner" -> 6,
        "retainAndRouteBoundedRetry.retainedReachesOwner" -> 2,
        "retainAndRouteWithDeadline.outcomeReachesOwner" -> 6,
        "servedDispatch.startsWithinOneStep" -> 1,
        "unservedDispatch.startsWithinOneStep" -> 1
      )
    )
    assertEquals(
      liveness.map(_.assumptions),
      List(
        List(features.activity.standalone.protocol.servedDispatchRuns),
        List(features.activity.standalone.protocol.unservedDispatchRuns),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.deliveryFair, features.nexus.workflow.protocol.recovery),
        List(features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair),
        List(features.nexus.workflow.protocol.deliveryFair, features.nexus.workflow.protocol.recovery),
        List(features.nexus.workflow.protocol.reporting, features.nexus.workflow.protocol.deliveryFair)
      )
    )
  }
