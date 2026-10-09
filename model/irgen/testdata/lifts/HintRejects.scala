// API behavior hints the lifter must refuse at the line that declares them (fn-118.2): a method
// descriptor built at run time names a method, but no constant the gRPC generator wrote, so the
// lifter cannot read the method from the generated API. The lifter's tests lift them with the other
// rejected declarations and compare the diagnostics with expected/rejects.txt.
package fixture.hintrejects

import framework.realize.*
import temporal.realize.*
import temporal.features.activity.standalone.activity
import temporal.features.activity.standalone.system.ActivitySystem as activitySystem
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
private def realizing(hint: Visibility) = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(workflowService, taskQueue),
  scripts = Vector(controller()),
  evidence = Vector.empty,
  behavior = ApiBehavior(visibility = Vector(hint), causes = Vector.empty)
)

private val METHOD_DESCRIBE_NOTHING =
  METHOD_DESCRIBE_ACTIVITY_EXECUTION.toBuilder().setFullMethodName("fixture.None/Nothing").build()

// A write no generated constant declares.
val builtWrite: Realization =
  realizing(METHOD_DESCRIBE_NOTHING.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce))

// A read no generated constant declares.
val builtRead: Realization =
  realizing(METHOD_START_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_NOTHING, Visible.atOnce))
