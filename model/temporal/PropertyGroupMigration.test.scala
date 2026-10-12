package temporal

import framework.*
import munit.FunSuite

class PropertyGroupMigrationTest extends FunSuite:
  private lazy val safety: List[(String, Property[?])] = List(
    "activityProtocol.cancelRequested" -> temporal.features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.cancelRequested,
    "activityProtocol.canceledByWorker" -> temporal.features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.canceledByWorker,
    "activityProtocol.canceledByID" -> temporal.features.activity.standalone.protocol.ActivityProtocol.cancellation.properties.safety.canceledByID,
    "activityProtocol.completes" -> temporal.features.activity.standalone.protocol.ActivityProtocol.completion.properties.safety.completes,
    "activityProtocol.dispatchRequiresReady" -> temporal.features.activity.standalone.protocol.ActivityProtocol.dispatching.properties.safety.dispatchRequiresReady,
    "activityProtocol.scheduleToStartRequiresDispatch" -> temporal.features.activity.standalone.protocol.ActivityProtocol.dispatching.properties.safety.scheduleToStartRequiresDispatch,
    "activityProtocol.cancelIsNotUndone" -> temporal.features.activity.standalone.protocol.ActivityProtocol.properties.safety.cancelIsNotUndone,
    "activityProtocol.resetSettles" -> temporal.features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetSettles,
    "activityProtocol.resetResumes" -> temporal.features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetResumes,
    "activityProtocol.resetKeepsPaused" -> temporal.features.activity.standalone.protocol.ActivityProtocol.properties.safety.resetKeepsPaused,
    "activityProtocol.controlPrecedence" -> temporal.features.activity.standalone.protocol.ActivityProtocol.properties.safety.controlPrecedence,
    "activityProtocol.resetStaysPending" -> temporal.features.activity.standalone.protocol.ActivityProtocol.reset.properties.safety.resetStaysPending,
    "activityProtocol.retryExhausts" -> temporal.features.activity.standalone.protocol.ActivityProtocol.retry.properties.safety.retryExhausts,
    "standaloneActivity.startedByPollingWorker" -> temporal.features.activity.standalone.protocol.StandaloneActivity.properties.safety.startedByPollingWorker,
    "activityProtocol.scheduleToStartFires" -> temporal.features.activity.standalone.protocol.ActivityProtocol.timeouts.properties.safety.scheduleToStartFires,
    "activityProtocol.scheduleToCloseTimesOut" -> temporal.features.activity.standalone.protocol.ActivityProtocol.timeouts.properties.safety.scheduleToCloseTimesOut,
    "nexusProduct.terminalIsFinal" -> temporal.features.nexus.shared.product.NexusProduct.properties.safety.terminalIsFinal,
    "nexusCaller.repliedByPollingWorker" -> temporal.features.nexus.workflow.protocol.NexusCaller.properties.safety.repliedByPollingWorker,
    "nexusProtocol.asyncStarts" -> temporal.features.nexus.workflow.protocol.NexusProtocol.properties.safety.asyncStarts,
    "nexusProtocol.completionSucceeds" -> temporal.features.nexus.workflow.protocol.NexusProtocol.properties.safety.completionSucceeds
  )

  private lazy val liveness: List[Progress[?]] = List(
    temporal.features.activity.standalone.protocol.ServedDispatch.properties.liveness.startsWithinOneStep,
    temporal.features.activity.standalone.protocol.UnservedDispatch.properties.liveness.startsWithinOneStep,
    temporal.features.nexus.workflow.protocol.AckByOriginal.properties.liveness.ackByOriginalProgress,
    temporal.features.nexus.workflow.protocol.AckByOriginalWithDeadline.properties.liveness.ackByOriginalWithDeadlineProgress,
    temporal.features.nexus.workflow.protocol.RejectAfterClose.properties.liveness.rejectAfterCloseProgress,
    temporal.features.nexus.workflow.protocol.RejectAfterCloseWithDeadline.properties.liveness.rejectAfterCloseWithDeadlineProgress,
    temporal.features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainAndRouteProgress,
    temporal.features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainedReachesOwner,
    temporal.features.nexus.workflow.protocol.RetainAndRoute.properties.liveness.retainedWaitsWithoutRecovery,
    temporal.features.nexus.workflow.protocol.RetainAndRouteBoundedRetry.properties.liveness.retainAndRouteBoundedRetryProgress,
    temporal.features.nexus.workflow.protocol.RetainAndRouteBoundedRetry.properties.liveness.retainedReachesOwnerBoundedRetry,
    temporal.features.nexus.workflow.protocol.RetainAndRouteWithDeadline.properties.liveness.retainAndRouteWithDeadlineProgress
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
        List(temporal.features.activity.standalone.protocol.servedDispatchRuns),
        List(temporal.features.activity.standalone.protocol.unservedDispatchRuns),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.deliveryFair,
          temporal.features.nexus.workflow.protocol.recovery
        ),
        List(temporal.features.nexus.workflow.protocol.deliveryFair),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        ),
        List(
          temporal.features.nexus.workflow.protocol.deliveryFair,
          temporal.features.nexus.workflow.protocol.recovery
        ),
        List(
          temporal.features.nexus.workflow.protocol.reporting,
          temporal.features.nexus.workflow.protocol.deliveryFair
        )
      )
    )
  }
