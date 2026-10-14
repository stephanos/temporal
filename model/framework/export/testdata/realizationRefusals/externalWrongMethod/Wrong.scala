package fixture.externalwrongmethod

import framework.realize.*
import temporal.realize.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID

val failed = rpc(workflowService, METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID) {}
val cleanup = command(failed, regardless = true)
val crossed = ActivityExternalSettlement.Scheduled(failed, failed, failed, cleanup)
