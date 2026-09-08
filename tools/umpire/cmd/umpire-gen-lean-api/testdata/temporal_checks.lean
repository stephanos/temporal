import Temporal.API

open Temporal.Api.Workflowservice.V1

private def startOperation : Except Umpire.Operation.Error
    (Umpire.Operation.RpcDeclaration Temporal.API.rpcOwner
      StartWorkflowExecutionRequest StartWorkflowExecutionResponse String) :=
  (Temporal.API.bindUnary WorkflowService.startWorkflowExecution).map
    (fun binding => Umpire.Operation.rpc String binding)

#guard startOperation.toOption.map (·.identity.value) ==
  some "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution"

example : WorkflowService.startWorkflowExecution.fullName =
    "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution" := rfl

private def reference : Temporal.API.MethodReference WorkflowService.startWorkflowExecution :=
  by constructor

example : reference.schema.request.root =
    "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest" := rfl

open Umpire.Operation

private def forgedSchema : RpcSchema := {
  fullName := "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution"
  request := ⟨"temporal.api.workflowservice.v1.StartWorkflowExecutionRequest", []⟩
  response := ⟨"temporal.api.workflowservice.v1.StartWorkflowExecutionResponse", []⟩
  schemaInputs := []
  clientStreaming := false
  serverStreaming := false
}

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example : Except Error (Declaration .unaryRpc Bool Nat Empty) :=
  (checkRpc Bool Nat forgedSchema forgedSchema).map (rpc Empty)

namespace Forged.Temporal.API

private def rpcOwner : RpcOwner where
  Witness _ _ := Unit
  schema _ := forgedSchema

private def declaration (Request Response : Type) :=
  (checkRpc rpcOwner (Request := Request) (Response := Response) () forgedSchema).map (rpc Empty)

#guard (declaration Bool Nat).toOption.map (·.identity.value) == some forgedSchema.fullName

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example : Except Error (RpcDeclaration _root_.Temporal.API.rpcOwner Bool Nat Empty) :=
  declaration Bool Nat

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example : Except Error (RpcDeclaration _root_.Temporal.API.rpcOwner
    StartWorkflowExecutionRequest StartWorkflowExecutionResponse Empty) :=
  declaration StartWorkflowExecutionRequest StartWorkflowExecutionResponse

end Forged.Temporal.API

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example : Temporal.API.rpcOwner.Witness Bool Nat :=
  ⟨WorkflowService.startWorkflowExecution, reference⟩

example (declaration : RpcDeclaration Temporal.API.rpcOwner Request Response Failure) :
    declaration.schema = declaration.reference.2.schema := declaration.schema_eq

example (declaration : RpcDeclaration Temporal.API.rpcOwner Request Response Failure) :
    declaration.schema.fullName = declaration.reference.1.fullName := by
  rw [declaration.schema_eq]
  exact declaration.reference.2.fullName_eq
