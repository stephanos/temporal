//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../build/model-scala.jar
//> using jar ../../../build/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
package fixture.retiredInvalid

import umpire.*
import umpire.realize.*

private def join(left: String, right: String): String = left + right

val schema = action("start", Party("caller"))
  .schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest")
val rpc = Instruction.Rpc(
  "endpoint",
  join("/temporal.api.workflowservice.v1.WorkflowService/", "StartActivityExecution"),
  Vector.empty
)
val poll = Instruction.Poll("evidence", "endpoint", Vector.empty, Operand.Run, 1)
val read = Recorded.Read("method", "path")
val single = Recorded.Single("method", "path")
val history = Recorded.History("nexus_operation_scheduled_event_attributes")
val observed = Observed("observed", "temporal.server.api.testpilot.v1.InstructionOutcome")
val assignment = Assignment("namespace", Operand.Run)
val response = ResponseRead("run_id", Cardinality.one, Vector.empty)
val evidenceField = EvidenceField("operation", "execution.workflow_id")
val proto = Proto("temporal.api.common.v1.Payload", ProtoField("data", ProtoValue.Text("x")))
val entry = ProtoEntry("encoding", ProtoValue.Utf8("json/plain"))
val path = Operand.Path(Operand.Projected, join("task_queue", ".name"))
val enumName = ProtoValue.EnumName(join("COMMAND_TYPE_", "SCHEDULE_NEXUS_OPERATION"))
