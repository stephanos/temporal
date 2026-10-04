//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../gen/model-scala.jar
//> using jar ../../../gen/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
// R4: a realization refers to its declarations by value, so a reference to one that does not exist
// fails to compile at its source. Each form is written once against a declaration that exists, then
// misspelled; each misspelling is the one error on its line.
package fixture.referenceInvalid

import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION
import io.temporal.api.history.v1.HistoryEvent
import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.{AwaitCommand, AwaitLearned, Fault}
import temporal.realize.*

val startWorkflow = rpc(workflowService, METHOD_START_WORKFLOW_EXECUTION) {
  field(_.namespace) := workerNamespace
}
val start = action(Party())
val completionAuthority = Learned("completion-authority", LearnedKind.handle)
val historyEvent = Observed[HistoryEvent]("history-event")
val started = Evidence.history(
  "fixture.referenceInvalid.evidence.started",
  "started",
  "fixture.referenceInvalid.source.history",
  Recorded.history(Field[HistoryEvent, Long](_.eventId)),
  Field[HistoryEvent, Long](_.eventId),
  Commitment.reported
)
val startNexusOperation = Command("start-nexus-operation", AwaitLearned(completionAuthority.id))

val stepByValue = always(startWorkflow)
val stepMisspelled = always(startWorkfow)
val performedByValue = perform(start -> startWorkflow)
val performedMisspelled = perform(start -> startWorkfow)
val learnedByValue = AwaitLearned(completionAuthority.id)
val learnedMisspelled = AwaitLearned(completionAuthorty.id)
val observedByValue = Target.Observe(historyEvent.id)
val observedMisspelled = Target.Observe(historyEvnt.id)
val closesByValue = command(startWorkflow, closes = Vector(started))
val closesMisspelled = command(startWorkflow, closes = Vector(startd))
val commandByValue = AwaitCommand(startNexusOperation.id)
val commandMisspelled = AwaitCommand(startNexusOpration.id)
val roleByValue = Fault(handlerTaskQueue, FaultKind.workerStop)
val roleMisspelled = Fault(handlerTaskQeue, FaultKind.workerStop)
