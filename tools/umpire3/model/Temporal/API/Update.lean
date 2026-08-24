import Temporal.API.Generated.Wire
import Temporal.API.Interpretation

namespace Umpire3.Temporal.API.Update

open Umpire3.Temporal.API

inductive InterpretationError where
  | missingNamespace
  | missingWorkflow
  | missingRequest
  | missingMetadata
  | missingUpdateID
  | missingUpdateName
  deriving DecidableEq, Repr

structure UpdateCommand where
  namespaceName : RequiredText
  workflowId : RequiredText
  runId : String
  updateId : RequiredText
  updateName : RequiredText
  hasSensitiveArguments : Bool

private def parse (request : Generated.UpdateWorkflowExecutionRequest) :
    Except InterpretationError UpdateCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some execution := request.workflowExecution
    | throw .missingWorkflow
  let some workflowId := RequiredText.fromString execution.workflowId
    | throw .missingWorkflow
  let some updateRequest := request.request
    | throw .missingRequest
  let some metadata := updateRequest.metadata
    | throw .missingMetadata
  let some updateId := RequiredText.fromString metadata.updateId
    | throw .missingUpdateID
  let some input := updateRequest.input
    | throw .missingUpdateName
  let some updateName := RequiredText.fromString input.name
    | throw .missingUpdateName
  return {
    namespaceName
    workflowId
    runId := execution.runId
    updateId
    updateName
    hasSensitiveArguments := input.args.isSome
  }

def interpret (request : Generated.UpdateWorkflowExecutionRequest) :
    Interpretation UpdateCommand InterpretationError :=
  match parse request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

theorem accepted_has_update_identity {request command}
    (_accepted : interpret request = .accepted command) :
    command.updateId.value ≠ "" ∧ command.workflowId.value ≠ "" :=
  ⟨command.updateId.present, command.workflowId.present⟩

end Umpire3.Temporal.API.Update
