import Temporal.API.Generated.Nexus
import Temporal.Product.Nexus

namespace Umpire3.Temporal.API.Nexus

inductive InterpretationError where
  | missingNamespace
  | missingOperation
  | missingRequestID
  deriving DecidableEq, Repr

structure CancelCommand where
  namespaceName : String
  operationId : String
  runId : String
  requestId : String
  reason : String
  deriving DecidableEq, Repr

def CancelCommand.Valid (command : CancelCommand) : Prop :=
  command.namespaceName ≠ "" ∧ command.operationId ≠ "" ∧ command.requestId ≠ ""

def interpretCancel (request : Generated.RequestCancelNexusOperationExecutionRequest) :
    Except InterpretationError CancelCommand :=
  if request.namespaceName = "" then .error .missingNamespace
  else if request.operationId = "" then .error .missingOperation
  else if request.requestId = "" then .error .missingRequestID
  else .ok {
    namespaceName := request.namespaceName
    operationId := request.operationId
    runId := request.runId
    requestId := request.requestId
    reason := request.reason
  }

def semanticEffect (request : Generated.RequestCancelNexusOperationExecutionRequest) :
    Option Temporal.Product.Nexus.Command :=
  match interpretCancel request with
  | .ok _ => some .acceptCancellation
  | .error _ => none

theorem accepted_is_valid {request command} (accepted : interpretCancel request = .ok command) :
    command.Valid := by
  simp only [interpretCancel] at accepted
  split at accepted
  · simp at accepted
  · rename_i namespacePresent
    split at accepted
    · simp at accepted
    · rename_i operationPresent
      split at accepted
      · simp at accepted
      · rename_i requestPresent
        simp at accepted
        cases accepted
        exact ⟨namespacePresent, operationPresent, requestPresent⟩

theorem rejected_has_no_effect {request failure}
    (rejected : interpretCancel request = .error failure) : semanticEffect request = none := by
  simp [semanticEffect, rejected]

end Umpire3.Temporal.API.Nexus
