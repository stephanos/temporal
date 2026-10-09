// A realization object is typed by its machine (fn-133 R9): evidence of a fact of another machine,
// and a described status keyed by one, do not compile, each at its line.
package fixture.typedrealizations

import framework.realize.*
import temporal.realize.*
import temporal.features.activity.standalone.system.{ActivitySystem, AdmissionFact}
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus.*

private val calls = RequestBase(workflowService, "activity_id" -> run)
private val started = rpc(calls, METHOD_START_ACTIVITY_EXECUTION) {}

object Foreign extends Realizes(ActivitySystem):
  object controller extends Controller(everyCase(started))
  object evidence extends Evidences(answered(AdmissionFact.dispatchSent, started))

private val describedForeign = DescribedStatus(
  machine = ActivitySystem,
  calls = calls,
  method = METHOD_DESCRIBE_ACTIVITY_EXECUTION,
  info = Field(_.getInfo),
  operation = Field(_.activityId),
  status = Field(_.status)
)(AdmissionFact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED)
