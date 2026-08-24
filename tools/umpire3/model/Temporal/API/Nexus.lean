import Temporal.API.Generated.Wire
import Temporal.API.Interpretation
import Temporal.Families.NexusCancellation.Feature

namespace Umpire3.Temporal.API.Nexus

open Umpire3.Temporal.API

inductive InterpretationError where
  | missingNamespace
  | missingOperation
  | missingEndpoint
  | missingRequestID
  | missingService
  | missingAction
  | invalidScheduleToCloseTimeout
  | invalidScheduleToStartTimeout
  | invalidStartToCloseTimeout
  deriving DecidableEq, Repr

structure CancelCommand where
  namespaceName : String
  operationId : String
  runId : String
  requestId : String
  reason : String
  deriving DecidableEq, Repr

structure StartCommand where
  namespaceName : RequiredText
  operationId : RequiredText
  endpoint : RequiredText
  service : RequiredText
  operation : RequiredText
  requestId : RequiredText
  scheduleToCloseTimeout : Option Generated.Duration
  scheduleToStartTimeout : Option Generated.Duration
  startToCloseTimeout : Option Generated.Duration
  hasSensitiveInput : Bool

def durationValid (duration : Generated.Duration) : Bool :=
  decide (duration.seconds ≥ 0 ∧ duration.nanos ≥ 0)

def optionalDurationValid : Option Generated.Duration → Bool
  | none => true
  | some duration => durationValid duration

private def parseStart (request : Generated.StartNexusOperationExecutionRequest) :
    Except InterpretationError StartCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some operationId := RequiredText.fromString request.operationId
    | throw .missingOperation
  let some endpoint := RequiredText.fromString request.endpoint
    | throw .missingEndpoint
  let some service := RequiredText.fromString request.service
    | throw .missingService
  let some operation := RequiredText.fromString request.operation
    | throw .missingAction
  let some requestId := RequiredText.fromString request.requestId
    | throw .missingRequestID
  unless optionalDurationValid request.scheduleToCloseTimeout do
    throw .invalidScheduleToCloseTimeout
  unless optionalDurationValid request.scheduleToStartTimeout do
    throw .invalidScheduleToStartTimeout
  unless optionalDurationValid request.startToCloseTimeout do
    throw .invalidStartToCloseTimeout
  return {
    namespaceName
    operationId
    endpoint
    service
    operation
    requestId
    scheduleToCloseTimeout := request.scheduleToCloseTimeout
    scheduleToStartTimeout := request.scheduleToStartTimeout
    startToCloseTimeout := request.startToCloseTimeout
    hasSensitiveInput := request.input.isSome
  }

def interpretStart (request : Generated.StartNexusOperationExecutionRequest) :
    Interpretation StartCommand InterpretationError :=
  match parseStart request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

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
    Option Temporal.Feature.NexusCancellationFencing.Action :=
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

theorem accepted_start_has_operation_identity {request command}
    (_accepted : interpretStart request = .accepted command) :
    command.operationId.value ≠ "" := command.operationId.present

end Umpire3.Temporal.API.Nexus
